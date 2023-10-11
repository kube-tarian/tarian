package importcommand

import (
	"os"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// YAML content to write to the temporary file
const constraint1 = `kind: FakeKind
namespace: test-ns1
name: constraint-1
selector:
  matchlabels:
  - key: key1
    value: value1
allowedprocesses:
- regex: regex-1
allowedfiles:
- name: file-1
  sha256sum: hash-1`

const constraint2 = `kind: FakeKind
namespace: test-ns1
name: constraint-2
selector:
  matchlabels:
  - key: key1
    value: value1
allowedprocesses:
- regex: regex-1
allowedfiles:
- name: file-1
  sha256sum: hash-1`

func generateTempFile(directory, content string) string {
	tempFile, err := os.CreateTemp(directory, "import-file-*.yaml")
	if err != nil {
		panic(err)
	}
	_, err = tempFile.WriteString(content)
	if err != nil {
		panic(err)
	}
	return tempFile.Name()
}
func TestImportCommandRun(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "import-dir-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient ugrpc.Client
		args       []string
	}{
		{
			name:        "Use real gRPC client",
			args:        []string{generateTempFile(tempDir, constraint1)},
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
		{
			name:        "Zero files",
			args:        []string{},
			expectedErr: "specify file paths to import",
		},
		{
			name:        "One file",
			args:        []string{generateTempFile(tempDir, constraint1)},
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			expectedLog: "1 constraint(s) imported successfully",
		},
		{
			name:        "File not found",
			args:        []string{"not-found.yaml"},
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			expectedErr: "failed to open file not-found.yaml: open not-found.yaml: no such file or directory",
		},
		{
			name:        "empty file",
			args:        []string{generateTempFile(tempDir, "")},
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			expectedLog: "No constraints imported",
		},
		{
			name:       "Multiple constraints",
			grpcClient: ugrpc.NewFakeGrpcClient(),
			args:       []string{generateTempFile(tempDir, constraint1+"\n---\n"+constraint2)},
		},
	}

	serverAddr := "localhost:50056"
	go utesting.StartFakeServer(t, serverAddr)
	defer utesting.CloseFakeServer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &importCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:     log.GetLogger(),
				grpcClient: tt.grpcClient,
			}

			logOutput := []byte{}
			cmd.logger.Out = &utesting.LogOutputWriter{Output: &logOutput}
			log.MiniLogFormat()

			err = cmd.run(nil, tt.args)

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

func TestNewImportCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := NewImportCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	flags := cmd.HasAvailableFlags()
	assert.False(t, flags)
}
