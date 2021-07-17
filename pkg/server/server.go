package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

type Server struct {
	tarianpb.UnimplementedConfigServer
	constraintStore store.ConstraintStore
}

func NewServer() *Server {
	return &Server{constraintStore: store.NewMemoryConstraintStore()}
}

func (s *Server) GetConstraints(ctx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Info("Received get config RPC")

	var constraints []*tarianpb.Constraint

	if request.GetNamespace() == "" {
		constraints, _ = s.constraintStore.GetAll()
	} else {
		constraints, _ = s.constraintStore.FindByNamespace(request.GetNamespace())
	}

	return &tarianpb.GetConstraintsResponse{
		Constraints: constraints,
	}, nil
}

func (s *Server) AddConstraint(ctx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	s.constraintStore.Add(request.GetConstraint())

	return &tarianpb.AddConstraintResponse{Success: true}, nil
}
