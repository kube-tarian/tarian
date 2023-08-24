package cmd

import (
	"fmt"

	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/clusteragent/webhookserver"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

type runWebhookServerCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	port int

	podAgentContainerName  string
	podAgentContainerImage string

	clusterAgentHost string
	clusterAgentPort string

	healthProbeBindAddress string
	podNamespace           string

	enableLeaderElection  bool
	enableCertRotator     bool
	certRotatorSecretName string

	mutatingWebhookConfigurationName string
}

func newWebhookServerCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &runWebhookServerCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	runWebhookServerCmd := &cobra.Command{
		Use:   "run-webhook-server",
		Short: "Run kubernetes admission webhook server",
		RunE:  cmd.run,
	}

	// Add flags
	runWebhookServerCmd.Flags().IntVar(&cmd.port, "port", 9443, "Webhook server port")
	runWebhookServerCmd.Flags().StringVar(&cmd.podAgentContainerName, "pod-agent-container-name", "tarian-pod-agent", "Pod agent container name to be injected")
	runWebhookServerCmd.Flags().StringVar(&cmd.podAgentContainerImage, "pod-agent-container-image", "localhost:5000/tarian-pod-agent:latest", "Pod agent container image to be injected")
	runWebhookServerCmd.Flags().StringVar(&cmd.clusterAgentHost, "cluster-agent-host", "tarian-cluster-agent.tarian-system.svc", "Host address of cluster-agent")
	runWebhookServerCmd.Flags().StringVar(&cmd.clusterAgentPort, "cluster-agent-port", "80", "Port of cluster-agent")
	runWebhookServerCmd.Flags().StringVar(&cmd.healthProbeBindAddress, "health-probe-bind-address", ":8081", "Health probe bind address")
	runWebhookServerCmd.Flags().StringVar(&cmd.podNamespace, "pod-namespace", "tarian-system", "Pod namespace where it runs. This is intended to be set from a downward API.")
	runWebhookServerCmd.Flags().BoolVar(&cmd.enableLeaderElection, "enable-leader-election", false, "Enable leader election")
	runWebhookServerCmd.Flags().BoolVar(&cmd.enableCertRotator, "enable-cert-rotator", true, "Enable cert rotator")
	runWebhookServerCmd.Flags().StringVar(&cmd.certRotatorSecretName, "cert-rotator-secret-name", "tarian-webhook-server-cert", "The tls secret name to be managed by cert rotator")
	runWebhookServerCmd.Flags().StringVar(&cmd.mutatingWebhookConfigurationName, "mutating-webhook-configuration-name", "tarian-mutating-webhook-configuration", "Name of mutating webhook configuration")

	return runWebhookServerCmd
}

func (c *runWebhookServerCommand) run(cmd *cobra.Command, args []string) error {
	podAgentContainerConfig := webhookserver.PodAgentContainerConfig{
		Name:        c.podAgentContainerName,
		Image:       c.podAgentContainerImage,
		LogEncoding: c.globalFlags.LogFormatter,
		Host:        c.clusterAgentHost,
		Port:        c.clusterAgentPort,
	}

	mgr, err := webhookserver.NewManager(c.logger, c.port, c.healthProbeBindAddress, c.enableLeaderElection)
	if err != nil {
		return fmt.Errorf("run-webhook-server: %w", err)
	}

	isReady := make(chan struct{})
	if c.enableCertRotator {
		namespace := c.podNamespace
		err = webhookserver.RegisterCertRotator(c.logger, mgr, isReady, namespace,
			c.mutatingWebhookConfigurationName,
			c.certRotatorSecretName)
		if err != nil {
			return fmt.Errorf("run-webhook-server: %w", err)
		}
	} else {
		close(isReady)
	}

	go func() {
		<-isReady
		// register the rest of the controllers after cert is ready
		webhookserver.RegisterControllers(c.logger, mgr, podAgentContainerConfig)
	}()

	c.logger.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		c.logger.Error("problem running manager: ", err)
		return fmt.Errorf("run-webhook-server: problem running manager: %w", err)
	}

	c.logger.Warn("manager shutdown gracefully")
	return nil

}
