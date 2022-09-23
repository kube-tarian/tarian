package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kube-tarian/tarian/pkg/logger"
	"github.com/kube-tarian/tarian/pkg/server"
	"github.com/kube-tarian/tarian/pkg/server/dbstore"
	"github.com/kube-tarian/tarian/pkg/server/dgraphstore"
	"github.com/kube-tarian/tarian/pkg/store"
	cli "github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kelseyhightower/envconfig"
)

const (
	defaultPort = "50051"
	defaultHost = ""
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

func main() {
	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarian Server",
		Usage:   "The Tarian Server is the central component which manages config DB, users, etc.",
		Version: version + " (" + commit + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level: debug, info, warn, error",
				Value: "info",
			},
			&cli.StringFlag{
				Name:  "log-encoding",
				Usage: "log-encoding: json, console",
				Value: "console",
			},
		},
		DefaultCommand: "run",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "Host address to listen at",
						Value: defaultHost,
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Host port to listen at",
						Value: defaultPort,
					},
					&cli.StringFlag{
						Name:  "alertmanager-address",
						Usage: "Alert manager address to send alerts to. For example: http://localhost:9093. Setting this enables alerting",
						Value: "",
					},
					&cli.DurationFlag{
						Name:  "alert-evaluation-interval",
						Usage: "The interval for evaluating and sending alerts",
						Value: 30 * time.Second,
					},
					&cli.StringFlag{
						Name:  "tls-cert-file",
						Usage: "File containing the default x509 Certificate for TLS. (CA cert concatenated after the cert)",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "tls-private-key-file",
						Usage: "Private key file in x509 format matching --tls-cert-file",
						Value: "",
					},
				},
				Action: run,
			},
			{
				Name:  "db",
				Usage: "Command group related to database",
				Subcommands: []*cli.Command{
					{
						Name:   "migrate",
						Usage:  "Run database migration",
						Action: dbmigrate,
					},
				},
			},
			{
				Name:  "dgraph",
				Usage: "Command group related to Dgraph database",
				Subcommands: []*cli.Command{
					{
						Name:   "apply-schema",
						Usage:  "Apply the schema for Dgraph database",
						Action: applyDgraphSchema,
						Flags: []cli.Flag{
							&cli.DurationFlag{
								Name:  "timeout",
								Usage: "How long it should wait for the operation to complete",
								Value: 5 * time.Minute,
							},
						},
					},
				},
			},
		},
	}
}

func run(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	server.SetLogger(logger)

	// Create server
	host := c.String("host")
	port := c.String("port")

	dgraphAddress := os.Getenv("DGRAPH_ADDRESS")
	storeSet := store.StoreSet{}
	if dgraphAddress != "" {
		cfg := dgraphstore.DgraphConfig{Address: dgraphAddress}

		err := envconfig.Process("Dgraph", &cfg)
		if err != nil {
			logger.Fatalw("dgraph config error", "err", err)
		}

		dialOpts := buildDgraphDialOpts(cfg, logger)
		grpcClient, err := dgraphstore.NewGrpcClient(cfg.Address, dialOpts)

		if err != nil {
			logger.Fatalw("error while initiating dgraph client", "err", err)
		}

		dg := dgraphstore.NewDgraphClient(grpcClient)
		storeSet.EventStore = dgraphstore.NewDgraphEventStore(dg)
		storeSet.ActionStore = dgraphstore.NewDgraphActionStore(dg)
		storeSet.ConstraintStore = dgraphstore.NewDgraphConstraintStore(dg)
	} else {
		var cfg server.PostgresqlConfig
		err := envconfig.Process("Postgres", &cfg)
		if err != nil {
			logger.Fatalw("database config error", "err", err)
		}

		storeSet.ActionStore, err = dbstore.NewDbActionStore(cfg.GetDsn())
		if err != nil {
			logger.Fatalw("error while initiating database access", "err", err)

		}

		storeSet.EventStore, err = dbstore.NewDbEventStore(cfg.GetDsn())
		if err != nil {
			logger.Fatalw("error while initiating database access", "err", err)

		}

		storeSet.ConstraintStore, err = dbstore.NewDbConstraintStore(cfg.GetDsn())
		if err != nil {
			logger.Fatalw("error while initiating database access", "err", err)

		}
	}

	server, err := server.NewServer(storeSet, c.String("tls-cert-file"), c.String("tls-private-key-file"))
	if err != nil {
		logger.Fatalw("error while initiating tarian-server", "err", err)
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		server.Stop()
	}()

	if c.String("alertmanager-address") != "" {
		url, err := url.Parse(c.String("alertmanager-address"))
		if err != nil {
			logger.Fatalw("invalid url in alertmanager-address", "err", err)
		}

		server.WithAlertDispatcher(url, c.Duration("alert-evaluation-interval")).StartAlertDispatcher()
	}

	// Run server
	logger.Infow("tarian-server is listening at", "address", host+":"+port)

	if err := server.Start(host + ":" + port); err != nil {
		logger.Fatalw("failed to start server", "err", err)
	}

	logger.Info("tarian-server shutdown gracefully")

	return nil
}

func dbmigrate(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	var cfg server.PostgresqlConfig
	err := envconfig.Process("Postgres", &cfg)
	if err != nil {
		logger.Fatalw("database config error", "err", err)
	}

	count, err := dbstore.RunMigration(cfg.GetDsn())

	if err != nil {
		logger.Fatalw("error while running database migration", "err", err)
	} else {
		logger.Infow("completed database migration", "applied", count)
	}

	return nil
}

func applyDgraphSchema(c *cli.Context) error {
	l := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	var cfg dgraphstore.DgraphConfig
	err := envconfig.Process("Dgraph", &cfg)
	if err != nil {
		l.Fatalw("dgraph config error", "err", err)
	}

	dialOpts := buildDgraphDialOpts(cfg, l)
	grpcClient, err := dgraphstore.NewGrpcClient(cfg.Address, dialOpts)
	if err != nil {
		l.Fatalw("error while creating grpc client for applying dgraph schema", "err", err)
	}

	dg := dgraphstore.NewDgraphClient(grpcClient)

	ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
	defer cancel()

	err = dgraphstore.ApplySchema(ctx, dg)

	if err != nil {
		l.Fatalw("error while applying dgraph schema", "err", err)
	} else {
		l.Infow("dgraph schema applied")
	}

	return nil
}

func buildDgraphDialOpts(dgraphCfg dgraphstore.DgraphConfig, l *zap.SugaredLogger) []grpc.DialOption {
	dialOpts := []grpc.DialOption{}
	if dgraphCfg.TLSCertFile == "" {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		if dgraphCfg.TLSCAFile != "" {
			serverCACert, err := ioutil.ReadFile(dgraphCfg.TLSCAFile)
			if err != nil {
				l.Fatalw("failed to read Dgraph TLS CA file", "filename", dgraphCfg.TLSCAFile, "err", err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				l.Errorw("failed to append Dgraph TLS CA file")
			}
		}

		cert, err := tls.LoadX509KeyPair(dgraphCfg.TLSCertFile, dgraphCfg.TLSKeyFile)
		if err != nil {
			l.Fatalw("error while creating Dgraph client TLS cert", "err", err)
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
	return dialOpts
}
