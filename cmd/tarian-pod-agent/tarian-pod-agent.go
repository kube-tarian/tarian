package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	defaultClusterAgentPort = "50052"
	defaultClusterAgentHost = ""
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

var grpcConn *grpc.ClientConn

func main() {
	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarian Pod Agent",
		Usage:   "The Tarian pod agent is the component which runs as a sidecar to monitor your main container.",
		Version: version + " (" + commit + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level: debug, info, warn, error",
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
				Usage: "Run the pod agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host address of the cluster agent to communicate with",
						Value: defaultClusterAgentHost,
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Host port of the cluster agent to communicate with",
						Value: defaultClusterAgentPort,
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
		host = defaultClusterAgentHost
	}

	port := c.String("port")
	if port == "" {
		port = defaultClusterAgentPort
	}

	logLevel := c.String("log-level")
	logEncoding := c.String("log-encoding")
	logger := getLogger(logLevel, logEncoding)

	logger.Infow("tarian-pod-agent is running")

	var err error

	grpcConn, err = grpc.Dial(host+":"+port, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}

	defer grpcConn.Close()

	client := tarianpb.NewConfigClient(grpcConn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		r, err := client.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: "default"})

		if err != nil {
			logger.Fatalw("error while getting constraints from the cluster agent", "err", err)
		}

		logger.Infow("received constraints from the cluster agent", "constraint", r.GetConstraints())

		cancel()
		time.Sleep(3 * time.Second)
	}
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
