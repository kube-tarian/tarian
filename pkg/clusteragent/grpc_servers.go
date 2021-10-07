package clusteragent

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/scylladb/go-set/strset"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

type ConfigServer struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient

	enableAddConstraint bool
}

func NewConfigServer(tarianServerAddress string, opts []grpc.DialOption) *ConfigServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")

	return &ConfigServer{grpcConn: grpcConn, configClient: tarianpb.NewConfigClient(grpcConn)}
}

func (cs *ConfigServer) EnableAddConstraint(value bool) {
	cs.enableAddConstraint = value
}

func (cs *ConfigServer) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Debug("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.GetConstraints(ctx, request)

	return r, err
}

func (cs *ConfigServer) AddConstraint(reqCtx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	if !cs.enableAddConstraint {
		return nil, status.Errorf(codes.Unimplemented, "Method AddConstraint is disabled in tarian-cluster-agent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.AddConstraint(ctx, request)

	return r, err
}

func (cs *ConfigServer) RemoveConstraint(reqCtx context.Context, request *tarianpb.RemoveConstraintRequest) (*tarianpb.RemoveConstraintResponse, error) {
	if !cs.enableAddConstraint {
		return nil, status.Errorf(codes.Unimplemented, "Method RemoveConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.RemoveConstraint(ctx, request)

	return r, err
}

func (cs *ConfigServer) Close() {
	cs.grpcConn.Close()
}

type EventServer struct {
	tarianpb.UnimplementedEventServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
	eventClient  tarianpb.EventClient
	k8sClientset *kubernetes.Clientset

	actions     []*tarianpb.Action
	actionsLock sync.RWMutex

	cancelFunc context.CancelFunc
	cancelCtx  context.Context
}

func NewEventServer(tarianServerAddress string, opts []grpc.DialOption, k8sClientset *kubernetes.Clientset) *EventServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")
	ctx, cancel := context.WithCancel(context.Background())

	return &EventServer{
		grpcConn:     grpcConn,
		configClient: tarianpb.NewConfigClient(grpcConn),
		eventClient:  tarianpb.NewEventClient(grpcConn),
		cancelFunc:   cancel, cancelCtx: ctx,
		k8sClientset: k8sClientset,
	}
}

func (es *EventServer) IngestEvent(requestContext context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Debug("Received ingest violation event RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := es.eventClient.IngestEvent(ctx, request)

	es.processActions(request.GetEvent())

	return r, err
}

func (es *EventServer) processActions(event *tarianpb.Event) {
	if event.GetTargets() == nil {
		return
	}

	for _, target := range event.GetTargets() {
		if target.GetPod() == nil {
			continue
		}

		pod := target.GetPod()

		es.actionsLock.RLock()
		for _, action := range es.actions {
			if actionMatchesPod(action, pod) {
				es.runAction(action, pod)
			}
		}
		es.actionsLock.RUnlock()
	}
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

func (es *EventServer) runAction(action *tarianpb.Action, pod *tarianpb.Pod) {
	// the only supported action now
	if action.GetAction() != "delete-pod" {
		return
	}

	if es.k8sClientset == nil {
		logger.Warnw("action due to run, but kubernetes client is nil", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace())
		return
	}

	logger.Infow("run action", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace())

	ctx, cancel := context.WithTimeout(es.cancelCtx, 120*time.Second)
	defer cancel()

	err := es.k8sClientset.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{})
	if err != nil {
		logger.Infow("error while executing delete-pod action", "actionName", action.GetName(), "action", action.GetAction(), "pod", pod.GetNamespace(), "namespace", pod.GetNamespace(), "error", err)
	}
}

func (es *EventServer) Close() {
	es.grpcConn.Close()
	es.cancelFunc()
}

func (es *EventServer) LoopSyncActions() error {
	for {
		es.SyncActions()

		select {
		case <-time.After(3 * time.Second):
		case <-es.cancelCtx.Done():
			return es.cancelCtx.Err()
		}
	}
}

func (es *EventServer) SyncActions() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := es.configClient.GetActions(ctx, &tarianpb.GetActionsRequest{})

	if err != nil {
		logger.Errorw("error while getting actions from the server", "err", err)
	}

	logger.Debugw("received actions from the server", "actions", r.GetActions())
	cancel()

	es.SetActions(r.GetActions())
}

func (es *EventServer) SetActions(actions []*tarianpb.Action) {
	es.actionsLock.Lock()
	defer es.actionsLock.Unlock()

	es.actions = actions
}
