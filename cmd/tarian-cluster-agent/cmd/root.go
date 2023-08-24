package cmd

import (
	"os"

	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var globalFlags *flags.GlobalFlags

func newRootCommand(logger *logrus.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "tarian-cluster-agent",
		Short:         "Tarian Cluster Agent",
		Long:          "The Tarian Cluster Agent is the controller that runs in each kubernetes cluster that controls the pod agents",
		Version:       versionStr,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := globalFlags.ValidateGlobalFlags()
			if err != nil {
				return err
			}

			logLevel, _ := logrus.ParseLevel(globalFlags.LogLevel)
			logger.SetLevel(logrus.Level(logLevel))

			if globalFlags.LogFormatter == "json" {
				logger.SetFormatter(&logrus.JSONFormatter{
					PrettyPrint: true,
				})
			}
			return nil
		},
	}

	return rootCmd
}

func buildRootCommand(logger *logrus.Logger) *cobra.Command {
	rootCmd := newRootCommand(logger)

	rootCmd.SetVersionTemplate("tarian cluster agent version: {{.Version}}\n")

	// Add global flags
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	// Add subcommand to the root command
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newRunCommand(globalFlags))
	rootCmd.AddCommand(newWebhookServerCommand(globalFlags))

	return rootCmd
}

func Execute() {
	logger := log.GetLogger()
	rootCmd := buildRootCommand(logger)
	err := rootCmd.Execute()
	if err != nil {
		logger.Errorf("command failed: %s", err)
		os.Exit(1)
	}
}
