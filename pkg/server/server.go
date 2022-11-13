package server

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	ConfigServer    *ConfigServer
	AlertDispatcher *AlertDispatcher

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	eventStore store.EventStore
}

func NewServer(storeSet store.StoreSet, certFile string, privateKeyFile string) (*Server, error) {
	opts := []grpc.ServerOption{}
	if certFile != "" && privateKeyFile != "" {
		creds, _ := credentials.NewServerTLSFromFile(certFile, privateKeyFile)
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	configServer := NewConfigServer(storeSet.ConstraintStore, storeSet.ActionStore)
	eventServer := NewEventServer(storeSet.EventStore)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	server := &Server{
		GrpcServer:   grpcServer,
		EventServer:  eventServer,
		ConfigServer: configServer,
		cancelCtx:    cancelCtx,
		cancelFunc:   cancelFunc,
		eventStore:   storeSet.EventStore,
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
