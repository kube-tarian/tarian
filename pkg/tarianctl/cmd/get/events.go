package get

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianctl/util"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	cli "github.com/urfave/cli/v2"
)

func NewGetEventsCommand() *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "Get events from the Tarian Server.",
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewEventClient(c.String("server-address"), opts...)

			response, err := client.GetEvents(context.Background(), &tarianpb.GetEventsRequest{})

			if err != nil {
				logger.Fatal(err)
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Time", "Namespace", "Name", "Violating Processes", "Violated Files"})
			table.SetColumnSeparator(" ")
			table.SetCenterSeparator("-")
			table.SetAlignment(tablewriter.ALIGN_LEFT)

			for _, e := range response.GetEvents() {
				for _, t := range e.GetTargets() {
					table.Append(
						[]string{
							e.GetServerTimestamp().AsTime().Format(time.RFC3339),
							t.GetPod().GetNamespace(),
							t.GetPod().GetName(),
							violatingProcessesToString(t.GetViolatingProcesses()),
							violatedFilesToString(t.GetViolatedFiles()),
						},
					)
				}
			}

			table.Render()

			return nil
		},
	}
}

func violatingProcessesToString(processes []*tarianpb.Process) string {
	str := strings.Builder{}

	for i, p := range processes {
		str.WriteString(strconv.Itoa(int(p.GetPid())))
		str.WriteString(":")
		str.WriteString(p.GetName())

		if i < len(processes)-1 {
			str.WriteString(", ")
		}

		if i >= 10 {
			str.WriteString("... ")
			str.WriteString(strconv.Itoa(int(len(processes) - i - 1)))
			str.WriteString(" more")
			break
		}
	}

	return str.String()
}

func violatedFilesToString(violatedFiles []*tarianpb.ViolatedFile) string {
	str := strings.Builder{}

	for i, f := range violatedFiles {
		str.WriteString(f.GetName())

		if i < len(violatedFiles)-1 {
			str.WriteString(", ")
		}

		if i >= 10 {
			str.WriteString("... ")
			str.WriteString(strconv.Itoa(int(len(violatedFiles) - i - 1)))
			str.WriteString(" more")
			break
		}
	}

	return str.String()
}
