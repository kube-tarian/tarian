package tarianpb

import (
	context "context"

	grpc "google.golang.org/grpc"
)

type fakeConfigClient struct{}

// NewFakeConfigClient returns a new instance of fakeConfigClient.
func NewFakeConfigClient() ConfigClient {
	return &fakeConfigClient{}
}

// GetConstraints returns the constraints for the specified namespace.
func (f *fakeConfigClient) GetConstraints(ctx context.Context, in *GetConstraintsRequest, opts ...grpc.CallOption) (*GetConstraintsResponse, error) {
	var regex = "regex-1"
	var hash = "hash-1"
	return &GetConstraintsResponse{
		Constraints: []*Constraint{
			{
				Kind:      "FakeKind",
				Namespace: "test-ns1",
				Name:      "constraint-1",
				Selector: &Selector{
					MatchLabels: []*MatchLabel{
						{Key: "key1", Value: "value1"},
					},
				},
				AllowedProcesses: []*AllowedProcessRule{
					{Regex: &regex},
				},
				AllowedFiles: []*AllowedFileRule{
					{Name: "file-1", Sha256Sum: &hash},
				},
			},
		},
	}, nil
}

// AddConstraint adds a constraint to the specified namespace.
func (f *fakeConfigClient) AddConstraint(ctx context.Context, in *AddConstraintRequest, opts ...grpc.CallOption) (*AddConstraintResponse, error) {
	out := &AddConstraintResponse{
		Success: true,
	}
	return out, nil
}

// RemoveConstraint removes a constraint from the specified namespace.
func (f *fakeConfigClient) RemoveConstraint(ctx context.Context, in *RemoveConstraintRequest, opts ...grpc.CallOption) (*RemoveConstraintResponse, error) {
	return nil, nil
}

// AddAction adds an action to the specified namespace.
func (f *fakeConfigClient) AddAction(ctx context.Context, in *AddActionRequest, opts ...grpc.CallOption) (*AddActionResponse, error) {
	out := &AddActionResponse{
		Success: true,
	}
	return out, nil
}

// GetActions returns the actions for the specified namespace.
func (f *fakeConfigClient) GetActions(ctx context.Context, in *GetActionsRequest, opts ...grpc.CallOption) (*GetActionsResponse, error) {
	return &GetActionsResponse{
		Actions: []*Action{
			{
				Kind:      "FakeKind",
				Namespace: "default",
				Name:      "action1",
				Selector: &Selector{
					MatchLabels: []*MatchLabel{
						{Key: "key1", Value: "value1"},
					},
				},
				OnViolatedProcess: true,
				OnViolatedFile:    false,
				OnFalcoAlert:      false,
				Action:            "delete-pod",
			},
		},
	}, nil
}

// RemoveAction removes an action from the specified namespace.
func (f *fakeConfigClient) RemoveAction(ctx context.Context, in *RemoveActionRequest, opts ...grpc.CallOption) (*RemoveActionResponse, error) {
	return nil, nil
}

type fakeEventClient struct{}

// NewFakeEventClient returns a new instance of fakeEventClient.
func NewFakeEventClient() EventClient {
	return &fakeEventClient{}
}

// IngestEvent ingests an event to the Tarian Server.
func (f *fakeEventClient) IngestEvent(ctx context.Context, in *IngestEventRequest, opts ...grpc.CallOption) (*IngestEventResponse, error) {
	return nil, nil
}

// GetEvents returns the events from the Tarian Server.
func (f *fakeEventClient) GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error) {
	return &GetEventsResponse{
		Events: []*Event{
			{
				Kind: "FakeKind",
				Type: "FakeType",
				Uid:  "FakeUid",
				Targets: []*Target{
					{
						Pod: &Pod{
							Namespace: "default",
							Name:      "nginx-1",
							Labels: []*Label{
								{Key: "app", Value: "nginx"},
							},
						},
						ViolatedProcesses: []*Process{
							{
								Pid:  123,
								Name: "Unknown",
							},
						},
						ViolatedFiles: []*ViolatedFile{
							{
								Name:              "/etc/unknownFile",
								ActualSha256Sum:   "1234567890",
								ExpectedSha256Sum: "0987654321",
							},
						},
					},
				},
			},
		},
	}, nil
}
