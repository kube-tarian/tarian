package dgraphstore

import (
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
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

// NewGrpcClient creates a new gRPC client connection to the specified address with custom options.
//
// Parameters:
// - address: The address of the gRPC server to connect to, e.g., "localhost:9080".
// - opts: Additional gRPC DialOptions (optional).
//
// Returns:
// - A new gRPC client connection.
// - An error if there was an issue establishing the connection.
func NewGrpcClient(address string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	grpcClient, err := grpc.Dial(address, opts...)
	return grpcClient, err
}

// NewDgraphClient creates a new Dgraph client using a provided gRPC client connection.
//
// Parameters:
// - grpcClient: An established gRPC client connection to a Dgraph server.
//
// Returns:
// - A new Dgraph client.
func NewDgraphClient(grpcClient *grpc.ClientConn) *dgo.Dgraph {
	return dgo.NewDgraphClient(
		api.NewDgraphClient(grpcClient),
	)
}
