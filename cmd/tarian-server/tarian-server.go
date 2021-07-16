package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

type server struct {
	tarianpb.UnimplementedConfigServer
}

func (s *server) GetConfig(context.Context, *empty.Empty) (*tarianpb.GetConfigResponse, error) {
	log.Printf("Received get config RPC")

	constraints := []*tarianpb.Constraint{}

	exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}

	allowedProcessRegex := "nginx"
	exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
	constraints = append(constraints, &exampleConstraint)

	return &tarianpb.GetConfigResponse{
		Config: &tarianpb.Config{
			Constraints: constraints,
		},
	}, nil
}

func main() {
	fmt.Println("tarian-server")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	tarianpb.RegisterConfigServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
