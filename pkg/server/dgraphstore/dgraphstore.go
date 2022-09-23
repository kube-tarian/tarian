package dgraphstore

import (
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
)

type DgraphConfig struct {
	Address     string `default:"localhost:9080"`
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string
}

const defaultTimeout = 30 * time.Second

func NewGrpcClient(address string, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	grpcClient, err := grpc.Dial(address, opts...)

	return grpcClient, err
}

func NewDgraphClient(grpcClient *grpc.ClientConn) *dgo.Dgraph {
	return dgo.NewDgraphClient(
		api.NewDgraphClient(grpcClient),
	)
}
