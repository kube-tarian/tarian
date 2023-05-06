package server

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sethvargo/go-retry"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

type Server struct {
	GrpcServer      *grpc.Server
	EventServer     *EventServer
	IngestionWorker *IngestionWorker
	ConfigServer    *ConfigServer
	AlertDispatcher *AlertDispatcher

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	eventStore store.EventStore
}

func NewServer(storeSet store.StoreSet, certFile string, privateKeyFile string, natsURL string) (*Server, error) {
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
		jetstreamQueue, err := protoqueue.NewJetstream(natsURL, "tarian-server-event-ingestion")

		if err == nil {
			queuePublisher = jetstreamQueue
			queueSubscriber = jetstreamQueue
		} else {
			logger.Errorw("failed to create Jetstream queue", "err", err)
		}

		ctx := context.Background()
		backoffConnect := retry.NewConstant(5 * time.Second)
		backoffConnect = retry.WithCappedDuration(1*time.Minute, backoffConnect)
		err = retry.Do(ctx, backoffConnect, func(ctx context.Context) error {
			err = jetstreamQueue.Connect()
			if err != nil {
				logger.Warnw("failed to connect to NATS, retrying...", "err", err)
				return retry.RetryableError(err)
			}

			return nil
		})

		if err != nil {
			logger.Errorw("failed to connect to NATS, giving up after retrying", "err", err)
			return nil, err
		}

		backoffInit := retry.NewConstant(5 * time.Second)
		backoffInit = retry.WithCappedDuration(1*time.Minute, backoffInit)
		err = retry.Do(ctx, backoffInit, func(ctx context.Context) error {
			err = jetstreamQueue.Init()
			if err != nil {
				logger.Warnw("failed to init stream, retrying...", "err", err)
				return retry.RetryableError(err)
			}

			return nil
		})
		if err != nil {
			logger.Errorw("failed to init stream and subscription, giving up after retrying", "err", err)
			return nil, err
		}
	}

	configServer := NewConfigServer(storeSet.ConstraintStore, storeSet.ActionStore)
	eventServer := NewEventServer(storeSet.EventStore, queuePublisher)
	ingestionWorker := NewIngestionWorker(storeSet.EventStore, queueSubscriber)

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
	}

	return server, nil
}

func (s *Server) Start(grpcListenAddress string) error {
	listener, err := net.Listen("tcp", grpcListenAddress)
	if err != nil {
		logger.Errorw("failed to listen", "err", err)
		return err
	}

	logger.Infow("tarian-server is listening at", "address", listener.Addr())

	go s.IngestionWorker.Start()

	if err := s.GrpcServer.Serve(listener); err != nil {
		logger.Errorw("failed to serve", "err", err)
		return err
	}

	return nil
}

func (s *Server) WithAlertDispatcher(alertManagerAddress *url.URL, alertEvaluationInterval time.Duration) *Server {
	s.AlertDispatcher = NewAlertDispatcher(alertManagerAddress, alertEvaluationInterval)

	return s
}

func (s *Server) StartAlertDispatcher() {
	go s.AlertDispatcher.LoopSendAlerts(s.cancelCtx, s.eventStore)
}

func (s *Server) Stop() {
	s.GrpcServer.GracefulStop()
	s.cancelFunc()
}
