package server

import (
	"context"
	"fmt"

	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/store"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
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

	err := es.eventStore.Add(request.GetEvent())

	if err != nil {
		fmt.Println(err)
		return &tarianpb.IngestEventResponse{Success: false}, nil
	}

	return &tarianpb.IngestEventResponse{Success: true}, nil
}
