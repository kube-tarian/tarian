package clusteragent

import (
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
	actionHandler        *actionHandler
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
		var kubeconfig string
		var err error

		if os.Getenv("KUBECONFIG") != "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		} else {
			kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}

		// use the current context in kubeconfig
		k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Warnw("error configuring kubernetes clientset config", "err", err)
		}
	}

	var k8sClientset *kubernetes.Clientset
	if k8sClientConfig != nil {
		var err error

		k8sClientset, err = kubernetes.NewForConfig(k8sClientConfig)
		if err != nil {
			logger.Warnw("error configuring kubernetes clientset", "err", err)
		}
	}

	actionHandler := newActionHandler(config.ServerAddress, config.ServerGrpcDialOptions, k8sClientset)
	eventServer := NewEventServer(config.ServerAddress, config.ServerGrpcDialOptions, actionHandler)

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	ca := &ClusterAgent{
		grpcServer:    grpcServer,
		configServer:  configServer,
		eventServer:   eventServer,
		actionHandler: actionHandler,
	}

	if config.EnableFalcoIntegration {
		var err error

		ca.falcoAlertsSubsriber, err = NewFalcoAlertsSubscriber(config.ServerAddress, config.ServerGrpcDialOptions, config.FalcoClientConfig, actionHandler)

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

func (ca *ClusterAgent) RunActionHandler() {
	ca.actionHandler.Run()
}
