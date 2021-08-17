package clusteragent

import (
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

type ClusterAgent struct {
	grpcServer   *grpc.Server
	configServer *ConfigServer
	eventServer  *EventServer
}

func NewClusterAgent(serverAddress string, opts []grpc.DialOption) *ClusterAgent {
	grpcServer := grpc.NewServer()

	configServer := NewConfigServer(serverAddress, opts)
	eventServer := NewEventServer(serverAddress, opts)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	ca := &ClusterAgent{
		grpcServer:   grpcServer,
		configServer: configServer,
		eventServer:  eventServer,
	}

	return ca
}

func (ca *ClusterAgent) Close() {
	ca.configServer.Close()
	ca.eventServer.Close()
}

func (ca *ClusterAgent) GetGrpcServer() *grpc.Server {
	return ca.grpcServer
}
