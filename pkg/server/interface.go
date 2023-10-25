package server

import (
	"net/url"
	"time"

	"google.golang.org/grpc"
)

// Server is the interface for the Tarian server.
type Server interface {
	Start(grpcListenAddress string) error
	WithAlertDispatcher(alertManagerAddress *url.URL, alertEvaluationInterval time.Duration) Server
	StartAlertDispatcher()
	Stop()
	GetGrpcServer() *grpc.Server
}
