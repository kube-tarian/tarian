package remove

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/cmd/tarianctl/util"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianctl/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type removeConstraintsCmd struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

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
		err := errors.New("please specify the names of the constraint to be removed")
		return fmt.Errorf("remove constraint: %w", err)
	}
	opts, err := util.ClientOptionsFromCliContext(c.logger, c.globalFlags)
	if err != nil {
		return fmt.Errorf("remove constraint: %w", err)
	}

	client, err := client.NewConfigClient(c.globalFlags.ServerAddr, opts...)
	if err != nil {
		return fmt.Errorf("remove constraint: %w", err)
	}

	for _, name := range args {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		response, err := client.RemoveConstraint(ctx, &tarianpb.RemoveConstraintRequest{Namespace: c.namespace, Name: name})
		cancel()
		if err != nil {
			return fmt.Errorf("remove constraint: %w", err)
		}

		if response.GetSuccess() {
			c.logger.Infof("Constraint %s is deleted successfully\n", name)
		} else {
			c.logger.Warnf("Constraint %s is not deleted\n", name)
		}
	}
	return nil
}
