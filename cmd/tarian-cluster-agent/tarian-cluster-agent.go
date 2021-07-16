package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

const (
	port = ":50052"
)

type server struct {
	tarianpb.UnimplementedConfigServer
}

var grpcConn *grpc.ClientConn

func (s *server) GetConfig(context.Context, *empty.Empty) (*tarianpb.GetConfigResponse, error) {
	log.Printf("Received get config RPC")

	c := tarianpb.NewConfigClient(grpcConn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.GetConfig(ctx, &empty.Empty{})

	return r, err
}

func main() {
	fmt.Println("tarian-cluster-agent")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	tarianpb.RegisterConfigServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())

	grpcConn, err = grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer grpcConn.Close()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
