package clusteragent

import (
	"flag"
	"os"
	"path/filepath"

	falcoclient "github.com/falcosecurity/client-go/pkg/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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

	var k8sClientConfig *rest.Config
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// In cluster
		var err error

		k8sClientConfig, err = rest.InClusterConfig()

		if err != nil {
			logger.Fatalw("error configuring kubernetes clientset config", "err", err)
		}
	} else {
		var kubeconfig *string
		var err error

		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}

		// use the current context in kubeconfig
		k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			logger.Fatalw("error configuring kubernetes clientset config", "err", err)
		}
	}

	k8sClientset, err := kubernetes.NewForConfig(k8sClientConfig)
	if err != nil {
		logger.Fatalw("error configuring kubernetes clientset", "err", err)
	}

	eventServer := NewEventServer(config.ServerAddress, config.ServerGrpcDialOptions, k8sClientset)

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
