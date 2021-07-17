package server

import (
	"context"
	"log"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
)

type Server struct {
	tarianpb.UnimplementedConfigServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) GetConfig(context.Context, *empty.Empty) (*tarianpb.GetConfigResponse, error) {
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
