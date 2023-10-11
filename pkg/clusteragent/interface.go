package clusteragent

import (
	"net"
)

// Agent is the interface for the cluster agent.
type Agent interface {
	// Run starts the cluster agent.
	Run()
	// Close stops the cluster agent.
	Close()
	// GetGrpcServer returns the gRPC server instance used by the cluster agent.
	GetGrpcServer() GRPCServer
}

// GRPCServer is an interface for gRPC server
type GRPCServer interface {
	// Serve starts the gRPC server
	Serve(net.Listener) error
	// GracefulStop stops the gRPC server gracefully
	GracefulStop()
}
