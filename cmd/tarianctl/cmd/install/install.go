package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/util/helm"
	"github.com/kube-tarian/tarian/pkg/util/kubeclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/util/homedir"
)

// installCmd represents the install command
type installCmd struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	// k8s cluster related options
	namespace   string
	kubeContext string
	kubeconfig  string

	charts       string
	serverValues []string
	agentsValues []string
	natsValues   []string
	dgraphValues []string

	// install options
	onlyAgents bool // install only agents

	// clients
	helmClient *helm.Client
	kubeClient *kubeclient.Client
}

// NewInstallCommand creates a new install command
func NewInstallCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &installCmd{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tarian on Kubernetes.",
		PreRun: func(_ *cobra.Command, _ []string) {
			// cmd.logger.Warn("WARNING: This command is in alpha version and may have some bugs and breaking changes in future releases.")
			yellow := "\033[33m"
			reset := "\033[0m"
			text := "WARNING: This command is in alpha version and may have some bugs and breaking changes in future releases."
			fmt.Printf("%s%s%s\n", yellow, text, reset)
		},
		RunE: cmd.run,
	}

	// Define command-line flags
	installCmd.Flags().StringVarP(&cmd.namespace, "namespace", "n", "tarian-system", "Namespace to install Tarian.")
	installCmd.Flags().BoolVar(&cmd.onlyAgents, "only-agents", false, "Set to install Tarian Cluster Agent and Node Agent only.")
	installCmd.Flags().StringSliceVar(&cmd.serverValues, "server-values", nil, "Path to the helm values file for Tarian Server.")
	installCmd.Flags().StringSliceVar(&cmd.agentsValues, "agents-values", nil, "Path to the helm values file for Tarian Cluster Agent and Node agent .")
	installCmd.Flags().StringVar(&cmd.charts, "charts", "", "Path to the tarian helm charts directory.")
	installCmd.Flags().StringSliceVar(&cmd.natsValues, "nats-values", nil, "Path to the helm values file for Nats.")
	installCmd.Flags().StringSliceVar(&cmd.dgraphValues, "dgraph-values", nil, "Path to the helm values file for DGraph.")
	installCmd.Flags().StringVar(&cmd.kubeContext, "kube-context", "", "Name of the kubeconfig context to use.")
	installCmd.Flags().StringVar(&cmd.kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use")

	return installCmd
}

// run executes the install command
func (c *installCmd) run(cmd *cobra.Command, args []string) error {
	// Create a temporary directory to store the values file
	tempDir, err := os.MkdirTemp("", "helm-values-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
				return fmt.Errorf("install: could not find kubeconfig")
			}
		}
	}

	if c.kubeContext != "" {
		c.logger.Debug("using kubeconfig context: ", c.kubeContext)
	}
	c.logger.Debug("Using Namespace: ", c.namespace)
	c.logger.Infof("Installing Tarian in namespace '%v'...", c.namespace)

	// Create Helm and Kubernetes client instances
	c.helmClient, err = helm.NewHelmClient(c.logger, c.kubeconfig, c.kubeContext)
	if err != nil {
		return fmt.Errorf("install: failed to create helm client: %w", err)
	}

	c.kubeClient, err = kubeclient.NewKubeClient(c.logger, c.kubeconfig, c.kubeContext)
	if err != nil {
		return fmt.Errorf("install: failed to get kubeclient: %w", err)
	}

	var g errgroup.Group
	for _, f := range []func(string) error{
		c.installNats,
		c.installDgraph,
	} {
		f := f
		g.Go(func() error {
			return f(tempDir)
		})
	}
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("install: failed to install dependencies: %w", err)
	}

	if c.onlyAgents {
		// Install Tarian Cluster Agent and Node Agent
		c.logger.Warn("Please don't forget to provide tarian server address via helm values file.")
		err = c.installAgents(tempDir)
		if err != nil {
			return err
		}
	} else {
		// Install Tarian Server and Cluster Agent and Node Agent
		err = c.installServer(tempDir)
		if err != nil {
			return err
		}
		err = c.installAgents(tempDir)
		if err != nil {
			return err
		}
	}

	c.logger.Info("Tarian successfully installed.")
	return nil
}

func (c *installCmd) installNats(tempDir string) error {
	c.logger.Info("Installing Nats...")
	// Add Helm repository for Nats
	err := c.helmClient.AddRepo("nats", "https://nats-io.github.io/k8s/helm/charts/")
	if err != nil {
		return fmt.Errorf("install: failed to add nats repo: %w", err)
	}

	// Install Nats using Helm
	natsValuesFile := filepath.Join(tempDir, "nats-values.yaml")
	err = natsHelmDefaultValues(natsValuesFile)
	if err != nil {
		return fmt.Errorf("install: failed to get nats helm default values file: %w", err)
	}
	valuesFiles := []string{natsValuesFile}
	if c.natsValues != nil {
		valuesFiles = append(valuesFiles, c.natsValues...)
	}
	natsVer := "0.19.16"

	err = c.helmClient.Install("nats", "nats/nats", c.namespace, valuesFiles, natsVer, nil)
	if err != nil {
		return fmt.Errorf("install: failed to install nats: %w", err)
	}

	c.logger.Info("Waiting for Nats pods to be ready...")

	// Wait for Nats pods to become ready
	err = c.kubeClient.WaitForPodsToBeReady(c.namespace, "app.kubernetes.io/name=nats")
	if err != nil {
		return fmt.Errorf("install: nats pods failed to be ready: %w", err)
	}
	return nil
}

