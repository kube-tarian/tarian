package add

import (
	"strings"
	"testing"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestConstraintCommand_Run(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr string
		expectedLog string

		grpcClient            ugrpc.Client
		constraintName        string
		matchLabels           []string
		allowedProcesses      []string
		allowedFileSha256Sums []string
		fromViolatedPod       string
		dryRun                bool
	}{
		{
			name:             "Add Constraint Successfully",
			grpcClient:       ugrpc.NewFakeGrpcClient(),
			constraintName:   "test-constraint",
			matchLabels:      []string{"key1=val1"},
			allowedProcesses: []string{"process1"},
			expectedLog:      "Constraint was added successfully",
		},
		{
			name:        "Add Constraint without name or from-violated-pod",
			grpcClient:  ugrpc.NewFakeGrpcClient(),
			expectedErr: "either constraint name or from-violated-pod is required",
		},
		{
			name:            "Add Constraint with both name and from-violated-pod",
			grpcClient:      ugrpc.NewFakeGrpcClient(),
			constraintName:  "test-constraint",
			fromViolatedPod: "test-pod",
			expectedErr:     "constraint name and from-violated-pod cannot be used together",
		},
		{
			name:                  "Add Constraint Successfully",
			grpcClient:            ugrpc.NewFakeGrpcClient(),
			constraintName:        "test-constraint",
			matchLabels:           []string{"key1=val1"},
			allowedProcesses:      []string{"process1", "process2"},
			allowedFileSha256Sums: []string{"file1=sha256sum1", "file2=sha256sum2"},
			dryRun:                true,
			expectedLog: `kind: Constraint
namespace: test-namespace
name: test-constraint
selector:
  matchlabels:
  - key: key1
    value: val1
allowedprocesses:
- regex: process1
- regex: process2
allowedfiles:
- name: file1
  sha256sum: sha256sum1
- name: file2
  sha256sum: sha256sum2
`,
		},
		{
			name:           "Add Constraint without allowedProcesses and allowedFileSha256Sums",
			grpcClient:     ugrpc.NewFakeGrpcClient(),
			constraintName: "test-constraint",
			expectedErr:    "no allowed processes or files found, use --allowed-processes or --allowed-file-sha256sums or both",
		},
		{
			name:                  "Add Constraint without matchLabels",
			grpcClient:            ugrpc.NewFakeGrpcClient(),
			constraintName:        "test-constraint",
			allowedProcesses:      []string{"process1", "process2"},
			allowedFileSha256Sums: []string{"file1=sha256sum1", "file2=sha256sum2"},
			expectedErr:           "no match labels found, use --match-labels",
		},
		{
			name:             "Use real gRPC client",
			constraintName:   "test-constraint",
			matchLabels:      []string{"key1=val1"},
			allowedProcesses: []string{"process1", "process2"},
			expectedErr:      "rpc error: code = Unimplemented desc = unknown service tarianpb.api.Config",
		},
		// TODO: Add test for from-violated-pod after faking GetEvents()
		// {
		// 	name:            "Add Constraint with from-violated-pod",
		// 	grpcClient:      ugrpc.NewFakeGrpcClient(),
		// 	fromViolatedPod: "test-pod",
		// },

		// TODO: Add test for Duplicate rules

	}
	serverAddr := "localhost:50052"
	go startFakeServer(t, serverAddr)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &constraintsCommand{
				globalFlags: &flags.GlobalFlags{
					ServerAddr: serverAddr,
				},
				logger:                log.GetLogger(),
				grpcClient:            tt.grpcClient,
				name:                  tt.constraintName,
				namespace:             "test-namespace",
				matchLabels:           tt.matchLabels,
				allowedProcesses:      tt.allowedProcesses,
				allowedFileSha256Sums: tt.allowedFileSha256Sums,
				fromViolatedPod:       tt.fromViolatedPod,
				dryRun:                tt.dryRun,
			}

			// Capture log output
			logOutput := []byte{}
			cmd.logger.Out = &logOutputWriter{&logOutput}

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
				assert.Equal(t, strings.TrimSpace(cleanLog(string(logOutput))), strings.TrimSpace(tt.expectedLog))
			}
		})
	}
}

func TestAllowedProcessesFromString(t *testing.T) {
	tests := []struct {
		name       string
		input      []string
		expected   []*tarianpb.AllowedProcessRule
		shouldFail bool
	}{
		{
			name:  "Allow both processes",
			input: []string{"process1", "process2"},
			expected: []*tarianpb.AllowedProcessRule{
				{Regex: strPtr("process1")},
				{Regex: strPtr("process2")},
			},
			shouldFail: false,
		},
		{
			name:       "No process, nil output",
			input:      nil,
			expected:   nil,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowedProcessesFromString(tt.input)

			if tt.shouldFail {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAllowedFilesFromString(t *testing.T) {
	tests := []struct {
		name       string
		input      []string
		expected   []*tarianpb.AllowedFileRule
		shouldFail bool
	}{
		{
			name:  "Allow both files",
			input: []string{"file1=hash1", "file2=hash2"},
			expected: []*tarianpb.AllowedFileRule{
				{Name: "file1", Sha256Sum: strPtr("hash1")},
				{Name: "file2", Sha256Sum: strPtr("hash2")},
			},
			shouldFail: false,
		},
		{
			name:  "Allow files with without hash",
			input: []string{"file1=hash1", "file2=hash2", "file1"},
			expected: []*tarianpb.AllowedFileRule{
				{Name: "file1", Sha256Sum: strPtr("hash1")},
				{Name: "file2", Sha256Sum: strPtr("hash2")},
			},
			shouldFail: false,
		},
		{
			name:       "Don't allow files with without hash",
			input:      []string{"file1="},
			expected:   nil,
			shouldFail: true,
		},
		{
			name:       "No file, nil output",
			input:      nil,
			expected:   nil,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowedFilesFromString(tt.input)

			if tt.shouldFail {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

// TODO: Add test for buildConstraintFromViolatedPod
