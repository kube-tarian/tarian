package remove

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRemoveActionsCommandRun(t *testing.T) {

	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient ugrpc.Client
		args       []string
	}{
		{
			name:        "remove actions successfully",
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			args:        []string{"action1", "action2"},
			expectedLog: "Successfully removed actions: [action1 action2]",
		},
		{
			name:        "Use real gRPC client",
			args:        []string{"action1", "action2"},
			expectedErr: "unknown service tarianpb.api.Config",
		},
		{
			name:        "no actions specified",
			args:        []string{},
			expectedErr: "please specify the name(s) of the action to be removed",
		},
	}

	serverAddr := "localhost:50057"
	go utesting.StartFakeServer(t, serverAddr)
	defer utesting.CloseFakeServer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &removeActionsCmd{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:     log.GetLogger(),
				grpcClient: tt.grpcClient,
			}

			logOutput := []byte{}
			cmd.logger.Out = &utesting.LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()

			err := cmd.run(nil, tt.args)

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

func TestNewRemoveActionsCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newRemoveActionsCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "default", namespaceFlag.DefValue)
}
