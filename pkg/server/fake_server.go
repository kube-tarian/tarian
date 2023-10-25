package server

import (
	"net/url"
	"time"

	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type fakeServer struct {
	logger *logrus.Logger
}

// NewFakeServer creates a new Tarian server instance.
func NewFakeServer(logger *logrus.Logger, storeSet store.Set, certFile string, privateKeyFile string, natsURL string, natsOptions []nats.Option, natsStreamConfig nats.StreamConfig) (Server, error) {
	return &fakeServer{
		logger: logger,
	}, nil
}

// Start implements Server.
func (f *fakeServer) Start(grpcListenAddress string) error {
	f.logger.Info("Starting Tarian server")

	return nil
}

// StartAlertDispatcher implements Server.
func (f *fakeServer) StartAlertDispatcher() {
	f.logger.Infof("Starting alert dispatcher")
}

// Stop implements Server.
func (f *fakeServer) Stop() {
	f.logger.Infof("Stopping Tarian server")
}

// WithAlertDispatcher implements Server.
func (f *fakeServer) WithAlertDispatcher(alertManagerAddress *url.URL, alertEvaluationInterval time.Duration) Server {
	return f
}

// GetGrpcServer implements Server.
func (*fakeServer) GetGrpcServer() *grpc.Server {
	panic("unimplemented")
}
