package uninstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/util/helm"
	"golang.org/x/sync/errgroup"

	"github.com/kube-tarian/tarian/pkg/util/kubeclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

type uninstallCmd struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	// k8s cluster related options
	namespace   string
	kubeContext string
	kubeconfig  string

	// uninstall options
	onlyAgents bool

	// clients
	helmClient helm.Client
	kubeClient kubeclient.Client
}

func NewUninstallCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &uninstallCmd{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Tarian from a Kubernetes cluster",
		PreRun: func(_ *cobra.Command, _ []string) {
			// cmd.logger.Warn("WARNING: This command is in alpha version and may have some bugs and breaking changes in future releases.")
			yellow := "\033[33m"
			reset := "\033[0m"
			text := "WARNING: This command is in alpha version and may have some bugs and breaking changes in future releases."
			fmt.Printf("%s%s%s\n", yellow, text, reset)
		},
		RunE: cmd.run,
	}
	uninstallCmd.Flags().StringVar(&cmd.namespace, "namespace", "tarian-system", "Namespace to uninstall Tarian from")
	uninstallCmd.Flags().BoolVar(&cmd.onlyAgents, "only-agents", false, "Set to uninstall Tarian Cluster Agent and Node Agent only")
	uninstallCmd.Flags().StringVar(&cmd.kubeContext, "kube-context", "", "Kubernetes context to use")
	uninstallCmd.Flags().StringVar(&cmd.kubeconfig, "kubeconfig", "", "Kubernetes configuration file")

	return uninstallCmd
}

func (c *uninstallCmd) run(cmd *cobra.Command, args []string) error {
	var err error
	// set kubeconfig
	if c.kubeconfig == "" {
		if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
			c.logger.Debugf("using kubeconfig from environment variable, KUBECONFIG=%s", kubeconfigEnv)
			c.kubeconfig = kubeconfigEnv
		} else {
			c.logger.Debug("using kubeconfig from default kubeconfig path ~/.kube/config")
			home := homedir.HomeDir()
			c.kubeconfig = filepath.Join(home, ".kube", "config")
			_, err := os.Stat(c.kubeconfig)
			if err != nil {
				c.logger.Debug("could not find kubeconfig in default path ~/.kube/config")
				c.logger.Error("Please set the --kubeconfig flag or the KUBECONFIG environment variable")
				return fmt.Errorf("uninstall: could not find kubeconfig")
			}
		}
	}

	if c.kubeContext != "" {
		c.logger.Debug("using kubeconfig context: ", c.kubeContext)
	}
	c.logger.Debug("Using Namespace: ", c.namespace)
	c.logger.Infof("Uninstalling Tarian in namespace '%v'...", c.namespace)

	// Create Helm and Kubernetes client instances
	if c.helmClient == nil {
		c.helmClient, err = helm.NewHelmClient(c.logger, c.kubeconfig, c.kubeContext)
		if err != nil {
			return fmt.Errorf("uninstall: failed to create helm client: %w", err)
		}
	}

	if c.kubeClient == nil {
		c.kubeClient, err = kubeclient.NewKubeClient(c.logger, c.kubeconfig, c.kubeContext)
		if err != nil {
			return fmt.Errorf("uninstall: failed to get kubeclient: %w", err)
		}
	}

	var g errgroup.Group

	for _, f := range []func() error{
		c.uninstallAgents,
		c.uninstallServer,
		c.uninstallDgraph,
		c.uninstallNats,
	} {
		f := f
		g.Go(func() error {
			return f()
		})
	}
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("uninstall: failed to uninstall: %w", err)
	}

	if c.onlyAgents {
		// Uninstall Tarian Cluster Agent and Node Agent only
		c.logger.Info("Uninstalling Tarian Cluster Agent and Node Agent only...")
		err = c.uninstallAgents()
		if err != nil {
			return fmt.Errorf("uninstall: failed to uninstall Tarian Cluster Agent and Node Agent: %w", err)
		}
	} else {
		// Uninstall Tarian Cluster Agent, Node Agent and Tarian Server, DGraph and NATS
		var g errgroup.Group

		for _, f := range []func() error{
			c.uninstallAgents,
			c.uninstallServer,
			c.uninstallDgraph,
			c.uninstallNats,
		} {
			f := f
			g.Go(func() error {
				return f()
			})
		}
		err = g.Wait()
		if err != nil {
			return fmt.Errorf("uninstall: failed to uninstall: %w", err)
		}
	}

	c.logger.Info("Uninstallation complete")
	return nil
}

func (c *uninstallCmd) uninstallNats() error {
	c.logger.Info("Uninstalling NATS...")
	err := c.helmClient.Uninstall("nats", c.namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall NATS: %w", err)
	}

	// Wait for NATS to be uninstalled
	err = c.kubeClient.WaitForPodsToBeDeleted(c.namespace, "app.kubernetes.io/name=nats")
	if err != nil {
		return fmt.Errorf("failed to wait for NATS to be uninstalled: %w", err)
	}
	return nil
}

func (c *uninstallCmd) uninstallDgraph() error {
	c.logger.Info("Uninstalling DGraph...")
	err := c.helmClient.Uninstall("dgraph", c.namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall DGraph: %w", err)
	}

	// Wait for DGraph to be uninstalled
	err = c.kubeClient.WaitForPodsToBeDeleted(c.namespace, "app=dgraph")
	if err != nil {
		return fmt.Errorf("failed to wait for DGraph to be uninstalled: %w", err)
	}

	return nil
}

func (c *uninstallCmd) uninstallServer() error {
	c.logger.Info("Uninstalling Tarian Server...")
	err := c.helmClient.Uninstall("tarian-server", c.namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall Tarian Server: %w", err)
	}

	// Wait for Tarian Server to be uninstalled
	err = c.kubeClient.WaitForPodsToBeDeleted(c.namespace, "app=tarian-server")
	if err != nil {
		return fmt.Errorf("failed to wait for Tarian Server to be uninstalled: %w", err)
	}

	return nil
}

func (c *uninstallCmd) uninstallAgents() error {
	c.logger.Info("Uninstalling Agents...")
	err := c.helmClient.Uninstall("tarian", c.namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall Agents: %w", err)
	}

	// Wait for Agents to be uninstalled
	err = c.kubeClient.WaitForPodsToBeDeleted(c.namespace, "app=tarian")
	if err != nil {
		return fmt.Errorf("failed to wait for Agents to be uninstalled: %w", err)
	}
	c.logger.Debug("Tarian Cluster Agent and Node Agent successfully uninstalled.")

	return nil
}
