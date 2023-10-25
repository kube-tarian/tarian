package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd/dgraph"
	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/server"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/store"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const (
	defaultPort = "50051"
	defaultHost = ""
)

type runCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	host string
	port string

	alertManagerAddress     string
	alertEvaluationInterval time.Duration

	tlsCertFile       string
	tlsPrivateKeyFile string

	natsURL                    string
	natsTLSRootCAs             []string
	natsTLSClientCert          string
	natsTLSClientKey           string
	natsStramConfigMaxMsg      int64
	natsStramConfigMaxBytes    int64
	natsStreamConfigReplicas   int
	natsStramConfigMaxAge      time.Duration
	natsStreamConfigDuplicates time.Duration

	dgraphClient dgraphstore.Client
	server       server.Server
}

func newRunCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &runCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the server",
		RunE:  cmd.run,
	}

	// Add flags
	cmd.addFlags(runCmd)

	return runCmd
}

func (o *runCommand) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.host, "host", defaultHost, "The host to listen on")
	cmd.Flags().StringVar(&o.port, "port", defaultPort, "The port to listen on")

	cmd.Flags().StringVar(&o.alertManagerAddress, "alertmanager-address", "", "Alert manager address to send alerts to. For example: http://localhost:9093. Setting this enables alerting")
	cmd.Flags().DurationVar(&o.alertEvaluationInterval, "alert-evaluation-interval", 30*time.Second, "The interval for evaluating and sending alerts")

	cmd.Flags().StringVar(&o.tlsCertFile, "tls-cert-file", "", "File containing the default x509 Certificate for TLS. (CA cert concatenated after the cert)")
	cmd.Flags().StringVar(&o.tlsPrivateKeyFile, "tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file")

	cmd.Flags().StringVar(&o.natsURL, "nats-url", "", "If specified, tarian-server will use NATS to queue the incoming events")
	cmd.Flags().StringSliceVar(&o.natsTLSRootCAs, "nats-tls-root-cas", []string{}, "The root CA certificates to be used to connect to NATS")
	cmd.Flags().StringVar(&o.natsTLSClientCert, "nats-tls-client-cert", "", "The client certificate to be used to connect to NATS")
	cmd.Flags().StringVar(&o.natsTLSClientKey, "nats-tls-client-key", "", "The client key to be used to connect to NATS")
	cmd.Flags().Int64Var(&o.natsStramConfigMaxMsg, "nats-stream-config-max-msg-size", 10000, "")
	cmd.Flags().Int64Var(&o.natsStramConfigMaxBytes, "nats-stream-config-max-bytes", 50*1000*1000, "")
	cmd.Flags().DurationVar(&o.natsStramConfigMaxAge, "nats-stream-config-max-age", 24*time.Hour, "")
	cmd.Flags().IntVar(&o.natsStreamConfigReplicas, "nats-stream-config-replicas", 1, "")
	cmd.Flags().DurationVar(&o.natsStreamConfigDuplicates, "nats-stream-config-duplicates", 1*time.Minute, "")
}

func (o *runCommand) run(_ *cobra.Command, args []string) error {
	host := o.host
	port := o.port

	if o.dgraphClient == nil {
		dgraphAddress := os.Getenv("DGRAPH_ADDRESS")
		cfg := dgraphstore.DgraphConfig{Address: dgraphAddress}

		err := envconfig.Process("Dgraph", &cfg)
		if err != nil {
			return fmt.Errorf("run: dgraph config error: %w", err)
		}

		dialOpts, err := dgraph.BuildDgraphDialOpts(cfg, o.logger)
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}

		grpcClient, err := grpc.Dial(cfg.Address, dialOpts...)
		if err != nil {
			return fmt.Errorf("run: failed to connect to server: %w", err)
		}
		defer grpcClient.Close()
		o.dgraphClient = dgraphstore.NewDgraphClient(grpcClient)
		o.logger.Debug("Created dgraphstore client")
	}

	storeSet := store.Set{}
	storeSet.EventStore = o.dgraphClient.NewDgraphEventStore()
	storeSet.ActionStore = o.dgraphClient.NewDgraphActionStore()
	storeSet.ConstraintStore = o.dgraphClient.NewDgraphConstraintStore()

	natsOpts := []nats.Option{}
	for _, rootCA := range o.natsTLSRootCAs {
		natsOpts = append(natsOpts, nats.RootCAs(rootCA))
	}

	if o.natsTLSClientCert != "" {
		cert := nats.ClientCert(o.natsTLSClientCert, o.natsTLSClientKey)
		natsOpts = append(natsOpts, cert)
	}

	streamName := "tarian-server-event-ingestion"
	streamConfig := nats.StreamConfig{
		Name:       streamName,
		Subjects:   []string{streamName},
		Retention:  nats.LimitsPolicy,
		Discard:    nats.DiscardOld,
		MaxMsgs:    o.natsStramConfigMaxMsg,
		MaxAge:     o.natsStramConfigMaxAge,
		MaxBytes:   o.natsStramConfigMaxBytes,
		Storage:    nats.FileStorage,
		Replicas:   o.natsStreamConfigReplicas,
		Duplicates: o.natsStreamConfigDuplicates,
	}

	if o.server == nil {
		serv, err := server.NewServer(o.logger, storeSet, o.tlsCertFile, o.tlsPrivateKeyFile, o.natsURL, natsOpts, streamConfig)
		if err != nil {
			return fmt.Errorf("run: error while initiating tarian-server: %w", err)
		}
		o.server = serv
	}

	var sigCh = make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		o.logger.Warn("got sigterm signal, attempting graceful shutdown: signal: ", sig)
		o.server.Stop()
	}()

	if o.alertManagerAddress != "" {
		url, err := url.Parse(o.alertManagerAddress)
		if err != nil {
			return fmt.Errorf("run: invalid url in alertmanager-address: %w", err)
		}

		o.server.WithAlertDispatcher(url, o.alertEvaluationInterval).StartAlertDispatcher()
	}

	addr := host + ":" + port
	// Run server
	o.logger.Infof("tarian-server is listening at: %s", addr)
	if err := o.server.Start(addr); err != nil {
		return fmt.Errorf("run: failed to start server: %w", err)
	}

	o.logger.Info("tarian-server shutdown gracefully")
	return nil
}
