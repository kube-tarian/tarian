package get

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianctl/util"
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
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)
			response, err := client.GetConstraints(context.Background(), &tarianpb.GetConstraintsRequest{})

			if err != nil {
				logger.Fatal(err)
			}

			outputFormat := c.String("output")
			if outputFormat == "" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Namespace", "Selector", "Allowed Processes", "Allowed Files"})
				table.SetColumnSeparator(" ")
				table.SetCenterSeparator("-")
				table.SetAlignment(tablewriter.ALIGN_LEFT)

				for _, c := range response.GetConstraints() {
					table.Append([]string{c.GetNamespace(), matchLabelsToString(c.GetSelector().GetMatchLabels()), allowedProcessesToString(c.GetAllowedProcesses()), allowedFilesToString(c.GetAllowedFiles())})
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

	for i, r := range rules {
		str.WriteString("regex:")
		str.WriteString(r.GetRegex())

		if i < len(rules)-1 {
			str.WriteString(",")
		}
	}

	return str.String()
}

func allowedFilesToString(rules []*tarianpb.AllowedFileRule) string {
	str := strings.Builder{}

	for i, r := range rules {
		str.WriteString(r.GetName())
		str.WriteString(":")
		str.WriteString(r.GetSha256Sum())

		if i < len(rules)-1 {
			str.WriteString(",")
		}
	}

	return str.String()
}
