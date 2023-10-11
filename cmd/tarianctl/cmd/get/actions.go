package get

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/kube-tarian/tarian/pkg/util"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type actionCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	namespace string
	output    string
}

// actionsCmd represents the actions command
func newGetActionsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &actionCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Get actions from the Tarian Server.",
		Example: `tarianctl get actions
tarianctl get action -o yaml
tctl get a -o yaml
`,
		RunE: cmd.run,
	}

	// add flags
	actionsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "", "Filter by namespace")
	actionsCmd.Flags().StringVarP(&cmd.output, "output", "o", "", "Output format. Valid values: yaml")
	return actionsCmd
}

func (c *actionCommand) run(_ *cobra.Command, args []string) error {
	if c.grpcClient == nil {
		opts, err := util.GetDialOptions(c.logger, c.globalFlags.ServerTLSEnabled, c.globalFlags.ServerTLSInsecureSkipVerify, c.globalFlags.ServerTLSCAFile)
		if err != nil {
			return fmt.Errorf("add constraints: %w", err)
		}

		grpcConn, err := grpc.Dial(c.globalFlags.ServerAddr, opts...)
		if err != nil {
			return fmt.Errorf("add constraints: failed to connect to server: %w", err)
		}
		defer grpcConn.Close()
		c.grpcClient = ugrpc.NewGRPCClient(grpcConn)
	}
	client := c.grpcClient.NewConfigClient()

	request := &tarianpb.GetActionsRequest{}

	if c.namespace != "" {
		request.Namespace = c.namespace
	}

	response, err := client.GetActions(context.Background(), request)
	if err != nil {
		return fmt.Errorf("get actions: failed to get actions: %w", err)
	}

	outputFormat := c.output
	if outputFormat == "" {
		actionsTableOutput(response.GetActions(), c.logger.Out)
	} else if outputFormat == "yaml" {
		err = actionsYamlOutput(response.GetActions(), c.logger)
		if err != nil {
			return fmt.Errorf("get actions: %w", err)
		}
	} else {
		return fmt.Errorf("get actions: invalid output format: %s", outputFormat)
	}

	return nil
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

func actionsTableOutput(actions []*tarianpb.Action, out io.Writer) {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Namespace", "Action Name", "Selector", "Trigger", "Action"})
	table.SetColumnSeparator(" ")
	table.SetCenterSeparator("-")
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, a := range actions {
		table.Append([]string{a.GetNamespace(), a.GetName(), matchLabelsToString(a.GetSelector().GetMatchLabels()), formatActionTrigger(a), a.GetAction()})
	}

	table.Render()
}

func actionsYamlOutput(actions []*tarianpb.Action, logger *logrus.Logger) error {
	logger.SetFormatter(&log.NoTimestampFormatter{})
	for _, action := range actions {
		d, err := yaml.Marshal(action)
		if err != nil {
			log.DefaultFormat()
			return err
		}

		logger.Info(string(d))
		logger.Info("---\n")
	}
	log.DefaultFormat()
	return nil
}
