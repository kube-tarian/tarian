package get

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	cli "github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func NewGetConstraintsCommand() *cli.Command {
	return &cli.Command{
		Name:  "constraints",
		Usage: "Get constraints from the Tarian Server.",
		Flags: []cli.Flag{&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format. Valid values: yaml",
			Value:   "",
		}},
		Action: func(c *cli.Context) error {
			client, _ := client.NewConfigClient(c.String("server-address"))
			response, err := client.GetConstraints(context.Background(), &tarianpb.GetConstraintsRequest{})

			if err != nil {
				return err
			}

			outputFormat := c.String("output")
			if outputFormat == "" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Namespace", "Selector", "Allowed Processes"})
				table.SetColumnSeparator(" ")
				table.SetCenterSeparator("-")
				table.SetAlignment(tablewriter.ALIGN_LEFT)

				for _, c := range response.GetConstraints() {
					table.Append([]string{c.GetNamespace(), matchLabelsToString(c.GetSelector().GetMatchLabels()), allowedProcessesToString(c.GetAllowedProcesses())})
				}

				table.Render()
			} else if outputFormat == "yaml" {
				for _, c := range response.GetConstraints() {
					d, err := yaml.Marshal(c)
					if err != nil {
						return err
					}

					fmt.Print(string(d))
					fmt.Println("---")
				}
			}

			return nil
		},
	}
}

func matchLabelsToString(labels []*tarianpb.MatchLabel) string {
	if len(labels) == 0 {
		return ""
	}

	str := strings.Builder{}
	str.WriteString("matchLabels:")

	for i, l := range labels {
		str.WriteString(l.GetKey())
		str.WriteString("=")
		str.WriteString(l.GetValue())

		if i < len(labels)-1 {
			str.WriteString(",")
		}
	}

	return str.String()
}

func allowedProcessesToString(rules []*tarianpb.AllowedProcessRule) string {
	str := strings.Builder{}

	for i, l := range rules {
		str.WriteString("regex:")
		str.WriteString(l.GetRegex())

		if i < len(rules)-1 {
			str.WriteString(",")
		}
	}

	return str.String()
}
