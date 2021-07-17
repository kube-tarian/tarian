package main

import (
	"log"
	"net"
	"os"

	"github.com/devopstoday11/tarian/pkg/clusteragent"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
		Action: run,
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
		},
	}
}

func run(c *cli.Context) error {
	host := c.String("host")
	if host == "" {
		host = defaultHost
	}

	port := c.String("port")
	if port == "" {
		port = defaultPort
	}

	serverAddress := c.String("server-address")
	if serverAddress == "" {
		serverAddress = defaultServerAddress
	}

	logLevel := c.String("log-level")
	logEncoding := c.String("log-encoding")
	logger := getLogger(logLevel, logEncoding)

	clusteragent.SetLogger(logger)

	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		logger.Fatalw("failed to listen", "err", err)
	}

	clusterAgentServer := clusteragent.NewServer(serverAddress)
	defer clusterAgentServer.Close()

	s := grpc.NewServer()

	tarianpb.RegisterConfigServer(s, clusterAgentServer)
	logger.Infow("tarian-cluster-agent is listening at", "address", listener.Addr())

	if err := s.Serve(listener); err != nil {
		logger.Fatalw("failed to serve", "err", err)
	}

	return nil
}

func getLogger(level string, encoding string) *zap.SugaredLogger {
	zapLevel := zap.InfoLevel
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         encoding,
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()

	if err != nil {
		log.Fatal(err)
	}

	return logger.Sugar()
}
