package cmd

import (
	"os"

	version "github.com/kube-tarian/tarian/cmd"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/add"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/get"
	importcommand "github.com/kube-tarian/tarian/cmd/tarianctl/cmd/import"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/remove"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var globalFlags *flags.GlobalFlags

func newRootCommand(logger *logrus.Logger) *cobra.Command {
	return &cobra.Command{
		Use:           "tarianctl",
		Aliases:       []string{"tctl"},
		Version:       version.GetVersion(),
		Short:         "tarianctl is the CLI tool to control the Tarian Server.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := globalFlags.ValidateGlobalFlags()
			if err != nil {
				return err
			}

			globalFlags.GetFlagValuesFromEnvVar()

			logLevel, _ := logrus.ParseLevel(globalFlags.LogLevel)
			logger.SetLevel(logrus.Level(logLevel))

			if globalFlags.LogFormatter == "json" {
				logger.SetFormatter(&logrus.JSONFormatter{})
			}
			return nil
		},
		Long: `
 _                    _                          _     _
| |_    __ _   _ __  (_)   __ _   _ __     ___  | |_  | |
| __|  / _  | | '__| | |  / _  | | '_ \   / __| | __| | |
| |_  | (_| | | |    | | | (_| | | | | | | (__  | |_  | |
 \__|  \__,_| |_|    |_|  \__,_| |_| |_|  \___|  \__| |_|

tarianctl is the CLI tool to control the Tarian Server.
`,
	}
}

func buildRootCommand(logger *logrus.Logger) *cobra.Command {
	rootCmd := newRootCommand(logger)

	// Add global flags
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)

	rootCmd.SetVersionTemplate("tarianctl version: {{.Version}}\n")

	// Add subcommand to the root command
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(get.NewGetCommand(globalFlags))
	rootCmd.AddCommand(add.NewAddCommand(globalFlags))
	rootCmd.AddCommand(remove.NewRemoveCommand(globalFlags))
	rootCmd.AddCommand(importcommand.NewImportCommand(globalFlags))
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
