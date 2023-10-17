package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kube-tarian/tarian/cmd/tarian-pod-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/podagent"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type registerCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	host                    string
	port                    string
	podLabelsFile           string
	podName                 string
	podUID                  string
	namespace               string
	registerRules           string
	registerFilePaths       string
	registerFileIgnorePaths string
	fileValidationInterval  time.Duration

	podAgent podagent.Agent
}

func newRegisterCommand(globalFlag *flags.GlobalFlags) *cobra.Command {
	cmd := &registerCommand{
		globalFlags: globalFlag,
		logger:      log.GetLogger(),
	}

	registerCommand := &cobra.Command{
		Use:   "register",
		Short: "Register the pod to the Tarian server",
		RunE:  cmd.runRegisterCommand,
	}

	// Add flags
	registerCommand.Flags().StringVar(&cmd.host, "host", defaultClusterAgentHost, "The host of the Tarian server")
	registerCommand.Flags().StringVar(&cmd.port, "port", defaultClusterAgentPort, "The port of the Tarian server")
	registerCommand.Flags().StringVar(&cmd.podLabelsFile, "pod-labels-file", "", "The path to the file containing the pod labels, This is intended to be a file from Kubernetes DownwardAPIVolumeFile")
	registerCommand.Flags().StringVar(&cmd.podName, "pod-name", "", "The name of the pod")
	registerCommand.Flags().StringVar(&cmd.podUID, "pod-uid", "", "The UID of the pod")
	registerCommand.Flags().StringVar(&cmd.namespace, "namespace", "", "The namespace of the pod")
	registerCommand.Flags().StringVar(&cmd.registerRules, "register-rules", "", "Type of rules that should be automatically registered.")
	registerCommand.Flags().StringVar(&cmd.registerFilePaths, "register-file-paths", "", "The root directories of which pod-agent should register the checksums")
	registerCommand.Flags().StringVar(&cmd.registerFileIgnorePaths, "register-file-ignore-paths", "", "Paths that should be ignored while registering file checksums")
	registerCommand.Flags().DurationVar(&cmd.fileValidationInterval, "file-validation-interval", 3*time.Second, "The interval to validate the pod labels file based on constraints")

	return registerCommand
}

func (c *registerCommand) runRegisterCommand(cmd *cobra.Command, args []string) error {
	c.logger.Info("tarian-pod-agent is running in register mode")
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

	registerRules := strings.Split(c.registerRules, ",")
	for _, rule := range registerRules {
		switch strings.TrimSpace(rule) {
		case "files":
			c.logger.Warn("enabled auto register for files")
			c.podAgent.EnableRegisterFiles()
		case "all":
			c.logger.Info("enabled auto register for all rules")
			c.podAgent.EnableRegisterFiles()
		}
	}

	registerFilePathsArg := strings.Split(c.registerFilePaths, ",")
	registerFilePaths := []string{}
	for _, path := range registerFilePathsArg {
		registerFilePaths = append(registerFilePaths, strings.TrimSpace(path))
	}
	c.podAgent.SetRegisterFilePaths(registerFilePaths)

	registerFileIgnorePathsArg := strings.Split(c.registerFileIgnorePaths, ",")
	registerFileIgnorePaths := []string{}
	for _, path := range registerFileIgnorePathsArg {
		registerFileIgnorePaths = append(registerFileIgnorePaths, strings.TrimSpace(path))
	}
	c.podAgent.SetRegisterFileIgnorePaths(registerFileIgnorePaths)

	c.podAgent.SetFileValidationInterval(c.fileValidationInterval)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		c.logger.WithField("signal", sig).Info("got sigterm signal, attempting graceful shutdown")

		c.podAgent.GracefulStop()
	}()

	c.podAgent.RunRegister()
	c.logger.Info("tarian-pod-agent shutdown gracefully")
	return nil
}
