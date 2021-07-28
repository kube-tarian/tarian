package main

import (
	"log"
	"os"

	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd/add"
	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd/get"
	cli "github.com/urfave/cli/v2"
)

const (
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
		Name:    "Tarianctl",
		Usage:   "tarianctl is the CLI tool to control the Tarian Server.",
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
			&cli.StringFlag{
				Name:  "server-address",
				Usage: "Tarian server address to communicate with",
				Value: defaultServerAddress,
			},
		},
		Commands: []*cli.Command{
			get.NewGetCommand(),
			add.NewAddCommand(),
		},
	}
}
