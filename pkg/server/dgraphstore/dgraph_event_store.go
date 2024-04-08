package dgraphstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DgraphEventStore is a store for managing Events using Dgraph as the backend.
type dgraphEventStore struct {
	dgraphClient *dgo.Dgraph
}

func newDgraphEventStore(dgraphClient *dgo.Dgraph) store.EventStore {
	return &dgraphEventStore{dgraphClient: dgraphClient}
}

// GetAll retrieves all events from the Dgraph store, ignoring events
// with target_detection_data_type and target_detection_data.
//
// Parameters:
// - limit: The maximum number of events to retrieve.
//
// Returns:
// - An array of protobuf Event messages representing the retrieved events.
// - An error if there was an issue with the database query.
func (d *dgraphEventStore) GetAll(limit uint) ([]*tarianpb.Event, error) {
	// Dgraph query to retrieve all events, ignoring events with eventType as tarian-detection/detection.
	q := fmt.Sprintf(`
	    {
			events(func: type(Event)) @filter(not eq(event_type, "tarian-detection/detection")) {
				%s
			}
		}
	`, eventFields)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tx := d.dgraphClient.NewReadOnlyTxn()
	resp, err := tx.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	var result dgraphEventList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, err
	}

	events := result.toPbEvents()

	return events, nil
}

// dgraphEventList is a helper struct to unmarshal Dgraph query results.
type dgraphEventList struct {
	Events []Event
}

// toPbEvents converts Dgraph Event entities to protobuf Event messages.
//
// Returns:
// - An array of protobuf Event messages.
func (d *dgraphEventList) toPbEvents() []*tarianpb.Event {
	logger := log.GetLogger()
	events := []*tarianpb.Event{}
	for _, evt := range d.Events {
		event := tarianpb.NewEvent()
		event.Type = evt.Type
		event.Uid = evt.EventUID

		if evt.ServerTimestamp != nil {
			event.ServerTimestamp = timestamppb.New(*evt.ServerTimestamp)
		}

		if evt.ClientTimestamp != nil {
			event.ClientTimestamp = timestamppb.New(*evt.ClientTimestamp)
		}

		if evt.AlertSentAt != nil {
			event.AlertSentAt = timestamppb.New(*evt.AlertSentAt)
		}

		event.Targets = []*tarianpb.Target{}

		for _, evtTarget := range evt.Targets {
			t := &tarianpb.Target{}

			if evtTarget.ViolatedProcesses != "" {
				err := json.Unmarshal([]byte(evtTarget.ViolatedProcesses), &t.ViolatedProcesses)
				if err != nil {
					logger.WithError(err).Warn("Failed to unmarshal violated processes")
				}
			}

			if evtTarget.ViolatedFiles != "" {
				err := json.Unmarshal([]byte(evtTarget.ViolatedFiles), &t.ViolatedFiles)
				if err != nil {
					logger.WithError(err).Warn("Failed to unmarshal violated processes")
				}
			}

			if evtTarget.FalcoAlert != "" {
				err := json.Unmarshal([]byte(evtTarget.FalcoAlert), &t.FalcoAlert)
				if err != nil {
					logger.WithError(err).Warn("Failed to unmarshal violated processes")
				}
			}

			if evtTarget.Pod != nil {
				t.Pod = &tarianpb.Pod{}
				t.Pod.Uid = evtTarget.Pod.PodUID
				t.Pod.Namespace = evtTarget.Pod.Namespace
				t.Pod.Name = evtTarget.Pod.Name
				err := json.Unmarshal([]byte(evtTarget.Pod.Labels), &t.Pod.Labels)
				if err != nil {
					logger.WithError(err).Warn("Failed to unmarshal violated processes")
				}
			}

			if t.DetectionDataType != "" {
				t.DetectionDataType = evtTarget.DetectionDataType
			}

			if t.DetectionData != "" {
				t.DetectionData = evtTarget.DetectionData
			}

			event.Targets = append(event.Targets, t)
		}

		events = append(events, event)
	}

	return events
}

