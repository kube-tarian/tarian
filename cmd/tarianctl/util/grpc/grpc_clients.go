package grpc

import (
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

type client struct {
	conn *grpc.ClientConn
}

// NewGRPCClient creates a new GRPCClient.
func NewGRPCClient(conn *grpc.ClientConn) Client {
	return &client{
		conn: conn,
	}
}

// NewConfigClient creates a new ConfigClient.
func (g *client) NewConfigClient() tarianpb.ConfigClient {
	return tarianpb.NewConfigClient(g.conn)
}

// NewEventClient creates a new EventClient.
func (g *client) NewEventClient() tarianpb.EventClient {
	return tarianpb.NewEventClient(g.conn)
}
