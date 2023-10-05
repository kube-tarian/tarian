package add

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	utesting "github.com/kube-tarian/tarian/cmd/tarianctl/util/testing"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/spf13/cobra"

	"github.com/stretchr/testify/assert"
)

func TestAddActionCommandRun(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient        ugrpc.Client
		dryRun            bool
		onViolatedFile    bool
		onViolatedProcess bool
		matchLabels       []string
		action            string
		onFalcoAlert      string
	}{
		{
			name:        "Add Action Successfully",
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			action:      "delete-pod",
			expectedLog: "Action was added successfully",
		},
		{
			name:       "Add Action with Dry Run",
			grpcClient: ugrpc.NewFakeGrpcClient(),
			dryRun:     true,
			action:     "delete-pod",
			matchLabels: []string{
				"key1=val1",
				"key2=val2",
			},
			expectedLog: `kind: Action
namespace: test-namespace
name: test-action
selector:
    matchlabels:
        - key: key1
          value: val1
        - key: key2
          value: val2
onviolatedprocess: false
onviolatedfile: false
onfalcoalert: false
falcopriority: 0
action: delete-pod

`,
		},
		{
			name:         "Add Action with Dry Run and On Falco Alert",
			grpcClient:   ugrpc.NewFakeGrpcClient(),
			dryRun:       true,
			action:       "delete-pod",
			onFalcoAlert: "alert",
			expectedLog: `kind: Action
namespace: test-namespace
name: test-action
selector:
    matchlabels: []
onviolatedprocess: false
onviolatedfile: false
onfalcoalert: true
falcopriority: 1
action: delete-pod
`,
		},
		{
			name:         "Add Action with Invalid Falco Alert",
			grpcClient:   ugrpc.NewFakeGrpcClient(),
			action:       "delete-pod",
			onFalcoAlert: "invalid",
			expectedErr:  "add action: invalid falco alert: invalid",
		},
		{
			name:              "Add Action on Violated Process and violated file",
			grpcClient:        ugrpc.NewFakeGrpcClient(),
			dryRun:            true,
			action:            "delete-pod",
			onViolatedFile:    true,
			onViolatedProcess: true,
			expectedLog: `kind: Action
namespace: test-namespace
name: test-action
selector:
    matchlabels: []
onviolatedprocess: true
onviolatedfile: true
onfalcoalert: false
falcopriority: 0
action: delete-pod
`,
		},
		{
			name:        "Add Action with Invalid Action",
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			action:      "invalid",
			expectedErr: "invalid action: invalid",
		},
		{
			name:        "Use real gRPC client",
			action:      "delete-pod",
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
	}

	serverAddr := "localhost:50051"
	go utesting.StartFakeServer(t, serverAddr)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the action command with the test configuration
			cmd := &actionCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:            log.GetLogger(),
				grpcClient:        tt.grpcClient,
				dryRun:            tt.dryRun,
				onFalcoAlert:      tt.onFalcoAlert,
				onViolatedProcess: tt.onViolatedProcess,
				onViolatedFile:    tt.onViolatedFile,
				action:            tt.action,
				name:              "test-action",
				namespace:         "test-namespace",
				matchLabels:       tt.matchLabels,
			}

			// Capture log output
			logOutput := []byte{}
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
				assert.Equal(t, utesting.CleanLog(tt.expectedLog), utesting.CleanLog(string(logOutput)))
				// assert.Equal(t, strings.TrimSpace(utesting.CleanLog(string(logOutput))), strings.TrimSpace(tt.expectedLog))
			}
		})
	}
}

func TestNewAddActionCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newAddActionCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "default", namespaceFlag.DefValue)

	nameFlag := cmd.Flags().Lookup("name")
	assert.NotNil(t, nameFlag)

	matchLabelsFlag := cmd.Flags().Lookup("match-labels")
	assert.NotNil(t, matchLabelsFlag)

	actionFlag := cmd.Flags().Lookup("action")
	assert.NotNil(t, actionFlag)

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	assert.NotNil(t, dryRunFlag)

	onViolatedProcessFlag := cmd.Flags().Lookup("on-violated-process")
	assert.NotNil(t, onViolatedProcessFlag)

	onViolatedFileFlag := cmd.Flags().Lookup("on-violated-file")
	assert.NotNil(t, onViolatedFileFlag)

	onFalcoAlertFlag := cmd.Flags().Lookup("on-falco-alert")
	assert.NotNil(t, onFalcoAlertFlag)
}
