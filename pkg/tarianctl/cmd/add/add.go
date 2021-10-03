package add

import (
	cli "github.com/urfave/cli/v2"
)

func NewAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add resources to the Tarian Server.",
		Flags: []cli.Flag{},
		Subcommands: []*cli.Command{
			NewAddConstraintCommand(),
			NewAddActionCommand(),
		},
	}
}
