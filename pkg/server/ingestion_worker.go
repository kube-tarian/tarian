package server

import (
	"encoding/json"

	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IngestionWorker handles the ingestion of events from a message queue.
type IngestionWorker struct {
	eventStore     store.EventStore
	IngestionQueue protoqueue.QueueSubscriber
	logger         *logrus.Logger
}

// NewIngestionWorker creates a new IngestionWorker instance.
//
// Parameters:
// - logger: The logger to use for logging.
// - eventStore: The EventStore to use for storing events.
// - queueSubscriber: The queue subscriber for event ingestion.
//
// Returns:
// - *IngestionWorker: A new instance of IngestionWorker.
func NewIngestionWorker(logger *logrus.Logger, eventStore store.EventStore, queueSubscriber protoqueue.QueueSubscriber) *IngestionWorker {
	return &IngestionWorker{
		eventStore:     eventStore,
		IngestionQueue: queueSubscriber,
		logger:         logger,
	}
}

// Start starts the IngestionWorker, continuously processing messages from the ingestion queue.
//
// Working:
// - The IngestionWorker continuously fetches messages from the ingestion queue.
// - It checks if the message is a valid event.
// - If it is a valid event, it updates the server timestamp and stores the event in the event store.
// - If there are errors during processing, they are logged.
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

		buf, err := json.Marshal(event)
		if err != nil {
			iw.logger.WithError(err).Error("marshaling error: error while processing event")
			continue
		}

		event.ServerTimestamp = timestamppb.Now()
		logrus.Info(">> DEBUG IngestionWorker", "event", string(buf))
		err = iw.eventStore.Add(event)

		if err != nil {
			iw.logger.WithError(err).Error("error while processing event")
		}
	}
}
