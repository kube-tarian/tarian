package clusteragent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ClusterAgentConfig struct {
	ServerAddress         string
	ServerGrpcDialOptions []grpc.DialOption
	EnableAddConstraint   bool
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
	logger       *logrus.Logger
}

func NewClusterAgent(logger *logrus.Logger, config *ClusterAgentConfig) (*ClusterAgent, error) {
	grpcServer := grpc.NewServer()

	configServer, err := NewConfigServer(logger, config.ServerAddress, config.ServerGrpcDialOptions)
	if err != nil {
		return nil, fmt.Errorf("NewClusterAgent: %w", err)
	}
	configServer.EnableAddConstraint(config.EnableAddConstraint)

	var k8sClientConfig *rest.Config
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// In cluster
		var err error
		k8sClientConfig, err = rest.InClusterConfig()
		if err != nil {
			logger.WithError(err).Error("error configuring kubernetes clientset config")
			return nil, fmt.Errorf("NewClusterAgent: %w", err)
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
			logger.WithError(err).Warn("error configuring kubernetes clientset config")
		}
	}

	var k8sClientset *kubernetes.Clientset
	if k8sClientConfig != nil {
		var err error

		k8sClientset, err = kubernetes.NewForConfig(k8sClientConfig)
		if err != nil {
			logger.WithError(err).Warn("error configuring kubernetes clientset")
		}
	}

	actionHandler, err := newActionHandler(logger, config.ServerAddress, config.ServerGrpcDialOptions, k8sClientset)
	if err != nil {
		return nil, fmt.Errorf("NewClusterAgent: %w", err)
	}
	eventServer, err := NewEventServer(logger, config.ServerAddress, config.ServerGrpcDialOptions, actionHandler)
	if err != nil {
		return nil, fmt.Errorf("NewClusterAgent: %w", err)
	}

	tarianpb.RegisterConfigServer(grpcServer, configServer)
	tarianpb.RegisterEventServer(grpcServer, eventServer)

	grpcConn, err := grpc.Dial(config.ServerAddress, config.ServerGrpcDialOptions...)
	if err != nil {
		logger.WithError(err).Warn("error creating grpc conn")
	}

	configClient := tarianpb.NewConfigClient(grpcConn)
	ctx, cancel := context.WithCancel(context.Background())
	configCache := NewConfigCache(ctx, logger, configClient)
	ca := &ClusterAgent{
		configCache:   configCache,
		grpcServer:    grpcServer,
		configServer:  configServer,
		eventServer:   eventServer,
		actionHandler: actionHandler,
		k8sInformers:  informers.NewSharedInformerFactory(k8sClientset, 12*time.Hour),
		context:       ctx,
		cancelFunc:    cancel,
		logger:        logger,
	}

	// Not sure why this is needed, but it doesn't work without this.
	ca.k8sInformers.Core().V1().Pods().Informer()
	ca.falcoSidekickListener, err = NewFalcoSidekickListener(logger, ":8088", config.ServerAddress, config.ServerGrpcDialOptions, ca.k8sInformers, ca.configCache, actionHandler)
	if err != nil {
		return nil, fmt.Errorf("NewClusterAgent: %w", err)
	}
	return ca, nil
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
			ca.logger.WithError(err).Fatal("error running falco sidekick listener")
		}
	}()
}
