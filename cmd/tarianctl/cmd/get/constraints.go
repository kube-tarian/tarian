package get

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type constraintsCommand struct {
	globalFlags *flags.GlobalFlags
	output      string
}

// constraintsCmd represents the constraints command
func newGetConstraintsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &constraintsCommand{
		globalFlags: globalFlags,
	}
	constraintsCmd := &cobra.Command{
		Use:     "constraints",
		Aliases: []string{"constraint", "c"},
		Short:   "Get constraints from the Tarian Server.",
		Long:    "Get constraints from the Tarian Server.",
		Run:     cmd.run,
		Example: `tarianctl get constraints
tarianctl get constraints -o yaml
tctl get c -o yaml
`,
	}

	// add flags
	constraintsCmd.Flags().StringVarP(&cmd.output, "output", "o", "", "Output format. Valid values: yaml")
	return constraintsCmd
}

func (c *constraintsCommand) run(cobraCmd *cobra.Command, args []string) {
	logger := logger.GetLogger(c.globalFlags.LogLevel, c.globalFlags.LogEncoding)
	util.SetLogger(logger)

	opts := util.ClientOptionsFromCliContext(c.globalFlags)
	client, _ := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)
	response, err := client.GetConstraints(context.Background(), &tarianpb.GetConstraintsRequest{})

	if err != nil {
		logger.Fatal(err)
	}

	outputFormat := c.output
	if outputFormat == "" {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Namespace", "Constraint Name", "Selector", "Allowed Processes", "Allowed Files"})
		table.SetColumnSeparator(" ")
		table.SetCenterSeparator("-")
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, c := range response.GetConstraints() {
			table.Append([]string{
				c.GetNamespace(),
				c.GetName(),
				matchLabelsToString(c.GetSelector().GetMatchLabels()),
				allowedProcessesToString(c.GetAllowedProcesses()),
				allowedFilesToString(c.GetAllowedFiles()),
			})
		}

		table.Render()
	} else if outputFormat == "yaml" {
		for _, c := range response.GetConstraints() {
			d, err := yaml.Marshal(c)
			if err != nil {
				logger.Fatal(err)
			}
			fmt.Print(string(d))
			fmt.Println("---")
		}
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
