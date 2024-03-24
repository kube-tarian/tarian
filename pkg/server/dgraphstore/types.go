package dgraphstore

import (
	"context"
	"time"

	"github.com/kube-tarian/tarian/pkg/store"
)

// Constraint represents a constraint in the Dgraph database.
type Constraint struct {
	UID              string   `json:"uid,omitempty"`                          // Unique identifier of the constraint.
	DType            []string `json:"dgraph.type,omitempty"`                  // Type information for Dgraph.
	Namespace        string   `json:"constraint_namespace,omitempty"`         // Namespace of the constraint.
	Name             string   `json:"constraint_name,omitempty"`              // Name of the constraint.
	Selector         string   `json:"constraint_selector,omitempty"`          // Selector associated with the constraint.
	AllowedProcesses string   `json:"constraint_allowed_processes,omitempty"` // Allowed processes defined by the constraint.
	AllowedFiles     string   `json:"constraint_allowed_files,omitempty"`     // Allowed files defined by the constraint.
}

// Action represents an action in the Dgraph database.
type Action struct {
	UID                string   `json:"uid,omitempty"`                         // Unique identifier of the action.
	DType              []string `json:"dgraph.type,omitempty"`                 // Type information for Dgraph.
	Namespace          string   `json:"action_namespace,omitempty"`            // Namespace of the action.
	Name               string   `json:"action_name,omitempty"`                 // Name of the action.
	Selector           string   `json:"action_selector,omitempty"`             // Selector associated with the action.
	OnViolatedProcess  bool     `json:"action_on_violated_process,omitempty"`  // Indicates whether the action applies to violated processes.
	OnViolatedFile     bool     `json:"action_on_violated_file,omitempty"`     // Indicates whether the action applies to violated files.
	OnFalcoAlert       bool     `json:"action_on_falco_alert,omitempty"`       // Indicates whether the action applies to Falco alerts.
	FalcoAlertPriority int      `json:"action_falco_alert_priority,omitempty"` // Priority of the action for Falco alerts.
	Action             string   `json:"action_action,omitempty"`               // The action to be taken.
}

// Pod represents a pod in the Dgraph database.
type Pod struct {
	UID       string   `json:"uid,omitempty"`           // Unique identifier of the pod.
	DType     []string `json:"dgraph.type,omitempty"`   // Type information for Dgraph.
	Namespace string   `json:"pod_namespace,omitempty"` // Namespace of the pod.
	Name      string   `json:"pod_name,omitempty"`      // Name of the pod.
	PodUID    string   `json:"pod_uid,omitempty"`       // Unique identifier of the pod (UID).
	Labels    string   `json:"pod_labels,omitempty"`    // Labels associated with the pod.
}

// Event represents an event in the Dgraph database.
type Event struct {
	UID             string     `json:"uid,omitempty"`                    // Unique identifier of the event.
	DType           []string   `json:"dgraph.type,omitempty"`            // Type information for Dgraph.
	Type            string     `json:"event_type,omitempty"`             // Type of the event.
	EventUID        string     `json:"event_uid,omitempty"`              // Unique identifier of the event.
	ClientTimestamp *time.Time `json:"event_client_timestamp,omitempty"` // Client timestamp of the event.
	ServerTimestamp *time.Time `json:"event_server_timestamp,omitempty"` // Server timestamp of the event.
	AlertSentAt     *time.Time `json:"event_alert_sent_at,omitempty"`    // Timestamp when an alert was sent for the event.
	Targets         []Target   `json:"targets,omitempty"`                // List of targets associated with the event.
}

// Target represents a target in the Dgraph database.
type Target struct {
	UID               string   `json:"uid,omitempty"`                        // Unique identifier of the target.
	DType             []string `json:"dgraph.type,omitempty"`                // Type information for Dgraph.
	ViolatedProcesses string   `json:"target_violated_processes,omitempty"`  // Violated processes associated with the target (in JSON format).
	ViolatedFiles     string   `json:"target_violated_files,omitempty"`      // Violated files associated with the target (in JSON format).
	FalcoAlert        string   `json:"target_falco_alert,omitempty"`         // Falco alert associated with the target (in JSON format).
	Pod               *Pod     `json:"pod,omitempty"`                        // Pod associated with the target.
	DetectionDataType string   `json:"tarian_detection_data_type,omitempty"` // Type of the tarian detection data.
	DetectionData     string   `json:"tarian_detection_data,omitempty"`      // The tarian detection data in JSON format.
}

// Client is an interface for creating Dgraph clients.
type Client interface {
	// NewDgraphActionStore returns a new Dgraph action store.
	NewDgraphActionStore() store.ActionStore
	// NewDgraphConstraintStore returns a new Dgraph constraint store.
	NewDgraphConstraintStore() store.ConstraintStore
	// NewDgraphEventStore returns a new Dgraph event store.
	NewDgraphEventStore() store.EventStore
	// ApplySchema applies the Dgraph schema.
	ApplySchema(ctx context.Context) error
}
