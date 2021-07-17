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

type Server struct {
	tarianpb.UnimplementedConfigServer

	grpcConn     *grpc.ClientConn
	configClient tarianpb.ConfigClient
}

func NewServer(tarianServerAddress string) *Server {
	grpcConn, err := grpc.Dial(tarianServerAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		logger.Fatalw("couldn't not connect to tarian-server", "err", err)
	}

	return &Server{grpcConn: grpcConn, configClient: tarianpb.NewConfigClient(grpcConn)}
}

func (s *Server) GetConstraints(reqCtx context.Context, request *tarianpb.GetConstraintsRequest) (*tarianpb.GetConstraintsResponse, error) {
	logger.Info("Received get config RPC")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := s.configClient.GetConstraints(ctx, request)

	return r, err
}

func (s *Server) AddConstraint(ctx context.Context, request *tarianpb.AddConstraintRequest) (*tarianpb.AddConstraintResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Method AddConstraint is not supported in tarian-cluster-agent, send it to tarian-server instead.")
}

func (s *Server) Close() {
	s.grpcConn.Close()
}
