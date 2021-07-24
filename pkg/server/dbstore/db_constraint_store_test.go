package dbstore

import (
	"testing"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
	columns := []string{"id", "namespace", "selector", "allowed_processes"}
	pgxRows := pgxpoolmock.NewRows(columns).
		AddRow(1, "default", `{"match_labels": [{"key": "app", "value": "nginx"}]}`, `[{"regex": "(.*)nginx(.*)"}]`).
		AddRow(2, "default2", `{"match_labels": [{"key": "app", "value": "worker"}]}`, `[{"regex": "(.*)worker(.*)"}]`).
		ToPgxRows()
	mockPool.EXPECT().Query(gomock.Any(), "SELECT * FROM constraints ORDER BY id ASC", gomock.Any()).Return(pgxRows, nil)

	s := DbConstraintStore{pool: mockPool}
	constraints, err := s.GetAll()

	if err != nil {
		t.Error(err)
	}

	require.Nil(t, err)
	require.Len(t, constraints, 2)

	constraint := constraints[0]
	assert.Equal(t, "default", constraint.GetNamespace())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "nginx", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)nginx(.*)", constraint.GetAllowedProcesses()[0].GetRegex())

	constraint = constraints[1]
	assert.Equal(t, "default2", constraint.GetNamespace())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "worker", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)worker(.*)", constraint.GetAllowedProcesses()[0].GetRegex())
}

func TestFindByNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
	columns := []string{"id", "namespace", "selector", "allowed_processes"}
	pgxRows := pgxpoolmock.NewRows(columns).
		AddRow(1, "default", `{"match_labels": [{"key": "app", "value": "nginx"}]}`, `[{"regex": "(.*)nginx(.*)"}]`).
		ToPgxRows()
	mockPool.EXPECT().Query(gomock.Any(), "SELECT * FROM constraints WHERE namespace = $1 ORDER BY id ASC", "default").Return(pgxRows, nil)

	s := DbConstraintStore{pool: mockPool}
	constraints, err := s.FindByNamespace("default")

	if err != nil {
		t.Error(err)
	}

	require.Nil(t, err)
	require.Len(t, constraints, 1)

	constraint := constraints[0]
	assert.Equal(t, "default", constraint.GetNamespace())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "nginx", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)nginx(.*)", constraint.GetAllowedProcesses()[0].GetRegex())
}
