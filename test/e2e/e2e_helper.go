package e2e

import (
	"fmt"
	"net"
	"testing"

	"github.com/devopstoday11/tarian/pkg/clusteragent"
	"github.com/devopstoday11/tarian/pkg/podagent"
	"github.com/devopstoday11/tarian/pkg/server"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var cfg server.PostgresqlConfig = server.PostgresqlConfig{
	User:     "postgres",
	Password: "tarian",
	Name:     "tarian",
	Host:     "localhost",
	Port:     "5432",
	SslMode:  "disable",
}

const (
	e2eServerPort       = "60051"
	e2eClusterAgentPort = "60052"
)

type E2eHelper struct {
	server       *grpc.Server
	clusterAgent *clusteragent.ClusterAgent
	podAgent     *podagent.PodAgent
	t            *testing.T
}

func NewE2eHelper(t *testing.T) *E2eHelper {
	grpcServer, err := server.NewGrpcServer(cfg.GetDsn())
	require.Nil(t, err)

	clusterAgent := clusteragent.NewClusterAgent("localhost:" + e2eServerPort)
	podAgent := podagent.NewPodAgent("localhost:" + e2eClusterAgentPort)

	return &E2eHelper{server: grpcServer, t: t, clusterAgent: clusterAgent, podAgent: podAgent}
}

func (e *E2eHelper) RunServer() {
	listener, err := net.Listen("tcp", ":"+e2eServerPort)
	require.Nil(e.t, err)

	fmt.Println("tarian-server is serving")
	err = e.server.Serve(listener)
	require.Nil(e.t, err)
}

func (e *E2eHelper) RunClusterAgent() {
	caListener, err := net.Listen("tcp", ":"+e2eClusterAgentPort)
	require.Nil(e.t, err)

	fmt.Println("tarian-cluster-agent is serving")
	err = e.clusterAgent.GetGrpcServer().Serve(caListener)
	require.Nil(e.t, err)
}

func (e *E2eHelper) Stop() {
	e.server.GracefulStop()
	e.clusterAgent.GetGrpcServer().GracefulStop()
	e.podAgent.Close()
}

func (e *E2eHelper) Run() {
	go e.RunServer()
	go e.RunClusterAgent()

	e.podAgent.Dial()
}
