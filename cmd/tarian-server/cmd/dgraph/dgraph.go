package dgraph

import (
	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd/flags"
	"github.com/spf13/cobra"
)

// NewDgraphCommand creates a new `dgraph` command
func NewDgraphCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	dgraphCmd := &cobra.Command{
		Use:   "dgraph",
		Short: "Command group related to Dgraph database",
	}

	// Add subcommand to the root command
	dgraphCmd.AddCommand(newApplySchemaCommand(globalFlags))
	return dgraphCmd
}
