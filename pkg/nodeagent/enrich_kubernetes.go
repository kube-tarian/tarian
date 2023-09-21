package nodeagent

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// containerIdx is the name of the index used for pod informers to index by container IDs.
	containerIdx = "container-ids"

	// containerIDLen is the maximum length of a container ID.
	containerIDLen = 15
)

var (
	errNotPod = errors.New("object is not a *corev1.Pod")
)

// containerIndexFunc is a function used by the pod informer to index pods by container IDs.
// It extracts container IDs from different types of containers in a pod.
//
// Parameters:
//   - obj: The object to index (expected to be a *corev1.Pod).
//
// Returns:
//   - []string: An array of container IDs found in the pod.
//   - error: An error if the object is not a pod or if there's a problem extracting container IDs.
func containerIndexFunc(obj interface{}) ([]string, error) {
	var containerIDs []string
	appendContainerID := func(fullContainerID string) error {
		if fullContainerID == "" {
			// This is expected if the container hasn't been started yet.
			// This function will be called again after the container starts.
			return nil
		}

		containerID, err := cleanContainerIDFromPod(fullContainerID)
		if err != nil {
			return err
		}

		containerIDs = append(containerIDs, containerID)

		return nil
	}

	switch t := obj.(type) {
	case *corev1.Pod:
		for _, container := range t.Status.InitContainerStatuses {
			err := appendContainerID(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		for _, container := range t.Status.ContainerStatuses {
			err := appendContainerID(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		for _, container := range t.Status.EphemeralContainerStatuses {
			err := appendContainerID(container.ContainerID)
			if err != nil {
				return nil, err
			}
		}
		return containerIDs, nil
	}
	return nil, fmt.Errorf("%w - found %T", errNotPod, obj)
}

// cleanContainerIDFromPod extracts and cleans the container ID from the format "docker://<name>"
// to ensure it's a maximum of 15 characters long.
//
// Parameters:
//   - podContainerID: The container ID in the format "docker://<name>".
//
// Returns:
//   - string: The cleaned container ID as a string.
//   - error: An error if the container ID format is unexpected.
func cleanContainerIDFromPod(podContainerID string) (string, error) {
	parts := strings.Split(podContainerID, "//")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected containerID format, expecting 'docker://<name>', got %q", podContainerID)
	}

	containerID := parts[1]
	if len(containerID) > containerIDLen {
		containerID = containerID[:containerIDLen]
	}

	return containerID, nil
}

// K8sPodWatcher is an interface for finding pods based on container IDs.
type K8sPodWatcher interface {
	FindPod(containerID string) *corev1.Pod
}

// PodWatcher watches Kubernetes pods and allows finding a pod by its container ID.
type PodWatcher struct {
	podInformer     cache.SharedIndexInformer
	informerFactory informers.SharedInformerFactory
	logger          *logrus.Logger
}

// NewPodWatcher creates a new PodWatcher instance for watching Kubernetes pods and finding pods by container ID.
//
// Parameters:
//   - logger: The logger instance for logging messages.
//   - k8sClient: The Kubernetes client used to create informers.
//   - nodeName: The name of the Kubernetes node (optional).
//
// Returns:
//   - *PodWatcher: A new PodWatcher instance.
//   - error: An error if creating the PodWatcher or informers fails.
func NewPodWatcher(logger *logrus.Logger, k8sClient *kubernetes.Clientset, nodeName string) (*PodWatcher, error) {
	k8sInformerFactory := informers.NewSharedInformerFactoryWithOptions(k8sClient, 60*time.Second,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			if nodeName != "" {
				options.FieldSelector = "spec.nodeName=" + nodeName
			}
		}))
	podInformer := k8sInformerFactory.Core().V1().Pods().Informer()
	err := podInformer.AddIndexers(map[string]cache.IndexFunc{
		containerIdx: containerIndexFunc,
	})
	if err != nil {
		return nil, fmt.Errorf("NewPodWatcher: %w", err)
	}

	return &PodWatcher{
		podInformer:     podInformer,
		informerFactory: k8sInformerFactory,
		logger:          logger,
	}, nil
}

// Start starts the PodWatcher and waits for cache synchronization.
func (watcher *PodWatcher) Start() {
	watcher.informerFactory.Start(wait.NeverStop)
	watcher.informerFactory.WaitForCacheSync(wait.NeverStop)
	watcher.logger.WithField("num", len(watcher.podInformer.GetStore().ListKeys())).Info("PodWatcher: initial pods sync")
}

// FindPod finds a pod by its container ID.
//
// Parameters:
//   - containerID: The container ID to search for.
//
// Returns:
//   - *corev1.Pod: The matching pod, or nil if not found.
func (watcher *PodWatcher) FindPod(containerID string) *corev1.Pod {
	indexedContainerID := containerID
	if len(containerID) > containerIDLen {
		indexedContainerID = containerID[:containerIDLen]
	}

	pods, err := watcher.podInformer.GetIndexer().ByIndex(containerIdx, indexedContainerID)
	if err != nil {
		return nil
	}

	return findContainer(containerID, pods)
}

// findContainer finds a pod by its container ID among a list of pods.
//
// Parameters:
//   - containerID: The container ID to search for.
//   - pods: The list of pods to search in.
//
// Returns:
//   - *corev1.Pod: The matching pod, or nil if not found.
func findContainer(containerID string, pods []interface{}) *corev1.Pod {
	if containerID == "" {
		return nil
	}

	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}

		for _, container := range pod.Status.ContainerStatuses {
			if containerIDContains(container.ContainerID, containerID) {
				return pod
			}
		}
		for _, container := range pod.Status.InitContainerStatuses {
			if containerIDContains(container.ContainerID, containerID) {
				return pod
			}
		}
		for _, container := range pod.Status.EphemeralContainerStatuses {
			if containerIDContains(container.ContainerID, containerID) {
				return pod
			}
		}
	}

	return nil
}

// containerIDContains checks if a container ID contains a specific prefix.
//
// Parameters:
//   - containerID: The container ID to check.
//   - prefix: The prefix to search for in the container ID.
//
// Returns:
//   - bool: True if the container ID contains the prefix, false otherwise.
func containerIDContains(containerID string, prefix string) bool {
	parts := strings.Split(containerID, "//")
	if len(parts) == 2 && strings.HasPrefix(parts[1], prefix) {
		return true
	}

	return false
}
