package dgraph

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore/fakestore"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySchema(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		dgraphClient dgraphstore.Client
	}{
		{
			name:        "Use real dgraph client",
			expectedErr: "code = DeadlineExceeded desc = context deadline exceeded",
		},
		{
			name:         "Apply schema successfully",
			dgraphClient: fakestore.NewFakeClient(),
			expectedLog:  "successfully applied schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &applySchemaCommand{
				logger:       log.GetLogger(),
				dgraphClient: tt.dgraphClient,
			}

			logOutput := []byte{}
			cmd.logger.Out = &utesting.LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()

			err := cmd.run(nil, nil)

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				if !assert.NoError(t, err) {
					assert.FailNow(t, "error not expected")
				}
			}

			if tt.expectedLog != "" {
				assert.Equal(t, utesting.CleanLog(tt.expectedLog), utesting.CleanLog(string(logOutput)))
			}
		})
	}
}

func TestNewDgraphCommand(t *testing.T) {
	cmd := NewDgraphCommand(&flags.GlobalFlags{})
	assert.NotNil(t, cmd)
	assert.Equal(t, "dgraph", cmd.Use)
	assert.Equal(t, "Command group related to Dgraph database", cmd.Short)
	assert.Len(t, cmd.Commands(), 1)
	assert.IsType(t, &cobra.Command{}, cmd.Commands()[0])
}

func TestNewApplySchemaCommand(t *testing.T) {
	cmd := newApplySchemaCommand(&flags.GlobalFlags{})
	assert.NotNil(t, cmd)

	assert.Equal(t, "apply-schema", cmd.Use)
	assert.Equal(t, "Apply the schema for Dgraph database", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	timeoutFlag := cmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)

	assert.Equal(t, "5m0s", cmd.Flags().Lookup("timeout").Value.String())

	err := cmd.Flags().Set("timeout", "10m")
	require.NoError(t, err)
	assert.Equal(t, "10m0s", cmd.Flags().Lookup("timeout").Value.String())
}
