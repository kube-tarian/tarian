package fakestore

import (
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

type fakeConstraintStore struct{}

func newFakeConstraintStore() store.ConstraintStore {
	return &fakeConstraintStore{}
}

// Add implements store.ConstraintStore.
func (*fakeConstraintStore) Add(constraint *tarianpb.Constraint) error {
	panic("unimplemented")
}

// FindByNamespace implements store.ConstraintStore.
func (*fakeConstraintStore) FindByNamespace(namespace string) ([]*tarianpb.Constraint, error) {
	panic("unimplemented")
}

// GetAll implements store.ConstraintStore.
func (*fakeConstraintStore) GetAll() ([]*tarianpb.Constraint, error) {
	panic("unimplemented")
}

// NamespaceAndNameExist implements store.ConstraintStore.
func (*fakeConstraintStore) NamespaceAndNameExist(namespace string, name string) (bool, error) {
	panic("unimplemented")
}

// RemoveByNamespaceAndName implements store.ConstraintStore.
func (*fakeConstraintStore) RemoveByNamespaceAndName(namespace string, name string) error {
	panic("unimplemented")
}
