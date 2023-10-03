package add

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewAddCommand(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedSubcommand string
		expectedErr        string
	}{
		{
			name:               "No subcommand provided",
			args:               []string{},
			expectedSubcommand: "",
			expectedErr:        `tarianctl needs exactly one argument, use "tarianctl add --help" for command usage`,
		},
		{
			name:               "Valid subcommand provided constraint",
			args:               []string{"constraint"},
			expectedSubcommand: "constraint",
			expectedErr:        "failed to connect to server",
		},
		{
			name:               "Valid subcommand provided action",
			args:               []string{"action"},
			expectedSubcommand: "action",
			expectedErr:        `required flag(s) "action" not set`,
		},
		{
			name:               "Invalid subcommand provided",
			args:               []string{"invalid-subcommand"},
			expectedSubcommand: "",
			expectedErr:        `unknown command "invalid-subcommand" for "add"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewAddCommand(&flags.GlobalFlags{})

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
