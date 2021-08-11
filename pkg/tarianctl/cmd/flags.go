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
		&cli.BoolFlag{
			Name:  "server-tls-enabled",
			Usage: "If enabled, it will communicate with the server using TLS",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "server-tls-insecure-skip-verify",
			Usage: "If set to true, it will skip server's certificate chain and hostname verification",
			Value: true,
		},
	}
}
