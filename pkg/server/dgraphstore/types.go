package dgraphstore

import "time"

type Constraint struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	Namespace        string `json:"constraint_namespace,omitempty"`
	Name             string `json:"constraint_name,omitempty"`
	Selector         string `json:"constraint_selector,omitempty"`
	AllowedProcesses string `json:"constraint_allowed_processes,omitempty"`
	AllowedFiles     string `json:"constraint_allowed_files,omitempty"`
}

type Action struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	Namespace          string `json:"action_namespace,omitempty"`
	Name               string `json:"action_name,omitempty"`
	Selector           string `json:"action_selector,omitempty"`
	OnViolatedProcess  bool   `json:"action_on_violated_process,omitempty"`
	OnViolatedFile     bool   `json:"action_on_violated_file,omitempty"`
	OnFalcoAlert       bool   `json:"action_on_falco_alert,omitempty"`
	FalcoAlertPriority int    `json:"action_falco_alert_priority,omitempty"`
	Action             string `json:"action_action,omitempty"`
}

type Pod struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	Namespace string `json:"pod_namespace,omitempty"`
	Name      string `json:"pod_name,omitempty"`
	PodUID    string `json:"pod_uid,omitempty"`
	Labels    string `json:"pod_labels,omitempty"`
}

type Event struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	Type            string     `json:"event_type,omitempty"`
	EventUID        string     `json:"event_uid,omitempty"`
	ClientTimestamp *time.Time `json:"event_client_timestamp,omitempty"`
	ServerTimestamp *time.Time `json:"event_server_timestamp,omitempty"`
	AlertSentAt     *time.Time `json:"event_alert_sent_at,omitempty"`
	Targets         []Target   `json:"targets,omitempty"`
}

type Target struct {
	UID   string   `json:"uid,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`

	ViolatedProcesses string `json:"target_violated_processes,omitempty"`
	ViolatedFiles     string `json:"target_violated_files,omitempty"`
	FalcoAlert        string `json:"target_falco_alert,omitempty"`
	Pod               *Pod   `json:"pod,omitempty"`
}
