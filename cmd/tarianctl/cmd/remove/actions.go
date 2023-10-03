package remove

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"

	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type removeActionsCmd struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	namespace string
}

func newRemoveActionsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &removeActionsCmd{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	actionsCmd := &cobra.Command{
		Use:     "actions",
		Aliases: []string{"action", "a"},
		Short:   "Remove actions from the Tarian Server.",
		Example: "Tarianctl remove actions [command options] names...",
		RunE:    cmd.run,
	}

	// add flags
	actionsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "default", "The namespace scope for the action to be removed")
	return actionsCmd
}

func (c *removeActionsCmd) run(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := errors.New("please specify the name of the action to be removed")
		return fmt.Errorf("remove action: %w", err)
	}

	opts, err := util.ClientOptionsFromCliContext(c.logger, c.globalFlags)
	if err != nil {
		return fmt.Errorf("remove action: %w", err)
	}

	grpcConn, err := grpc.Dial(c.globalFlags.ServerAddr, opts...)
	if err != nil {
		return fmt.Errorf("remove action: failed to connect to server: %w", err)
	}
	defer grpcConn.Close()

	c.grpcClient = ugrpc.NewGRPCClient(grpcConn)
	client := c.grpcClient.NewConfigClient()

	for _, name := range args {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err := client.RemoveAction(ctx, &tarianpb.RemoveActionRequest{Namespace: c.namespace, Name: name})
		cancel()
		if err != nil {
			return fmt.Errorf("remove action: %w", err)
		}

		if response.GetSuccess() {
			c.logger.Infof("Action '%s' is deleted successfully\n", name)
		} else {
			c.logger.Warnf("Action '%s' is not deleted\n", name)
		}
	}
	return nil
}
