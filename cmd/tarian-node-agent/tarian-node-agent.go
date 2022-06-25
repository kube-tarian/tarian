package main

import "C"

import (
	"fmt"
	"log"
	"os"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/nodeagent"
	cli "github.com/urfave/cli/v2"
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
		Name:    "Tarian Node Agent",
		Usage:   "The Tarian Node Agent is the component which runs as a daemonset, detecting new processes from all containers in the node.",
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
		Action: func(ctx *cli.Context) error {
			return ctx.App.Command("run").Run(ctx)
		},
		Commands: []*cli.Command{
			{
				Name:   "run",
				Usage:  "Run the node agent",
				Action: run,
			},
		},
	}
}

func run(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	nodeagent.SetLogger(logger)

	if !isDebugFsMounted() {
		logger.Infow("debugfs is not mounted, will try to mount")

		err := mountDebugFs()
		if err != nil {
			logger.Fatal(err)
		}

		logger.Infow("successfully mounted debugfs", "path", DebugFSRoot)
	}

	captureExec, err := nodeagent.NewCaptureExec()
	if err != nil {
		logger.Fatal(err)
	}

	execEvent := captureExec.GetEventsChannel()
	go captureExec.Start()

	logger.Infow("tarian-node-agent is running")
	logger.Infow("detecting new processes")

	for {
		e := <-execEvent

		fmt.Printf("%d %s %s %s %s %s\n", e.Pid, e.Comm, e.Filename, e.ContainerID, e.K8sPodName, e.K8sNamespace)
	}

	return nil
}
