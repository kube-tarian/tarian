package cmd

import (
	"os"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarian-pod-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/podagent"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCommandRun(t *testing.T) {
	logger := log.GetLogger()
	file := createTestLabelsFile(t, true)
	defer os.Remove(file.Name())
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		podLabelsFile string

		podAgent podagent.Agent
	}{
		{
			name:        "Run Register mode Successfully",
			podAgent:    podagent.NewFakePodAgent(logger),
			expectedLog: "tarian-pod-agent is running in register mode",
		},
		{
			name:          "Run Register mode with a empty Labels File",
			podAgent:      podagent.NewFakePodAgent(logger),
			expectedErr:   "no labels found in file",
			podLabelsFile: file.Name(),
		},
		{
			name:          "Run register mode with Invalid Labels File",
			expectedErr:   "failed to open file",
			podLabelsFile: "nonexistent_labels.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &registerCommand{
				logger:        logger,
				podLabelsFile: tt.podLabelsFile,
				podAgent:      tt.podAgent,
			}

			logOutput := []byte{}
			cmd.logger.Out = &utesting.LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()

			err := cmd.runRegisterCommand(nil, nil)

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				if !assert.NoError(t, err) {
					assert.FailNow(t, "error not expected")
				}
			}

			if tt.expectedLog != "" {
				assert.Contains(t, utesting.CleanLog(string(logOutput)), utesting.CleanLog(tt.expectedLog))
			}
		})
	}
}

func TestNewRegisterModeCommand(t *testing.T) {
	globalFlags := &flags.GlobalFlags{}
	cmd := newRegisterCommand(globalFlags)

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)

	assert.Equal(t, "register", cmd.Use)
	assert.Equal(t, "Register the pod to the Tarian server", cmd.Short)

	testFlags(t, cmd)
}

func testFlags(t *testing.T, cmd *cobra.Command) {
	assert.Equal(t, defaultClusterAgentHost, cmd.Flags().Lookup("host").DefValue)
	err := cmd.Flags().Set("host", "localhost")
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cmd.Flags().Lookup("host").Value.String())

	assert.Equal(t, defaultClusterAgentPort, cmd.Flags().Lookup("port").DefValue)
	err = cmd.Flags().Set("port", "50053")
	assert.NoError(t, err)
	assert.Equal(t, "50053", cmd.Flags().Lookup("port").Value.String())

	assert.Equal(t, "", cmd.Flags().Lookup("pod-labels-file").DefValue)
	err = cmd.Flags().Set("pod-labels-file", "test_labels.txt")
	assert.NoError(t, err)
	assert.Equal(t, "test_labels.txt", cmd.Flags().Lookup("pod-labels-file").Value.String())

	assert.Equal(t, "", cmd.Flags().Lookup("pod-name").DefValue)
	err = cmd.Flags().Set("pod-name", "test_pod")
	assert.NoError(t, err)
	assert.Equal(t, "test_pod", cmd.Flags().Lookup("pod-name").Value.String())

	assert.Equal(t, "", cmd.Flags().Lookup("pod-uid").DefValue)
	err = cmd.Flags().Set("pod-uid", "test_pod_uid")
	assert.NoError(t, err)
	assert.Equal(t, "test_pod_uid", cmd.Flags().Lookup("pod-uid").Value.String())

	assert.Equal(t, "", cmd.Flags().Lookup("namespace").DefValue)
	err = cmd.Flags().Set("namespace", "test_namespace")
	assert.NoError(t, err)
	assert.Equal(t, "test_namespace", cmd.Flags().Lookup("namespace").Value.String())

	assert.Equal(t, "3s", cmd.Flags().Lookup("file-validation-interval").DefValue)
	err = cmd.Flags().Set("file-validation-interval", "5s")
	assert.NoError(t, err)
	assert.Equal(t, "5s", cmd.Flags().Lookup("file-validation-interval").Value.String())
}
