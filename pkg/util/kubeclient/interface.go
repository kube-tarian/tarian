package kubeclient

// Client represents a Kubernetes client for interacting with Kubernetes clusters.
type Client interface {
	// WaitForPodsToBeReady waits for pods to be ready in the specified namespace and with the given label selector.
	WaitForPodsToBeReady(namespace, labelSelector string) error
	// ExecPodWithOneContainer executes a command in a pod with one container.
	ExecPodWithOneContainer(namespace, podName string, cmd []string) (string, error)
	// GetPodName returns the name of a pod in the specified namespace and with the given label selector.
	GetPodName(namespace, labelSelector string) (string, error)
}
