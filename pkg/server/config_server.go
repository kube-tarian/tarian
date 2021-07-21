package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
)

type ConfigServer struct {
	tarianpb.UnimplementedConfigServer
	constraintStore store.ConstraintStore
}

func NewConfigServer(dsn string) (*ConfigServer, error) {
	dbStore, err := dbstore.NewDbConstraintStore(dsn)

	if err != nil {
		return nil, err
	}

	return &ConfigServer{constraintStore: dbStore}, nil
}

func (cs *ConfigServer) GetConstraints(ctx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Info("Received get config RPC")

	var constraints []*tarianpb.Constraint

	if request.GetNamespace() == "" {
		constraints, _ = cs.constraintStore.GetAll()
	} else {
		constraints, _ = cs.constraintStore.FindByNamespace(request.GetNamespace())
	}

	return &tarianpb.GetConstraintsResponse{
		Constraints: constraints,
	}, nil
}

func (cs *ConfigServer) AddConstraint(ctx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	cs.constraintStore.Add(request.GetConstraint())

	return &tarianpb.AddConstraintResponse{Success: true}, nil
}
