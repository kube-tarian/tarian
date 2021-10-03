package clusteragent

import (
	falcoclient "github.com/falcosecurity/client-go/pkg/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

type ClusterAgentConfig struct {
	ServerAddress          string
	ServerGrpcDialOptions  []grpc.DialOption
	EnableFalcoIntegration bool
	EnableAddConstraint    bool
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
	configServer.EnableAddConstraint(config.EnableAddConstraint)

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

		ca.falcoAlertsSubsriber, err = NewFalcoAlertsSubscriber(config.ServerAddress, config.ServerGrpcDialOptions, config.FalcoClientConfig)

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

func (ca *ClusterAgent) LoopSyncActions() error {
	return ca.eventServer.LoopSyncActions()
}
