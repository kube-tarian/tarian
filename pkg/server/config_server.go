package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"

	"github.com/scylladb/go-set/strset"
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
	var err error

	if request.GetNamespace() == "" {
		constraints, err = cs.constraintStore.GetAll()
	} else {
		constraints, err = cs.constraintStore.FindByNamespace(request.GetNamespace())
	}

	matchedConstraints := []*tarianpb.Constraint{}

	// filter matchLabels
	if request.GetLabels() != nil {
		requestLabelSet := strset.New()
		for _, l := range request.GetLabels() {
			requestLabelSet.Add(l.GetKey() + "=" + l.GetValue())
		}

		for _, constraint := range constraints {
			if constraint.GetSelector() == nil || constraint.GetSelector().GetMatchLabels() == nil {
				continue
			}

			constraintSelectorLabelSet := strset.New()
			for _, l := range constraint.GetSelector().GetMatchLabels() {
				constraintSelectorLabelSet.Add(l.GetKey() + "=" + l.GetValue())
			}

			if constraintSelectorLabelSet.IsSubset(requestLabelSet) {
				matchedConstraints = append(matchedConstraints, constraint)
			}
		}
	} else {
		matchedConstraints = constraints
	}

	if err != nil {
		logger.Errorw("error while handling get constraints RPC", "error", err)
	}

	return &tarianpb.GetConstraintsResponse{
		Constraints: matchedConstraints,
	}, nil
}

func (cs *ConfigServer) AddConstraint(ctx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	err := cs.constraintStore.Add(request.GetConstraint())
	if err != nil {
		logger.Errorw("error while handling add constraint RPC", "error", err)
		return &tarianpb.AddConstraintResponse{Success: false}, nil
	}

	return &tarianpb.AddConstraintResponse{Success: true}, nil
}
