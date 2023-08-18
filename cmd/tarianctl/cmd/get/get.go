package get

import (
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
func NewGetCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources from the Tarian Server.",
		Long:  "Get resources from the Tarian Server.",
		Run: func(cobraCmd *cobra.Command, args []string) {
			fmt.Println("get called")
		},
		Args: cobra.MinimumNArgs(1),
	}

	// add subcommands
	getCmd.AddCommand(newGetConstraintsCommand(globalFlags))
	getCmd.AddCommand(newGetActionsCommand(globalFlags))
	getCmd.AddCommand(newGetEventsCommand(globalFlags))

	return getCmd
}
