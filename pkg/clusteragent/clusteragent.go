package clusteragent

import (
	"context"
	"log"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type Server struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
}

func NewServer(tarianServerAddress string) *Server {
	grpcConn, err := grpc.Dial(tarianServerAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	return &Server{grpcConn: grpcConn, configClient: tarianpb.NewConfigClient(grpcConn)}
}

func (s *Server) GetConfig(context.Context, *empty.Empty) (*tarianpb.GetConfigResponse, error) {
	log.Printf("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := s.configClient.GetConfig(ctx, &empty.Empty{})

	return r, err
}

func (s *Server) Close() {
	s.grpcConn.Close()
}
