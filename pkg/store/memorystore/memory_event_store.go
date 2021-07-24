package memorystore

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

func (m *MemoryEventStore) FindByNamespace(namespace string) ([]*tarianpb.Event, error) {
	namespacedEvents := []*tarianpb.Event{}
	for _, event := range m.data {
		for _, target := range event.GetTargets() {
			pod := target.GetPod()

			if pod != nil && pod.GetNamespace() == namespace {
				namespacedEvents = append(namespacedEvents, event)
				continue
			}
		}
	}

	return namespacedEvents, nil
}

func (m *MemoryEventStore) Add(event *tarianpb.Event) error {
	m.data = append(m.data, event)

	return nil
}
