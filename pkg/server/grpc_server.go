package server

import (
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

func NewGrpcServer(dsn string) (*grpc.Server, error) {
	grpcServer := grpc.NewServer()

	configServer, err := NewConfigServer(dsn)
	if err != nil {
		logger.Fatalw("failed to initiate config server", "err", err)
	}

	eventServer, err := NewEventServer(dsn)
	if err != nil {
		logger.Fatalw("failed to initiate event server", "err", err)
	}

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	return grpcServer, err
}