// Constants for the fields needed in Dgraph query.
const eventFields = `
	uid
	dgraph.type

	event_type
	event_uid
	event_client_timestamp
	event_server_timestamp
	event_alert_sent_at

	targets {
		uid
		target_violated_processes
		target_violated_files
		target_falco_alert
		pod {
			uid
			pod_uid
			pod_namespace
			pod_name
			pod_labels
		}
		target_detection_data_type
		target_detection_data
	}
`

// FindByNamespace retrieves events from the Dgraph store by namespace and returns them as protobuf Events.
//
// Parameters:
// - namespace: The namespace to filter events by.
// - limit: The maximum number of events to retrieve.
//
// Returns:
// - An array of protobuf Event messages representing matching events.
// - An error if this function is unimplemented.
func (d *dgraphEventStore) FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error) {
	return nil, errors.New("Unimplemented")
}

// Add adds a new event to the Dgraph store.
//
// Parameters:
// - evt: The protobuf Event message to add to the store.
//
// Returns:
// - An error if there was an issue storing the event in the database.
func (d *dgraphEventStore) Add(evt *tarianpb.Event) error {
	dgraphEvent, err := dgraphEventFromPb(evt)
	if err != nil {
		return err
	}

	// Upsert target pods
	for _, evtTarget := range dgraphEvent.Targets {
		if evtTarget.Pod == nil {
			continue
		}

		p, err := d.UpsertPod(*evtTarget.Pod)
		if err != nil {
			return err
		}

		evtTarget.Pod.UID = p.UID
	}

	payload, err := json.Marshal(dgraphEvent)

	if err != nil {
		return err
	}

	mu := &api.Mutation{
		CommitNow: true,
	}

	mu.SetJson = payload

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = d.dgraphClient.NewTxn().Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return nil
}

// dgraphEventFromPb converts a protobuf Event to a Dgraph Event.
//
// Parameters:
// - pbEvent: The protobuf Event message to convert.
//
// Returns:
// - A pointer to the Dgraph Event.
// - An error if there was an issue with the conversion.
func dgraphEventFromPb(pbEvent *tarianpb.Event) (*Event, error) {
	dgraphEvent := &Event{
		UID:   "_:event",
		DType: []string{"Event"},

		Type:     pbEvent.Type,
		EventUID: pbEvent.Uid,

		Targets: []Target{},
	}

	for i, pbTarget := range pbEvent.Targets {
		t := Target{
			UID:   fmt.Sprintf("_:target%d", i),
			DType: []string{"Target"},
		}

		if len(pbTarget.ViolatedProcesses) > 0 {
			violatedProcessesJSON, err := json.Marshal(pbTarget.ViolatedProcesses)
			if err != nil {
				continue
			}

			t.ViolatedProcesses = string(violatedProcessesJSON)
		}

		if len(pbTarget.ViolatedFiles) > 0 {
			violatedFilesJSON, err := json.Marshal(pbTarget.ViolatedFiles)
			if err != nil {
				continue
			}

			t.ViolatedFiles = string(violatedFilesJSON)
		}

		if pbTarget.FalcoAlert != nil {
			violatedFalcoAlertsJSON, err := json.Marshal(pbTarget.FalcoAlert)
			if err != nil {
				continue
			}

			t.FalcoAlert = string(violatedFalcoAlertsJSON)
		}

		if pbTarget.Pod != nil {
			labelsJSON, _ := json.Marshal(pbTarget.Pod.Labels)

			t.Pod = &Pod{
				UID:       fmt.Sprintf("_:pod%d", i),
				DType:     []string{"Pod"},
				Namespace: pbTarget.Pod.Namespace,
				Name:      pbTarget.Pod.Name,
				PodUID:    pbTarget.Pod.Uid,
				Labels:    string(labelsJSON),
			}
		}

		if pbTarget.DetectionDataType != "" {
			t.DetectionDataType = pbTarget.GetDetectionDataType()
		}

		if pbTarget.DetectionData != "" {
			t.DetectionData = pbTarget.GetDetectionData()
		}

		dgraphEvent.Targets = append(dgraphEvent.Targets, t)
	}

	if pbEvent.ClientTimestamp != nil {
		t := pbEvent.ClientTimestamp.AsTime()
		dgraphEvent.ClientTimestamp = &t
	}

	if pbEvent.ServerTimestamp != nil {
		t := pbEvent.ServerTimestamp.AsTime()
		dgraphEvent.ServerTimestamp = &t
	}

	if pbEvent.AlertSentAt != nil {
		t := pbEvent.AlertSentAt.AsTime()
		dgraphEvent.AlertSentAt = &t
	}

	return dgraphEvent, nil
}

