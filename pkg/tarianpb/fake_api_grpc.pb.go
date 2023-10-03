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

	return nil, nil
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
	return nil, nil
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
	return nil, nil
}
