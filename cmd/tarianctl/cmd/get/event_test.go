package get

import (
	"bytes"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	utesting "github.com/kube-tarian/tarian/cmd/tarianctl/util/testing"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const expectedOutput = `--------------------------------------------------------------------------------------------------------------------------
          TIME           NAMESPACE     POD                                      EVENTS                                    
--------------------------------------------------------------------------------------------------------------------------
  1970-01-01T00:00:00Z   default     nginx-1   violated processes                                                         
                                                                                                                          
                                               123:Unknownviolated files                                                  
                                                                                                                          
                                               name=/etc/unknownFile actual-sha256=1234567890 expected-sha256=0987654321  
                                                                                                                          
                                                                                                                          
--------------------------------------------------------------------------------------------------------------------------
`

func TesGetEventsCommandRun(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient grpc.Client
		limit      uint
	}{
		{
			name:        "Successful execution with default values",
			grpcClient:  grpc.NewFakeGrpcClient(),
			limit:       200,
			expectedLog: expectedOutput,
		},
		{
			name:        "Use real gRPC client",
			expectedErr: "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Event",
		},
	}

	serverAddr := "localhost:50055"
	go utesting.StartFakeServer(t, serverAddr)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &eventsCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:     log.GetLogger(),
				grpcClient: tt.grpcClient,
				limit:      tt.limit,
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

func TestNewGetEventsCommand(t *testing.T) {
	mockGlobalFlags := &flags.GlobalFlags{
		ServerAddr: "mock-server-address",
	}

	cmd := newGetEventsCommand(mockGlobalFlags)

	assert.IsType(t, &cobra.Command{}, cmd)

	limitFlag := cmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag)
}

func TestEventsTableOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.Out = &buf

	events := []*tarianpb.Event{
		{
			Kind: "FakeKind",
			Type: "FakeType",
			Uid:  "FakeUid",
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Namespace: "default",
						Name:      "nginx-1",
						Labels: []*tarianpb.Label{
							{Key: "app", Value: "nginx"},
						},
					},
					ViolatedProcesses: []*tarianpb.Process{
						{
							Pid:  123,
							Name: "Unknown",
						},
					},
					ViolatedFiles: []*tarianpb.ViolatedFile{
						{
							Name:              "/etc/unknownFile",
							ActualSha256Sum:   "1234567890",
							ExpectedSha256Sum: "0987654321",
						},
					},
				},
			},
		},
	}

	eventsTableOutput(events, logger)

	assert.Equal(t, utesting.CleanLog(expectedOutput), utesting.CleanLog(buf.String()))
}

func TestViolatedProcessesToString(t *testing.T) {
	processes := []*tarianpb.Process{
		{Pid: 123, Name: "process-1"},
		{Pid: 456, Name: "process-2"},
		{Pid: 789, Name: "process-3"},
	}

	result := violatedProcessesToString(processes)
	expectedResult := "123:process-1, 456:process-2, 789:process-3"

	assert.Equal(t, expectedResult, result)
}

func TestViolatedFilesToString(t *testing.T) {
	files := []*tarianpb.ViolatedFile{
		{Name: "file-1", ActualSha256Sum: "actual-1", ExpectedSha256Sum: "expected-1"},
		{Name: "file-2", ActualSha256Sum: "actual-2", ExpectedSha256Sum: "expected-2"},
		{Name: "file-3", ActualSha256Sum: "actual-3", ExpectedSha256Sum: "expected-3"},
	}

	result := violatedFilesToString(files)

	expectedResult := "name=file-1 actual-sha256=actual-1 expected-sha256=expected-1, " +
		"name=file-2 actual-sha256=actual-2 expected-sha256=expected-2, " +
		"name=file-3 actual-sha256=actual-3 expected-sha256=expected-3"

	assert.Equal(t, expectedResult, result)
}

func TestFalcoAlertToString(t *testing.T) {
	tests := []struct {
		name           string
		alert          *tarianpb.FalcoAlert
		expectedResult string
	}{
		{
			name: "Emergency",
			alert: &tarianpb.FalcoAlert{
				Priority: tarianpb.FalcoPriority(tarianpb.FalcoPriority.Number(0)),
				Output:   "alert output",
			},
			expectedResult: "EMERGENCY: alert output",
		},
		{
			name: "Critical",
			alert: &tarianpb.FalcoAlert{
				Priority: tarianpb.FalcoPriority(tarianpb.FalcoPriority.Number(2)),
				Output:   "alert output",
			},
			expectedResult: "CRITICAL: alert output",
		},
		{
			name: "Alert",
			alert: &tarianpb.FalcoAlert{
				Priority: tarianpb.FalcoPriority(tarianpb.FalcoPriority.Number(1)),
				Output:   "alert output",
			},
			expectedResult: "ALERT: alert output",
		},
		{
			name:           "Nil",
			alert:          nil,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := falcoAlertToString(tt.alert)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
