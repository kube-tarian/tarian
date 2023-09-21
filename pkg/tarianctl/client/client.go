package client

import (
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

// NewConfigClient creates a new ConfigClient.
func NewConfigClient(serverAddress string, opts ...grpc.DialOption) (tarianpb.ConfigClient, error) {
	grpcConn, err := grpc.Dial(serverAddress, opts...)

	return tarianpb.NewConfigClient(grpcConn), err
}

// NewEventClient creates a new EventClient.
func NewEventClient(serverAddress string, opts ...grpc.DialOption) (tarianpb.EventClient, error) {
	grpcConn, err := grpc.Dial(serverAddress, opts...)

	return tarianpb.NewEventClient(grpcConn), err
}
