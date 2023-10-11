package cmd

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/log"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/stretchr/testify/assert"
)

func TestClusterAgentCommandRun(t *testing.T) {
	logger := log.GetLogger()
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		host string
		port string

		serverAddress               string
		serverTLSEnabled            bool
		serverTLSCAFile             string
		serverTLSInsecureSkipVerify bool

		enableAddConstraint bool

		falcoListenerHTTPPort string

		clusterAgent clusteragent.Agent
	}{
		{
			name:         "Run Cluster Agent Successfully",
			clusterAgent: clusteragent.NewFakeClusterAgent(logger),
			host:         "localhost",
			port:         "44355",
			expectedLog:  "tarian-cluster-agent is listening at address=127.0.0.1:44355 tarian-cluster-agent shutdown gracefully",
		},
		{
			name:        "Use Real Cluster Agent",
			expectedErr: "failed to build resolver: passthrough: received empty target in Build()",
		},
		{
			name:        "Invalid host",
			host:        "invalid host",
			port:        "44355",
			expectedErr: "lookup invalid host: no such host",
		},
		{
			name:             "TLS enabled",
			serverTLSEnabled: true,
			serverTLSCAFile:  "fakeCA/ca.crt",
			expectedErr:      "failed to read server TLS CA file: serverCAFile: fakeCA/ca.crt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &runCommand{
				globalFlags:                 &flags.GlobalFlags{},
				logger:                      logger,
				host:                        tt.host,
				port:                        tt.port,
				serverAddress:               tt.serverAddress,
				serverTLSEnabled:            tt.serverTLSEnabled,
				serverTLSCAFile:             tt.serverTLSCAFile,
				serverTLSInsecureSkipVerify: tt.serverTLSInsecureSkipVerify,
				enableAddConstraint:         tt.enableAddConstraint,
				falcoListenerHTTPPort:       tt.falcoListenerHTTPPort,
				clusterAgent:                tt.clusterAgent,
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

// TestNewRunCommand tests the creation of the runCommand cobra command.
func TestNewRunCommand(t *testing.T) {
	cmd := newRunCommand(&flags.GlobalFlags{})

	if cmd.Use != "run" {
		t.Errorf("Expected cmd.Use to be 'run', got %s", cmd.Use)
	}

	if cmd.Short != "Run the cluster agent" {
		t.Errorf("Expected cmd.Short to be 'Run the cluster agent', got %s", cmd.Short)
	}

	flags := cmd.Flags()

	expectedFlags := map[string]string{
		"host":                            defaultHost,
		"port":                            defaultPort,
		"server-address":                  defaultServerAddress,
		"server-tls-enabled":              "false",
		"server-tls-ca-file":              "",
		"server-tls-insecure-skip-verify": "true",
		"enable-add-constraint":           "false",
		"falco-listener-http-port":        defaultSidekickListenerHTTPPort,
	}

	for flagName, defaultValue := range expectedFlags {
		flag := flags.Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected '%s' flag to be defined", flagName)
		} else {
			if flag.DefValue != defaultValue {
				t.Errorf("Expected '%s' flag default value to be '%s', got '%s'", flagName, defaultValue, flag.DefValue)
			}
		}
	}
}
