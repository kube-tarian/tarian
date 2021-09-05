package remove

import (
	"context"
	"fmt"
	"time"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianctl/util"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
)

func NewRemoveConstraintsCommand() *cli.Command {
	return &cli.Command{
		Name:  "constraints",
		Usage: "Remove constraints from the Tarian Server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the constraint to be removed",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "The name scope for the constraint to be removed",
				Value:    "",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			response, err := client.RemoveConstraint(ctx, &tarianpb.RemoveConstraintRequest{Namespace: c.String("namespace"), Name: c.String("name")})
			cancel()

			if err != nil {
				logger.Fatal(err)
			}

			if response.GetSuccess() {
				fmt.Println("Constraint is deleted succesfully")
			}

			return nil
		},
	}
}
