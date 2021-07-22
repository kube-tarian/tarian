package memorystore

import (
	"testing"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEventStoreGetAll(t *testing.T) {
	store := NewMemoryEventStore()

	event1 := tarianpb.Event{Type: "violation"}
	event2 := tarianpb.Event{Type: "other_type"}

	store.Add(&event1)
	store.Add(&event2)

	events, _ := store.GetAll()
	require.Len(t, events, 2)

	assert.Equal(t, "violation", events[0].Type)
	assert.Equal(t, "other_type", events[1].Type)
}
