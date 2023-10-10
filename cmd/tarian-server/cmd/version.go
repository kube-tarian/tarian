package cmd

import (
	version "github.com/kube-tarian/tarian/cmd"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Args:  cobra.NoArgs,
	Short: "Prints version of tarian server",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.GetLogger()
		logger.Infof("tarian server version: %s\n", version.GetVersion())
	},
}
