package remove

import (
	"net"
	"regexp"
	"strings"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRemoveActionsCommand_Run(t *testing.T) {
	t.Parallel()
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
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
		{
			name:        "no actions specified",
			args:        []string{},
			expectedErr: "please specify the name(s) of the action to be removed",
		},
	}

	serverAddr := "localhost:50057"
	go startFakeServer(t, serverAddr)

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
			cmd.logger.Out = &logOutputWriter{&logOutput}

			err := cmd.run(nil, tt.args)

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				if !assert.NoError(t, err) {
					assert.FailNow(t, "error not expected")
				}
			}

			if tt.expectedLog != "" {
				assert.Equal(t, cleanLog(string(logOutput)), tt.expectedLog)
			}
		})
	}
}

func startFakeServer(t *testing.T, serverAddr string) {
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		assert.NoError(t, err)
	}

	srv := grpc.NewServer()

	if err := srv.Serve(lis); err != nil {
		assert.NoError(t, err)
	}
}

type logOutputWriter struct {
	output *[]byte
}

func (w *logOutputWriter) Write(p []byte) (n int, err error) {
	*w.output = append(*w.output, p...)
	return len(p), nil
}

func cleanLog(input string) string {
	index := strings.Index(input, "]")
	input = input[index+2:]

	spaceRe := regexp.MustCompile(`\s+`)
	input = spaceRe.ReplaceAllString(input, " ")

	newlineRe := regexp.MustCompile(`\n+`)
	input = newlineRe.ReplaceAllString(input, "\n")

	input = strings.TrimSpace(input)

	return input
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
