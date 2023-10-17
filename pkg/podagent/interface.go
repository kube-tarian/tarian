package podagent

import (
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

// Agent is the interface that wraps the basic methods for a pod agent.
type Agent interface {
	// SetPodLabels sets the pod labels for the agent.
	SetPodLabels(labels []*tarianpb.Label)
	// SetPodName sets the pod name for the agent.
	SetPodName(name string)
	// SetPodUID sets the pod UID for the agent.
	SetPodUID(uid string)
	// SetNamespace sets the namespace for the agent.
	SetNamespace(namespace string)
	// SetFileValidationInterval sets the interval for file validation.
	SetFileValidationInterval(t time.Duration)
	// EnableRegisterFiles enables the agent to register files.
	EnableRegisterFiles()
	// SetRegisterFilePaths sets the paths to register.
	SetRegisterFilePaths(paths []string)
	// SetRegisterFileIgnorePaths sets the paths to ignore while registering.
	SetRegisterFileIgnorePaths(paths []string)
	// Dial establishes a connection to the cluster agent.
	Dial()
	// GracefulStop stops the agent gracefully.
	GracefulStop()
	// RunThreatScan starts the threat scan loop.
	RunThreatScan()
	// RunRegister starts the registration loop.
	RunRegister()
	// SetConstraints sets the constraints for the agent.
	SetConstraints(constraints []*tarianpb.Constraint)
	// GetConstraints returns the constraints for the agent.
	GetConstraints() []*tarianpb.Constraint
	// SyncConstraints retrieves and synchronizes constraints from the cluster agent.
	SyncConstraints()
	// ReportViolatedFilesToClusterAgent reports violated files to the cluster agent.
	ReportViolatedFilesToClusterAgent(violatedFiles map[string]*violatedFile)
}
