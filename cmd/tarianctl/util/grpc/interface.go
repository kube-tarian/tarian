package grpc

import (
	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

// Client is an interface for gRPC client
type Client interface {
	// NewConfigClient returns a new ConfigClient
	NewConfigClient() tarianpb.ConfigClient
	// NewEventClient returns a new EventClient
	NewEventClient() tarianpb.EventClient
}
