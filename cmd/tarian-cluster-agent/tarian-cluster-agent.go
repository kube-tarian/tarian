package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/devopstoday11/tarian/pkg/clusteragent"
	"github.com/devopstoday11/tarian/pkg/clusteragent/webhookserver"
	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/go-logr/zapr"
	cli "github.com/urfave/cli/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	defaultPort = "50052"
	defaultHost = ""

	defaultServerAddress = "localhost:50051"
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
		Action: func(ctx *cli.Context) error {
			return ctx.App.Command("run").Run(ctx)
		},
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

	clusterAgent := clusteragent.NewClusterAgent(c.String("server-address"))
	defer clusterAgent.Close()

	grpcServer := clusterAgent.GetGrpcServer()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		grpcServer.GracefulStop()
		wg.Done()
	}()

	logger.Infow("tarian-cluster-agent is listening at", "address", listener.Addr())

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatalw("failed to serve", "err", err)
	}

	wg.Wait()
	logger.Info("tarian-cluster-agent shutdown gracefully")

	return nil
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
		webhookserver.RegisterCertRotator(mgr, isReady, namespace)
	}

	go func() {
		<-isReady
		// register the rest of the controllers after cert is ready
		webhookserver.RegisterControllers(mgr, podAgentContainerConfig)
	}()

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}

	logger.Info("manager shutdown gracefully")
	return nil
}
