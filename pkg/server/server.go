package server

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sethvargo/go-retry"
)

// TarianServer represents the Tarian server, which includes gRPC server, event server, ingestion worker, config server, and alert dispatcher.
type TarianServer struct {
	GrpcServer      *grpc.Server
	EventServer     *EventServer
	IngestionWorker *IngestionWorker
	ConfigServer    *ConfigServer
	AlertDispatcher *AlertDispatcher

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	eventStore store.EventStore
	logger     *logrus.Logger
}

// NewServer creates a new Tarian server instance.
//
// Parameters:
// - logger: The logger to use for logging.
// - storeSet: A set of stores including constraint, action, and event stores.
// - certFile: The path to the server certificate file.
// - privateKeyFile: The path to the server private key file.
// - natsURL: The NATS server URL.
// - natsOptions: Options for configuring the NATS connection.
// - natsStreamConfig: Configuration for the NATS JetStream stream.
//
// Returns:
// - *TarianServer: A new instance of the Tarian server.
// - error: An error if any occurs during server creation.
func NewServer(logger *logrus.Logger, storeSet store.Set, certFile string, privateKeyFile string, natsURL string, natsOptions []nats.Option, natsStreamConfig nats.StreamConfig) (Server, error) {
	opts := []grpc.ServerOption{}
	if certFile != "" && privateKeyFile != "" {
		creds, _ := credentials.NewServerTLSFromFile(certFile, privateKeyFile)
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	var queuePublisher protoqueue.QueuePublisher
	var queueSubscriber protoqueue.QueueSubscriber

	if natsURL == "" {
		channelQueue := protoqueue.NewChannelQueue()
		queuePublisher = channelQueue
		queueSubscriber = channelQueue
	} else {
		jetstreamQueue, err := protoqueue.NewJetstream(logger, natsURL, natsOptions, natsStreamConfig.Name)

		if err == nil {
			queuePublisher = jetstreamQueue
			queueSubscriber = jetstreamQueue
		} else {
			logger.WithError(err).Error("failed to create Jetstream queue")
		}

		ctx := context.Background()
		backoffConnect := retry.NewConstant(5 * time.Second)
		backoffConnect = retry.WithCappedDuration(1*time.Minute, backoffConnect)
		err = retry.Do(ctx, backoffConnect, func(ctx context.Context) error {
			err = jetstreamQueue.Connect()
			if err != nil {
				logger.Errorf("failed to connect to NATS: %s", err)
				logger.Info("retrying to connect to NATS.......")
				logger.WithFields(logrus.Fields{
					"natsURL": natsURL,
					"error":   err,
				}).Warn("retrying to connect to NATS.......")
				err = fmt.Errorf("NewServer: %w", err)
				return retry.RetryableError(err)
			}

			return nil
		})

		if err != nil {
			logger.WithError(err).Error("failed to connect to NATS")
			return nil, err
		}

		backoffInit := retry.NewConstant(5 * time.Second)
		backoffInit = retry.WithCappedDuration(1*time.Minute, backoffInit)
		err = retry.Do(ctx, backoffInit, func(ctx context.Context) error {
			err = jetstreamQueue.Init(natsStreamConfig)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"natsURL": natsURL,
					"error":   err,
				}).Warn("retrying to connect to NATS.......")
				err = fmt.Errorf("NewServer: %w", err)
				return retry.RetryableError(err)
			}

			return nil
		})
		if err != nil {
			logger.WithError(err).Error("failed to init stream and subscription")
			err = fmt.Errorf("NewServer: %w", err)
			return nil, err
		}
	}

	configServer := NewConfigServer(logger, storeSet.ConstraintStore, storeSet.ActionStore)
	eventServer := NewEventServer(logger, storeSet.EventStore, queuePublisher)
	ingestionWorker := NewIngestionWorker(logger, storeSet.EventStore, queueSubscriber)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	server := &TarianServer{
		GrpcServer:      grpcServer,
		EventServer:     eventServer,
		IngestionWorker: ingestionWorker,
		ConfigServer:    configServer,
		cancelCtx:       cancelCtx,
		cancelFunc:      cancelFunc,
		eventStore:      storeSet.EventStore,
		logger:          logger,
	}

	return server, nil
}

// Start starts the Tarian server to listen on the given gRPC server address.
//
// Parameters:
// - grpcListenAddress: The gRPC server address to listen on.
//
// Returns:
// - error: An error if any occurs during server startup.
func (s *TarianServer) Start(grpcListenAddress string) error {
	listener, err := net.Listen("tcp", grpcListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.WithField("address", listener.Addr()).Info("tarian-server is listening")

	go s.IngestionWorker.Start()

	if err := s.GrpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// WithAlertDispatcher sets up the alert dispatcher for the Tarian server.
//
// Parameters:
// - alertManagerAddress: The URL of the Alertmanager.
// - alertEvaluationInterval: The interval for evaluating and sending alerts.
//
// Returns:
// - *TarianServer: The server with the alert dispatcher configured.
func (s *TarianServer) WithAlertDispatcher(alertManagerAddress *url.URL, alertEvaluationInterval time.Duration) Server {
	s.AlertDispatcher = NewAlertDispatcher(s.logger, alertManagerAddress, alertEvaluationInterval)

	return s
}

// StartAlertDispatcher starts the alert dispatcher for the Tarian server.
func (s *TarianServer) StartAlertDispatcher() {
	go s.AlertDispatcher.LoopSendAlerts(s.cancelCtx, s.eventStore)
}

// Stop stops the Tarian server gracefully.
func (s *TarianServer) Stop() {
	s.GrpcServer.GracefulStop()
	s.cancelFunc()
}

// GetGrpcServer returns the gRPC server instance.
func (s *TarianServer) GetGrpcServer() *grpc.Server {
	return s.GrpcServer
}
