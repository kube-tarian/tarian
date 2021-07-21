package store

import "github.com/devopstoday11/tarian/pkg/tarianpb"

// MemoryEventStore implements EventStore
type MemoryEventStore struct {
	data []*tarianpb.Event
}

func NewMemoryEventStore() *MemoryEventStore {
	m := &MemoryEventStore{}

	return m
}

func (m *MemoryEventStore) GetAll() ([]*tarianpb.Event, error) {
	return m.data, nil
}

func (m *MemoryEventStore) Add(event *tarianpb.Event) error {
	m.data = append(m.data, event)

	return nil
}
