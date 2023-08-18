package add

import (
	"context"
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type actionCommand struct {
	globalFlags *flags.GlobalFlags

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
	}
	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Add an action to the Tarian Server.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("actions called")
		},
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

func (c *actionCommand) run(cmd *cobra.Command, args []string) {
	logger := logger.GetLogger(c.globalFlags.LogLevel, c.globalFlags.LogEncoding)
	util.SetLogger(logger)

	opts := util.ClientOptionsFromCliContext(c.globalFlags)
	configClient, err := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)
	if err != nil {
		logger.Fatal(err)
	}

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
		req.Action.FalcoPriority = tarianpb.FalcoPriorityFromString(c.onFalcoAlert)
	}

	if c.dryRun {
		d, err := yaml.Marshal(req.GetAction())
		if err != nil {
			logger.Fatal(err)
		}

		fmt.Println(string(d))
	} else {
		response, err := configClient.AddAction(context.Background(), req)

		if err != nil {
			logger.Fatal(err)
		}

		if response.GetSuccess() {
			logger.Info("Action was added successfully")
		} else {
			logger.Fatal("failed to add Action")
		}
	}
}
