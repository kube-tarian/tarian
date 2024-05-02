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

// ConfigServer handles gRPC requests related to configuration constraints.
type ConfigServer struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient

	enableAddConstraint bool

	logger *logrus.Logger
}

// NewConfigServer creates a new ConfigServer instance and establishes a gRPC connection to the Tarian server.
// It takes a logger, Tarian server address, and gRPC dial options as input.
// Parameters:
//   - logger: A logger instance for logging.
//   - tarianServerAddress: The address of the Tarian server to connect to.
//   - opts: gRPC dial options for configuring the connection.
//
// Returns:
//   - *ConfigServer: A new instance of ConfigServer.
//   - error: An error if connection or initialization fails.
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

// EnableAddConstraint enables or disables the ability to add constraints.
// It takes a boolean value as input to determine whether adding constraints is allowed.
// Parameters:
//   - value: A boolean value indicating whether to enable or disable adding constraints.
func (cs *ConfigServer) EnableAddConstraint(value bool) {
	cs.enableAddConstraint = value
}

// GetConstraints retrieves constraints from the Tarian server.
// Parameters:
//   - reqCtx: The request context.
//   - request: The request object for getting constraints.
//
// Returns:
//   - *tarianpb.GetConstraintsResponse: The response containing constraints.
//   - error: An error if the request fails.
func (cs *ConfigServer) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	cs.logger.Trace("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.GetConstraints(ctx, request)

	return r, err
}

// AddConstraint adds a new constraint to the Tarian server.
// It checks whether adding constraints is enabled and returns an error if not.
// Parameters:
//   - reqCtx: The request context.
//   - request: The request object for adding a constraint.
//
// Returns:
//   - *tarianpb.AddConstraintResponse: The response indicating the result of adding the constraint.
//   - error: An error if the request fails or adding constraints is disabled.
func (cs *ConfigServer) AddConstraint(reqCtx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	if !cs.enableAddConstraint {
		err := status.Errorf(codes.Unimplemented, "Method AddConstraint is disabled in tarian-cluster-agent")
		return nil, fmt.Errorf("AddConstraint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.AddConstraint(ctx, request)
	return r, err
}

// RemoveConstraint removes a constraint from the Tarian server.
// It checks whether adding constraints is enabled and returns an error if not.
// Parameters:
//   - reqCtx: The request context.
//   - request: The request object for removing a constraint.
//
// Returns:
//   - *tarianpb.RemoveConstraintResponse: The response indicating the result of removing the constraint.
//   - error: An error if the request fails or removing constraints is disabled.
func (cs *ConfigServer) RemoveConstraint(reqCtx context.Context, request *tarianpb.RemoveConstraintRequest) (*tarianpb.RemoveConstraintResponse, error) {
	if !cs.enableAddConstraint {
		err := status.Errorf(codes.Unimplemented, "Method RemoveConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
		return nil, fmt.Errorf("RemoveConstraint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.RemoveConstraint(ctx, request)

	return r, err
}

// Close closes the gRPC connection.
func (cs *ConfigServer) Close() {
	cs.grpcConn.Close()
}

// EventServer handles gRPC requests related to events and ingesting violation events.
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

// NewEventServer creates a new EventServer instance and establishes a gRPC connection to the Tarian server.
// It takes a logger, Tarian server address, gRPC dial options, and an actionHandler as input.
// Parameters:
//   - logger: A logger instance for logging.
//   - tarianServerAddress: The address of the Tarian server to connect to.
//   - opts: gRPC dial options for configuring the connection.
//   - actionHandler: An instance of actionHandler for handling queued events.
//
// Returns:
//   - *EventServer: A new instance of EventServer.
//   - error: An error if connection or initialization fails.
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

// IngestEvent ingests a violation event to the Tarian server and queues it for processing.
// Parameters:
//   - requestContext: The request context.
//   - request: The request object for ingesting the violation event.
//
// Returns:
//   - *tarianpb.IngestEventResponse: The response indicating the result of ingesting the event.
//   - error: An error if the request fails.
func (es *EventServer) IngestEvent(requestContext context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	es.logger.Trace("Received ingest event RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := es.eventClient.IngestEvent(ctx, request)
	es.actionHandler.QueueEvent(request.GetEvent())

	return r, err
}

// Close closes the gRPC connection and cancels the context.
func (es *EventServer) Close() {
	es.grpcConn.Close()
	es.cancelFunc()
}
