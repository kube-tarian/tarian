package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/zapr"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/clusteragent/webhookserver"
	"github.com/kube-tarian/tarian/pkg/logger"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	ctrl "sigs.k8s.io/controller-runtime"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	defaultPort = "50052"
	defaultHost = ""

	defaultServerAddress = "localhost:50051"

	defaultSidekickListenerHttpPort = "8088"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

func main() {
	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarian Cluster Agent",
		Usage:   "The Tarian Cluster Agent is the controller that runs in each kubernetes cluster that controls the pod agents",
		Version: version + " (" + commit + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level: debug, info, warn, error.",
				Value: "info",
			},
			&cli.StringFlag{
				Name:  "log-encoding",
				Usage: "log-encoding: json, console",
				Value: "console",
			},
		},
		DefaultCommand: "run",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the cluster agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host address to listen at",
						Value: defaultHost,
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Host port to listen at",
						Value: defaultPort,
					},
					&cli.StringFlag{
						Name:  "server-address",
						Usage: "Tarian server address to communicate with",
						Value: defaultServerAddress,
					},
					&cli.BoolFlag{
						Name:  "server-tls-enabled",
						Usage: "If enabled, it will communicate with the server using TLS",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "server-tls-ca-file",
						Usage: "The CA the server uses for TLS connection.",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "server-tls-insecure-skip-verify",
						Usage: "If set to true, it will skip server's certificate chain and hostname verification",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "enable-add-constraint",
						Usage: "Enable add constraint RPC. This is needed to support pod agent running in register mode.",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "falco-listener-http-port",
						Usage: "Falco listener http port",
						Value: defaultSidekickListenerHttpPort,
					},
				},
				Action: run,
			},
			{
				Name:  "run-webhook-server",
				Usage: "Run kubernetes admission webhook server",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "port",
						Usage: "Host port to listen at",
						Value: 9443,
					},
					&cli.StringFlag{
						Name:  "pod-agent-container-name",
						Usage: "The name of pod-agent container that will be injected",
						Value: "tarian-pod-agent",
					},
					&cli.StringFlag{
						Name:  "pod-agent-container-image",
						Usage: "The image of pod-agent container that will be injected",
						Value: "localhost:5000/tarian-pod-agent:latest",
					},
					&cli.StringFlag{
						Name:  "cluster-agent-host",
						Usage: "Host address of cluster-agent",
						Value: "tarian-cluster-agent.tarian-system.svc",
					},
					&cli.StringFlag{
						Name:  "cluster-agent-port",
						Usage: "Port of cluster-agent",
						Value: "80",
					},
					&cli.StringFlag{
						Name:  "health-probe-bind-address",
						Usage: "Health probe bind address",
						Value: ":8081",
					},
					&cli.StringFlag{
						Name:  "pod-namespace",
						Usage: "Pod namespace where it runs. This is intended to be set from a downward API.",
						Value: "tarian-system",
					},
					&cli.BoolFlag{
						Name:  "enable-leader-election",
						Usage: "Enable leader election",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "enable-cert-rotator",
						Usage: "Enable cert rotator",
						Value: true,
					},
					&cli.StringFlag{
						Name:  "cert-rotator-secret-name",
						Usage: "The tls secret name to be managed by cert rotator",
						Value: "tarian-webhook-server-cert",
					},
					&cli.StringFlag{
						Name:  "mutating-webhook-configuration-name",
						Usage: "Name of MutatingWebhookConfiguration to which it will inject the ca bundle",
						Value: "tarian-mutating-webhook-configuration",
					},
				},
				Action: runWebhookServer,
			},
		},
	}
}

func run(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	clusteragent.SetLogger(logger)

	listener, err := net.Listen("tcp", c.String("host")+":"+c.String("port"))
	if err != nil {
		logger.Fatalw("failed to listen", "err", err)
	}

	clusterAgentConfig := newClusterAgentConfigFromCliContext(c, logger)
	clusterAgent := clusteragent.NewClusterAgent(clusterAgentConfig)
	defer clusterAgent.Close()

	grpcServer := clusterAgent.GetGrpcServer()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		grpcServer.GracefulStop()
	}()

	go clusterAgent.Run()

	logger.Infow("tarian-cluster-agent is listening at", "address", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatalw("failed to serve", "err", err)
	}

	logger.Info("tarian-cluster-agent shutdown gracefully")

	return nil
}

func newClusterAgentConfigFromCliContext(c *cli.Context, logger *zap.SugaredLogger) *clusteragent.ClusterAgentConfig {
	dialOpts := []grpc.DialOption{}

	if c.Bool("server-tls-enabled") {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		serverCAFile := c.String("server-tls-ca-file")

		if serverCAFile != "" {
			serverCACert, err := os.ReadFile(serverCAFile)
			if err != nil {
				logger.Fatalw("failed to read server tls ca files", "filename", serverCAFile, "err", err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				logger.Errorw("failed to append server ca file")
			}
		}

		tlsConfig := &tls.Config{ServerName: "", RootCAs: certPool}
		tlsConfig.InsecureSkipVerify = c.Bool("server-tls-insecure-skip-verify")

		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	config := &clusteragent.ClusterAgentConfig{
		ServerAddress:         c.String("server-address"),
		ServerGrpcDialOptions: dialOpts,
		EnableAddConstraint:   c.Bool("enable-add-constraint"),
	}

	return config
}

func runWebhookServer(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	clusteragent.SetLogger(logger)

	ctrlLogger := zapr.NewLogger(logger.Desugar())
	ctrl.SetLogger(ctrlLogger)

	podAgentContainerConfig := webhookserver.PodAgentContainerConfig{
		Name:        c.String("pod-agent-container-name"),
		Image:       c.String("pod-agent-container-image"),
		LogEncoding: c.String("log-encoding"),
		Host:        c.String("cluster-agent-host"),
		Port:        c.String("cluster-agent-port"),
	}

	mgr := webhookserver.NewManager(c.Int("port"), c.String("health-probe-bind-address"), c.Bool("enable-leader-election"))

	isReady := make(chan struct{})

	if c.Bool("enable-cert-rotator") {
		namespace := c.String("pod-namespace")
		webhookserver.RegisterCertRotator(mgr, isReady, namespace, c.String("mutating-webhook-configuration-name"), c.String("cert-rotator-secret-name"))
	} else {
		close(isReady)
	}

	go func() {
		<-isReady
		// register the rest of the controllers after cert is ready
		webhookserver.RegisterControllers(mgr, podAgentContainerConfig, logger)
	}()

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal(err, "problem running manager")
	}

	logger.Info("manager shutdown gracefully")
	return nil
}
