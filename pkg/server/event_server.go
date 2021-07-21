package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
)

type EventServer struct {
	tarianpb.UnimplementedEventServer
}

func NewEventServer() (*EventServer, error) {
	return &EventServer{}, nil
}

func (es *EventServer) IngestEvent(ctx context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Infow("ingest event", "request", request)

	return nil, nil
}
