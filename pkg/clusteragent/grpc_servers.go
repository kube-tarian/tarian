package clusteragent

import (
	"context"
	"sync"
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

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
	eventClient  tarianpb.EventClient

	actions     []*tarianpb.Action
	actionsLock sync.RWMutex

	cancelFunc context.CancelFunc
	cancelCtx  context.Context
}

func NewEventServer(tarianServerAddress string, opts []grpc.DialOption) *EventServer {
	logger.Infow("connecting to the tarian server", "address", tarianServerAddress)
	grpcConn, err := grpc.Dial(tarianServerAddress, opts...)

	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	logger.Info("connected to the tarian server")
	ctx, cancel := context.WithCancel(context.Background())

	return &EventServer{
		grpcConn:     grpcConn,
		configClient: tarianpb.NewConfigClient(grpcConn),
		eventClient:  tarianpb.NewEventClient(grpcConn),
		cancelFunc:   cancel, cancelCtx: ctx,
	}
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
	es.cancelFunc()
}

func (es *EventServer) LoopSyncActions() error {
	for {
		es.SyncActions()

		select {
		case <-time.After(3 * time.Second):
		case <-es.cancelCtx.Done():
			return es.cancelCtx.Err()
		}
	}
}

func (es *EventServer) SyncActions() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	r, err := es.configClient.GetActions(ctx, &tarianpb.GetActionsRequest{})

	if err != nil {
		logger.Errorw("error while getting actions from the server", "err", err)
	}

	logger.Debugw("received actions from the server", "actions", r.GetActions())
	cancel()

	es.SetActions(r.GetActions())
}

func (es *EventServer) SetActions(actions []*tarianpb.Action) {
	es.actionsLock.Lock()
	defer es.actionsLock.Unlock()

	es.actions = actions
}
