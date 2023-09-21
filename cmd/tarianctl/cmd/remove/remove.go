package remove

import (
	"errors"
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

// NewRemoveCommand creates a new `remove` command
func NewRemoveCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove resources from the Tarian Server.",
		Long:    "Remove resources from the Tarian Server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				err := errors.New("no resource specified, use `tarianctl remove --help` for command usage")
				return fmt.Errorf("remove: %w", err)
			}
			return nil
		},
		Args: cobra.MinimumNArgs(1),
	}

	// add subcommands
	removeCmd.AddCommand(newRemoveConstraintsCommand(globalFlags))
	removeCmd.AddCommand(newRemoveActionsCommand(globalFlags))
	return removeCmd
}
