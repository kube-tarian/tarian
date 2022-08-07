// Package importcmd provides import commands
package importcmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func NewImportCommand() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import resources to the Tarian Server.",
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			files := []*os.File{}

			for _, path := range c.Args().Slice() {
				f, err := os.Open(path)

				if err != nil {
					logger.Fatal(err)
				}

				files = append(files, f)
			}

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)

			for _, f := range files {
				importFile(f, client)
				f.Close()
			}

			return nil
		},
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
