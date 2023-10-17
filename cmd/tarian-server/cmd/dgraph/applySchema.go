package dgraph

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type applySchemaCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	timeout time.Duration
}

func newApplySchemaCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &applySchemaCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	applySchemaCmd := &cobra.Command{
		Use:   "apply-schema",
		Short: "Apply the schema for Dgraph database",
		RunE:  cmd.run,
	}

	// Add flags
	applySchemaCmd.Flags().DurationVar(&cmd.timeout, "timeout", 5*time.Minute, "Timeout for applying schema")
	return applySchemaCmd
}

func (o *applySchemaCommand) run(_ *cobra.Command, args []string) error {
	var cfg dgraphstore.DgraphConfig
	err := envconfig.Process("Dgraph", &cfg)
	if err != nil {
		return fmt.Errorf("apply-schema: dgraph config error: %w", err)
	}

	dialOpts, err := BuildDgraphDialOpts(cfg, o.logger)
	if err != nil {
		return fmt.Errorf("apply-schema: %w", err)
	}
	grpcClient, err := dgraphstore.NewGrpcClient(cfg.Address, dialOpts)
	if err != nil {
		return fmt.Errorf("apply-schema: error while creating grpc client: %w", err)
	}

	dg := dgraphstore.NewDgraphClient(grpcClient)

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	err = dgraphstore.ApplySchema(ctx, dg)
	if err != nil {
		return fmt.Errorf("apply-schema: failed to apply schema: %w", err)
	}

	o.logger.Info("successfully applied schema")
	return nil
}

// BuildDgraphDialOpts builds the dial options for Dgraph client
func BuildDgraphDialOpts(dgraphCfg dgraphstore.DgraphConfig, logger *logrus.Logger) ([]grpc.DialOption, error) {
	dialOpts := []grpc.DialOption{}
	if dgraphCfg.TLSCertFile == "" {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		if dgraphCfg.TLSCAFile != "" {
			serverCACert, err := os.ReadFile(dgraphCfg.TLSCAFile)
			if err != nil {

				return nil, fmt.Errorf("BuildDgraphDialOpts: failed to read Dgraph TLS CA file: %s, error: %w", dgraphCfg.TLSCAFile, err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				logger.Error("BuildDgraphDialOpts: failed to append Dgraph TLS CA file")
			}
		}

		cert, err := tls.LoadX509KeyPair(dgraphCfg.TLSCertFile, dgraphCfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("BuildDgraphDialOpts: error while creating Dgraph client TLS cert: %w", err)
		}

		// Get server name, without port
		splitAddress := strings.Split(dgraphCfg.Address, ":")
		serverName := ""

		if len(splitAddress) > 0 {
			serverName = splitAddress[0]
		}

		tlsConfig := &tls.Config{ServerName: serverName, RootCAs: certPool, Certificates: []tls.Certificate{cert}}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
	return dialOpts, nil
}
