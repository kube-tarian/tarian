package add

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type constraintsCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	name                  string
	namespace             string
	matchLabels           []string
	allowedProcesses      []string
	allowedFileSha256Sums []string
	fromViolatedPod       string
	dryRun                bool
}

// constraintsCmd represents the constraints command
func newAddConstraintCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &constraintsCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	constraintsCmd := &cobra.Command{
		Use:     "constraints",
		Aliases: []string{"constraint", "c"},
		Short:   "Add a constraint to the Tarian Server.",
		Example: `tarianctl add constraint --name NAME --namespace NAMESPACE --match-labels=KEY_1=VAL_1,... --allowed-processes=REGEX_1,...
  
tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx
  
tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-file-sha256sums=/etc/nginx/nginx.conf=c01b39c7a35ccc3b081a3e83d2c71fa9a767ebfeb45c69f08e17dfe3ef375a7b`,
		RunE: cmd.run,
	}

	// add flags
	constraintsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "default", "The namespace scope for the constraint submitted")
	constraintsCmd.Flags().StringVar(&cmd.name, "name", "", "The name scope for the constraint submitted")
	constraintsCmd.Flags().StringArrayVar(&cmd.matchLabels, "match-labels", []string{}, "The matchLabels selector for the constraint submitted. `KEY_1=VAL_1`,...,KEY_N=VAL_N")
	constraintsCmd.Flags().StringArrayVar(&cmd.allowedProcesses, "allowed-processes", []string{}, "The allowed processes for the constraint submitted. `REGEX_1`,...,REGEX_N")
	constraintsCmd.Flags().StringArrayVar(&cmd.allowedFileSha256Sums, "allowed-file-sha256sums", []string{}, "The allowed file sha256 sums for the constraint submitted. `PATH_1=SUM_1`,...,PATH_N=SUM_N")
	constraintsCmd.Flags().StringVar(&cmd.fromViolatedPod, "from-violated-pod", "", "Pod name of which recent violations will be converted into a constraint")
	constraintsCmd.Flags().BoolVar(&cmd.dryRun, "dry-run", false, "If true, only print the constraint without sending it to tarian server")
	return constraintsCmd
}

