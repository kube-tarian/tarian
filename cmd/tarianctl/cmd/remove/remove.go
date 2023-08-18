package remove

import (
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
func NewRemoveCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove resources from the Tarian Server.",
		Long:    "Remove resources from the Tarian Server.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("remove called")
		},
		Args: cobra.MinimumNArgs(1),
	}

	// add subcommands
	removeCmd.AddCommand(newRemoveConstraintsCommand(globalFlags))
	removeCmd.AddCommand(newRemoveActionsCommand(globalFlags))
	return removeCmd
}
