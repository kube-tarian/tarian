package nodeagent

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
