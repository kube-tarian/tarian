package memorystore

import "github.com/devopstoday11/tarian/pkg/tarianpb"

// MemoryConstraintStore implements ConstraintStore
type MemoryConstraintStore struct {
	data map[string][]*tarianpb.Constraint
}

func NewMemoryConstraintStore() *MemoryConstraintStore {
	m := &MemoryConstraintStore{data: make(map[string][]*tarianpb.Constraint)}

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

func (m *MemoryConstraintStore) Add(constraint *tarianpb.Constraint) error {
	m.data[constraint.GetNamespace()] = append(m.data[constraint.GetNamespace()], constraint)

	return nil
}
