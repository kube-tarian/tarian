package e2e

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/podagent"
	"github.com/kube-tarian/tarian/pkg/server"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var cfg dgraphstore.DgraphConfig = dgraphstore.DgraphConfig{
	Address: "localhost:9080",
}

const (
	e2eServerPort       = "60051"
	e2eClusterAgentPort = "60052"
)

// TestHelper is a helper struct for setting up and managing testing components.
type TestHelper struct {
	server       *grpc.Server             // The gRPC server used for testing.
	clusterAgent clusteragent.Agent       // The ClusterAgent for cluster management.
	podAgent     podagent.Agent           // The PodAgent for managing pods.
	t            *testing.T               // The testing.T instance for reporting test failures.
	dgraphConfig dgraphstore.DgraphConfig // Configuration for Dgraph database.
	dg           dgraphstore.Client       // The Dgraph client.
}

// NewE2eHelper creates a new TestHelper instance for e2e tests.
func NewE2eHelper(t *testing.T) *TestHelper {
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcClient, err := grpc.Dial(cfg.Address, dialOpts...)
	if err != nil {
		t.Fatal("error while initiating dgraph client", "err", err)
	}

	dg := dgraphstore.NewDgraphClient(grpcClient)

	storeSet := store.Set{}
	storeSet.EventStore = dg.NewDgraphEventStore()
	storeSet.ActionStore = dg.NewDgraphActionStore()
	storeSet.ConstraintStore = dg.NewDgraphConstraintStore()

	srv, err := server.NewServer(log.GetLogger(), storeSet, "", "", "", []nats.Option{}, nats.StreamConfig{})
	grpcServer := srv.GetGrpcServer()
	require.Nil(t, err)

	clusterAgentConfig := &clusteragent.Config{
		ServerAddress:         "localhost:" + e2eServerPort,
		ServerGrpcDialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}

	clusterAgent, err := clusteragent.NewClusterAgent(log.GetLogger(), clusterAgentConfig)
	require.Nil(t, err)
	podAgent := podagent.NewPodAgent(log.GetLogger(), "localhost:"+e2eClusterAgentPort)

	return &TestHelper{t: t, dgraphConfig: cfg, dg: dg, server: grpcServer, clusterAgent: clusterAgent, podAgent: podAgent}
}

// RunServer runs the Tarian server.
func (th *TestHelper) RunServer() {
	listener, err := net.Listen("tcp", ":"+e2eServerPort)
	require.Nil(th.t, err)

	fmt.Println("tarian-server is serving")
	err = th.server.Serve(listener)
	require.Nil(th.t, err)
}

// RunClusterAgent runs the Tarian cluster agent.
func (th *TestHelper) RunClusterAgent() {
	caListener, err := net.Listen("tcp", ":"+e2eClusterAgentPort)
	require.Nil(th.t, err)

	fmt.Println("tarian-cluster-agent is serving")
	err = th.clusterAgent.GetGrpcServer().Serve(caListener)
	require.Nil(th.t, err)
}

// Stop stops the Tarian server and cluster agent.
func (th *TestHelper) Stop() {
	th.server.GracefulStop()
	th.clusterAgent.GetGrpcServer().GracefulStop()
	th.podAgent.GracefulStop()
}

// Run runs the Tarian server and cluster agent.
func (th *TestHelper) Run() {
	go th.RunServer()
	go th.RunClusterAgent()

	time.Sleep(2 * time.Second)
	th.podAgent.Dial()
}

// PrepareDatabase prepares the database for e2e tests.
func (th *TestHelper) PrepareDatabase() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := th.dg.ApplySchema(ctx)
	require.Nil(th.t, err)
}
