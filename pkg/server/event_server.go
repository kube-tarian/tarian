package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
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
	logger.Infow("ingest event", "request", request)

	event := request.GetEvent()
	if event == nil {
		return &tarianpb.IngestEventResponse{Success: false}, nil
	}

	event.ServerTimestamp = timestamppb.Now()

	err := es.eventStore.Add(request.GetEvent())

	if err != nil {
		logger.Errorw("error while handling ingest event", "err", err)
		return &tarianpb.IngestEventResponse{Success: false}, nil
	}

	return &tarianpb.IngestEventResponse{Success: true}, nil
}

func (es *EventServer) GetEvents(ctxt context.Context, request *tarianpb.GetEventsRequest) (*tarianpb.GetEventsResponse, error) {
	var events []*tarianpb.Event

	if request.GetNamespace() == "" {
		events, _ = es.eventStore.GetAll()
	} else {
		events, _ = es.eventStore.FindByNamespace(request.GetNamespace())
	}

	return &tarianpb.GetEventsResponse{
		Events: events,
	}, nil
}
