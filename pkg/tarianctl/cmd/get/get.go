package get

import (
	cli "github.com/urfave/cli/v2"
)

func NewGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get resources from the Tarian Server.",
		Flags: []cli.Flag{},
		Subcommands: []*cli.Command{
			NewGetConstraintsCommand(),
		},
	}
}
