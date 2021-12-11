package clusteragent

import (
	"context"
	"os"
	"path/filepath"
	"time"

	falcoclient "github.com/falcosecurity/client-go/pkg/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
	"k8s.io/client-go/informers"
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
	grpcServer            *grpc.Server
	configServer          *ConfigServer
	eventServer           *EventServer
	actionHandler         *actionHandler
	configCache           *ConfigCache
	falcoSidekickListener *FalcoSidekickListener

	k8sInformers informers.SharedInformerFactory
	context      context.Context
	cancelFunc   context.CancelFunc
}

func NewClusterAgent(config *ClusterAgentConfig) *ClusterAgent {
	ctx, cancel := context.WithCancel(context.Background())

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

	grpcConn, err := grpc.Dial(config.ServerAddress, config.ServerGrpcDialOptions...)
	if err != nil {
		logger.Warnw("error creating grpc conn", "err", err)
	}
	configClient := tarianpb.NewConfigClient(grpcConn)
	configCache := NewConfigCache(ctx, configClient)

	ca := &ClusterAgent{
		configCache:   configCache,
		grpcServer:    grpcServer,
		configServer:  configServer,
		eventServer:   eventServer,
		actionHandler: actionHandler,
		k8sInformers:  informers.NewSharedInformerFactory(k8sClientset, 12*time.Hour),
		context:       ctx,
		cancelFunc:    cancel,
	}

	// Not sure why this is needed, but it doesn't work without this.
	ca.k8sInformers.Core().V1().Pods().Informer()

	ca.falcoSidekickListener = NewFalcoSidekickListener(":8088", config.ServerAddress, config.ServerGrpcDialOptions, ca.k8sInformers, ca.configCache, actionHandler)

	return ca
}

func (ca *ClusterAgent) Close() {
	ca.configServer.Close()
	ca.eventServer.Close()

	ca.cancelFunc()
}

func (ca *ClusterAgent) GetGrpcServer() *grpc.Server {
	return ca.grpcServer
}

func (ca *ClusterAgent) Run() {
	go ca.configCache.Run()
	go ca.actionHandler.Run()
	go ca.k8sInformers.Start(ca.context.Done())

	go func() {
		if err := ca.falcoSidekickListener.server.ListenAndServe(); err != nil {
			logger.Fatalw("error running falco sidekick listener", "err", err)
		}
	}()
}
