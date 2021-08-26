package clusteragent

import (
	"context"
	"fmt"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/falcosecurity/client-go/pkg/api/outputs"
	"github.com/falcosecurity/client-go/pkg/client"
	"github.com/gogo/protobuf/jsonpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FalcoAlertsSubscriber struct {
	client *client.Client
}

func NewFalcoAlertsSubscriber(config *client.Config) (*FalcoAlertsSubscriber, error) {
	falcoClient, err := client.NewForConfig(context.Background(), config)

	if err != nil {
		return nil, err
	}

	return &FalcoAlertsSubscriber{
		client: falcoClient,
	}, nil
}

func (f *FalcoAlertsSubscriber) Start() {
	ctx := context.Background()

	err := f.client.OutputsWatch(ctx, f.ProcessFalcoOutput, time.Second*1)
	if err != nil {
		logger.Fatalw("falco: outputs watch error", err, "err")
	}
}

func (f *FalcoAlertsSubscriber) Close() {
	f.client.Close()
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
					Priority:     res.GetPriority().String(),
					Output:       res.GetOutput(),
					OutputFields: res.GetOutputFields(),
				},
			},
		},
	}

	out, err := (&jsonpb.Marshaler{}).MarshalToString(event)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
