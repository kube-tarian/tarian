package get

import (
	"bytes"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	utesting "github.com/kube-tarian/tarian/pkg/testing"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetActionCommandRun(t *testing.T) {

	textOut := `--------------------------------------------------------------------------------------
  NAMESPACE   ACTION NAME          SELECTOR                TRIGGER          ACTION    
--------------------------------------------------------------------------------------
  default     action1       matchLabels:key1=value1   onViolatedProcess   delete-pod  
--------------------------------------------------------------------------------------
`
	yamlOut := `kind: FakeKind
namespace: default
name: action1
selector:
  matchlabels:
  - key: key1
    value: value1
onviolatedprocess: true
onviolatedfile: false
onfalcoalert: false
falcopriority: 0
action: delete-pod
---
`
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient ugrpc.Client

		namespace string
		output    string
	}{
		{
			name:        "Successful execution with default values",
			expectedErr: "",
			expectedLog: textOut,
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			namespace:   "",
			output:      "",
		},
		{
			name:        "Successful execution with output flag set to 'yaml'",
			expectedErr: "",
			expectedLog: yamlOut,
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			namespace:   "",
			output:      "yaml",
		},
		{
			name:        "Use real gRPC client",
			expectedErr: "unknown service tarianpb.api.Config",
		},
	}
	serverAddr := "localhost:50053"
	go utesting.StartFakeServer(t, serverAddr)
	defer utesting.CloseFakeServer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &actionCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:     log.GetLogger(),
				grpcClient: tt.grpcClient,
				namespace:  tt.namespace,
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

func TestNewGetActionsCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newGetActionsCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)
	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)

	outoutFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outoutFlag)

}

func TestTableOutput(t *testing.T) {
	actions := []*tarianpb.Action{
		{
			Namespace: "test-ns1",
			Name:      "action-1",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key1", Value: "value1"},
				},
			},
			OnViolatedProcess: true,
			OnViolatedFile:    true,
			Action:            "test-action-1",
		},
		{
			Namespace: "test-ns2",
			Name:      "action-2",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key2", Value: "value2"},
				},
			},
			Action: "test-action-2",
		},
	}

	var buf bytes.Buffer

	actionsTableOutput(actions, &buf)
	expectedOutput := `------------------------------------------------------------------------------------------------------
  NAMESPACE   ACTION NAME          SELECTOR                      TRIGGER                  ACTION      
------------------------------------------------------------------------------------------------------
  test-ns1    action-1      matchLabels:key1=value1   onViolatedProcess,               test-action-1  
                                                      onViolatedFile                                  
  test-ns2    action-2      matchLabels:key2=value2                                    test-action-2  
------------------------------------------------------------------------------------------------------`
	assert.Equal(t, utesting.CleanLog(expectedOutput), utesting.CleanLog(buf.String()))
}

func TestYAMLOutput(t *testing.T) {
	actions := []*tarianpb.Action{
		{
			Namespace: "test-ns1",
			Name:      "action-1",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key1", Value: "value1"},
				},
			},
			Action: "test-action-1",
		},
		{
			Namespace: "test-ns2",
			Name:      "action-2",
			Selector: &tarianpb.Selector{
				MatchLabels: []*tarianpb.MatchLabel{
					{Key: "key2", Value: "value2"},
				},
			},
			Action: "test-action-2",
		},
	}

	var buf bytes.Buffer

	logger := logrus.New()
	logger.Out = &buf

	err := actionsYamlOutput(actions, logger)
	require.NoError(t, err)

	expectedLog := `kind: ""
namespace: test-ns1
name: action-1
selector:
  matchlabels:
  - key: key1
    value: value1
onviolatedprocess: false
onviolatedfile: false
onfalcoalert: false
falcopriority: 0
action: test-action-1
---
kind: ""
namespace: test-ns2
name: action-2
selector:
  matchlabels:
  - key: key2
    value: value2
onviolatedprocess: false
onviolatedfile: false
onfalcoalert: false
falcopriority: 0
action: test-action-2
---
`
	assert.Equal(t, expectedLog, buf.String())
}
