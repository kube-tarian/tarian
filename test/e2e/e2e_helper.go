package e2e

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/podagent"
	"github.com/kube-tarian/tarian/pkg/server"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/store"
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

type TestHelper struct {
	server       *grpc.Server
	clusterAgent *clusteragent.ClusterAgent
	podAgent     *podagent.PodAgent
	t            *testing.T
	dgraphConfig dgraphstore.DgraphConfig
	dg           *dgo.Dgraph
}

func NewE2eHelper(t *testing.T) *TestHelper {
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcClient, err := dgraphstore.NewGrpcClient(cfg.Address, dialOpts)

	if err != nil {
		t.Fatal("error while initiating dgraph client", "err", err)
	}

	dg := dgraphstore.NewDgraphClient(grpcClient)

	storeSet := store.StoreSet{}
	storeSet.EventStore = dgraphstore.NewDgraphEventStore(dg)
	storeSet.ActionStore = dgraphstore.NewDgraphActionStore(dg)
	storeSet.ConstraintStore = dgraphstore.NewDgraphConstraintStore(dg)

	srv, err := server.NewServer(storeSet, "", "", "")
	grpcServer := srv.GrpcServer
	require.Nil(t, err)

	clusterAgentConfig := &clusteragent.ClusterAgentConfig{
		ServerAddress:         "localhost:" + e2eServerPort,
		ServerGrpcDialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}

	clusterAgent := clusteragent.NewClusterAgent(clusterAgentConfig)
	podAgent := podagent.NewPodAgent("localhost:" + e2eClusterAgentPort)

	return &TestHelper{t: t, dgraphConfig: cfg, dg: dg, server: grpcServer, clusterAgent: clusterAgent, podAgent: podAgent}
}

func (th *TestHelper) RunServer() {
	listener, err := net.Listen("tcp", ":"+e2eServerPort)
	require.Nil(th.t, err)

	fmt.Println("tarian-server is serving")
	err = th.server.Serve(listener)
	require.Nil(th.t, err)
}

func (th *TestHelper) RunClusterAgent() {
	caListener, err := net.Listen("tcp", ":"+e2eClusterAgentPort)
	require.Nil(th.t, err)

	fmt.Println("tarian-cluster-agent is serving")
	err = th.clusterAgent.GetGrpcServer().Serve(caListener)
	require.Nil(th.t, err)
}

func (th *TestHelper) Stop() {
	th.server.GracefulStop()
	th.clusterAgent.GetGrpcServer().GracefulStop()
	th.podAgent.GracefulStop()
}

func (th *TestHelper) Run() {
	go th.RunServer()
	go th.RunClusterAgent()

	time.Sleep(2 * time.Second)
	th.podAgent.Dial()
}

func (th *TestHelper) PrepareDatabase() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := dgraphstore.ApplySchema(ctx, th.dg)
	require.Nil(th.t, err)
}
