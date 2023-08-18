package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type importCommand struct {
	globalFlags *flags.GlobalFlags
}

// importCmd represents the import command
func NewImportCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &importCommand{
		globalFlags: globalFlags,
	}

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import resources to the Tarian Server.",
		Long:  "Import resources to the Tarian Server.",
		Run:   cmd.run,
	}

	return importCmd
}

func (c *importCommand) run(cmd *cobra.Command, args []string) {
	logger := logger.GetLogger(c.globalFlags.LogLevel, c.globalFlags.LogEncoding)
	util.SetLogger(logger)

	files := []*os.File{}

	for _, path := range args {
		f, err := os.Open(path)

		if err != nil {
			logger.Fatal(err)
		}

		files = append(files, f)
	}

	opts := util.ClientOptionsFromCliContext(c.globalFlags)
	client, _ := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)

	for _, f := range files {
		importFile(f, client)
		f.Close()
	}
}

func importFile(f *os.File, client tarianpb.ConfigClient) error {
	decoder := yaml.NewDecoder(f)

	imported := 0

	for {
		var constraint tarianpb.Constraint
		err := decoder.Decode(&constraint)
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		req := &tarianpb.AddConstraintRequest{Constraint: &constraint}
		response, err := client.AddConstraint(context.Background(), req)

		if err != nil {
			return err
		}

		if response.GetSuccess() {
			imported++
		}
	}

	if imported > 0 {
		fmt.Println("Imported constraint successfully")
	}

	return nil
}
