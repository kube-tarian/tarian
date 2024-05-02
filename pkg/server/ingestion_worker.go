package server

import (
	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IngestionWorker handles the ingestion of events from a message queue.
type IngestionWorker struct {
	eventStore                      store.EventStore
	IngestionQueue                  protoqueue.QueueSubscriber
	IngestionQueueForEventDetection protoqueue.QueueSubscriber
	logger                          *logrus.Logger
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
func NewIngestionWorker(logger *logrus.Logger, eventStore store.EventStore, queueSubscriber protoqueue.QueueSubscriber, queueSubscriberForEventDetection protoqueue.QueueSubscriber) *IngestionWorker {
	return &IngestionWorker{
		eventStore:                      eventStore,
		IngestionQueue:                  queueSubscriber,
		IngestionQueueForEventDetection: queueSubscriberForEventDetection,
		logger:                          logger,
	}
}

// Start starts the IngestionWorker, continuously processing messages from the ingestion queue.
// It uses a goroutine and a buffered channel to read events from the queue in the background.
func (iw *IngestionWorker) Start() {
	go iw.loopConsumeQueue(iw.IngestionQueue)
	go iw.loopConsumeQueue(iw.IngestionQueueForEventDetection)
}

func (iw *IngestionWorker) loopConsumeQueue(queue protoqueue.QueueSubscriber) {
	eventChan := make(chan *tarianpb.Event, 1000) // buffered channel with capacity 1000

	go func() {
		defer close(eventChan) // close the channel on exit

		for {
			event, err := queue.NextMessage(&tarianpb.Event{})
			if err != nil {
				iw.logger.WithError(err).Error("error while processing event")
				continue
			}

			eventChan <- event.(*tarianpb.Event)
		}
	}()

	defer iw.logger.Info("stopped consuming events from ingestion queue")

	for event := range eventChan {
		iw.processEvent(event)
	}
}

func (iw *IngestionWorker) processEvent(event *tarianpb.Event) {
	event.ServerTimestamp = timestamppb.Now()
	uid := uuid.NewV4()
	event.Uid = uid.String()
	err := iw.eventStore.Add(event)

	if err != nil {
		iw.logger.WithError(err).Error("error while processing event")
	}
}
