package server

import (
	"context"
	"log"

	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
)

type Server struct {
	tarianpb.UnimplementedConfigServer
	constraintStore store.ConstraintStore
}

func NewServer() *Server {
	return &Server{constraintStore: store.NewMemoryConstraintStore()}
}

func (s *Server) GetConstraints(context.Context, *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	log.Printf("Received get config RPC")

	constraints, _ := s.constraintStore.FindByNamespace("default")

	return &tarianpb.GetConstraintsResponse{
		Constraints: constraints,
	}, nil
}
