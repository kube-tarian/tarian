package kubeclient

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type client struct {
	client     *kubernetes.Clientset
	restConfig *rest.Config
	logger     *logrus.Logger
}

// NewKubeClient returns a new Kubernetes client.
func NewKubeClient(logger *logrus.Logger, kubeconfig string, kubeContext string) (Client, error) {
	var restConfig *rest.Config
	var err error
	if kubeContext != "" {
		restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: kubeContext,
			},
		).ClientConfig()
	} else {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	return &client{
		client:     clientSet,
		restConfig: restConfig,
		logger:     logger,
	}, nil
}

// WaitForPodsToBeReady waits for pods to be ready in the specified namespace and with the given label selector.
func (k *client) WaitForPodsToBeReady(namespace, labelSelector string) error {
	ctx := context.Background()
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, false, wait.ConditionWithContextFunc(func(ctx context.Context) (bool, error) {
		k.logger.Debugf(`Waiting for pods "%v" to be in the "Running" state...`, labelSelector)

		podList, err := k.client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return false, nil
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				k.logger.Debugf("Waiting for container '%v' to be in the running state.", containerStatus.Name)
				if !containerStatus.Ready {
					return false, nil
				}
			}
		}

		k.logger.Infof("All pods '%v' are in the 'Running' state.", labelSelector)
		return true, nil
	}))
}

// ExecPodWithOneContainer executes a command in a pod with one container.
func (k *client) ExecPodWithOneContainer(namespace, podName string, cmd []string) (string, error) {
	podOpts := &corev1.PodExecOptions{
		Command: cmd,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}

	req := k.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(podOpts, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	k.logger.Debugf("stdout: %s", stdout.String())
	k.logger.Debugf("stderr: %s", stderr.String())

	if err != nil {
		return "", fmt.Errorf("failed to stream: %w", err)
	}

	if stderr.String() != "" {
		return "", fmt.Errorf("stderr: %s", stderr.String())
	}

	return stdout.String(), nil
}

// GetPodName returns the name of a pod in the specified namespace and with the given label selector.
func (k *client) GetPodName(namespace, labelSelector string) (string, error) {
	podList, err := k.client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil || len(podList.Items) == 0 {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}
	return podList.Items[0].Name, nil
}
