package dbstore

import (
	"testing"

	"github.com/driftprogramming/pgxpoolmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var constraintColumns = []string{"id", "namespace", "name", "selector", "allowed_processes", "allowed_files"}

func TestGetAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
	pgxRows := pgxpoolmock.NewRows(constraintColumns).
		AddRow(1, "default", "nginx", `{"match_labels": [{"key": "app", "value": "nginx"}]}`, `[{"regex": "(.*)nginx(.*)"}]`, `[{"name": "/etc/nginx/nginx.conf", "sha256sum": "c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"}]`).
		AddRow(2, "default2", "worker", `{"match_labels": [{"key": "app", "value": "worker"}]}`, `[{"regex": "(.*)worker(.*)"}]`, `[{"name": "/etc/worker/worker.yaml", "sha256sum": "c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"}]`).
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
	assert.Equal(t, "nginx", constraint.GetName())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "nginx", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)nginx(.*)", constraint.GetAllowedProcesses()[0].GetRegex())
	assert.Equal(t, "/etc/nginx/nginx.conf", constraint.GetAllowedFiles()[0].GetName())

	constraint = constraints[1]
	assert.Equal(t, "default2", constraint.GetNamespace())
	assert.Equal(t, "worker", constraint.GetName())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "worker", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)worker(.*)", constraint.GetAllowedProcesses()[0].GetRegex())
	assert.Equal(t, "/etc/worker/worker.yaml", constraint.GetAllowedFiles()[0].GetName())
}

func TestFindByNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup
	mockPool := pgxpoolmock.NewMockPgxPool(ctrl)
	pgxRows := pgxpoolmock.NewRows(constraintColumns).
		AddRow(1, "default", "nginx", `{"match_labels": [{"key": "app", "value": "nginx"}]}`, `[{"regex": "(.*)nginx(.*)"}]`, `[{"name": "/etc/nginx/nginx.conf", "sha256sum": "c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b"}]`).
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
	assert.Equal(t, "nginx", constraint.GetName())
	assert.Equal(t, "app", constraint.GetSelector().GetMatchLabels()[0].GetKey())
	assert.Equal(t, "nginx", constraint.GetSelector().GetMatchLabels()[0].GetValue())
	assert.Equal(t, "(.*)nginx(.*)", constraint.GetAllowedProcesses()[0].GetRegex())
	assert.Equal(t, "/etc/nginx/nginx.conf", constraint.GetAllowedFiles()[0].GetName())
}
