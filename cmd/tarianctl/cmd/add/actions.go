package add

import (
	"context"
	"errors"
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"google.golang.org/grpc"

	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/kube-tarian/tarian/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type actionCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	name              string
	namespace         string
	matchLabels       []string
	action            string
	dryRun            bool
	onViolatedProcess bool
	onViolatedFile    bool
	onFalcoAlert      string
}

// actionsCmd represents the actions command
func newAddActionCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &actionCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Add an action to the Tarian Server.",
		RunE:    cmd.run,
		Example: "tarianctl add action --name NAME --namespace NAMESPACE --match-labels=KEY_1=VAL_1,... --action=delete-pod",
	}

	// add flags
	actionsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "default", "The namespace scope for the action submitted")
	actionsCmd.Flags().StringVar(&cmd.name, "name", "", "The name scope for the action submitted")
	actionsCmd.Flags().StringSliceVar(&cmd.matchLabels, "match-labels", []string{}, "The matchLabels selector for the action submitted. `KEY_1=VAL_1` ... KEY_N=VAL_N")
	actionsCmd.Flags().StringVar(&cmd.action, "action", "", "The action to run on event. Valid values: delete-pod")
	_ = actionsCmd.MarkFlagRequired("action")
	actionsCmd.Flags().BoolVar(&cmd.dryRun, "dry-run", false, "If true, only print the action without sending it to tarian server")
	actionsCmd.Flags().BoolVar(&cmd.onViolatedProcess, "on-violated-process", true, "If true, the action will run on violated process")
	actionsCmd.Flags().BoolVar(&cmd.onViolatedFile, "on-violated-file", true, "If true, the action will run on violated file")
	actionsCmd.Flags().StringVar(&cmd.onFalcoAlert, "on-falco-alert", "", "If specified, the action will run on falco alert with the specified priority and above. Valid values: alert, critical, emergency")
	return actionsCmd
}

func (c *actionCommand) run(_ *cobra.Command, args []string) error {
	// TODO: Remove this check when we support more actions
	if c.action != "delete-pod" {
		c.logger.Errorf("invalid action: %s", c.action)
		return fmt.Errorf("add action: invalid action: %s", c.action)
	}

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
	configClient := c.grpcClient.NewConfigClient()

	req := &tarianpb.AddActionRequest{
		Action: &tarianpb.Action{
			Kind:      tarianpb.KindAction,
			Namespace: c.namespace,
			Name:      c.name,
			Selector: &tarianpb.Selector{
				MatchLabels: matchLabelsFromString(c.matchLabels),
			},
			OnViolatedProcess: c.onViolatedProcess,
			OnViolatedFile:    c.onViolatedFile,
			Action:            c.action,
		},
	}

	if c.onFalcoAlert != "" {
		req.Action.OnFalcoAlert = true
		falcoAlert := map[string]bool{
			"alert":     true,
			"critical":  true,
			"emergency": true,
		}
		if !falcoAlert[c.onFalcoAlert] {
			c.logger.Errorf("invalid falco alert: %s", c.onFalcoAlert)
			return fmt.Errorf("add action: invalid falco alert: %s", c.onFalcoAlert)
		}
		req.Action.FalcoPriority = tarianpb.FalcoPriorityFromString(c.onFalcoAlert)
	}

	if c.dryRun {
		d, err := yaml.Marshal(req.GetAction())
		if err != nil {
			return fmt.Errorf("add action: %w", err)
		}
		c.logger.Info(string(d))
	} else {
		response, err := configClient.AddAction(context.Background(), req)
		if err != nil {
			return fmt.Errorf("add action: failed to add action: %w", err)
		}

		if response.GetSuccess() {
			c.logger.Info("Action was added successfully")
		} else {
			err := errors.New("failed to add action")
			return fmt.Errorf("add action: %w", err)
		}
	}
	return nil
}
