package add

import (
	"context"
	"fmt"
	"strings"

	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
)

func NewAddConstraintCommand() *cli.Command {
	return &cli.Command{
		Name:  "constraint",
		Usage: "Add a constraint to the Tarian Server.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the constraint submitted",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:     "match-labels",
				Usage:    "The matchLabels selector for the constraint submitted. `KEY_1=VAL_1` ... KEY_N=VAL_N",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "allowed-processes",
				Usage:    "The allowed processes for the constraint submitted. `REGEX_1` ... REGEX_N",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			client, _ := client.NewConfigClient(c.String("server-address"))

			req := &tarianpb.AddConstraintRequest{
				Constraint: &tarianpb.Constraint{
					Namespace: c.String("namespace"),
					Selector: &tarianpb.Selector{
						MatchLabels: matchLabelsFromString(c.String("match-labels")),
					},
					AllowedProcesses: allowedProcessesFromString(c.String("allowed-processes")),
				},
			}

			response, err := client.AddConstraint(context.Background(), req)

			if err != nil {
				fmt.Println("error")
				return err
			}

			if response.GetSuccess() {
				fmt.Println("Constraint was added successfully.")
			} else {
				fmt.Println("Failed to add Constraint.")
			}

			return nil
		},
	}
}

func matchLabelsFromString(labelsStr string) []*tarianpb.MatchLabel {
	labels := []*tarianpb.MatchLabel{}

	splitByComma := strings.Split(labelsStr, ",")

	for _, s := range splitByComma {
		idx := strings.Index(s, "=")

		if idx < 0 {
			continue
		}

		key := s[:idx]
		value := strings.Trim(s[idx+1:], "\"")

		labels = append(labels, &tarianpb.MatchLabel{Key: key, Value: value})
	}

	return labels
}

func allowedProcessesFromString(str string) []*tarianpb.AllowedProcessRule {
	allowedProcesses := []*tarianpb.AllowedProcessRule{}

	splitByComma := strings.Split(str, ",")

	for _, s := range splitByComma {
		token := strings.Trim(s, "\" ")

		allowedProcesses = append(allowedProcesses, &tarianpb.AllowedProcessRule{Regex: &token})
	}

	return allowedProcesses
}
