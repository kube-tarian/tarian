package server

import (
	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IngestionWorker struct {
	eventStore     store.EventStore
	IngestionQueue protoqueue.QueueSubscriber
	logger         *logrus.Logger
}

func NewIngestionWorker(logger *logrus.Logger, eventStore store.EventStore, queueSubscriber protoqueue.QueueSubscriber) *IngestionWorker {
	return &IngestionWorker{
		eventStore:     eventStore,
		IngestionQueue: queueSubscriber,
		logger:         logger,
	}
}

func (iw *IngestionWorker) Start() {
	for {
		msg, err := iw.IngestionQueue.NextMessage(&tarianpb.Event{})
		if err != nil {
			iw.logger.WithError(err).Error("error while processing event")
			continue
		}

		event, ok := msg.(*tarianpb.Event)
		if !ok {
			iw.logger.WithError(err).Error("error while processing event")
			continue
		}

		event.ServerTimestamp = timestamppb.Now()
		err = iw.eventStore.Add(event)

		if err != nil {
			iw.logger.WithError(err).Error("error while processing event")
		}
	}
}
