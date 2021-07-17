package store

import "github.com/devopstoday11/tarian/pkg/tarianpb"

type ConstraintStore interface {
	GetAll() ([]*tarianpb.Constraint, error)
	FindByNamespace(namespace string) ([]*tarianpb.Constraint, error)
}

type MemoryConstraintStore struct {
	data map[string][]*tarianpb.Constraint
}

func NewMemoryConstraintStore() *MemoryConstraintStore {
	m := &MemoryConstraintStore{data: make(map[string][]*tarianpb.Constraint)}
	exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}

	allowedProcessRegex := "nginx"
	exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}

	m.data["default"] = append(m.data["default"], &exampleConstraint)

	return m
}

func (m *MemoryConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	allConstraints := []*tarianpb.Constraint{}

	for _, nsConstraints := range m.data {
		allConstraints = append(allConstraints, nsConstraints...)
	}

	return allConstraints, nil
}

func (m *MemoryConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	allConstraints := []*tarianpb.Constraint{}
	allConstraints = append(allConstraints, m.data[namespace]...)

	return allConstraints, nil
}
