package add

import (
	"context"
	"fmt"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func NewAddActionCommand() *cli.Command {
	return &cli.Command{
		Name:      "action",
		Usage:     "Add an action to the Tarian Server.",
		UsageText: `tarianctl add action --name NAME --namespace NAMESPACE --match-labels=KEY_1=VAL_1,... --action=delete-pod`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the action submitted",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "The name scope for the action submitted",
				Value:    "",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "match-labels",
				Usage: "The matchLabels selector for the action submitted. `KEY_1=VAL_1` ... KEY_N=VAL_N",
			},
			&cli.StringFlag{
				Name:     "action",
				Usage:    "The action to run on event. Valid values: delete-pod",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "If true, only print the action without sending it to tarian server",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "on-violated-process",
				Usage: "If true, the action will run on violated process",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "on-violated-file",
				Usage: "If true, the action will run on violated file",
				Value: true,
			},
			&cli.StringFlag{
				Name:  "on-falco-alert",
				Usage: "If specified, the action will run on falco alert with the specified priority and above. Valid values: alert, critical, emergency",
			},
		},
		Action: runAddAction,
	}
}

func runAddAction(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	util.SetLogger(logger)

	opts := util.ClientOptionsFromCliContext(c)
	configClient, err := client.NewConfigClient(c.String("server-address"), opts...)
	if err != nil {
		logger.Fatal(err)
	}

	req := &tarianpb.AddActionRequest{
		Action: &tarianpb.Action{
			Kind:      tarianpb.KindAction,
			Namespace: c.String("namespace"),
			Name:      c.String("name"),
			Selector: &tarianpb.Selector{
				MatchLabels: matchLabelsFromString(c.String("match-labels")),
			},
			OnViolatedProcess: c.Bool("on-violated-process"),
			OnViolatedFile:    c.Bool("on-violated-file"),
			Action:            c.String("action"),
		},
	}

	if c.String("on-falco-alert") != "" {
		req.Action.OnFalcoAlert = true
		req.Action.FalcoPriority = tarianpb.FalcoPriorityFromString(c.String("on-falco-alert"))
	}

	if req != nil {
		if c.Bool("dry-run") {
			d, err := yaml.Marshal(req.GetAction())
			if err != nil {
				logger.Fatal(err)
			}

			fmt.Println(string(d))
		} else {
			response, err := configClient.AddAction(context.Background(), req)

			if err != nil {
				logger.Fatal(err)
			}

			if response.GetSuccess() {
				logger.Info("Action was added successfully")
			} else {
				logger.Fatal("failed to add Action")
			}
		}
	} else {
		fmt.Println("No new Action")
	}

	return nil
}
