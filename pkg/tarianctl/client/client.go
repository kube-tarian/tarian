package client

import (
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

func NewConfigServer(serverAddress string) (tarianpb.ConfigClient, error) {
	grpcConn, err := grpc.Dial(serverAddress, grpc.WithInsecure())

	return tarianpb.NewConfigClient(grpcConn), err
}
