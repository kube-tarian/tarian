package dgraphstore

import (
	"context"

	"github.com/dgraph-io/dgo/v210/protos/api"
)

// schema defines the schema for various Dgraph types such as Constraint, Action, Pod, Event, and Target.
var schema = `
	# Constraint type schema definition
	constraint_namespace: string @index(exact) .
	constraint_name: string @index(exact) .
	constraint_selector: string .
	constraint_allowed_processes: string .
	constraint_allowed_files: string .

	type Constraint {
		constraint_namespace
		constraint_name
		constraint_selector
		constraint_allowed_processes
		constraint_allowed_files
	}

	# Action type schema definition
	action_namespace: string @index(exact) .
	action_name: string @index(exact) .
	action_selector: string .
	action_on_violated_process: bool .
	action_on_violated_file: bool .
	action_on_falco_alert: bool .
	action_falco_alert_priority: int .
	action_action: string .

	type Action {
		action_namespace
		action_name
		action_selector
		action_on_violated_process
		action_on_violated_file
		action_on_falco_alert
		action_falco_alert_priority
		action_action
	}

	# Pod type schema definition
	pod_name: string @index(exact) .
	pod_namespace: string @index(exact) .
	pod_uid: string @index(exact) @upsert .
	pod_labels: string .
	pod: uid .

	type Pod {
		pod_name
		pod_namespace
		pod_uid
		pod_labels
	}

	# Event type schema definition
	event_uid: string @index(exact) @upsert .
	event_type: string @index(exact) .
	event_client_timestamp: dateTime @index(hour) .
	event_server_timestamp: dateTime @index(hour) .
	event_alert_sent_at: dateTime @index(hour) .
	targets: [uid] .

	type Event {
		event_type
		event_uid
		event_client_timestamp
		event_server_timestamp
		event_alert_sent_at

		targets: [Target]
	}

	# Target type schema definition
	target_violated_processes: string . # JSON
	target_violated_files: string . # JSON
	target_falco_alert: string .
	target_detection_data_type: string .
	target_detection_data: string .

	type Target {
		pod: Pod

		target_violated_processes
		target_violated_files
		target_falco_alert
		target_detection_data_type
		target_detection_data
	}
`

// ApplySchema applies the specified schema to the Dgraph database using the provided Dgraph client.
//
// Parameters:
// - ctx: The context for the operation.
// - dg: The Dgraph client instance.
//
// Returns:
// - An error if there was an issue applying the schema.
func (c *client) ApplySchema(ctx context.Context) error {
	op := &api.Operation{}
	op.Schema = schema

	return c.dg.Alter(ctx, op)
}
