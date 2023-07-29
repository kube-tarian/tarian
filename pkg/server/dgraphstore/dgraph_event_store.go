package dgraphstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"golang.org/x/net/context"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DgraphEventStore struct {
	dgraphClient *dgo.Dgraph
}

func NewDgraphEventStore(dgraphClient *dgo.Dgraph) *DgraphEventStore {
	d := &DgraphEventStore{dgraphClient: dgraphClient}

	return d
}

func (d *DgraphEventStore) GetAll(limit uint) ([]*tarianpb.Event, error) {
	q := fmt.Sprintf(`
		{
			events(func: type(Event)) {
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

type dgraphEventList struct {
	Events []Event
}

func (d *dgraphEventList) toPbEvents() []*tarianpb.Event {
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
				json.Unmarshal([]byte(evtTarget.ViolatedProcesses), &t.ViolatedProcesses)
			}

			if evtTarget.ViolatedFiles != "" {
				json.Unmarshal([]byte(evtTarget.ViolatedFiles), &t.ViolatedFiles)
			}

			if evtTarget.FalcoAlert != "" {
				json.Unmarshal([]byte(evtTarget.FalcoAlert), &t.FalcoAlert)
			}

			if evtTarget.Pod != nil {
				t.Pod = &tarianpb.Pod{}
				t.Pod.Uid = evtTarget.Pod.PodUID
				t.Pod.Namespace = evtTarget.Pod.Namespace
				t.Pod.Name = evtTarget.Pod.Name
				json.Unmarshal([]byte(evtTarget.Pod.Labels), &t.Pod.Labels)
			}

			t.DetectionDataType = evtTarget.DetectionDataType
			t.DetectionData = evtTarget.DetectionData

			event.Targets = append(event.Targets, t)
		}

		events = append(events, event)
	}

	return events
}

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

func (d *DgraphEventStore) FindByNamespace(namespace string, limit uint) ([]*tarianpb.Event, error) {
	return nil, errors.New("Unimplemented")
}

func (d *DgraphEventStore) Add(evt *tarianpb.Event) error {
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

		t.DetectionDataType = pbTarget.GetDetectionDataType()
		t.DetectionData = pbTarget.GetDetectionData()

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

func (d *DgraphEventStore) FindWhereAlertNotSent() ([]*tarianpb.Event, error) {
	q := fmt.Sprintf(`
	    {
			events(func: type(Event)) @filter(not has(event_alert_sent_at)) {
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

func (d *DgraphEventStore) UpdateAlertSent(uid string) error {
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

func (d *DgraphEventStore) UpsertPod(dgraphPod Pod) (Pod, error) {
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
