package clusteragent

import (
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	falcoclient "github.com/falcosecurity/client-go/pkg/client"
	"google.golang.org/grpc"
)

type ClusterAgentConfig struct {
	ServerAddress          string
	ServerGrpcDialOptions  []grpc.DialOption
	EnableFalcoIntegration bool
	FalcoClientConfig      *falcoclient.Config
}

type ClusterAgent struct {
	grpcServer           *grpc.Server
	configServer         *ConfigServer
	eventServer          *EventServer
	falcoAlertsSubsriber *FalcoAlertsSubscriber
}

func NewClusterAgent(config *ClusterAgentConfig) *ClusterAgent {
	grpcServer := grpc.NewServer()

	configServer := NewConfigServer(config.ServerAddress, config.ServerGrpcDialOptions)
	eventServer := NewEventServer(config.ServerAddress, config.ServerGrpcDialOptions)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	ca := &ClusterAgent{
		grpcServer:   grpcServer,
		configServer: configServer,
		eventServer:  eventServer,
	}

	if config.EnableFalcoIntegration {
		var err error

		ca.falcoAlertsSubsriber, err = NewFalcoAlertsSubscriber(config.FalcoClientConfig)

		if err != nil {
			logger.Fatalw("falco: unable to connect to falco grpc server", "err", err)
		}
	}

	return ca
}

func (ca *ClusterAgent) Close() {
	ca.configServer.Close()
	ca.eventServer.Close()

	if ca.falcoAlertsSubsriber != nil {
		ca.falcoAlertsSubsriber.Close()
	}
}

func (ca *ClusterAgent) GetGrpcServer() *grpc.Server {
	return ca.grpcServer
}

func (ca *ClusterAgent) GetFalcoAlertsSubscriber() *FalcoAlertsSubscriber {
	return ca.falcoAlertsSubsriber
}
