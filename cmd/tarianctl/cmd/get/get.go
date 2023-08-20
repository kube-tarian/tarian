package get

import (
	"errors"
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

func NewGetCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources from the Tarian Server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				err := errors.New("no resource specified, use `tarianctl get --help` for command usage")
				return fmt.Errorf("get: %w", err)
			}
			return nil
		},
	}

	// add subcommands
	getCmd.AddCommand(newGetConstraintsCommand(globalFlags))
	getCmd.AddCommand(newGetActionsCommand(globalFlags))
	getCmd.AddCommand(newGetEventsCommand(globalFlags))

	return getCmd
}
