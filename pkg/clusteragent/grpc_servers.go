package clusteragent

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type ConfigServer struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient

	enableAddConstraint bool

	logger *logrus.Logger
}

func NewConfigServer(logger *logrus.Logger, tarianServerAddress string, opts []grpc.DialOption) (*ConfigServer, error) {
	logger.WithField("address", tarianServerAddress).Info("connecting to the tarian server")

	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	if err != nil {
		logger.WithError(err).Error("couldn't not connect to tarian-server")
		return nil, fmt.Errorf("NewConfigServer: couldn't not connect to tarian-server: %w", err)
	}

	logger.Info("connected to the tarian server")

	return &ConfigServer{
		grpcConn:     grpcConn,
		configClient: tarianpb.NewConfigClient(grpcConn),
		logger:       logger,
	}, nil
}

func (cs *ConfigServer) EnableAddConstraint(value bool) {
	cs.enableAddConstraint = value
}

func (cs *ConfigServer) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	cs.logger.Debug("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.GetConstraints(ctx, request)

	return r, err
}

func (cs *ConfigServer) AddConstraint(reqCtx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	if !cs.enableAddConstraint {
		err := status.Errorf(codes.Unimplemented, "Method AddConstraint is disabled in tarian-cluster-agent")
		return nil, fmt.Errorf("AddConstraint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.AddConstraint(ctx, request)
	return r, fmt.Errorf("AddConstraint: %w", err)
}

func (cs *ConfigServer) RemoveConstraint(reqCtx context.Context, request *tarianpb.RemoveConstraintRequest) (*tarianpb.RemoveConstraintResponse, error) {
	if !cs.enableAddConstraint {
		err := status.Errorf(codes.Unimplemented, "Method RemoveConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
		return nil, fmt.Errorf("RemoveConstraint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.RemoveConstraint(ctx, request)

	return r, fmt.Errorf("RemoveConstraint: %w", err)
}

func (cs *ConfigServer) Close() {
	cs.grpcConn.Close()
}

type EventServer struct {
	tarianpb.UnimplementedEventServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
	eventClient  tarianpb.EventClient

	cancelFunc context.CancelFunc
	cancelCtx  context.Context

	actionHandler *actionHandler
	logger        *logrus.Logger
}

func NewEventServer(logger *logrus.Logger, tarianServerAddress string, opts []grpc.DialOption, actionHandler *actionHandler) (*EventServer, error) {
	logger.WithField("address", tarianServerAddress).Info("connecting to the tarian server")

	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)
	if err != nil {
		logger.WithError(err).Error("couldn't not connect to tarian-server")
		return nil, fmt.Errorf("NewEventServer: couldn't not connect to tarian-server: %w", err)
	}

	logger.Info("connected to the tarian server")

	ctx, cancel := context.WithCancel(context.Background())

	return &EventServer{
		grpcConn:     grpcConn,
		configClient: tarianpb.NewConfigClient(grpcConn),
		eventClient:  tarianpb.NewEventClient(grpcConn),
		cancelFunc:   cancel, cancelCtx: ctx,
		actionHandler: actionHandler,
		logger:        logger,
	}, nil
}

func (es *EventServer) IngestEvent(requestContext context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	es.logger.Debug("Received ingest violation event RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := es.eventClient.IngestEvent(ctx, request)
	es.actionHandler.QueueEvent(request.GetEvent())

	return r, fmt.Errorf("IngestEvent: %w", err)
}

func (es *EventServer) Close() {
	es.grpcConn.Close()
	es.cancelFunc()
}
