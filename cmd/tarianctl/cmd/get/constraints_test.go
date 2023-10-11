package get

import (
	"bytes"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	utesting "github.com/kube-tarian/tarian/pkg/testing"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConstraintCommandRun(t *testing.T) {
	// t.Parallel()
	textOut := `---------------------------------------------------------------------------------------------
  NAMESPACE   CONSTRAINT NAME          SELECTOR           ALLOWED PROCESSES   ALLOWED FILES  
---------------------------------------------------------------------------------------------
  test-ns1    constraint-1      matchLabels:key1=value1   regex:regex-1       file-1:hash-1  
---------------------------------------------------------------------------------------------
`

	yamlOut := `kind: FakeKind
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
  sha256sum: hash-1
---
`

	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient grpc.Client
		output     string
	}{
		{
			name:        "Successful execution with default values",
			grpcClient:  grpc.NewFakeGrpcClient(),
			output:      "",
			expectedLog: textOut,
		},
		{
			name:        "Successful execution with output flag set to 'yaml'",
			grpcClient:  grpc.NewFakeGrpcClient(),
			output:      "yaml",
			expectedLog: yamlOut,
		},
		{
			name:        "Use real gRPC client",
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
	}

	serverAddr := "localhost:50054"
	go utesting.StartFakeServer(t, serverAddr)
	defer utesting.CloseFakeServer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &constraintsCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:     log.GetLogger(),
				grpcClient: tt.grpcClient,
				output:     tt.output,
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

func TestConstraintsTableOutput(t *testing.T) {
	regex1 := "regex-1"
	hash1 := "hash-1"
	regex2 := "regex-2"
	hash2 := "file-2"
	constraints := []*tarianpb.Constraint{
		{
			Namespace: "test-ns1",
			Name:      "constraint-1",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key1", Value: "value1"},
				},
			},
			AllowedProcesses: []*tarianpb.AllowedProcessRule{
				{Regex: &regex1},
				{Regex: &regex2},
			},
			AllowedFiles: []*tarianpb.AllowedFileRule{
				{Name: "file-1", Sha256Sum: &hash1},
				{Name: "file-2", Sha256Sum: &hash2},
			},
		},
		{
			Namespace: "test-ns2",
			Name:      "constraint-2",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key2", Value: "value2"},
				},
			},
		},
	}

	var buf bytes.Buffer

	constraintsTableOutput(constraints, &buf)
	expectedOutput := `---------------------------------------------------------------------------------------------------------------------
  NAMESPACE   CONSTRAINT NAME          SELECTOR                ALLOWED PROCESSES               ALLOWED FILES         
---------------------------------------------------------------------------------------------------------------------
  test-ns1    constraint-1      matchLabels:key1=value1   regex:regex-1,regex:regex-2   file-1:hash-1,file-2:file-2  
  test-ns2    constraint-2      matchLabels:key2=value2                                                              
---------------------------------------------------------------------------------------------------------------------
`
	assert.Equal(t, utesting.CleanLog(expectedOutput), utesting.CleanLog(buf.String()))
}

func TestConstraintsYAMLOutput(t *testing.T) {
	regex1 := "regex-1"
	hash1 := "hash-1"
	regex2 := "regex-2"
	hash2 := "file-2"
	constraints := []*tarianpb.Constraint{
		{
			Namespace: "test-ns1",
			Name:      "constraint-1",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key1", Value: "value1"},
				},
			},
			AllowedProcesses: []*tarianpb.AllowedProcessRule{
				{Regex: &regex1},
				{Regex: &regex2},
			},
			AllowedFiles: []*tarianpb.AllowedFileRule{
				{Name: "file-1", Sha256Sum: &hash1},
				{Name: "file-2", Sha256Sum: &hash2},
			},
		},
		{
			Namespace: "test-ns2",
			Name:      "constraint-2",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key2", Value: "value2"},
				},
			},
		},
	}

	var buf bytes.Buffer

	logger := logrus.New()
	logger.Out = &buf

	err := constraintsYamlOutput(constraints, logger)
	require.NoError(t, err)

	expectedLog := `kind: ""
namespace: test-ns1
name: constraint-1
selector:
  matchlabels:
  - key: key1
    value: value1
allowedprocesses:
- regex: regex-1
- regex: regex-2
allowedfiles:
- name: file-1
  sha256sum: hash-1
- name: file-2
  sha256sum: file-2
---
kind: ""
namespace: test-ns2
name: constraint-2
selector:
  matchlabels:
  - key: key2
    value: value2
allowedprocesses: []
allowedfiles: []
---
`
	assert.Equal(t, expectedLog, buf.String())
}

func TestNewGetConstraintsCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newGetConstraintsCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	outoutFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outoutFlag)
}
