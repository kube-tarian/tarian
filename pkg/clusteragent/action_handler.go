package clusteragent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ActionHandler is an interface for handling Tarian actions.
type ActionHandler interface {
	QueueEvent(*tarianpb.Event)
}

type actionHandler struct {
	eventsChan chan *tarianpb.Event

	actions     []*tarianpb.Action
	actionsLock sync.RWMutex

	configClient tarianpb.ConfigClient
	eventClient  tarianpb.EventClient

	k8sClientset *kubernetes.Clientset

	cancelFunc context.CancelFunc
	cancelCtx  context.Context
	logger     *logrus.Logger
}

// newActionHandler creates a new actionHandler instance.
// It establishes a connection to the Tarian server, initializes clients,
// and prepares the actionHandler for event processing.
func newActionHandler(logger *logrus.Logger, tarianServerAddress string, opts []grpc.DialOption, k8sClientset *kubernetes.Clientset) (*actionHandler, error) {
	logger.WithField("address", tarianServerAddress).Info("connecting to the tarian server")
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	configClient := tarianpb.NewConfigClient(grpcConn)
	eventClient := tarianpb.NewEventClient(grpcConn)

	if err != nil {
		logger.WithError(err).Error("couldn't not connect to tarian-server")
		return nil, fmt.Errorf("newActionHandler: couldn't not connect to tarian-server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &actionHandler{
		eventsChan:   make(chan *tarianpb.Event, 4096),
		configClient: configClient,
		eventClient:  eventClient,
		k8sClientset: k8sClientset,
		cancelFunc:   cancel,
		cancelCtx:    ctx,
		logger:       logger,
	}, nil
}

// QueueEvent queues a Tarian event for processing.
// It logs the queued event and adds it to the events channel.
func (ah *actionHandler) QueueEvent(event *tarianpb.Event) {
	ah.logger.WithField("event", event).Trace("event queued")
	ah.eventsChan <- event
}

// Run starts processing Tarian events and actions.
// It launches a goroutine to synchronize actions and processes queued events.
func (ah *actionHandler) Run() {
	go func() {
		_ = ah.LoopSyncActions()
	}()

	for e := range ah.eventsChan {
		ah.ProcessActions(e)
	}
}

// ProcessActions processes Tarian actions based on Tarian events.
// It checks if actions should be executed for each event target and invokes the appropriate action.
func (ah *actionHandler) ProcessActions(event *tarianpb.Event) {
	ah.logger.WithField("event", event).Trace("event processed")
	if event.GetTargets() == nil {
		return
	}

	for _, target := range event.GetTargets() {
		if target.GetPod() == nil {
			continue
		}

		pod := target.GetPod()

		ah.actionsLock.RLock()
		for _, action := range ah.actions {
			actionEventFulfilled := false

			if action.OnViolatedProcess && len(target.GetViolatedProcesses()) > 0 {
				actionEventFulfilled = true
			}

			if action.OnViolatedFile && len(target.GetViolatedFiles()) > 0 {
				actionEventFulfilled = true
			}

			if action.OnFalcoAlert && target.GetFalcoAlert() != nil {
				if target.GetFalcoAlert().GetPriority() <= action.GetFalcoPriority() {
					actionEventFulfilled = true
				}
			}

			if actionEventFulfilled && actionMatchesPod(action, pod) {
				// check if action timestamp is greater than pod's creation timestamp
				// to prevent the pod from being terminated multiple times
				if ah.isEventTimestampRecent(event.GetClientTimestamp(), pod) {
					ah.runAction(action, pod)
				}
			}
		}
		ah.actionsLock.RUnlock()
	}
}

// isEventTimestampRecent checks if the event timestamp is recent compared to the pod's creation timestamp.
// It ensures that the same action is not executed multiple times on the same pod.
func (ah *actionHandler) isEventTimestampRecent(t *timestamppb.Timestamp, podInfo *tarianpb.Pod) bool {
	if ah.k8sClientset == nil {
		ah.logger.WithFields(logrus.Fields{
			"pod":       podInfo.GetNamespace(),
			"namespace": podInfo.GetNamespace(),
		}).Warn("about to determine action timestamp is recent, but kubernetes client is nil")
		return false
	}

	ctx, cancel := context.WithTimeout(ah.cancelCtx, 10*time.Second)
	defer cancel()

	pod, err := ah.k8sClientset.CoreV1().Pods(podInfo.GetNamespace()).Get(ctx, podInfo.GetName(), metaV1.GetOptions{})
	if err != nil {
		ah.logger.WithFields(logrus.Fields{
			"pod":       podInfo.GetNamespace(),
			"namespace": podInfo.GetNamespace(),
			"err":       err,
		}).Warn("about to determine action timestamp is recent, but kubernetes client is nil")
		return false
	}

	return t.AsTime().After(pod.CreationTimestamp.Time)
}

// actionMatchesPod checks if a Tarian action matches a pod based on labels and namespace.
// It compares labels from the action and the pod to determine if the action applies to the pod.
func actionMatchesPod(action *tarianpb.Action, pod *tarianpb.Pod) bool {
	if action.GetNamespace() != pod.GetNamespace() {
		return false
	}

	if action.GetSelector() == nil || action.GetSelector().GetMatchLabels() == nil {
		return true
	}

	actionLabels := strset.New()
	for _, l := range action.GetSelector().GetMatchLabels() {
		actionLabels.Add(l.GetKey() + "=" + l.GetValue())
	}

	podLabels := strset.New()
	for _, l := range pod.GetLabels() {
		podLabels.Add(l.GetKey() + "=" + l.GetValue())
	}

	return podLabels.IsSubset(actionLabels)
}

// runAction executes a Tarian action, such as deleting a pod.
// It checks if the action is "delete-pod" and if the Kubernetes client is available.
func (ah *actionHandler) runAction(action *tarianpb.Action, pod *tarianpb.Pod) {
	// the only supported action now
	if action.GetAction() != "delete-pod" {
		return
	}

	if ah.k8sClientset == nil {
		ah.logger.WithFields(logrus.Fields{
			"pod":       pod.GetNamespace(),
			"namespace": pod.GetNamespace(),
		}).Warn("action due to run, but kubernetes client is nil")
		return
	}

	ah.logger.WithFields(logrus.Fields{
		"actionName": action.GetName(),
		"action":     action.GetAction(),
		"pod":        pod.GetNamespace(),
		"namespace":  pod.GetNamespace(),
	}).Info("run action")

	ctx, cancel := context.WithTimeout(ah.cancelCtx, 120*time.Second)
	defer cancel()

	err := ah.k8sClientset.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{})
	if err == nil {
		err2 := ah.ingestPodDeletedEvent(pod)
		if err2 != nil {
			ah.logger.WithFields(logrus.Fields{
				"actionName": action.GetName(),
				"action":     action.GetAction(),
				"pod":        pod.GetNamespace(),
				"namespace":  pod.GetNamespace(),
				"error":      err,
			}).Error("error while logging pod-deleted event")
		}
	} else {
		ah.logger.WithFields(logrus.Fields{
			"actionName": action.GetName(),
			"action":     action.GetAction(),
			"pod":        pod.GetNamespace(),
			"namespace":  pod.GetNamespace(),
			"error":      err,
		}).Error("error while executing delete-pod action")
	}
}

