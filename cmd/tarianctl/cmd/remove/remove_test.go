package remove

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewRemoveCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "No subcommand provided",
			args:        []string{},
			expectedErr: "requires at least 1 arg(s), only received 0",
		},
		{
			name:        "Valid subcommand provided constraint",
			args:        []string{"constraint"},
			expectedErr: "please specify the name(s) of the constraint to be removed",
		},
		{
			name:        "Valid subcommand provided action",
			args:        []string{"action"},
			expectedErr: `please specify the name(s) of the action to be removed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRemoveCommand(&flags.GlobalFlags{})

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
