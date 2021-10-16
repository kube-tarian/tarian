package get

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/olekukonko/tablewriter"
	cli "github.com/urfave/cli/v2"
)

func NewGetEventsCommand() *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "Get events from the Tarian Server.",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:  "limit",
				Usage: "Limit the events returned from the server",
				Value: 200,
			},
		},
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			opts := util.ClientOptionsFromCliContext(c)
			client, _ := client.NewEventClient(c.String("server-address"), opts...)

			response, err := client.GetEvents(context.Background(), &tarianpb.GetEventsRequest{Limit: uint32(c.Uint("limit"))})

			if err != nil {
				logger.Fatal(err)
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Time", "Namespace", "Pod", "Events"})
			table.SetColWidth(80)
			table.SetColumnSeparator(" ")
			table.SetCenterSeparator("-")
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetReflowDuringAutoWrap(false)

			for _, e := range response.GetEvents() {
				for _, t := range e.GetTargets() {
					evt := strings.Builder{}
					if t.GetViolatedProcesses() != nil {
						evt.WriteString("violated processes\n")
						evt.WriteString(violatedProcessesToString(t.GetViolatedProcesses()))
					}

					if t.GetViolatedFiles() != nil {
						evt.WriteString("violated files\n")
						evt.WriteString(violatedFilesToString(t.GetViolatedFiles()))
					}

					if t.GetFalcoAlert() != nil {
						evt.WriteString("falco alert\n")
						evt.WriteString(falcoAlertToString(t.GetFalcoAlert()))
					}

					if e.GetType() == tarianpb.EventTypePodDeleted {
						evt.WriteString("pod deleted")
					}

					evt.WriteString("\n")

					table.Append(
						[]string{
							e.GetServerTimestamp().AsTime().Format(time.RFC3339),
							t.GetPod().GetNamespace(),
							t.GetPod().GetName(),
							evt.String(),
						},
					)
				}
			}

			table.Render()

			return nil
		},
	}
}

func violatedProcessesToString(processes []*tarianpb.Process) string {
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
		str.WriteString("name=")
		str.WriteString(f.GetName())
		str.WriteString(" actual-sha256=")
		str.WriteString(f.GetActualSha256Sum())
		str.WriteString(" expected-sha256=")
		str.WriteString(f.GetExpectedSha256Sum())

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

func falcoAlertToString(f *tarianpb.FalcoAlert) string {
	if f == nil {
		return ""
	}

	return f.GetPriority().ToString() + ": " + f.GetOutput()
}
