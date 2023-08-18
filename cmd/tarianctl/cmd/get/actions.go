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

type actionCommand struct {
	globalFlags *flags.GlobalFlags
	namespace   string
	output      string
}

// actionsCmd represents the actions command
func newGetActionsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &actionCommand{
		globalFlags: globalFlags,
	}
	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Get actions from the Tarian Server.",
		Example: `tarianctl get actions
tarianctl get action -o yaml
tctl get a -o yaml
`,
		Run: cmd.run,
	}

	// add flags
	actionsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "", "Filter by namespace")
	actionsCmd.Flags().StringVarP(&cmd.output, "output", "o", "", "Output format. Valid values: yaml")
	return actionsCmd
}

func (c *actionCommand) run(cmd *cobra.Command, args []string) {
	logger := logger.GetLogger(c.globalFlags.LogLevel, c.globalFlags.LogEncoding)
	util.SetLogger(logger)

	opts := util.ClientOptionsFromCliContext(c.globalFlags)
	client, _ := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)

	request := &tarianpb.GetActionsRequest{}

	ns := c.namespace
	if ns != "" {
		request.Namespace = ns
	}

	response, err := client.GetActions(context.Background(), request)

	if err != nil {
		logger.Fatal(err)
	}

	outputFormat := c.output
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
				logger.Fatal(err)
			}

			fmt.Print(string(d))
			fmt.Println("---")
		}
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