func (c *constraintsCommand) run(cobraCmd *cobra.Command, args []string) error {
	if c.grpcClient == nil {
		opts, err := util.ClientOptionsFromCliContext(c.logger, c.globalFlags)
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
	configClient := c.grpcClient.NewConfigClient()
	eventClient := c.grpcClient.NewEventClient()

	fromViolatedPod := c.fromViolatedPod
	var req *tarianpb.AddConstraintRequest

	// validate required fields
	if fromViolatedPod == "" && c.name == "" {
		err := errors.New("either constraint name or from-violated-pod is required")
		return fmt.Errorf("add constraints: %w", err)
	}

	if fromViolatedPod != "" && c.name != "" {
		err := errors.New("constraint name and from-violated-pod cannot be used together")
		return fmt.Errorf("add constraints: %w", err)
	}

	if fromViolatedPod != "" {
		constraint, err := c.buildConstraintFromViolatedPod(fromViolatedPod, eventClient, configClient, c.logger)
		if err != nil {
			return fmt.Errorf("add constraints: %w", err)
		}
		if constraint != nil {
			req = &tarianpb.AddConstraintRequest{
				Constraint: constraint,
			}
		}
	} else {
		req = &tarianpb.AddConstraintRequest{
			Constraint: &tarianpb.Constraint{
				Kind:      tarianpb.KindConstraint,
				Namespace: c.namespace,
				Name:      c.name,
				Selector: &tarianpb.Selector{
					MatchLabels: matchLabelsFromString(c.matchLabels),
				},
				AllowedProcesses: allowedProcessesFromString(c.allowedProcesses),
				AllowedFiles:     allowedFilesFromString(c.allowedFileSha256Sums),
			},
		}
	}

	if req != nil {
		if c.dryRun {
			d, err := yaml.Marshal(req.GetConstraint())
			if err != nil {
				return fmt.Errorf("add constraints: %w", err)
			}
			c.logger.Info(string(d))
		} else {
			if c.allowedFileSha256Sums == nil && c.allowedProcesses == nil {
				err := errors.New("no allowed processes or files found, use --allowed-processes or --allowed-file-sha256sums or both")
				return fmt.Errorf("add constraints: %w", err)
			}

			if c.matchLabels == nil {
				err := errors.New("no match labels found, use --match-labels")
				return fmt.Errorf("add constraints: %w", err)
			}

			response, err := configClient.AddConstraint(context.Background(), req)
			if err != nil {
				return fmt.Errorf("add constraints: failed to add constraints: %w", err)
			}

			if response.GetSuccess() {
				c.logger.Info("Constraint was added successfully")
			} else {
				err := errors.New("failed to add Constraint")
				return fmt.Errorf("add constraints: %w", err)
			}
		}
	} else {
		c.logger.Warn("No new constraint")
	}
	return nil
}

func allowedProcessesFromString(strProcesses []string) []*tarianpb.AllowedProcessRule {
	if strProcesses == nil {
		return nil
	}

	allowedProcesses := []*tarianpb.AllowedProcessRule{}

	for _, s := range strProcesses {
		token := strings.Trim(s, "\" ")

		allowedProcesses = append(allowedProcesses, &tarianpb.AllowedProcessRule{Regex: &token})
	}

	if len(allowedProcesses) == 0 {
		return nil
	}

	return allowedProcesses
}

func allowedFilesFromString(strFiles []string) []*tarianpb.AllowedFileRule {
	if strFiles == nil {
		return nil
	}

	allowedFiles := []*tarianpb.AllowedFileRule{}

	for _, s := range strFiles {
		idx := strings.Index(s, "=")

		if idx < 0 || idx == len(s)-1 {
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

func (c *constraintsCommand) buildConstraintFromViolatedPod(podName string, eventClient tarianpb.EventClient, configClient tarianpb.ConfigClient, logger *logrus.Logger) (*tarianpb.Constraint, error) {
	// Pull recent violations
	// TODO: filter namespace
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	resp, err := eventClient.GetEvents(ctx, &tarianpb.GetEventsRequest{Limit: 1000})
	cancel()
	if err != nil {
		return nil, fmt.Errorf("buildConstraintFromViolatedPod: %w", err)
	}

	targets := []*tarianpb.Target{}
	if events := resp.GetEvents(); events != nil {
		targets = collectEventTargetsByPodName(events, podName)
	}

	if len(targets) == 0 {
		err := errors.New("zero target found")
		return nil, fmt.Errorf("buildConstraintFromViolatedPod: %w", err)
	}

	// build process rules
	allowedProcesses := []*tarianpb.AllowedProcessRule{}
	for _, target := range targets {
		for _, p := range target.ViolatedProcesses {
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
		err := errors.New("no pod found")
		return nil, fmt.Errorf("buildConstraintFromViolatedPod: %w", err)
	}

	labels := targets[0].GetPod().GetLabels()
	if labels == nil {
		err := errors.New("no labels found")
		return nil, fmt.Errorf("buildConstraintFromViolatedPod: %w", err)
	}

	ignoredLabel := "pod-template-hash"

	matchLabels := []*tarianpb.MatchLabel{}
	for _, l := range labels {
		if l.GetKey() == ignoredLabel {
			continue
		}

		matchLabels = append(matchLabels, &tarianpb.MatchLabel{Key: l.GetKey(), Value: l.GetValue()})
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	constraintResponse, _ := configClient.GetConstraints(ctx2, &tarianpb.GetConstraintsRequest{Namespace: c.namespace, Labels: labels})
	cancel2()

	if constraints := constraintResponse.GetConstraints(); constraints != nil {
		allowedProcesses, allowedFiles = deduplicateRules(allowedProcesses, allowedFiles, constraintResponse.GetConstraints())
	}

	if len(allowedProcesses) == 0 && len(allowedFiles) == 0 {
		err := errors.New("no allowed processes or files found")
		return nil, fmt.Errorf("add constraints: buildConstraintFromViolatedPod: %w", err)
	}

	constraintName := c.name
	if constraintName == "" {
		constraintName = podName + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	constraint := &tarianpb.Constraint{
		Kind:      tarianpb.KindConstraint,
		Namespace: c.namespace,
		Name:      constraintName,
		Selector: &tarianpb.Selector{
			MatchLabels: matchLabels,
		},
		AllowedProcesses: allowedProcesses,
		AllowedFiles:     allowedFiles,
	}

	return constraint, nil
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
