package add

import (
	"errors"
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

// NewAddCommand creates a new `add` command
func NewAddCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"create"},
		Short:   "Add resources to the Tarian Server.",
		Long:    "Add resources to the Tarian Server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				err := errors.New(`tarianctl needs exactly one argument, use "tarianctl add --help" for command usage`)
				return fmt.Errorf("add: %w", err)
			}
			return nil
		},
	}

	// add subcommands
	addCmd.AddCommand(newAddConstraintCommand(globalFlags))
	addCmd.AddCommand(newAddActionCommand(globalFlags))
	return addCmd
}
