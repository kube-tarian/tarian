package main

import (
	"log"
	"os"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/podagent"
	cli "github.com/urfave/cli/v2"
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

	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	podagent.SetLogger(logger)

	logger.Infow("tarian-pod-agent is running")

	agent := podagent.NewPodAgent(host + ":" + port)
	agent.Run()

	return nil
}
