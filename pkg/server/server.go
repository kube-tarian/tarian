package server

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

type PostgresqlConfig struct {
	User     string `default:"postgres"`
	Password string `default:"tarian"`
	Name     string `default:"tarian"`
	Host     string `default:"localhost"`
	Port     string `default:"5432"`
	SslMode  string `default:"disable"`
}

func (p *PostgresqlConfig) GetDsn() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", p.User, p.Password, p.Host, p.Port, p.Name, p.SslMode)
}

type Server struct {
	GrpcServer      *grpc.Server
	EventServer     *EventServer
	ConfigServer    *ConfigServer
	AlertDispatcher *AlertDispatcher

	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	eventStore *dbstore.DbEventStore
}

func NewServer(dsn string) (*Server, error) {
	grpcServer := grpc.NewServer()

	configServer, err := NewConfigServer(dsn)
	if err != nil {
		logger.Errorw("failed to initiate config server", "err", err)
		return nil, err
	}

	eventServer, err := NewEventServer(dsn)
	if err != nil {
		logger.Errorw("failed to initiate event server", "err", err)
		return nil, err
	}

	eventStore, err := dbstore.NewDbEventStore(dsn)
	if err != nil {
		logger.Errorw("failed to initiate db store", "err", err)
		return nil, err
	}

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	server := &Server{
		GrpcServer:   grpcServer,
		EventServer:  eventServer,
		ConfigServer: configServer,
		cancelCtx:    cancelCtx,
		cancelFunc:   cancelFunc,
		eventStore:   eventStore,
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
