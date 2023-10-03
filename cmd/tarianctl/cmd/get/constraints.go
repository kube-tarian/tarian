package get

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type constraintsCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	output string
}

// constraintsCmd represents the constraints command
func newGetConstraintsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &constraintsCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	constraintsCmd := &cobra.Command{
		Use:     "constraints",
		Aliases: []string{"constraint", "c"},
		Short:   "Get constraints from the Tarian Server.",
		RunE:    cmd.run,
		Example: `tarianctl get constraints
tarianctl get constraints -o yaml
tctl get c -o yaml
`,
	}

	// add flags
	constraintsCmd.Flags().StringVarP(&cmd.output, "output", "o", "", "Output format. Valid values: yaml")
	return constraintsCmd
}

func (c *constraintsCommand) run(cobraCmd *cobra.Command, args []string) error {
	opts, err := util.ClientOptionsFromCliContext(c.logger, c.globalFlags)
	if err != nil {
		return fmt.Errorf("get constraints: %w", err)
	}

	grpcConn, err := grpc.Dial(c.globalFlags.ServerAddr, opts...)
	if err != nil {
		return fmt.Errorf("get constraints: failed to connect to server: %w", err)
	}
	defer grpcConn.Close()

	c.grpcClient = ugrpc.NewGRPCClient(grpcConn)
	client := c.grpcClient.NewConfigClient()

	response, err := client.GetConstraints(context.Background(), &tarianpb.GetConstraintsRequest{})
	if err != nil {
		return fmt.Errorf("get constraints: failed to get constraints: %w", err)
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
		for _, constraint := range response.GetConstraints() {
			d, err := yaml.Marshal(constraint)
			if err != nil {
				return fmt.Errorf("get constraints: %w", err)
			}
			fmt.Print(string(d))
			fmt.Println("---")
		}
	}
	return nil
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
