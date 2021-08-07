package cmd

import cli "github.com/urfave/cli/v2"

const (
	defaultServerAddress = "localhost:50051"
)

func CmdFlags() []cli.Flag {
	return []cli.Flag{
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
	}
}
