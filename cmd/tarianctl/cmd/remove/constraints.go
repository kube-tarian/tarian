package remove

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	ugrpc "github.com/kube-tarian/tarian/cmd/tarianctl/util/grpc"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/kube-tarian/tarian/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type removeConstraintsCmd struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	grpcClient ugrpc.Client

	namespace string
}

func newRemoveConstraintsCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &removeConstraintsCmd{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	constraintsCmd := &cobra.Command{
		Use:     "constraints",
		Aliases: []string{"constraint", "c"},
		Short:   "Remove constraints from the Tarian Server.",
		Example: `Tarianctl remove constraints [command options] names...`,
		RunE:    cmd.run,
	}

	// add flags
	constraintsCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "default", "The namespace scope for the constraint to be removed")
	return constraintsCmd
}

func (c *removeConstraintsCmd) run(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := errors.New("please specify the name(s) of the constraint to be removed")
		return fmt.Errorf("remove constraint: %w", err)
	}

	if c.grpcClient == nil {
		opts, err := util.GetDialOptions(c.logger, c.globalFlags.ServerTLSEnabled, c.globalFlags.ServerTLSInsecureSkipVerify, c.globalFlags.ServerTLSCAFile)
		if err != nil {
			return fmt.Errorf("import: %w", err)
		}

		grpcConn, err := grpc.Dial(c.globalFlags.ServerAddr, opts...)
		if err != nil {
			return fmt.Errorf("import: failed to connect to server: %w", err)
		}
		defer grpcConn.Close()
		c.grpcClient = ugrpc.NewGRPCClient(grpcConn)
	}

	client := c.grpcClient.NewConfigClient()
	deletedConstraints := []string{}
	for _, name := range args {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err := client.RemoveConstraint(ctx, &tarianpb.RemoveConstraintRequest{Namespace: c.namespace, Name: name})
		cancel()
		if err != nil {
			return fmt.Errorf("remove constraint: %w", err)
		}

		if response.GetSuccess() {
			deletedConstraints = append(deletedConstraints, name)
			c.logger.Debugf("Constraint '%s' removed\n", name)
		} else {
			c.logger.Warnf("Constraint '%s' is not removed\n", name)
		}
	}
	if len(deletedConstraints) > 0 {
		c.logger.Infof("Successfully removed constraints: %v\n", deletedConstraints)
	} else {
		c.logger.Warnf("No Constraints removed\n")
	}
	return nil
}
