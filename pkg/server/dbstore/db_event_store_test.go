package dbstore

import (
	"database/sql"
	"testing"
	"time"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	eventColumns = []string{"id", "uid", "type", "server_timestamp", "client_timestamp", "alert_sent_at", "targets"}
)

func TestDbEventStoreGetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)

	timeNow := time.Now()
	timeNow2 := time.Now().Add(1 * time.Hour)

	uid1 := uuid.NewV4()
	uid2 := uuid.NewV4()

	nullTime := sql.NullTime{}
	nullTime.Scan(timeNow.Add(2 * time.Second))

	pgxRows := pgxpoolmock.NewRows(eventColumns).
		AddRow(1, uid1.String(), "violation", timeNow, timeNow.Add(1*time.Second), nullTime, "[{\"pod\":{\"namespace\":\"default\"}}]").
		AddRow(2, uid2.String(), "violation", timeNow2, timeNow2.Add(1*time.Second), nullTime, "[{\"pod\":{\"namespace\":\"monitoring\"}}]").
		ToPgxRows()
	mockPool.EXPECT().Query(gomock.Any(), "SELECT * FROM events ORDER BY id ASC", gomock.Any()).Return(pgxRows, nil)

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

func TestDbEventStoreFindByNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)

	timeNow := time.Now()
	uid1 := uuid.NewV4()

	nullTime := sql.NullTime{}
	nullTime.Scan(timeNow.Add(2 * time.Second))

	pgxRows := pgxpoolmock.NewRows(eventColumns).
		AddRow(1, uid1.String(), "violation", timeNow, timeNow.Add(1*time.Second), nullTime, "[{\"pod\":{\"namespace\":\"monitoring\"}}]").
		ToPgxRows()
	mockPool.EXPECT().Query(gomock.Any(), "SELECT * FROM events WHERE namespace = $1 ORDER BY id ASC", gomock.Any()).Return(pgxRows, nil)

	s := DbEventStore{pool: mockPool}
	events, err := s.FindByNamespace("monitoring")

	if err != nil {
		t.Error(err)
	}

	require.Nil(t, err)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "violation", event.GetType())
	assert.Equal(t, "monitoring", event.GetTargets()[0].GetPod().GetNamespace())
}
