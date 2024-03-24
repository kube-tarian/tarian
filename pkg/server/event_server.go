package server

import (
	"context"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/protoqueue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

// EventServer handles gRPC calls related to event ingestion and retrieval.
type EventServer struct {
	tarianpb.UnimplementedEventServer
	eventStore                      store.EventStore
	ingestionQueue                  protoqueue.QueuePublisher
	ingestionQueueForEventDetection protoqueue.QueuePublisher
	logger                          *logrus.Logger
}

// NewEventServer creates a new EventServer instance.
//
// Parameters:
// - logger: The logger to use for logging.
// - s: The EventStore to use for storing events.
// - ingestionQueue: The queue publisher for event ingestion.
//
// Returns:
// - *EventServer: A new instance of EventServer.
func NewEventServer(logger *logrus.Logger, s store.EventStore, ingestionQueue protoqueue.QueuePublisher, ingestionQueueForEventDetection protoqueue.QueuePublisher) *EventServer {
	return &EventServer{
		eventStore:                      s,
		ingestionQueue:                  ingestionQueue,
		ingestionQueueForEventDetection: ingestionQueueForEventDetection,
		logger:                          logger,
	}
}

// IngestEvent ingests a new event into the system.
//
// Parameters:
// - ctx: The context for the operation.
// - request: The IngestEventRequest containing the event to ingest.
//
// Returns:
// - *tarianpb.IngestEventResponse: The response indicating the success of the operation.
// - error: An error, if any, during the operation.
func (es *EventServer) IngestEvent(ctx context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	es.logger.WithFields(logrus.Fields{
		"request": request,
	}).Trace("ingest event")

	event := request.GetEvent()
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "required event is empty")
	}

	var err error
	if event.Type == tarianpb.EventTypeDetection {
		err = es.ingestionQueueForEventDetection.Publish(request.GetEvent())
	} else {
		err = es.ingestionQueue.Publish(request.GetEvent())
	}

	if err != nil {
		es.logger.WithError(err).Error("error while handling ingest event")
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &tarianpb.IngestEventResponse{Success: true}, nil
}

// GetEvents retrieves events based on the provided request.
//
// Parameters:
// - ctxt: The context for the operation.
// - request: The GetEventsRequest containing filter criteria.
//
// Returns:
// - *tarianpb.GetEventsResponse: The response containing matched events.
// - error: An error, if any, during the operation.
func (es *EventServer) GetEvents(ctxt context.Context, request *tarianpb.GetEventsRequest) (*tarianpb.GetEventsResponse, error) {
	var events []*tarianpb.Event
	var err error

	limit := request.GetLimit()

	// TODO: validate limit
	if limit == 0 {
		limit = 1000
	}

	if request.GetNamespace() == "" {
		events, err = es.eventStore.GetAll(uint(limit))
	} else {
		events, err = es.eventStore.FindByNamespace(request.GetNamespace(), uint(limit))
	}

	if err != nil {
		es.logger.WithError(err).Error("error while handling get events RPC")
	}

	return &tarianpb.GetEventsResponse{
		Events: events,
	}, nil
}
