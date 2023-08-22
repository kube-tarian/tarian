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

type Server struct {
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

func NewServer(logger *logrus.Logger, storeSet store.StoreSet, certFile string, privateKeyFile string, natsURL string, natsOptions []nats.Option, natsStreamConfig nats.StreamConfig) (*Server, error) {
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

	server := &Server{
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

func (s *Server) Start(grpcListenAddress string) error {
	listener, err := net.Listen("tcp", grpcListenAddress)
	if err != nil {
		return fmt.Errorf("server start: failed to listen: %w", err)
	}

	s.logger.WithField("address", listener.Addr()).Info("tarian-server is listening")

	go s.IngestionWorker.Start()

	if err := s.GrpcServer.Serve(listener); err != nil {
		return fmt.Errorf("server start: failed to serve: %w", err)
	}

	return nil
}

func (s *Server) WithAlertDispatcher(alertManagerAddress *url.URL, alertEvaluationInterval time.Duration) *Server {
	s.AlertDispatcher = NewAlertDispatcher(s.logger, alertManagerAddress, alertEvaluationInterval)

	return s
}

func (s *Server) StartAlertDispatcher() {
	go s.AlertDispatcher.LoopSendAlerts(s.cancelCtx, s.eventStore)
}

func (s *Server) Stop() {
	s.GrpcServer.GracefulStop()
	s.cancelFunc()
}
