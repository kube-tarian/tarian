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

func (es *EventServer) IngestViolationEvent(ctx context.Context, request *tarianpb.IngestViolationEventRequest) (*tarianpb.IngestViolationEventResponse, error) {
	logger.Infow("ingest event", "request", request)

	return nil, nil
}
