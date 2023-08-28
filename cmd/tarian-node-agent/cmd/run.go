package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kube-tarian/tarian/cmd/tarian-node-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/nodeagent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	clusterAgentHost    string
	clusterAgentPort    string
	nodeNmae            string
	enableAddConstraint bool
}

func newRunCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &runCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the node agent",
		RunE:  cmd.run,
	}

	// Add flags
	runCmd.Flags().StringVar(&cmd.clusterAgentHost, "cluster-agent-host", "tarian-cluster-agent.tarian-system.svc", "The host to listen on")
	runCmd.Flags().StringVar(&cmd.clusterAgentPort, "cluster-agent-port", "80", "The port to listen on")
	runCmd.Flags().StringVar(&cmd.nodeNmae, "node-name", "", "The node name")
	runCmd.Flags().BoolVar(&cmd.enableAddConstraint, "enable-add-constraint", false, "Enable add constraint RPC. Enable this to allow register mode.")
	return runCmd
}

func (c *runCommand) run(cmd *cobra.Command, args []string) error {
	if !isDebugFsMounted() {
		c.logger.Info("debugfs is not mounted, will try to mount")

		err := mountDebugFs()
		if err != nil {
			c.logger.Error(err)
			return fmt.Errorf("failed to mount debugfs: %w", err)
		}

		c.logger.WithField("path", DebugFSRoot).Info("successfully mounted debugfs")
	}

	// Check host proc dir
	_, err := os.Stat(nodeagent.HostProcDir)
	if err == nil {
		c.logger.WithField("path", nodeagent.HostProcDir).Info("host proc is mounted")
	} else if os.IsNotExist(err) {
		c.logger.WithField("path", nodeagent.HostProcDir).Error("host proc is not mounted")
		return fmt.Errorf("host proc is not mounted: %w", err)
	}

	addr := c.clusterAgentHost + ":" + c.clusterAgentPort
	agent := nodeagent.NewNodeAgent(c.logger, addr)
	agent.EnableAddConstraint(c.enableAddConstraint)
	agent.SetNodeName(c.nodeNmae)

	c.logger.WithField("node-name", c.nodeNmae).Info("tarian-node-agent is running")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		c.logger.WithField("signal", sig).Warn("got sigterm signal, attempting graceful shutdown")

		agent.GracefulStop()
	}()

	agent.Run()
	c.logger.Info("tarian-node-agent shutdown gracefully")

	return nil
}
