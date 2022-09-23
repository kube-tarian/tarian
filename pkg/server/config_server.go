package server

import (
	"context"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc/codes"

	"github.com/scylladb/go-set/strset"
)

type ConfigServer struct {
	tarianpb.UnimplementedConfigServer
	constraintStore store.ConstraintStore
	actionStore     store.ActionStore
}

func NewConfigServer(constraintStore store.ConstraintStore, actionStore store.ActionStore) *ConfigServer {
	return &ConfigServer{constraintStore: constraintStore, actionStore: actionStore}
}

func (cs *ConfigServer) GetConstraints(ctx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Debugw("Received get config RPC", "namespace", request.GetNamespace(), "labels", request.GetLabels())

	var constraints []*tarianpb.Constraint
	var err error

	if request.GetNamespace() == "" {
		constraints, err = cs.constraintStore.GetAll()
	} else {
		constraints, err = cs.constraintStore.FindByNamespace(request.GetNamespace())
	}

	matchedConstraints := []*tarianpb.Constraint{}

	// filter matchLabels
	// TODO: add unit test for this
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

			if requestLabelSet.IsSubset(constraintSelectorLabelSet) {
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
	if request.GetConstraint() == nil {
		return nil, status.Error(codes.InvalidArgument, "required constraint is empty")
	}

	if request.GetConstraint().GetNamespace() == "" {
		return nil, status.Error(codes.InvalidArgument, "required field is empty: namespace")
	}

	if request.GetConstraint().GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "required field is empty: name")
	}

	exist, err := cs.constraintStore.NamespaceAndNameExist(request.GetConstraint().GetNamespace(), request.GetConstraint().GetName())
	if err != nil {
		logger.Errorw("error while handling add constraint RPC", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if exist {
		return nil, status.Error(codes.InvalidArgument, "namespace and name already exists")
	}

	err = cs.constraintStore.Add(request.GetConstraint())
	if err != nil {
		logger.Errorw("error while handling add constraint RPC", "err", err)
		return &tarianpb.AddConstraintResponse{Success: false}, nil
	}

	return &tarianpb.AddConstraintResponse{Success: true}, nil
}

func (cs *ConfigServer) RemoveConstraint(ctx context.Context, request *tarianpb.RemoveConstraintRequest) (*tarianpb.RemoveConstraintResponse, error) {
	if request.GetNamespace() == "" || request.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "required namespace or name is empty")
	}

	exist, err := cs.constraintStore.NamespaceAndNameExist(request.GetNamespace(), request.GetName())
	if err != nil {
		logger.Errorw("error while handling remove constraint RPC", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if !exist {
		return &tarianpb.RemoveConstraintResponse{Success: false}, status.Error(codes.NotFound, "Constraint not found")
	}

	err = cs.constraintStore.RemoveByNamespaceAndName(request.GetNamespace(), request.GetName())
	if err != nil {
		logger.Errorw("error while handling remove constraint RPC", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &tarianpb.RemoveConstraintResponse{Success: true}, nil
}

func (cs *ConfigServer) AddAction(ctx context.Context, request *tarianpb.AddActionRequest) (*tarianpb.AddActionResponse, error) {
	if request.GetAction() == nil {
		return nil, status.Error(codes.InvalidArgument, "required action is empty")
	}

	if request.GetAction().GetNamespace() == "" {
		return nil, status.Error(codes.InvalidArgument, "required field is empty: namespace")
	}

	if request.GetAction().GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "required field is empty: name")
	}

	logger.Infow("add action", "request", request)

	// exist, err := cs.actionStore.NamespaceAndNameExist(request.GetAction().GetNamespace(), request.GetAction().GetName())
	// if err != nil {
	// 	logger.Errorw("error while handling add action RPC", "err", err)
	// 	return nil, status.Error(codes.Internal, "internal server error")
	// }

	// if exist {
	// 	return nil, status.Error(codes.InvalidArgument, "namespace and name already exists")
	// }

	err := cs.actionStore.Add(request.GetAction())
	if err != nil {
		logger.Errorw("error while handling add action RPC", "err", err)
		return &tarianpb.AddActionResponse{Success: false}, nil
	}

	return &tarianpb.AddActionResponse{Success: true}, nil
}

func (cs *ConfigServer) GetActions(ctx context.Context, request *tarianpb.GetActionsRequest) (*tarianpb.GetActionsResponse, error) {
	logger.Debugw("Received get actions RPC", "namespace", request.GetNamespace(), "labels", request.GetLabels())

	var actions []*tarianpb.Action
	var err error

	if request.GetNamespace() == "" {
		actions, err = cs.actionStore.GetAll()
	} else {
		actions, err = cs.actionStore.FindByNamespace(request.GetNamespace())
	}

	matchedActions := []*tarianpb.Action{}

	// filter matchLabels
	// TODO: add unit test for this
	if request.GetLabels() != nil {
		requestLabelSet := strset.New()
		for _, l := range request.GetLabels() {
			requestLabelSet.Add(l.GetKey() + "=" + l.GetValue())
		}

		for _, action := range actions {
			if action.GetSelector() == nil || action.GetSelector().GetMatchLabels() == nil {
				continue
			}

			actionSelectorLabelSet := strset.New()
			for _, l := range action.GetSelector().GetMatchLabels() {
				actionSelectorLabelSet.Add(l.GetKey() + "=" + l.GetValue())
			}

			if requestLabelSet.IsSubset(actionSelectorLabelSet) {
				matchedActions = append(matchedActions, action)
			}
		}
	} else {
		matchedActions = actions
	}

	if err != nil {
		logger.Errorw("error while handling get actions RPC", "error", err)
	}

	return &tarianpb.GetActionsResponse{
		Actions: matchedActions,
	}, nil
}

func (cs *ConfigServer) RemoveAction(ctx context.Context, request *tarianpb.RemoveActionRequest) (*tarianpb.RemoveActionResponse, error) {
	if request.GetNamespace() == "" || request.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "required namespace or name is empty")
	}

	exist, err := cs.actionStore.NamespaceAndNameExist(request.GetNamespace(), request.GetName())
	if err != nil {
		logger.Errorw("error while handling remove action RPC", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if !exist {
		return &tarianpb.RemoveActionResponse{Success: false}, status.Error(codes.NotFound, "Action not found")
	}

	err = cs.actionStore.RemoveByNamespaceAndName(request.GetNamespace(), request.GetName())
	if err != nil {
		logger.Errorw("error while handling remove action RPC", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &tarianpb.RemoveActionResponse{Success: true}, nil
}
