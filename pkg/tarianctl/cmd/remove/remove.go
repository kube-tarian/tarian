package remove

import (
	cli "github.com/urfave/cli/v2"
)

func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "Remove resources from the Tarian Server.",
		Flags: []cli.Flag{},
		Subcommands: []*cli.Command{
			NewRemoveConstraintsCommand(),
			NewRemoveActionsCommand(),
		},
	}
}
