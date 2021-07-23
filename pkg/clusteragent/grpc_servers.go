package clusteragent

import (
	"context"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/gogo/status"
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
}

func NewConfigServer(tarianServerAddress string) *ConfigServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, grpc.WithInsecure())

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")

	return &ConfigServer{grpcConn: grpcConn, configClient: tarianpb.NewConfigClient(grpcConn)}
}

func (cs *ConfigServer) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Info("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := cs.configClient.GetConstraints(ctx, request)

	return r, err
}

func (cs *ConfigServer) AddConstraint(ctx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Method AddConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
}

func (cs *ConfigServer) Close() {
	cs.grpcConn.Close()
}

type EventServer struct {
	tarianpb.UnimplementedEventServer

	grpcConn    *grpc.ClientConn
	eventClient tarianpb.EventClient
}

func NewEventServer(tarianServerAddress string) *EventServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, grpc.WithInsecure())

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")

	return &EventServer{grpcConn: grpcConn, eventClient: tarianpb.NewEventClient(grpcConn)}
}

func (es *EventServer) IngestEvent(requestContext context.Context, request *tarianpb.IngestEventRequest) (*tarianpb.IngestEventResponse, error) {
	logger.Info("Received ingest violation event RPC")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := es.eventClient.IngestEvent(ctx, request)

	return r, err
}

func (es *EventServer) Close() {
	es.grpcConn.Close()
}
