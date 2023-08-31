package cmd

import (
	"fmt"

	version "github.com/kube-tarian/tarian/cmd"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Args:  cobra.NoArgs,
	Short: "Prints version of tarian-node-agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tarian node agent version: %s\n", version.GetVersion())
	},
}
