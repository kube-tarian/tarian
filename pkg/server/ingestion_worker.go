package server

import (
	"encoding/json"

	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IngestionWorker struct {
	eventStore     store.EventStore
	IngestionQueue protoqueue.QueueSubscriber
}

func NewIngestionWorker(eventStore store.EventStore, queueSubscriber protoqueue.QueueSubscriber) *IngestionWorker {
	return &IngestionWorker{eventStore: eventStore, IngestionQueue: queueSubscriber}
}

func (iw *IngestionWorker) Start() {
	for {
		msg, err := iw.IngestionQueue.NextMessage(&tarianpb.Event{})
		if err != nil {
			logger.Errorw("error while processing event", "err", err)
			continue
		}

		event, ok := msg.(*tarianpb.Event)
		if !ok {
			logger.Errorw("error while processing event")
			continue
		}

		buf, err := json.Marshal(event)
		logger.Infow(">>>> DEBUG IngestionWorker", "event", string(buf), "err", err)

		event.ServerTimestamp = timestamppb.Now()
		err = iw.eventStore.Add(event)

		if err != nil {
			logger.Errorw("error while processing event", "err", err)
		}
	}
}
