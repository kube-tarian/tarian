package dgraphstore

import (
	"context"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
)

var schema = `
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

	target_violated_processes: string . # json
	target_violated_files: string . # json
	target_falco_alert: string .
	target_detection_data_type: string .
	target_detection_data: string . # json

	type Target {
		pod: Pod

		target_violated_processes
		target_violated_files
		target_falco_alert

		target_detection_data_type
		target_detection_data
	}
`

func ApplySchema(ctx context.Context, dg *dgo.Dgraph) error {
	op := &api.Operation{}
	op.Schema = schema

	return dg.Alter(ctx, op)
}
