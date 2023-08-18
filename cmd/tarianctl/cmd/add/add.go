package add

import (
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
func NewAddCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"create"},
		Short:   "Add resources to the Tarian Server.",
		Long:    "Add resources to the Tarian Server.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("add called")
		},
		Args: cobra.MinimumNArgs(1),
	}

	// add subcommands
	addCmd.AddCommand(newAddConstraintCommand(globalFlags))
	addCmd.AddCommand(newAddActionCommand(globalFlags))
	return addCmd
}
