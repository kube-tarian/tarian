package server

import (
	"github.com/kube-tarian/tarian/pkg/queue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IngestionWorker struct {
	eventStore     store.EventStore
	IngestionQueue queue.QueueSubscriber
}

func NewIngestionWorker(eventStore store.EventStore, queueSubscriber queue.QueueSubscriber) *IngestionWorker {
	return &IngestionWorker{eventStore: eventStore, IngestionQueue: queueSubscriber}
}

func (iw *IngestionWorker) Start() {
	for {
		msg, _ := iw.IngestionQueue.NextMessage()

		if event, ok := msg.(*tarianpb.Event); ok {
			event.ServerTimestamp = timestamppb.Now()
			err := iw.eventStore.Add(event)

			if err != nil {
				logger.Errorw("error while processing event", "err", err)
			}
		}
	}
}
