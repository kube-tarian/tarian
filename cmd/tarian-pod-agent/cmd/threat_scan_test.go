package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarian-pod-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/podagent"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestThreatScanRunCommand(t *testing.T) {
	logger := log.GetLogger()
	file := createTestLabelsFile(t, true)
	defer os.Remove(file.Name())
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		podLabelsFile          string
		fileValidationInterval time.Duration

		podAgent podagent.Agent
	}{
		{
			name:        "Run Threat Scan Successfully",
			podAgent:    podagent.NewFakePodAgent(logger),
			expectedLog: "tarian-pod-agent is running in threat-scan mode SetPodName SetFileValidationInterval RunThreatScan tarian-pod-agent shutdown gracefully",
		},
		{
			name:          "Run Threat Scan with a empty Labels File",
			podAgent:      podagent.NewFakePodAgent(logger),
			expectedErr:   "no labels found in file",
			podLabelsFile: file.Name(),
		},
		{
			name:          "Run Threat Scan with Invalid Labels File",
			expectedErr:   "failed to open file",
			podLabelsFile: "nonexistent_labels.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &threatScanCommand{
				logger:                 logger,
				podLabelsFile:          tt.podLabelsFile,
				podAgent:               tt.podAgent,
				fileValidationInterval: tt.fileValidationInterval,
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
				assert.Contains(t, utesting.CleanLog(string(logOutput)), utesting.CleanLog(tt.expectedLog))
			}
		})
	}
}

func TestNewThreatScanCommand(t *testing.T) {
	globalFlags := &flags.GlobalFlags{}
	cmd := newThreatScanCommand(globalFlags)

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)

	assert.Equal(t, "threat-scan", cmd.Use)
	assert.Equal(t, "Scan the container image for vulnerabilities", cmd.Short)
	testFlags(t, cmd)
}
