package cmd

import (
	"os"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/add"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/get"
	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/remove"
	"github.com/spf13/cobra"
)

var (
	LogLevel                    string
	LogEncoding                 string
	ServerAddr                  string
	ServerTLSEnabled            bool
	ServerTLSCAFile             string
	ServerTLSInsecureSkipVerify bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "tarianctl",
	Aliases: []string{"tctl"},
	Version: versionStr,
	Short:   "tarianctl is the CLI tool to control the Tarian Server.",
	Long: `
 _                    _                          _     _
| |_    __ _   _ __  (_)   __ _   _ __     ___  | |_  | |
| __|  / _  | | '__| | |  / _  | | '_ \   / __| | __| | |
| |_  | (_| | | |    | | | (_| | | | | | | (__  | |_  | |
 \__|  \__,_| |_|    |_|  \__,_| |_| |_|  \___|  \__| |_|

tarianctl is the CLI tool to control the Tarian Server.
`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func addSubCommand(globalFlags *flags.GlobalFlags) {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(get.NewGetCommand(globalFlags))
	rootCmd.AddCommand(add.NewAddCommand(globalFlags))
	rootCmd.AddCommand(remove.NewRemoveCommand(globalFlags))
	rootCmd.AddCommand(NewImportCommand(globalFlags))

	rootCmd.SetVersionTemplate("tarianctl version: {{.Version}}\n")
}

func init() {
	// Add global flags
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags := flags.SetGlobalFlags(persistentFlags)

	// Add subcommand to the root command
	addSubCommand(globalFlags)
}
