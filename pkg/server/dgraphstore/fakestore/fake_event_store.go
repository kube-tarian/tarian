package fakestore

import (
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

type fakeEventStore struct{}

func newFakeEventStore() store.EventStore {
	return &fakeEventStore{}
}

// Add implements store.EventStore.
func (*fakeEventStore) Add(event *tarianpb.Event) error {
	panic("unimplemented")
}

// FindByNamespace implements store.EventStore.
func (*fakeEventStore) FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error) {
	panic("unimplemented")
}

// FindWhereAlertNotSent implements store.EventStore.
func (*fakeEventStore) FindWhereAlertNotSent() ([]*tarianpb.Event, error) {
	panic("unimplemented")
}

// GetAll implements store.EventStore.
func (*fakeEventStore) GetAll(limit uint) ([]*tarianpb.Event, error) {
	panic("unimplemented")
}

// UpdateAlertSent implements store.EventStore.
func (*fakeEventStore) UpdateAlertSent(uid string) error {
	panic("unimplemented")
}
