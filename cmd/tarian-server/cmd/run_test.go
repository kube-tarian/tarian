package cmd

import (
	"testing"

	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/server"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore/fakestore"
	"github.com/kube-tarian/tarian/pkg/store"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestServerRunCommand(t *testing.T) {
	serv, err := server.NewFakeServer(log.GetLogger(), store.Set{}, "", "", "", nil, nats.StreamConfig{})
	assert.NoError(t, err)

	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		dgaClient           dgraphstore.Client
		server              server.Server
		alertManagerAddress string
	}{
		{
			name:        "TestRunCommand",
			dgaClient:   fakestore.NewFakeClient(),
			server:      serv,
			expectedLog: "Starting Tarian server",
		},
		{
			name:                "TestRunCommandWithAlertDispatcher",
			dgaClient:           fakestore.NewFakeClient(),
			server:              serv,
			alertManagerAddress: "http://localhost:9093",
			expectedLog:         "Starting alert dispatcher",
		},
		{
			name:        "TestRunCommandWithDgraphNil",
			server:      serv,
			expectedLog: "Created dgraphstore client",
		},
		// Add more tests here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &runCommand{
				logger:              log.GetLogger(),
				dgraphClient:        tt.dgaClient,
				server:              tt.server,
				host:                "localhost",
				port:                "50055",
				alertManagerAddress: tt.alertManagerAddress,
			}

			// Capture log output
			logOutput := []byte{}
			cmd.logger.Out = &LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()
			cmd.logger.SetLevel(logrus.DebugLevel)

			err := cmd.run(nil, nil)
			// Assert expected error, if any
			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				if !assert.NoError(t, err) {
					assert.FailNow(t, "error not expected")
				}
			}

			// Assert expected log output
			if tt.expectedLog != "" {
				assert.Contains(t, utesting.CleanLog(string(logOutput)), utesting.CleanLog(tt.expectedLog))
			}
		})
	}
}

// LogOutputWriter is a writer for log output.
type LogOutputWriter struct {
	// Output is the log output.
	Output *[]byte
}

// Write writes the log output.
func (w *LogOutputWriter) Write(p []byte) (n int, err error) {
	*w.Output = append(*w.Output, p...)
	return len(p), nil
}
