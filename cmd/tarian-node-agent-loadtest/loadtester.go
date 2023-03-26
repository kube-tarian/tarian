package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	pkglogger "github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/nodeagent"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type loadtester struct {
	clusterAgentAddress string
	nodeName            string
	grpcConn            *grpc.ClientConn
	configClient        tarianpb.ConfigClient
	eventClient         tarianpb.EventClient
	logger              *zap.SugaredLogger
	cancelFunc          context.CancelFunc
	cancelCtx           context.Context
}

func newLoadTester(clusterAgentAddress string) *loadtester {
	l := pkglogger.GetLogger("info", "json")
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	return &loadtester{
		clusterAgentAddress: clusterAgentAddress,
		logger:              l,
		cancelFunc:          cancelFunc,
		cancelCtx:           cancelCtx,
	}
}

func (l *loadtester) SetNodeName(name string) {
	l.nodeName = name
}

func (l *loadtester) SetLogger(logger *zap.SugaredLogger) {
	l.logger = logger
}

func (l *loadtester) Run() {
	l.Dial()
	defer l.grpcConn.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		l.loopGenerateLoad(l.cancelCtx)
		wg.Done()
	}()

	wg.Wait()
}

func (l *loadtester) GracefulStop() {
	l.cancelFunc()
	l.grpcConn.Close()
}

func (l *loadtester) Dial() {
	var err error
	l.grpcConn, err = grpc.Dial(
		l.clusterAgentAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)

	l.configClient = tarianpb.NewConfigClient(l.grpcConn)
	l.eventClient = tarianpb.NewEventClient(l.grpcConn)

	if err != nil {
		l.logger.Fatalw("couldn't connect to the cluster agent", "err", err)
	}
}

func (l *loadtester) loopGenerateLoad(ctx context.Context) error {
	for {
		l.generateLoad()

		select {
		case <-time.After(1 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *loadtester) generateLoad() {
	violation := &nodeagent.ProcessViolation{}
	violation.Pid = uint32(time.Now().Unix())
	violation.Comm = fmt.Sprintf("loadtest-%s-%d", l.nodeName, rand.Intn(50))
	violation.K8sPodUID = fmt.Sprintf("loadtest-%s-%d", l.nodeName, rand.Intn(50))
	violation.K8sPodName = violation.K8sPodUID
	violation.K8sNamespace = "loadtest"

	violatedProcesses := make([]*tarianpb.Process, 1)
	violatedProcesses[0] = &tarianpb.Process{Pid: int32(violation.Pid), Name: violation.Comm}

	pbPodLabels := make([]*tarianpb.Label, len(violation.K8sPodLabels))
	pbPodLabels = append(pbPodLabels, &tarianpb.Label{Key: "key1", Value: "value1"})
	pbPodLabels = append(pbPodLabels, &tarianpb.Label{Key: "key2", Value: "value2"})

	req := &tarianpb.IngestEventRequest{
		Event: &tarianpb.Event{
			Type:            tarianpb.EventTypeViolation,
			ClientTimestamp: timestamppb.Now(),
			Targets: []*tarianpb.Target{
				{
					Pod: &tarianpb.Pod{
						Uid:       violation.K8sPodUID,
						Name:      violation.K8sPodName,
						Namespace: violation.K8sNamespace,
						Labels:    pbPodLabels,
					},
					ViolatedProcesses: violatedProcesses,
				},
			},
		},
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	response, err := l.eventClient.IngestEvent(ctx, req)

	if err != nil {
		l.logger.Errorw("error while reporting violation events", "err", err)
	} else {
		l.logger.Infow("ingest event response", "response", response)
	}
}
