package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// nolint: gochecknoglobals
var (
	version    = "dev"
	commit     = "main"
	versionStr = version + " (" + commit + ")"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Args:  cobra.NoArgs,
	Short: "Prints version of tarian cluster agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tarian cluster agent version: %s\n", versionStr)
	},
}
