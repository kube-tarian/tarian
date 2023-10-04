package get

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewGetCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "No resource specified",
			args:        []string{},
			expectedErr: "no resource specified, use `tarianctl get --help` for command usage",
		},
		{
			name:        "Valid subcommand provided constraints",
			args:        []string{"constraints"},
			expectedErr: "failed to connect to server",
		},
		{
			name:        "Valid subcommand provided actions",
			args:        []string{"actions"},
			expectedErr: "failed to build resolver",
		},
		{
			name:        "Valid subcommand provided events",
			args:        []string{"events"},
			expectedErr: "failed to build resolver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGetCommand(&flags.GlobalFlags{})

			assert.IsType(t, &cobra.Command{}, cmd)

			cmd.SetArgs(tt.args)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			err := cmd.Execute()

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