func (c *installCmd) installDgraph(tempDir string) error {
	c.logger.Info("Installing DGraph...")

	// Add Helm repository for DGraph
	err := c.helmClient.AddRepo("dgraph", "https://charts.dgraph.io")
	if err != nil {
		return fmt.Errorf("install: failed to add dgraph repo: %w", err)
	}

	// Install DGraph using Helm
	dgraphValuesFile := filepath.Join(tempDir, "dgraph-values.yaml")
	err = dgraphHelmDefaultValues(dgraphValuesFile)
	if err != nil {
		return fmt.Errorf("install: failed to get dgraph helm default values file: %w", err)
	}

	valuesFiles := []string{dgraphValuesFile}
	if c.dgraphValues != nil {
		valuesFiles = append(valuesFiles, c.dgraphValues...)
	}
	err = c.helmClient.Install("dgraph", "dgraph/dgraph", c.namespace, valuesFiles, "", nil)
	if err != nil {
		return fmt.Errorf("install: failed to install dgraph: %w", err)
	}

	c.logger.Info("Waiting for DGraph pods to be ready...")

	// Wait for DGraph pods to become ready
	err = c.kubeClient.WaitForPodsToBeReady(c.namespace, "app=dgraph")
	if err != nil {
		return fmt.Errorf("install: dgraph pods failed to be ready: %w", err)
	}
	return nil
}

func (c *installCmd) installServer(tempDir string) error {
	// If charts path is not specified, add Tarian Helm repository
	c.logger.Info("Installing Tarian Server...")
	if c.charts == "" {
		err := c.helmClient.AddRepo("tarian", "https://kube-tarian.github.io/helm-charts")
		if err != nil {
			return fmt.Errorf("install: failed to add tarian repo: %w", err)
		}
		c.charts = "tarian"
	}

	chart := c.charts + "/tarian-server"
	set := []string{"server.dgraph.address=dgraph-dgraph-alpha:9080"}

	// Install Tarian Server using Helm
	err := c.helmClient.Install("tarian-server", chart, c.namespace, c.serverValues, "", set)
	if err != nil {
		return fmt.Errorf("install: failed to install tarian server: %w", err)
	}

	c.logger.Info("Waiting for Tarian Server pods to be ready...")

	// Wait for Tarian Server pods to become ready
	err = c.kubeClient.WaitForPodsToBeReady(c.namespace, "app=tarian-server")
	if err != nil {
		return fmt.Errorf("install: tarian server pods failed to be ready: %w", err)
	}

	c.logger.Info("Apply schema to DGraph...")

	// Get the name of the Tarian Server pod
	podName, err := c.kubeClient.GetPodName(c.namespace, "app=tarian-server")
	if err != nil {
		return fmt.Errorf("install: failed to get tarian server pod name: %w", err)
	}

	// Define the command to apply schema to DGraph
	dgraphCmd := []string{"./tarian-server", "dgraph", "apply-schema"}

	// Execute the command in the Tarian Server pod
	output, err := c.kubeClient.ExecPodWithOneContainer(c.namespace, podName, dgraphCmd)
	if err != nil {
		return fmt.Errorf("install: failed to apply schema to dgraph: %w", err)
	}

	if strings.Contains(output, "error while applying dgraph schema") {
		return fmt.Errorf("failed to execute command: '%s', in namespace/pod: %s/%s, error: %s", strings.Join(dgraphCmd, " "), c.namespace, podName, output)
	}
	return nil
}

func (c *installCmd) installAgents(tempDir string) error {
	c.logger.Info("Installing Tarian Cluster Agent...")
	if c.charts == "" {
		err := c.helmClient.AddRepo("tarian", "https://kube-tarian.github.io/helm-charts")
		if err != nil {
			return fmt.Errorf("install: failed to add tarian repo: %w", err)
		}
		c.charts = "tarian"
	}

	chart := c.charts + "/tarian-cluster-agent"

	// Install Tarian Cluster Agent using Helm
	err := c.helmClient.Install("tarian", chart, c.namespace, c.agentsValues, "", nil)
	if err != nil {
		return fmt.Errorf("install: failed to install tarian cluster agent: %w", err)
	}

	c.logger.Info("Waiting for Tarian Cluster/Node Agent pods to be ready...")

	// Wait for Tarian Cluster/Node Agent pods to become ready
	err = c.kubeClient.WaitForPodsToBeReady(c.namespace, "app=tarian")
	if err != nil {
		return fmt.Errorf("install: tarian cluster/node agent pods failed to be ready: %w", err)
	}
	return nil
}
