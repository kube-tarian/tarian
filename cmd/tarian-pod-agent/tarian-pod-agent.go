package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/podagent"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
)

const (
	defaultClusterAgentPort = "50052"
	defaultClusterAgentHost = ""
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

func main() {
	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarian Pod Agent",
		Usage:   "The Tarian pod agent is the component which runs as a sidecar to monitor your main container.",
		Version: version + " (" + commit + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level: debug, info, warn, error",
				Value: "info",
			},
			&cli.StringFlag{
				Name:  "log-encoding",
				Usage: "log-encoding: json, console",
				Value: "console",
			},
		},
		Action: func(ctx *cli.Context) error {
			return ctx.App.Command("threat-scan").Run(ctx)
		},
		Commands: []*cli.Command{
			{
				Name:  "threat-scan",
				Usage: "Run the pod agent to scan threats",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host address of the cluster agent to communicate with",
						Value: defaultClusterAgentHost,
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Host port of the cluster agent to communicate with",
						Value: defaultClusterAgentPort,
					},
					&cli.StringFlag{
						Name:  "pod-labels-file",
						Usage: "File path containing pod labels. This is intended to be a file from Kubernetes DownwardAPIVolumeFile",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "pod-name",
						Usage: "Pod name where it is running. This is intended to be set from Downward API",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "pod-uid",
						Usage: "Pod UID where it is running. This is intended to be set from Downward API",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "namespace",
						Usage: "Kubernetes namespace where it is running",
						Value: "tarian-system",
					},
					&cli.DurationFlag{
						Name:  "file-validation-interval",
						Usage: "How frequent podagent should validate files based on constraints",
						Value: 3 * time.Second,
					},
				},
				Action: threatScan,
			},
			{
				Name:  "register",
				Usage: "Run the pod agent to register known processes and files as a constraint",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host address of the cluster agent to communicate with",
						Value: defaultClusterAgentHost,
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Host port of the cluster agent to communicate with",
						Value: defaultClusterAgentPort,
					},
					&cli.StringFlag{
						Name:  "pod-labels-file",
						Usage: "File path containing pod labels. This is intended to be a file from Kubernetes DownwardAPIVolumeFile",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "pod-name",
						Usage: "Pod name where it is running. This is intended to be set from Downward API",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "pod-uid",
						Usage: "Pod UID where it is running. This is intended to be set from Downward API",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "namespace",
						Usage: "Kubernetes namespace where it is running",
						Value: "tarian-system",
					},
					&cli.DurationFlag{
						Name:  "file-validation-interval",
						Usage: "How frequent podagent should validate files based on constraints",
						Value: 3 * time.Second,
					},
					&cli.StringFlag{
						Name:  "register-rules",
						Usage: "Type of rules that should be automatically registered.",
						Value: "processes,files",
					},
					&cli.StringFlag{
						Name:  "register-file-paths",
						Usage: "The root directories of which pod-agent should register the checksums",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "register-file-ignore-paths",
						Usage: "Paths that should be ignored while registering file checksums",
						Value: "",
					},
				},
				Action: register,
			},
		},
	}
}

func threatScan(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	podagent.SetLogger(logger)
	logger.Infow("tarian-pod-agent is running in threat-scan mode")

	agent := podagent.NewPodAgent(c.String("host") + ":" + c.String("port"))

	podLabelsFile := c.String("pod-labels-file")
	if podLabelsFile != "" {
		podLabels, err := readLabelsFromFile(podLabelsFile)

		if err != nil {
			logger.Errorw("failed reading pod-labels-file", "err", err)
		}

		agent.SetPodLabels(podLabels)
	}

	podName := c.String("pod-name")
	if podName != "" {
		agent.SetPodName(podName)
	} else {
		hostname, err := os.Hostname()
		if err == nil {
			agent.SetPodName(hostname)
		}
	}

	podUID := c.String("pod-uid")
	if podUID != "" {
		agent.SetpodUID(podUID)
	}

	namespace := c.String("namespace")
	if namespace != "" {
		agent.SetNamespace(namespace)
	}

	agent.SetFileValidationInterval(c.Duration("file-validation-interval"))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		agent.GracefulStop()
	}()

	agent.RunThreatScan()
	logger.Info("tarian-pod-agent shutdown gracefully")

	return nil
}

func register(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	podagent.SetLogger(logger)
	logger.Infow("tarian-pod-agent is running in register mode")

	agent := podagent.NewPodAgent(c.String("host") + ":" + c.String("port"))

	podLabelsFile := c.String("pod-labels-file")
	if podLabelsFile != "" {
		podLabels, err := readLabelsFromFile(podLabelsFile)

		if err != nil {
			logger.Errorw("failed reading pod-labels-file", "err", err)
		}

		agent.SetPodLabels(podLabels)
	}

	podName := c.String("pod-name")
	if podName != "" {
		agent.SetPodName(podName)
	} else {
		hostname, err := os.Hostname()
		if err == nil {
			agent.SetPodName(hostname)
		}
	}

	podUID := c.String("pod-uid")
	if podUID != "" {
		agent.SetpodUID(podUID)
	}

	namespace := c.String("namespace")
	if namespace != "" {
		agent.SetNamespace(namespace)
	}

	registerRules := strings.Split(c.String("register-rules"), ",")
	for _, rule := range registerRules {
		switch strings.TrimSpace(rule) {
		case "processes":
			logger.Infow("enabled auto register for processes")
			agent.EnableRegisterProcesses()
		case "files":
			logger.Infow("enabled auto register for files")
			agent.EnableRegisterFiles()
		case "all":
			logger.Infow("enabled auto register for all rules")
			agent.EnableRegisterProcesses()
			agent.EnableRegisterFiles()
		}
	}

	registerFilePathsArg := strings.Split(c.String("register-file-paths"), ",")
	registerFilePaths := []string{}
	for _, path := range registerFilePathsArg {
		registerFilePaths = append(registerFilePaths, strings.TrimSpace(path))
	}
	agent.SetRegisterFilePaths(registerFilePaths)

	registerFileIgnorePathsArg := strings.Split(c.String("register-file-ignore-paths"), ",")
	registerFileIgnorePaths := []string{}
	for _, path := range registerFileIgnorePathsArg {
		registerFileIgnorePaths = append(registerFileIgnorePaths, strings.TrimSpace(path))
	}
	agent.SetRegisterFileIgnorePaths(registerFileIgnorePaths)

	agent.SetFileValidationInterval(c.Duration("file-validation-interval"))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		agent.GracefulStop()
	}()

	agent.RunRegister()
	logger.Info("tarian-pod-agent shutdown gracefully")

	return nil
}

func readLabelsFromFile(path string) ([]*tarianpb.Label, error) {
	labels := []*tarianpb.Label{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		idx := strings.Index(line, "=")

		if idx < 0 {
			continue
		}

		key := line[:idx]
		value := strings.Trim(line[idx+1:], "\"")

		labels = append(labels, &tarianpb.Label{Key: key, Value: value})
	}

	return labels, nil
}
