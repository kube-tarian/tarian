package uninstall

import (
	"fmt"
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

func TestUninstall(t *testing.T) {
	logger := log.GetLogger()
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		// k8s cluster related options
		namespace  string
		kubeconfig string

		// install options
		onlyAgents bool // install only agents

		// clients
		helmClient helm.Client
		kubeClient kubeclient.Client
	}{
		{
			name:        "Uninstall Tarian in test-namespace namespace",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			namespace:   "test-namespace",
			kubeconfig:  "fake/kubeconfig",
			expectedLog: "Uninstallation complete",
		},
		{
			name:        "Uninstall Tarian in test-namespace namespace",
			kubeClient:  kubeclient.NewFakeClient(logger),
			namespace:   "test-namespace",
			kubeconfig:  "fake/kubeconfig",
			expectedErr: "exit status 1",
		},
		{
			name:        "Use real helm client",
			kubeClient:  kubeclient.NewFakeClient(logger),
			kubeconfig:  "fake/kubeconfig",
			expectedErr: "exit status 1",
		},
		{
			name:        "Uninstall Tarian in test-namespace namespace",
			helmClient:  helm.NewFakeClient(logger),
			kubeClient:  kubeclient.NewFakeClient(logger),
			onlyAgents:  true,
			namespace:   "test-namespace",
			kubeconfig:  "fake/kubeconfig",
			expectedLog: "Uninstallation complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &uninstallCmd{
				logger:     logger,
				namespace:  tt.namespace,
				kubeconfig: tt.kubeconfig,
				onlyAgents: tt.onlyAgents,
				helmClient: tt.helmClient,
				kubeClient: tt.kubeClient,
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
				fmt.Println(tt.expectedErr, err.Error())
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				if !assert.NoError(t, err) {
					fmt.Println(err)
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

func TestNewUninstallCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := NewUninstallCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "tarian-system", namespaceFlag.DefValue)
	err := namespaceFlag.Value.Set("test-namespace")
	assert.NoError(t, err)
	assert.Equal(t, "test-namespace", namespaceFlag.Value.String())
}
