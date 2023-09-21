package nodeagent

import (
	"fmt"
	"path/filepath"

	"github.com/kube-tarian/tarian/pkg/nodeagent/ebpf"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ExecEvent represents the structure of an execution event captured by the CaptureExec.
// It stores information about a process execution event, including its process ID (Pid),
// command name (Command), executable filename (Filename), associated container ID (ContainerID),
// Kubernetes Pod UID (K8sPodUID), Pod name (K8sPodName), Pod namespace (K8sNamespace),
// Pod labels (K8sPodLabels), and Pod annotations (K8sPodAnnotations).
type ExecEvent struct {
	// Pid is the process ID of the executed command.
	Pid uint32

	// Command is the command name (e.g., binary name) of the executed process.
	Command string

	// Filename is the full path to the executable file that was executed.
	Filename string

	// ContainerID is the unique identifier of the container associated with the process.
	ContainerID string

	// K8sPodUID is the unique identifier (UID) of the Kubernetes Pod where the process was executed.
	K8sPodUID string

	// K8sPodName is the name of the Kubernetes Pod where the process was executed.
	K8sPodName string

	// K8sNamespace is the namespace of the Kubernetes Pod where the process was executed.
	K8sNamespace string

	// K8sPodLabels are the labels associated with the Kubernetes Pod.
	K8sPodLabels map[string]string

	// K8sPodAnnotations are the annotations associated with the Kubernetes Pod.
	K8sPodAnnotations map[string]string
}

// CaptureExec captures and processes execution events, associating them with Kubernetes Pods.
// It uses eBPF (Extended Berkeley Packet Filter) to capture execution events in the Linux kernel.
type CaptureExec struct {
	eventsChan     chan ExecEvent       // Channel for sending captured execution events
	shouldClose    bool                 // Flag indicating whether the capture should be closed
	bpfCaptureExec *ebpf.BpfCaptureExec // Instance of eBPF capture execution
	nodeName       string               // The name of the node where the capture is running
	logger         *logrus.Logger       // Logger instance for logging
}

// NewCaptureExec creates a new CaptureExec instance for capturing and processing execution events.
// It initializes the eBPF capture execution instance and sets up a channel for sending events.
//
// Parameters:
//   - logger: A logger instance for logging.
//
// Returns:
//   - *CaptureExec: A new instance of CaptureExec.
//   - error: An error if creating the eBPF capture execution instance fails.
func NewCaptureExec(logger *logrus.Logger) (*CaptureExec, error) {
	// Create a new instance of eBPF capture execution.
	bpfCaptureExec, err := ebpf.NewBpfCaptureExec(logger)
	if err != nil {
		return nil, fmt.Errorf("NewCaptureExec: failed to create bpf capture exec: %w", err)
	}

	return &CaptureExec{
		eventsChan:     make(chan ExecEvent, 1000),
		bpfCaptureExec: bpfCaptureExec,
		logger:         logger,
	}, nil
}

// SetNodeName sets the name of the node where the capture is running.
//
// Parameters:
//   - name: The name of the node.
func (c *CaptureExec) SetNodeName(name string) {
	c.nodeName = name
}

// Start begins capturing execution events and associating them with Kubernetes Pods.
// It returns an error if any of the setup steps fail.
func (c *CaptureExec) Start() error {
	// Get in-cluster configuration for Kubernetes.
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("CaptureExec.Start: failed to get in cluster config: %w", err)
	}

	// Create a Kubernetes client.
	k8sClient := kubernetes.NewForConfigOrDie(config)

	// Create a PodWatcher to watch for Pods on the node.
	watcher, err := NewPodWatcher(c.logger, k8sClient, c.nodeName)
	if err != nil {
		return fmt.Errorf("CaptureExec.Start: failed to create pod watcher: %w", err)
	}
	watcher.Start()

	// Start capturing execution events with eBPF.
	go c.bpfCaptureExec.Start()

	// Get the channel for eBPF execution events.
	bpfExecEventsChan := c.bpfCaptureExec.GetExecEventsChannel()

	for {
		// Wait for eBPF execution events.
		bpfEvt := <-bpfExecEventsChan

		// Check if the capture should be closed.
		if c.shouldClose {
			break
		}

		// Retrieve the container ID.
		containerID, err := procsContainerID(bpfEvt.Pid)
		if err != nil {
			continue
		}

		// Extract the command name and filename from the event.
		filename := unix.ByteSliceToString(bpfEvt.Filename[:])
		command := filepath.Base(filename)

		// Find the corresponding Kubernetes Pod.
		pod := watcher.FindPod(containerID)

		var podName string
		var podUID string
		var namespace string
		var podLabels map[string]string
		var podAnnotations map[string]string

		if pod != nil {
			podName = pod.GetName()
			podUID = string(pod.GetUID())
			namespace = pod.GetNamespace()
			podLabels = pod.GetLabels()
			podAnnotations = pod.GetAnnotations()
		}

		// Create an ExecEvent and send it to the events channel.
		execEvent := ExecEvent{
			Pid:               bpfEvt.Pid,
			Command:           command,
			Filename:          filename,
			ContainerID:       containerID,
			K8sPodName:        podName,
			K8sPodUID:         podUID,
			K8sNamespace:      namespace,
			K8sPodLabels:      podLabels,
			K8sPodAnnotations: podAnnotations,
		}

		c.eventsChan <- execEvent
	}
	return nil
}

// Close stops the capture process and closes associated resources.
func (c *CaptureExec) Close() {
	c.shouldClose = true
	c.bpfCaptureExec.Close()
}

// GetEventsChannel returns the channel for receiving execution events.
func (c *CaptureExec) GetEventsChannel() chan ExecEvent {
	return c.eventsChan
}
