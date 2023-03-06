package server

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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

	promRegistry   *prometheus.Registry
	grpcMetrics    *grpc_prometheus.ServerMetrics
	promHTTPServer *http.Server
}

func NewServer(storeSet store.StoreSet, certFile string, privateKeyFile string) (*Server, error) {
	opts := []grpc.ServerOption{}
	if certFile != "" && privateKeyFile != "" {
		creds, _ := credentials.NewServerTLSFromFile(certFile, privateKeyFile)
		opts = append(opts, grpc.Creds(creds))
	}

	reg := prometheus.NewRegistry()
	grpcMetrics := grpc_prometheus.NewServerMetrics()

	opts = append(opts, grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()))
	opts = append(opts, grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()))
	opts = append(opts, grpc.ChainStreamInterceptor(grpcMetrics.StreamServerInterceptor()))
	opts = append(opts, grpc.ChainUnaryInterceptor(grpcMetrics.UnaryServerInterceptor()))

	grpcServer := grpc.NewServer(opts...)

	configServer := NewConfigServer(storeSet.ConstraintStore, storeSet.ActionStore)
	eventServer := NewEventServer(storeSet.EventStore)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	grpc_prometheus.Register(grpcServer)
	grpcMetrics.InitializeMetrics(grpcServer)

	promHTTPServer := &http.Server{Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{})}

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	server := &Server{
		GrpcServer:     grpcServer,
		EventServer:    eventServer,
		ConfigServer:   configServer,
		cancelCtx:      cancelCtx,
		cancelFunc:     cancelFunc,
		eventStore:     storeSet.EventStore,
		promRegistry:   reg,
		grpcMetrics:    grpcMetrics,
		promHTTPServer: promHTTPServer,
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

func (s *Server) StartPromHTTPServer(address string) {
	s.promHTTPServer.Addr = address
	go func() {
		if err := s.promHTTPServer.ListenAndServe(); err != nil {
			logger.Fatalw("unable to start prometheus metrics http server", "address", address)
		}
	}()
}

func (s *Server) Stop() {
	shutdownTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.GrpcServer.GracefulStop()
	s.promHTTPServer.Shutdown(shutdownTimeout)
	s.cancelFunc()
}
