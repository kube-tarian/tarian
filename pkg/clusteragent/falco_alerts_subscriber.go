package clusteragent

import (
	"context"
	"time"

	"github.com/falcosecurity/client-go/pkg/api/outputs"
	"github.com/falcosecurity/client-go/pkg/client"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FalcoAlertsSubscriber struct {
	client      *client.Client
	grpcConn    *grpc.ClientConn
	eventClient tarianpb.EventClient
	cancelCtx   context.Context
	cancelFunc  context.CancelFunc

	actionHandler *actionHandler
}

func NewFalcoAlertsSubscriber(tarianServerAddress string, opts []grpc.DialOption, config *client.Config, actionHandler *actionHandler) (*FalcoAlertsSubscriber, error) {
	falcoClient, err := client.NewForConfig(context.Background(), config)

	if err != nil {
		return nil, err
	}

	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &FalcoAlertsSubscriber{
		client:        falcoClient,
		grpcConn:      grpcConn,
		eventClient:   tarianpb.NewEventClient(grpcConn),
		cancelCtx:     ctx,
		cancelFunc:    cancel,
		actionHandler: actionHandler,
	}, nil
}

func (f *FalcoAlertsSubscriber) Start() {
	for {
		select {
		case <-time.After(time.Second):
			err := f.client.OutputsWatch(f.cancelCtx, f.ProcessFalcoOutput, time.Second*1)
			if err != nil {
				logger.Error(err)
			}
		case <-f.cancelCtx.Done():
			return
		}
	}
}

func (f *FalcoAlertsSubscriber) Close() {
	f.cancelFunc()
	f.client.Close()
	f.grpcConn.Close()
}

func (f *FalcoAlertsSubscriber) ProcessFalcoOutput(res *outputs.Response) error {
	var t *timestamppb.Timestamp
	if res.GetTime() != nil {
		t = res.GetTime()
	} else {
		t = timestamppb.Now()
	}

	outputFields := res.GetOutputFields()
	var pod *tarianpb.Pod
	if outputFields != nil {
		pod = &tarianpb.Pod{
			Name:      outputFields["k8s.pod.name"],
			Namespace: outputFields["k8s.ns.name"],
		}
	}

	event := &tarianpb.Event{
		Type:            tarianpb.EventTypeFalcoAlert,
		ClientTimestamp: t,
		Targets: []*tarianpb.Target{
			{
				Pod: pod,
				FalcoAlert: &tarianpb.FalcoAlert{
					Rule:         res.GetRule(),
					Priority:     tarianpb.FalcoPriority(res.GetPriority()),
					Output:       res.GetOutput(),
					OutputFields: res.GetOutputFields(),
				},
			},
		},
	}

	req := &tarianpb.IngestEventRequest{
		Event: event,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	response, err := f.eventClient.IngestEvent(ctx, req)
	defer cancel()

	if err != nil {
		logger.Errorw("error while reporting falco alerts", "err", err)
	} else {
		logger.Debugw("ingest event response", "response", response)

		f.actionHandler.QueueEvent(event)
	}

	return nil
}
