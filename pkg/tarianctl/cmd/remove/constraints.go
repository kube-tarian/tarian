package remove

import (
	"context"
	"fmt"
	"time"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
)

func NewRemoveConstraintsCommand() *cli.Command {
	return &cli.Command{
		Name:      "constraints",
		Aliases:   []string{"constraint"},
		Usage:     "Remove constraints from the Tarian Server.",
		UsageText: "Tarianctl remove constraints [command options] names...",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the constraint to be removed",
				Value:   "default",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				cli.ShowSubcommandHelpAndExit(c, 1)
			}

			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)

			for _, name := range c.Args().Slice() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				response, err := client.RemoveConstraint(ctx, &tarianpb.RemoveConstraintRequest{Namespace: c.String("namespace"), Name: name})
				cancel()

				if err != nil {
					logger.Fatal(err)
				}

				if response.GetSuccess() {
					fmt.Printf("Constraint %s is deleted succesfully\n", name)
				}
			}

			return nil
		},
	}
}
