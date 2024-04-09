package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cilium/ebpf/rlimit"
	"github.com/kube-tarian/tarian/cmd/tarian-node-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/nodeagent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Uname contains system uname information.
type Uname struct {
	ub syscall.Utsname
}

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

func (c *runCommand) run(_ *cobra.Command, args []string) error {
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

	if err := c.setLinuxKernelVersion(); err != nil {
		c.logger.WithError(err).Error("failed to set linux kernel version")
		return fmt.Errorf("failed to set linux kernel version: %w", err)
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		c.logger.Fatal(err)
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

// setLinuxKernelVersion sets the Linux kernel version by parsing the uname information.
func (c *runCommand) setLinuxKernelVersion() error {
	u := &Uname{}
	err := syscall.Uname(&u.ub)

	if err != nil {
		c.logger.WithField("error while making syscall to get linux kernel version, err: ", err)
		return fmt.Errorf("error while making syscall to get linux kernel version: %w", err)
	}

	linuxKernelVersion := charsToString(u.ub.Release[:])
	strArr := strings.Split(linuxKernelVersion, ".")
	if len(strArr) < 3 {
		c.logger.WithField("version", linuxKernelVersion).Fatal("invalid linux kernel version")
		return fmt.Errorf("invalid linux kernel version: %s", linuxKernelVersion)
	}
	majorVersion := strArr[0]
	minorVersion := strArr[1]
	patch := strArr[2]
	// Split to get the patch version
	strArr = strings.Split(patch, "-")
	patchVersion := strArr[0]
	os.Setenv("LINUX_VERSION_MAJOR", majorVersion)
	os.Setenv("LINUX_VERSION_MINOR", minorVersion)
	os.Setenv("LINUX_VERSION_PATCH", patchVersion)

	return nil
}

// charsToString converts an array of int8 to a string.
//
// ca []int8: the array of int8 to be converted.
// string: the resulting string from the conversion.
func charsToString(ca []int8) string {
	s := make([]byte, len(ca))
	var i int
	for ; i < len(ca); i++ {
		if ca[i] == 0 {
			break
		}
		s[i] = uint8(ca[i])
	}
	return string(s[0:i])
}
