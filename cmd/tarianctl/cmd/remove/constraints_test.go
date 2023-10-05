package remove

import (
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	utesting "github.com/kube-tarian/tarian/cmd/tarianctl/util/testing"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRemoveConstraintsCommandRun(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient ugrpc.Client
		args       []string
	}{
		{
			name:        "remove constraints successfully",
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			args:        []string{"constraint1", "constraint2"},
			expectedLog: "Successfully removed constraints: [constraint1 constraint2]",
		},
		{
			name:        "Use real gRPC client",
			args:        []string{"constraint1", "constraint2"},
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
		{
			name:        "no constraints specified",
			args:        []string{},
			expectedErr: "please specify the name(s) of the constraint to be removed",
		},
	}

	serverAddr := "localhost:50058"
	go utesting.StartFakeServer(t, serverAddr)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &removeConstraintsCmd{
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

func TestNewRemoveConstraintsCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newRemoveConstraintsCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "default", namespaceFlag.DefValue)
}
