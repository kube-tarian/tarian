package clusteragent

import (
	"context"
	"log"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
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

func (s *Server) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	log.Printf("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := s.configClient.GetConstraints(ctx, request)

	return r, err
}

func (s *Server) Close() {
	s.grpcConn.Close()
}
