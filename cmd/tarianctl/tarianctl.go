package main

import (
	"log"
	"os"

	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd"
	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd/add"
	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd/get"
	"github.com/devopstoday11/tarian/pkg/tarianctl/cmd/importcmd"
	cli "github.com/urfave/cli/v2"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

func main() {
	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarianctl",
		Usage:   "tarianctl is the CLI tool to control the Tarian Server.",
		Version: version + " (" + commit + ")",
		Flags:   cmd.CmdFlags(),
		Commands: []*cli.Command{
			get.NewGetCommand(),
			add.NewAddCommand(),
			importcmd.NewImportCommand(),
		},
	}
}
