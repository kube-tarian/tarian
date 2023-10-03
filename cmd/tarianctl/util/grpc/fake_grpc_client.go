package grpc

import "github.com/kube-tarian/tarian/pkg/tarianpb"

// FakeGrpcClient is a fake implementation of GrpcClient.
type FakeGrpcClient struct{}

// NewFakeGrpcClient returns a new instance of FakeGrpcClient.
func NewFakeGrpcClient() *FakeGrpcClient {
	return &FakeGrpcClient{}
}

// NewConfigClient returns a new instance of fakeConfigClient.
func (f *FakeGrpcClient) NewConfigClient() tarianpb.ConfigClient {
	return tarianpb.NewFakeConfigClient()
}

// NewEventClient returns a new instance of fakeEventClient.
func (f *FakeGrpcClient) NewEventClient() tarianpb.EventClient {
	return tarianpb.NewFakeEventClient()
}
