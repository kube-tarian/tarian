package server

import (
	"context"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

type EventServer struct {
	tarianpb.UnimplementedEventServer
}

func NewEventServer() (*EventServer, error) {
	return &EventServer{}, nil
}

func (es *EventServer) IngestViolationEvent(context.Context, *tarianpb.IngestViolationEventRequest) (*tarianpb.IngestViolationEventResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IngestViolationEvent not implemented")
}
