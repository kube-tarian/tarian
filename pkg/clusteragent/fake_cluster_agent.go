package clusteragent

import (
	"net"

	"github.com/sirupsen/logrus"
)

type fakeClusterAgent struct{}

// NewFakeClusterAgent creates a new fake ClusterAgent instance.
func NewFakeClusterAgent(logger *logrus.Logger) Agent {
	return &fakeClusterAgent{}
}

// Close implements Agent.
func (f *fakeClusterAgent) Close() {}

// GetGrpcServer implements Agent.
func (f *fakeClusterAgent) GetGrpcServer() GRPCServer {
	return NewFakeGRPCServer()
}

// Run implements Agent.
func (f *fakeClusterAgent) Run() {}

type fakeGRPCServer struct{}

// NewFakeGRPCServer creates a new fake gRPC server
func NewFakeGRPCServer() GRPCServer {
	return &fakeGRPCServer{}
}

// Serve starts the fake gRPC server
func (s *fakeGRPCServer) Serve(lis net.Listener) error {
	return nil
}

// GracefulStop stops the fake gRPC server
func (s *fakeGRPCServer) GracefulStop() {}
