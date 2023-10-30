package helm

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

type client struct {
	helmBin     string         // path to the helm binary
	kubeconfig  string         // path to the kubeconfig file
	kubeContext string         // name of the kubeconfig context
	logger      *logrus.Logger // logger
}

// NewHelmClient returns a new Helm client.
func NewHelmClient(logger *logrus.Logger, kubeconfig string, kubeContext string) (Client, error) {
	helmBinaryPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, fmt.Errorf("seems like helm is not installed, please install helm first")
	}

	output, err := exec.Command(helmBinaryPath, "version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run helm version: %w", err)
	}

	if !(strings.Contains(string(output), "Version:\"v3.")) {
		return nil, fmt.Errorf("helm version >=v3.*.* is required, current version: %s", string(output))
	}

	return &client{
		helmBin:     helmBinaryPath,
		kubeconfig:  kubeconfig,
		kubeContext: kubeContext,
		logger:      logger,
	}, nil
}

// AddRepo adds a Helm repository.
func (h *client) AddRepo(name string, url string) error {
	h.logger.Debugf("Adding Helm repo %s with URL %s", name, url)
	args := []string{
		"repo",
		"add",
		name,
		url,
	}
	if h.kubeconfig != "" {
		args = append(args, "--kubeconfig", h.kubeconfig)
	}

	if h.kubeContext != "" {
		args = append(args, "--kube-context", h.kubeContext)
	}

	output, err := exec.Command(h.helmBin, args...).CombinedOutput()
	h.logger.Debug(string(output))
	return err
}

// Install installs a Helm chart.
func (h *client) Install(name string, chart string, namespace string, valuesFiles []string, version string, setArgs []string) error {
	args := []string{
		"upgrade", "--install",
		name, chart,
		"--namespace", namespace,
		"--create-namespace",
	}

	if h.kubeconfig != "" {
		args = append(args, "--kubeconfig", h.kubeconfig)
	}

	if h.kubeContext != "" {
		args = append(args, "--kube-context", h.kubeContext)
	}

	for _, valuesFile := range valuesFiles {
		args = append(args, "--values", valuesFile)
	}

	if version != "" {
		args = append(args, "--version", version)
	}

	for _, setArg := range setArgs {
		args = append(args, "--set", setArg)
	}

	output, err := exec.Command(h.helmBin, args...).CombinedOutput()
	h.logger.Debug(string(output))

	return err
}

// Uninstall uninstalls a Helm chart.
func (h *client) Uninstall(name string, namespace string) error {
	h.logger.Debugf("Uninstalling Helm chart %s in namespace %s", name, namespace)
	args := []string{
		"uninstall", name,
		"--namespace", namespace,
	}

	if h.kubeconfig != "" {
		args = append(args, "--kubeconfig", h.kubeconfig)
	}

	if h.kubeContext != "" {
		args = append(args, "--kube-context", h.kubeContext)
	}

	output, err := exec.Command(h.helmBin, args...).CombinedOutput()
	h.logger.Debug(string(output))

	return err
}