package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestTarianctlRootCommand(t *testing.T) {
	t.Run("TestCommandVersion", func(t *testing.T) {
		stdout := new(bytes.Buffer)

		err := runRootCommand(stdout, []string{"version"})
		if assert.NoError(t, err) {
			out, _ := io.ReadAll(stdout)
			assert.Contains(t, string(out), "tarianctl version:")
		}
	})

	t.Run("TestInvalidSubcommand", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		err := runRootCommand(stdout, []string{"invalidStderr-subcommand"})
		assert.EqualError(t, err, `unknown command "invalidStderr-subcommand" for "tarianctl"`)
	})
}

func runRootCommand(output *bytes.Buffer, args []string) error {
	logger := log.GetLogger()
	logger.SetOutput(output)
	rootCmd := buildRootCommand(logger)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}
