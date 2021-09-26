package clusteragent

import (
	"context"
	"time"

	"github.com/gogo/status"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

type ConfigServer struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient

	enableAddConstraint bool
}

func NewConfigServer(tarianServerAddress string, opts []grpc.DialOption) *ConfigServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")

	return &ConfigServer{grpcConn: grpcConn, configClient: tarianpb.NewConfigClient(grpcConn)}
}

func (cs *ConfigServer) EnableAddConstraint(value bool) {
	cs.enableAddConstraint = value
}

func (cs *ConfigServer) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Debug("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.GetConstraints(ctx, request)

	return r, err
}

func (cs *ConfigServer) AddConstraint(reqCtx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	if !cs.enableAddConstraint {
		return nil, status.Errorf(codes.Unimplemented, "Method AddConstraint is disabled in tarian-cluster-agent")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.AddConstraint(ctx, request)

	return r, err
}

func (cs *ConfigServer) RemoveConstraint(reqCtx context.Context, request *tarianpb.RemoveConstraintRequest) (*tarianpb.RemoveConstraintResponse, error) {
	if !cs.enableAddConstraint {
		return nil, status.Errorf(codes.Unimplemented, "Method RemoveConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := cs.configClient.RemoveConstraint(ctx, request)

	return r, err
}

func (cs *ConfigServer) Close() {
	cs.grpcConn.Close()
}

type EventServer struct {
	tarianpb.UnimplementedEventServer

	grpcConn    *grpc.ClientConn
	eventClient tarianpb.EventClient
}

func NewEventServer(tarianServerAddress string, opts []grpc.DialOption) *EventServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")

	return &EventServer{grpcConn: grpcConn, eventClient: tarianpb.NewEventClient(grpcConn)}
}

func (es *EventServer) IngestEvent(requestContext context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Debug("Received ingest violation event RPC")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := es.eventClient.IngestEvent(ctx, request)

	return r, err
}

func (es *EventServer) Close() {
	es.grpcConn.Close()
}
