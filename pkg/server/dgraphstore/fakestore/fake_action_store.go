package fakestore

import (
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

type fakeActionStore struct{}

func newFakeActionStore() store.ActionStore {
	return &fakeActionStore{}
}

// Add implements store.ActionStore.
func (*fakeActionStore) Add(action *tarianpb.Action) error {
	panic("unimplemented")
}

// FindByNamespace implements store.ActionStore.
func (*fakeActionStore) FindByNamespace(namespace string) ([]*tarianpb.Action, error) {
	panic("unimplemented")
}

// GetAll implements store.ActionStore.
func (*fakeActionStore) GetAll() ([]*tarianpb.Action, error) {
	panic("unimplemented")
}

// NamespaceAndNameExist implements store.ActionStore.
func (*fakeActionStore) NamespaceAndNameExist(namespace string, name string) (bool, error) {
	panic("unimplemented")
}

// RemoveByNamespaceAndName implements store.ActionStore.
func (*fakeActionStore) RemoveByNamespaceAndName(namespace string, name string) error {
	panic("unimplemented")
}
