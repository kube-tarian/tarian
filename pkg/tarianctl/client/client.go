package client

import (
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

func NewConfigClient(serverAddress string, opts ...grpc.DialOption) (tarianpb.ConfigClient, error) {
	grpcConn, err := grpc.Dial(serverAddress, opts...)

	return tarianpb.NewConfigClient(grpcConn), err
}

func NewEventClient(serverAddress string, opts ...grpc.DialOption) (tarianpb.EventClient, error) {
	grpcConn, err := grpc.Dial(serverAddress, opts...)

	return tarianpb.NewEventClient(grpcConn), err
}
