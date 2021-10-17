package clusteragent

import (
	"context"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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
}

func newActionHandler(tarianServerAddress string, opts []grpc.DialOption, k8sClientset *kubernetes.Clientset) *actionHandler {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	configClient := tarianpb.NewConfigClient(grpcConn)
	eventClient := tarianpb.NewEventClient(grpcConn)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ah := &actionHandler{eventsChan: make(chan *tarianpb.Event, 4096), configClient: configClient, eventClient: eventClient, k8sClientset: k8sClientset, cancelFunc: cancel, cancelCtx: ctx}

	return ah
}

func (ah *actionHandler) QueueEvent(event *tarianpb.Event) {
	logger.Debugw("event queued", "event", event)

	ah.eventsChan <- event
}

func (ah *actionHandler) Run() {
	go ah.LoopSyncActions()

	for e := range ah.eventsChan {
		ah.ProcessActions(e)
	}
}

func (ah *actionHandler) ProcessActions(event *tarianpb.Event) {
	logger.Debugw("event processed", "event", event)
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

func (ah *actionHandler) isEventTimestampRecent(t *timestamppb.Timestamp, podInfo *tarianpb.Pod) bool {
	if ah.k8sClientset == nil {
		logger.Warnw("about to determine action timestamp is recent, but kubernetes client is nil", "pod", podInfo.GetNamespace(), "namespace", podInfo.GetNamespace())
		return false
	}

	ctx, cancel := context.WithTimeout(ah.cancelCtx, 10*time.Second)
	defer cancel()

	pod, err := ah.k8sClientset.CoreV1().Pods(podInfo.GetNamespace()).Get(ctx, podInfo.GetName(), metaV1.GetOptions{})

	if err != nil {
		logger.Errorw("error while calling get pod to check the created timestamp", "pod", pod.GetNamespace(), "namespace", pod.GetNamespace(), "err", err)
		return false
	}

	return t.AsTime().After(pod.CreationTimestamp.Time)
}

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

	return actionLabels.IsSubset(podLabels)
}

func (ah *actionHandler) runAction(action *tarianpb.Action, pod *tarianpb.Pod) {
	// the only supported action now
	if action.GetAction() != "delete-pod" {
		return
	}

	if ah.k8sClientset == nil {
		logger.Warnw("action due to run, but kubernetes client is nil", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace())
		return
	}

	logger.Infow("run action", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace())

	ctx, cancel := context.WithTimeout(ah.cancelCtx, 120*time.Second)
	defer cancel()

	err := ah.k8sClientset.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{})
	if err == nil {
		err2 := ah.ingestPodDeletedEvent(pod)

		if err2 != nil {
			logger.Errorw("error while logging pod-deleted event", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace(), "error", err)
		}
	} else {
		logger.Infow("error while executing delete-pod action", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace(), "error", err)
	}
}

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

	return err
}

func (ah *actionHandler) LoopSyncActions() error {
	for {
		ah.SyncActions()

		select {
		case <-time.After(3 * time.Second):
		case <-ah.cancelCtx.Done():
			return ah.cancelCtx.Err()
		}
	}
}

func (ah *actionHandler) SyncActions() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := ah.configClient.GetActions(ctx, &tarianpb.GetActionsRequest{})

	if err != nil {
		logger.Errorw("error while getting actions from the server", "err", err)
	}

	logger.Debugw("received actions from the server", "actions", r.GetActions())
	cancel()

	ah.SetActions(r.GetActions())
}

func (ah *actionHandler) SetActions(actions []*tarianpb.Action) {
	ah.actionsLock.Lock()
	defer ah.actionsLock.Unlock()

	ah.actions = actions
}

func (ah *actionHandler) Close() {
	ah.cancelFunc()
}
