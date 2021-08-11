package add

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianctl/util"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
)

func NewAddConstraintCommand() *cli.Command {
	return &cli.Command{
		Name:  "constraint",
		Usage: "Add a constraint to the Tarian Server.",
		UsageText: `tarianctl add constraint --name NAME --namespace NAMESPACE --match-labels=KEY_1=VAL_1,... --allowed-processes=REGEX_1,...
   tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
   tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-file-sha256sums=/etc/nginx/nginx.conf=c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "The namespace scope for the constraint submitted",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:     "name",
				Usage:    "The name scope for the constraint submitted",
				Value:    "",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "match-labels",
				Usage:    "The matchLabels selector for the constraint submitted. `KEY_1=VAL_1` ... KEY_N=VAL_N",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "allowed-processes",
				Usage: "The allowed processes for the constraint submitted. `REGEX_1` ... REGEX_N",
			},
			&cli.StringFlag{
				Name:  "allowed-file-sha256sums",
				Usage: "The allowed file sha256 sums for the constraint submitted. `PATH_1=SUM_1` ... PATH_N=SUM_N",
			},
		},
		Action: func(c *cli.Context) error {
			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)

			req := &tarianpb.AddConstraintRequest{
				Constraint: &tarianpb.Constraint{
					Namespace: c.String("namespace"),
					Name:      c.String("name"),
					Selector: &tarianpb.Selector{
						MatchLabels: matchLabelsFromString(c.String("match-labels")),
					},
					AllowedProcesses: allowedProcessesFromString(c.String("allowed-processes")),
					AllowedFiles:     allowedFilesFromString(c.String("allowed-file-sha256sums")),
				},
			}

			response, err := client.AddConstraint(context.Background(), req)

			if err != nil {
				return err
			}

			if response.GetSuccess() {
				fmt.Println("Constraint was added successfully")
			} else {
				return errors.New("failed to add Constraint")
			}

			return nil
		},
	}
}

func matchLabelsFromString(labelsStr string) []*tarianpb.MatchLabel {
	if labelsStr == "" {
		return nil
	}

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
	if str == "" {
		return nil
	}

	allowedProcesses := []*tarianpb.AllowedProcessRule{}

	splitByComma := strings.Split(str, ",")

	for _, s := range splitByComma {
		token := strings.Trim(s, "\" ")

		allowedProcesses = append(allowedProcesses, &tarianpb.AllowedProcessRule{Regex: &token})
	}

	if len(allowedProcesses) == 0 {
		return nil
	}

	return allowedProcesses
}

func allowedFilesFromString(str string) []*tarianpb.AllowedFileRule {
	if str == "" {
		return nil
	}

	allowedFiles := []*tarianpb.AllowedFileRule{}

	splitByComma := strings.Split(str, ",")

	for _, s := range splitByComma {
		idx := strings.Index(s, "=")

		if idx < 0 {
			continue
		}

		name := strings.Trim(s[:idx], "\"")
		value := strings.Trim(s[idx+1:], "\"")

		allowedFiles = append(allowedFiles, &tarianpb.AllowedFileRule{Name: name, Sha256Sum: &value})
	}

	if len(allowedFiles) == 0 {
		return nil
	}

	return allowedFiles
}
