package server

import (
	"context"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/queue"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"google.golang.org/grpc/codes"
)

type EventServer struct {
	tarianpb.UnimplementedEventServer
	eventStore     store.EventStore
	ingestionQueue queue.QueuePublisher
}

func NewEventServer(s store.EventStore, ingestionQueue queue.QueuePublisher) *EventServer {
	return &EventServer{eventStore: s, ingestionQueue: ingestionQueue}
}

func (es *EventServer) IngestEvent(ctx context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Debugw("ingest event", "request", request)

	event := request.GetEvent()
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "required event is empty")
	}

	err := es.ingestionQueue.Publish(request.GetEvent())

	if err != nil {
		logger.Errorw("error while handling ingest event", "err", err)
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
		logger.Errorw("error while handling get events RPC", "err", err)
	}

	return &tarianpb.GetEventsResponse{
		Events: events,
	}, nil
}
