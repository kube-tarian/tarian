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

func NewRemoveActionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "actions",
		Usage: "Remove actions from the Tarian Server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the action to be removed",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "The name of the action to be removed",
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
			response, err := client.RemoveAction(ctx, &tarianpb.RemoveActionRequest{Namespace: c.String("namespace"), Name: c.String("name")})
			cancel()

			if err != nil {
				logger.Fatal(err)
			}

			if response.GetSuccess() {
				fmt.Println("Action is deleted succesfully")
			}

			return nil
		},
	}
}
