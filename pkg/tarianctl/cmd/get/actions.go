package get

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	cli "github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func NewGetActionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "actions",
		Usage: "Get actions from the Tarian Server.",
		Flags: []cli.Flag{&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format. Valid values: yaml",
			Value:   "",
		}},
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewConfigClient(c.String("server-address"), opts...)
			response, err := client.GetActions(context.Background(), &tarianpb.GetActionsRequest{})

			if err != nil {
				logger.Fatal(err)
			}

			outputFormat := c.String("output")
			if outputFormat == "" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Namespace", "Action Name", "Selector", "Trigger", "Action"})
				table.SetColumnSeparator(" ")
				table.SetCenterSeparator("-")
				table.SetAlignment(tablewriter.ALIGN_LEFT)

				for _, a := range response.GetActions() {
					table.Append([]string{a.GetNamespace(), a.GetName(), matchLabelsToString(a.GetSelector().GetMatchLabels()), formatActionTrigger(a), a.GetAction()})
				}

				table.Render()
			} else if outputFormat == "yaml" {
				for _, c := range response.GetActions() {
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

func formatActionTrigger(action *tarianpb.Action) string {
	delimiter := ", "

	str := strings.Builder{}
	if action.GetOnViolatedProcess() {
		str.WriteString("onViolatedProcess")
	}

	if action.GetOnViolatedFile() {
		if str.Len() > 0 {
			str.WriteString(delimiter)
		}

		str.WriteString("onViolatedFile")
	}

	if action.GetOnFalcoAlert() {
		if str.Len() > 0 {
			str.WriteString(delimiter)
		}

		str.WriteString("onFalcoAlert=")
		str.WriteString(action.GetFalcoPriority().ToString())
	}

	return str.String()
}
