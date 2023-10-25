package dgraphstore

import (
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/store"
	"google.golang.org/grpc"
)

// DgraphConfig represents the configuration options for connecting to a Dgraph database.
type DgraphConfig struct {
	Address     string `default:"localhost:9080"` // Address is the Dgraph server address, e.g., "localhost:9080".
	TLSCertFile string // TLSCertFile is the path to the TLS certificate file (optional for secure connections).
	TLSKeyFile  string // TLSKeyFile is the path to the TLS key file (optional for secure connections).
	TLSCAFile   string // TLSCAFile is the path to the TLS CA certificate file (optional for secure connections).
}

// defaultTimeout is the default timeout used for context operations (30 seconds).
const defaultTimeout = 30 * time.Second

type client struct {
	dg *dgo.Dgraph
}

// NewDgraphClient creates a new Dgraph client using a provided gRPC client connection.
func NewDgraphClient(conn *grpc.ClientConn) Client {
	return &client{dg: dgo.NewDgraphClient(api.NewDgraphClient(conn))}
}

// NewDgraphConstraintStore creates a new DgraphConstraintStore with the provided Dgraph client.
func (c *client) NewDgraphConstraintStore() store.ConstraintStore {
	return newDgraphConstraintStore(c.dg)
}

// NewDgraphActionStore creates a new DgraphActionStore with the provided Dgraph client.
func (c *client) NewDgraphActionStore() store.ActionStore {
	return newDgraphActionStore(c.dg)
}

// NewDgraphEventStore creates a new DgraphEventStore with the provided Dgraph client.
func (c *client) NewDgraphEventStore() store.EventStore {
	return newDgraphEventStore(c.dg)
}
