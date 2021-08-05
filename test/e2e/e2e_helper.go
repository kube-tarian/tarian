package e2e

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/devopstoday11/tarian/pkg/clusteragent"
	"github.com/devopstoday11/tarian/pkg/podagent"
	"github.com/devopstoday11/tarian/pkg/server"
	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/driftprogramming/pgxpoolmock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	uuid "github.com/satori/go.uuid"
)

var cfg server.PostgresqlConfig = server.PostgresqlConfig{
	User:     "postgres",
	Password: "tarian",
	Name:     "tarian", // only used to connect, it will create its own db
	Host:     "localhost",
	Port:     "5432",
	SslMode:  "disable",
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
	dbPool       pgxpoolmock.PgxPool
	dbConfig     server.PostgresqlConfig
}

func NewE2eHelper(t *testing.T) *TestHelper {
	dbPool, err := pgxpool.Connect(context.Background(), cfg.GetDsn())
	require.Nil(t, err)

	dbConfig := cfg
	dbConfig.Name += "_test_" + fmt.Sprintf("%d", time.Now().Unix()) + "_" + uuid.NewV4().String()[:8]

	_, err = dbPool.Exec(context.Background(), "CREATE DATABASE "+dbConfig.Name)
	require.Nil(t, err)

	srv, err := server.NewServer(dbConfig.GetDsn())
	grpcServer := srv.GrpcServer
	require.Nil(t, err)

	clusterAgent := clusteragent.NewClusterAgent("localhost:" + e2eServerPort)
	podAgent := podagent.NewPodAgent("localhost:" + e2eClusterAgentPort)

	return &TestHelper{t: t, dbPool: dbPool, dbConfig: dbConfig, server: grpcServer, clusterAgent: clusterAgent, podAgent: podAgent}
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
	_, err := dbstore.RunMigration(th.dbConfig.GetDsn())
	require.Nil(th.t, err)
}

func (th *TestHelper) DropDatabase() {
	_, err := th.dbPool.Exec(context.Background(), "DROP DATABASE "+th.dbConfig.Name+" WITH (FORCE)")

	require.Nil(th.t, err)
}
