package remove

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/spf13/cobra"
)

type removeActionsCmd struct {
	globalFlags *flags.GlobalFlags

	namespace string
}

func newRemoveActionsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &removeActionsCmd{
		globalFlags: globalFlags,
	}
	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Remove actions from the Tarian Server.",
		Example: "Tarianctl remove actions [command options] names...",
		Run:     cmd.run,
	}

	// add flags
	actionsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "default", "The namespace scope for the action to be removed")
	return actionsCmd
}

func (c *removeActionsCmd) run(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify the names of the constraint to be removed")
		os.Exit(1)
	}

	logger := logger.GetLogger(c.globalFlags.LogLevel, c.globalFlags.LogEncoding)
	util.SetLogger(logger)

	opts := util.ClientOptionsFromCliContext(c.globalFlags)
	client, _ := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)

	for _, name := range args {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err := client.RemoveAction(ctx, &tarianpb.RemoveActionRequest{Namespace: c.namespace, Name: name})
		cancel()

		if err != nil {
			logger.Fatal(err)
		}

		if response.GetSuccess() {
			fmt.Printf("Action %s is deleted succesfully\n", name)
		}
	}
}
