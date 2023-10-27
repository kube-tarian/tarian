package install

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/kube-tarian/tarian/pkg/util/helm"
	"github.com/kube-tarian/tarian/pkg/util/kubeclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestInstall(t *testing.T) {
	logger := log.GetLogger()
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		// k8s cluster related options
		namespace  string
		kubeconfig string

		charts       string
		serverValues []string
		agentsValues []string

		// install options
		onlyAgents bool // install only agents

		// clients
		helmClient helm.Client
		kubeClient kubeclient.Client
	}{
		{
			name:        "Install Tarian in test-namespace namespace",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			namespace:   "test-namespace",
			expectedLog: "Tarian successfully installed in namespace 'test-namespace'",
		},
		{
			name:        "Install Tarian using helm charts from https://kube-tarian.github.io/helm-charts",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			expectedLog: "Adding Helm repository tarian with URL https://kube-tarian.github.io/helm-charts",
		},
		{
			name:        "Install Tarian using helm charts from local directory",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			namespace:   "test-namespace",
			charts:      "/fake/charts",
			expectedLog: "Installing Helm chart /fake/charts/tarian-server with name tarian-server in namespace test-namespace",
		},
		{
			name:        "Install only Tarian agents",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			namespace:   "test-namespace",
			onlyAgents:  true,
			expectedLog: "Tarian Cluster Agent and Node Agent successfully installed.",
		},
		{
			name:         "Install Tarian server with server values",
			helmClient:   helm.NewFakeClient(logger),
			kubeClient:   kubeclient.NewFakeClient(logger),
			serverValues: []string{"./dev/values/server.yaml"},
			expectedLog:  "Installing Helm chart tarian/tarian-server with name tarian-server in namespace  with values file(s) ./dev/values/server.yaml",
		},
		{
			name:         "Install Tarian agents with agents values",
			helmClient:   helm.NewFakeClient(logger),
			kubeClient:   kubeclient.NewFakeClient(logger),
			agentsValues: []string{"./dev/values/agent.yaml"},
			expectedLog:  "Installing Helm chart tarian/tarian-cluster-agent with name tarian-cluster-agent in namespace  with values file(s) ./dev/values/agent.yaml",
		},
		{
			name:        "Use real helm client",
			kubeClient:  kubeclient.NewFakeClient(logger),
			expectedErr: "exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &installCmd{
				logger:       logger,
				namespace:    tt.namespace,
				kubeconfig:   "fake/kubeconfig",
				charts:       tt.charts,
				serverValues: tt.serverValues,
				agentsValues: tt.agentsValues,
				onlyAgents:   tt.onlyAgents,
				helmClient:   tt.helmClient,
				kubeClient:   tt.kubeClient,
			}

			// Capture log output
			logOutput := []byte{}
			cmd.logger.Level = logrus.DebugLevel
			cmd.logger.Out = &utesting.LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()

			// Call the run function
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

func TestNewInstallCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := NewInstallCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "tarian-system", namespaceFlag.DefValue)
	err := namespaceFlag.Value.Set("test-namespace")
	assert.NoError(t, err)
	assert.Equal(t, "test-namespace", namespaceFlag.Value.String())

	chartsFlag := cmd.Flags().Lookup("charts")
	assert.NotNil(t, chartsFlag)
	err = chartsFlag.Value.Set("/fake/charts")
	assert.NoError(t, err)
	assert.Equal(t, "/fake/charts", chartsFlag.Value.String())

	serverValuesFlag := cmd.Flags().Lookup("server-values")
	assert.NotNil(t, serverValuesFlag)
	err = serverValuesFlag.Value.Set("./dev/values/server.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "[./dev/values/server.yaml]", serverValuesFlag.Value.String())

	agentsValuesFlag := cmd.Flags().Lookup("agents-values")
	assert.NotNil(t, agentsValuesFlag)
	err = agentsValuesFlag.Value.Set("./dev/values/agent.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "[./dev/values/agent.yaml]", agentsValuesFlag.Value.String())

	onlyAgentsFlag := cmd.Flags().Lookup("only-agents")
	assert.NotNil(t, onlyAgentsFlag)
	assert.Equal(t, "false", onlyAgentsFlag.DefValue)
	err = onlyAgentsFlag.Value.Set("true")
	assert.NoError(t, err)
	assert.Equal(t, "true", onlyAgentsFlag.Value.String())
}
