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

type EventServer struct {
	tarianpb.UnimplementedEventServer
	eventStore     store.EventStore
	ingestionQueue protoqueue.QueuePublisher
	logger         *logrus.Logger
}

func NewEventServer(logger *logrus.Logger, s store.EventStore, ingestionQueue protoqueue.QueuePublisher) *EventServer {
	return &EventServer{
		eventStore:     s,
		ingestionQueue: ingestionQueue,
		logger:         logger,
	}
}

func (es *EventServer) IngestEvent(ctx context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	es.logger.WithFields(logrus.Fields{
		"request": request,
	}).Debug("ingest event")

	event := request.GetEvent()
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "required event is empty")
	}

	err := es.ingestionQueue.Publish(request.GetEvent())

	if err != nil {
		es.logger.WithError(err).Error("error while handling ingest event")
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &tarianpb.IngestEventResponse{Success: true}, nil
}

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
