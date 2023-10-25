package fakestore

import (
	"context"

	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/store"
)

type fakeClient struct{}

// NewFakeClient creates a new fake Dgraph client.
func NewFakeClient() dgraphstore.Client {
	return &fakeClient{}
}

// ApplySchema implements Client.
func (f *fakeClient) ApplySchema(ctx context.Context) error {
	return nil
}

// NewDgraphActionStore implements Client.
func (f *fakeClient) NewDgraphActionStore() store.ActionStore {
	return newFakeActionStore()
}

// NewDgraphConstraintStore implements Client.
func (f *fakeClient) NewDgraphConstraintStore() store.ConstraintStore {
	return newFakeConstraintStore()
}

// NewDgraphEventStore implements Client.
func (f *fakeClient) NewDgraphEventStore() store.EventStore {
	return newFakeEventStore()
}