// ingestPodDeletedEvent sends an event to Tarian when a pod is deleted.
// It creates and sends a pod-deleted event to the Tarian EventClient.
func (ah *actionHandler) ingestPodDeletedEvent(pod *tarianpb.Pod) error {
	event := &tarianpb.Event{
		Type:            tarianpb.EventTypePodDeleted,
		ClientTimestamp: timestamppb.Now(),
		Targets: []*tarianpb.Target{
			{
				Pod: pod,
			},
		},
	}

	req := &tarianpb.IngestEventRequest{
		Event: event,
	}

	ctx2, cancel2 := context.WithTimeout(ah.cancelCtx, 10*time.Second)
	defer cancel2()

	_, err := ah.eventClient.IngestEvent(ctx2, req)
	return fmt.Errorf("ingestPodDeletedEvent: %w", err)
}

// LoopSyncActions continuously synchronizes Tarian actions.
// It periodically calls SyncActions to retrieve the latest actions from the server.
func (ah *actionHandler) LoopSyncActions() error {
	for {
		ah.SyncActions()

		select {
		case <-time.After(3 * time.Second):
		case <-ah.cancelCtx.Done():
			return fmt.Errorf("LoopSyncActions: %w", ah.cancelCtx.Err())
		}
	}
}

// SyncActions synchronizes Tarian actions with the server.
// It fetches the latest actions from the server and updates the actions list.
func (ah *actionHandler) SyncActions() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := ah.configClient.GetActions(ctx, &tarianpb.GetActionsRequest{})
	if err != nil {
		ah.logger.WithError(err).Error("error while getting actions from the server")
	}

	ah.logger.WithField("actions", r.GetActions()).Trace("received actions from the server")
	cancel()

	ah.SetActions(r.GetActions())
}

// SetActions sets the list of Tarian actions.
// It updates the actions list while holding a lock to ensure thread safety.
func (ah *actionHandler) SetActions(actions []*tarianpb.Action) {
	ah.actionsLock.Lock()
	defer ah.actionsLock.Unlock()

	ah.actions = actions
}

// Close stops the action handler by canceling its context.
func (ah *actionHandler) Close() {
	ah.cancelFunc()
}