// FindWhereAlertNotSent retrieves events from the Dgraph store where the alert has not been sent yet.
//
// Returns:
// - An array of protobuf Event messages representing matching events.
// - An error if there was an issue with the database query.
func (d *dgraphEventStore) FindWhereAlertNotSent() ([]*tarianpb.Event, error) {
	q := fmt.Sprintf(`
	    {
			events(func: type(Event)) @filter(not eq(event_type, "tarian-detection/detection") AND not has(event_alert_sent_at)) {
				%s
			}
		}
	`, eventFields)

	tx := d.dgraphClient.NewReadOnlyTxn()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := tx.Query(ctx, q)
	if err != nil {
		return nil, err
	}

	var result dgraphEventList
	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return nil, err
	}

	events := result.toPbEvents()

	return events, nil
}

// UpdateAlertSent updates the alert sent timestamp for a specific event.
//
// Parameters:
// - uid: The UID of the event to update.
//
// Returns:
// - An error if there was an issue updating the event in the database.
func (d *dgraphEventStore) UpdateAlertSent(uid string) error {
	query := `
		query q($event_uid: string) {
			evt as var(func: eq(event_uid, $event_uid))
		}`

	now := time.Now()
	dgraphEvent := Event{
		UID:         "uid(evt)",
		AlertSentAt: &now,
	}

	payload, err := json.Marshal(dgraphEvent)

	if err != nil {
		return err
	}

	mu := &api.Mutation{
		SetJson: payload,
	}

	req := &api.Request{
		Query:     query,
		Mutations: []*api.Mutation{mu},
		Vars:      map[string]string{"$event_uid": uid},
		CommitNow: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_, err = d.dgraphClient.NewTxn().Do(ctx, req)
	return err
}

// UpsertPod upserts a Pod in the Dgraph store.
//
// Parameters:
// - dgraphPod: The Pod entity to upsert.
//
// Returns:
// - The upserted Pod entity.
// - An error if there was an issue with the database query.
func (d *dgraphEventStore) UpsertPod(dgraphPod Pod) (Pod, error) {
	query := `
		query q($pod_uid: string) {
			pod(func: eq(pod_uid, $pod_uid)) {
				p as uid
			}
		}`

	dgraphPod.UID = "uid(p)"
	payload, err := json.Marshal(dgraphPod)

	if err != nil {
		return dgraphPod, err
	}

	mu := &api.Mutation{
		SetJson: payload,
	}

	req := &api.Request{
		Query:     query,
		Mutations: []*api.Mutation{mu},
		Vars:      map[string]string{"$pod_uid": dgraphPod.PodUID},
		CommitNow: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := d.dgraphClient.NewTxn().Do(ctx, req)

	ok := false
	podUID, ok := resp.Uids["uid(p)"]
	if !ok {
		var r struct {
			Pod []struct {
				UID string `json:"uid,omitempty"`
			}
		}

		err = json.Unmarshal(resp.GetJson(), &r)
		if err != nil {
			return dgraphPod, err
		}

		if len(r.Pod) > 0 {
			podUID = r.Pod[0].UID
		}
	}

	if podUID != "" {
		dgraphPod.UID = podUID
	}

	return dgraphPod, err
}
