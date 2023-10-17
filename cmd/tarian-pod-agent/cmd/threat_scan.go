package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarian-pod-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/podagent"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultClusterAgentPort = "50052"
	defaultClusterAgentHost = ""
)

type threatScanCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	host                   string
	port                   string
	podLabelsFile          string
	podName                string
	podUID                 string
	namespace              string
	fileValidationInterval time.Duration

	podAgent podagent.Agent
}

func newThreatScanCommand(globalFlag *flags.GlobalFlags) *cobra.Command {
	cmd := &threatScanCommand{
		globalFlags: globalFlag,
		logger:      logrus.New(),
	}

	var threatScanCmd = &cobra.Command{
		Use:   "threat-scan",
		Short: "Scan the container image for vulnerabilities",
		RunE:  cmd.run,
	}

	threatScanCmd.Flags().StringVar(&cmd.host, "host", defaultClusterAgentHost, "The host of the Tarian server")
	threatScanCmd.Flags().StringVar(&cmd.port, "port", defaultClusterAgentPort, "The port of the Tarian server")
	threatScanCmd.Flags().StringVar(&cmd.podLabelsFile, "pod-labels-file", "", "The path to the file containing the pod labels, This is intended to be a file from Kubernetes DownwardAPIVolumeFile")
	threatScanCmd.Flags().StringVar(&cmd.podName, "pod-name", "", "The name of the pod")
	threatScanCmd.Flags().StringVar(&cmd.podUID, "pod-uid", "", "The UID of the pod")
	threatScanCmd.Flags().StringVar(&cmd.namespace, "namespace", "", "The namespace of the pod")
	threatScanCmd.Flags().DurationVar(&cmd.fileValidationInterval, "file-validation-interval", 3*time.Second, "The interval to validate the pod labels file based on constraints")

	return threatScanCmd
}

func (c *threatScanCommand) run(_ *cobra.Command, args []string) error {
	c.logger.Info("tarian-pod-agent is running in threat-scan mode")
	addr := c.host + ":" + c.port
	if c.podAgent == nil {
		c.podAgent = podagent.NewPodAgent(c.logger, addr)
	}

	if c.podLabelsFile != "" {
		podLabels, err := readLabelsFromFile(c.logger, c.podLabelsFile)
		if err != nil {
			return fmt.Errorf("failed reading pod-labels-file: %w", err)
		}

		// delete pod-template-hash
		for i, e := range podLabels {
			if e.GetKey() == "pod-template-hash" {
				newPodLabels := make([]*tarianpb.Label, 0)
				newPodLabels = append(newPodLabels, podLabels[:i]...)
				newPodLabels = append(newPodLabels, podLabels[i+1:]...)
				podLabels = newPodLabels

				break
			}
		}

		c.podAgent.SetPodLabels(podLabels)
	}

	if c.podName != "" {
		c.podAgent.SetPodName(c.podName)
	} else {
		hostname, err := os.Hostname()
		if err == nil {
			c.podAgent.SetPodName(hostname)
		}
	}

	if c.podUID != "" {
		c.podAgent.SetPodUID(c.podUID)
	}

	if c.namespace != "" {
		c.podAgent.SetNamespace(c.namespace)
	}

	c.podAgent.SetFileValidationInterval(c.fileValidationInterval)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		c.logger.WithField("signal", sig).Info("got sigterm signal, attempting graceful shutdown")
		c.podAgent.GracefulStop()
	}()

	c.podAgent.RunThreatScan()
	c.logger.Info("tarian-pod-agent shutdown gracefully")
	return nil
}
