package server

import (
	"context"
	"log"

	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
)

type Server struct {
	tarianpb.UnimplementedConfigServer
	constraintStore store.ConstraintStore
}

func NewServer() *Server {
	return &Server{constraintStore: store.NewMemoryConstraintStore()}
}

func (s *Server) GetConfig(context.Context, *empty.Empty) (*tarianpb.GetConfigResponse, error) {
	log.Printf("Received get config RPC")

	constraints, _ := s.constraintStore.GetAll()

	return &tarianpb.GetConfigResponse{
		Config: &tarianpb.Config{
			Constraints: constraints,
		},
	}, nil
}
