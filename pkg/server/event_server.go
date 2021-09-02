package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventServer struct {
	tarianpb.UnimplementedEventServer
	eventStore store.EventStore
}

func NewEventServer(dsn string) (*EventServer, error) {
	dbStore, err := dbstore.NewDbEventStore(dsn)

	if err != nil {
		return nil, err
	}
	return &EventServer{eventStore: dbStore}, nil
}

func (es *EventServer) IngestEvent(ctx context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Debugw("ingest event", "request", request)

	event := request.GetEvent()
	if event == nil {
		return nil, status.Error(codes.InvalidArgument, "required event is empty")
	}

	event.ServerTimestamp = timestamppb.Now()

	err := es.eventStore.Add(request.GetEvent())

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
