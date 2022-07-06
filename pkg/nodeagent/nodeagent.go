package nodeagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

type NodeAgent struct {
	clusterAgentAddress string
	grpcConn            *grpc.ClientConn
	configClient        tarianpb.ConfigClient
	eventClient         tarianpb.EventClient
	podName             string
	podUID              string
	podLabels           []*tarianpb.Label
	namespace           string

	constraints            []*tarianpb.Constraint
	constraintsLock        sync.RWMutex
	constraintsInitialized bool

	cancelFunc context.CancelFunc
	cancelCtx  context.Context

	enableRegisterProcesses bool
}

func NewNodeAgent(clusterAgentAddress string) *NodeAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &NodeAgent{clusterAgentAddress: clusterAgentAddress, cancelCtx: ctx, cancelFunc: cancel, constraintsInitialized: false}
}

func (n *NodeAgent) SetPodLabels(labels []*tarianpb.Label) {
	n.podLabels = labels
}

func (n *NodeAgent) SetPodName(name string) {
	n.podName = name
}

func (n *NodeAgent) SetpodUID(uid string) {
	n.podUID = uid
}

func (n *NodeAgent) SetNamespace(namespace string) {
	n.namespace = namespace
}

func (n *NodeAgent) Dial() {
	var err error
	n.grpcConn, err = grpc.Dial(n.clusterAgentAddress, grpc.WithInsecure())
	n.configClient = tarianpb.NewConfigClient(n.grpcConn)
	n.eventClient = tarianpb.NewEventClient(n.grpcConn)

	if err != nil {
		logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}
}

func (n *NodeAgent) GracefulStop() {
	n.cancelFunc()
}

func (n *NodeAgent) Run() {
	n.Dial()
	defer n.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		n.loopSyncConstraints(n.cancelCtx)
		wg.Done()
	}()

	go func() {
		n.loopValidateProcesses(n.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

func (n *NodeAgent) SetConstraints(constraints []*tarianpb.Constraint) {
	n.constraintsLock.Lock()
	defer n.constraintsLock.Unlock()

	n.constraints = constraints
}

func (n *NodeAgent) GetConstraints() []*tarianpb.Constraint {
	return n.constraints
}

func (n *NodeAgent) loopSyncConstraints(ctx context.Context) error {
	for {
		n.SyncConstraints()

		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *NodeAgent) SyncConstraints() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := n.configClient.GetConstraints(ctx, &tarianpb.GetConstraintsRequest{Namespace: n.namespace, Labels: n.podLabels})

	if err != nil {
		logger.Errorw("error while getting constraints from the cluster agent", "err", err)
	}

	logger.Debugw("received constraints from the cluster agent", "constraint", r.GetConstraints())
	cancel()

	n.SetConstraints(r.GetConstraints())

	n.constraintsInitialized = true
}

func (n *NodeAgent) loopValidateProcesses(ctx context.Context) error {
	captureExec, err := NewCaptureExec()
	if err != nil {
		logger.Fatal(err)
	}

	execEvent := captureExec.GetEventsChannel()
	go captureExec.Start()

	for {
		select {
		case <-ctx.Done():
			captureExec.Close()
			return ctx.Err()
		case e := <-execEvent:
			if !n.constraintsInitialized {
				continue
			}

			fmt.Printf("%d %s %s %s %s %s\n", e.Pid, e.Comm, e.Filename, e.ContainerID, e.K8sPodName, e.K8sNamespace)
		}
	}
}
