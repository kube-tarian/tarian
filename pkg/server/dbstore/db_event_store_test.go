package dbstore

import (
	"testing"
	"time"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbEventStoreGetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
	columns := []string{"id", "type", "server_timestamp", "client_timestamp", "targets"}

	timeNow := time.Now()
	timeNow2 := time.Now().Add(1 * time.Hour)

	pgxRows := pgxpoolmock.NewRows(columns).
		AddRow(1, "violation", timeNow, timeNow.Add(1*time.Second), "[{\"pod\":{\"namespace\":\"default\"}}]").
		AddRow(2, "violation", timeNow2, timeNow2.Add(1*time.Second), "[{\"pod\":{\"namespace\":\"monitoring\"}}]").
		ToPgxRows()
	mockPool.EXPECT().Query(gomock.Any(), "SELECT * FROM events", gomock.Any()).Return(pgxRows, nil)

	s := DbEventStore{pool: mockPool}
	events, err := s.GetAll()

	if err != nil {
		t.Error(err)
	}

	require.Nil(t, err)
	require.Len(t, events, 2)

	event := events[0]
	assert.Equal(t, "violation", event.GetType())
	assert.Equal(t, "default", event.GetTargets()[0].GetPod().GetNamespace())

	event2 := events[1]
	assert.Equal(t, "violation", event2.GetType())
	assert.Equal(t, "monitoring", event2.GetTargets()[0].GetPod().GetNamespace())
}
