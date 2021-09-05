package add

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/tarianctl/client"
	"github.com/devopstoday11/tarian/pkg/tarianctl/util"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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
				Name:  "name",
				Usage: "The name scope for the constraint submitted",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "match-labels",
				Usage: "The matchLabels selector for the constraint submitted. `KEY_1=VAL_1` ... KEY_N=VAL_N",
			},
			&cli.StringFlag{
				Name:  "allowed-processes",
				Usage: "The allowed processes for the constraint submitted. `REGEX_1` ... REGEX_N",
			},
			&cli.StringFlag{
				Name:  "allowed-file-sha256sums",
				Usage: "The allowed file sha256 sums for the constraint submitted. `PATH_1=SUM_1` ... PATH_N=SUM_N",
			},
			&cli.StringFlag{
				Name:  "from-violated-pod",
				Usage: "Pod name of which recent violations will be converted into a constraint",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "If true, only print the constraint without sending it to tarian server",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
			util.SetLogger(logger)

			opts := util.ClientOptionsFromCliContext(c)
			configClient, err := client.NewConfigClient(c.String("server-address"), opts...)
			if err != nil {
				logger.Fatal(err)
			}

			eventClient, err := client.NewEventClient(c.String("server-address"), opts...)
			if err != nil {
				logger.Fatal(err)
			}

			fromViolatedPod := c.String("from-violated-pod")

			var req *tarianpb.AddConstraintRequest

			if fromViolatedPod != "" {
				constraint := buildConstraintFromViolatedPod(fromViolatedPod, c, eventClient, configClient, logger)

				if constraint != nil {
					req = &tarianpb.AddConstraintRequest{
						Constraint: constraint,
					}
				}
			} else {
				req = &tarianpb.AddConstraintRequest{
					Constraint: &tarianpb.Constraint{
						Kind:      tarianpb.KindConstraint,
						Namespace: c.String("namespace"),
						Name:      c.String("name"),
						Selector: &tarianpb.Selector{
							MatchLabels: matchLabelsFromString(c.String("match-labels")),
						},
						AllowedProcesses: allowedProcessesFromString(c.String("allowed-processes")),
						AllowedFiles:     allowedFilesFromString(c.String("allowed-file-sha256sums")),
					},
				}
			}

			if req != nil {
				if c.Bool("dry-run") {
					d, err := yaml.Marshal(req.GetConstraint())
					if err != nil {
						logger.Fatal(err)
					}

					fmt.Println(string(d))
				} else {
					response, err := configClient.AddConstraint(context.Background(), req)

					if err != nil {
						logger.Fatal(err)
					}

					if response.GetSuccess() {
						logger.Info("Constraint was added successfully")
					} else {
						logger.Fatal("failed to add Constraint")
					}
				}
			} else {
				fmt.Println("No new constraint")
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

func collectEventTargetsByPodName(events []*tarianpb.Event, podName string) []*tarianpb.Target {
	targets := []*tarianpb.Target{}

	for _, event := range events {
		if eventTargets := event.GetTargets(); eventTargets != nil {
			for _, target := range eventTargets {
				pod := target.GetPod()
				if pod == nil {
					continue
				}

				if pod.GetName() == podName {
					targets = append(targets, target)
				}
			}
		}
	}

	return targets
}

func buildConstraintFromViolatedPod(podName string, c *cli.Context, eventClient tarianpb.EventClient, configClient tarianpb.ConfigClient, logger *zap.SugaredLogger) *tarianpb.Constraint {
	// Pull recent violations
	// TODO: filter namespace
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	resp, err := eventClient.GetEvents(ctx, &tarianpb.GetEventsRequest{Limit: 1000})
	cancel()

	if err != nil {
		logger.Fatal(err)
	}

	targets := []*tarianpb.Target{}
	if events := resp.GetEvents(); events != nil {
		targets = collectEventTargetsByPodName(events, podName)
	}

	if len(targets) == 0 {
		return nil
	}

	// build process rules
	allowedProcesses := []*tarianpb.AllowedProcessRule{}
	for _, target := range targets {
		for _, p := range target.ViolatingProcesses {
			processName := p.GetName()
			allowedProcesses = append(allowedProcesses, &tarianpb.AllowedProcessRule{Regex: &processName})
		}
	}

	// build files rules
	allowedFiles := []*tarianpb.AllowedFileRule{}
	for _, target := range targets {
		for _, f := range target.ViolatedFiles {
			fileName := f.GetName()
			actualSha := f.GetActualSha256Sum()
			allowedFiles = append(allowedFiles, &tarianpb.AllowedFileRule{Name: fileName, Sha256Sum: &actualSha})
		}
	}

	if targets[0].GetPod() == nil {
		return nil
	}

	labels := targets[0].GetPod().GetLabels()

	if labels == nil {
		return nil
	}

	matchLabels := []*tarianpb.MatchLabel{}
	for _, l := range labels {
		matchLabels = append(matchLabels, &tarianpb.MatchLabel{Key: l.GetKey(), Value: l.GetValue()})
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	constraintResponse, _ := configClient.GetConstraints(ctx2, &tarianpb.GetConstraintsRequest{Namespace: c.String("namespace"), Labels: labels})
	cancel2()

	if constraints := constraintResponse.GetConstraints(); constraints != nil {
		allowedProcesses, allowedFiles = deduplicateRules(allowedProcesses, allowedFiles, constraintResponse.GetConstraints())
	}

	if len(allowedProcesses) == 0 && len(allowedFiles) == 0 {
		return nil
	}

	constraintName := c.String("name")
	if constraintName == "" {
		constraintName = podName + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	constraint := &tarianpb.Constraint{
		Kind:      tarianpb.KindConstraint,
		Namespace: c.String("namespace"),
		Name:      constraintName,
		Selector: &tarianpb.Selector{
			MatchLabels: matchLabels,
		},
		AllowedProcesses: allowedProcesses,
		AllowedFiles:     allowedFiles,
	}

	return constraint
}

func deduplicateRules(allowedProcesses []*tarianpb.AllowedProcessRule, allowedFiles []*tarianpb.AllowedFileRule, constraints []*tarianpb.Constraint) ([]*tarianpb.AllowedProcessRule, []*tarianpb.AllowedFileRule) {
	// deduplicate process rules
	for _, allowedProcess := range allowedProcesses {
		found := false
		newAllowedProcesses := allowedProcesses[:0]

		for _, c := range constraints {
			if c.GetAllowedProcesses() != nil {
				for _, p := range c.GetAllowedProcesses() {
					if allowedProcess.GetRegex() == p.GetRegex() {
						found = true
						break
					}
				}
			}

			if found {
				break
			}
		}

		if !found {
			newAllowedProcesses = append(newAllowedProcesses, allowedProcess)
		}

		allowedProcesses = newAllowedProcesses
	}

	// deduplicate file rules
	for _, allowedFile := range allowedFiles {
		found := false
		newAllowedFiles := allowedFiles[:0]

		for _, c := range constraints {
			if c.GetAllowedFiles() != nil {
				for _, f := range c.GetAllowedFiles() {
					if allowedFile.GetName() == f.GetName() && allowedFile.GetSha256Sum() == f.GetSha256Sum() {
						found = true
						break
					}
				}
			}

			if found {
				break
			}
		}

		if !found {
			newAllowedFiles = append(newAllowedFiles, allowedFile)
		}

		allowedFiles = newAllowedFiles
	}

	return allowedProcesses, allowedFiles
}
