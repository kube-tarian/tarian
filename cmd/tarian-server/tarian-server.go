package main

import (
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/server"
	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"

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
		Action: func(ctx *cli.Context) error {
			return ctx.App.Command("run").Run(ctx)
		},
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
				Name:  "dev",
				Usage: "Command group for development environment. Do not do this on production.",
				Subcommands: []*cli.Command{
					{
						Name:   "seed-data",
						Usage:  "Add development data to the database",
						Action: devSeedData,
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

	var cfg server.PostgresqlConfig
	err := envconfig.Process("Postgres", &cfg)
	if err != nil {
		logger.Fatalw("database config error", "err", err)
	}

	server, err := server.NewServer(cfg.GetDsn(), c.String("tls-cert-file"), c.String("tls-private-key-file"))
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

func devSeedData(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	var cfg server.PostgresqlConfig
	err := envconfig.Process("Postgres", &cfg)
	if err != nil {
		logger.Fatalw("database config error", "err", err)
	}

	dbStore, err := dbstore.NewDbConstraintStore(cfg.GetDsn())
	if err != nil {
		logger.Fatalw("error creating database store", "err", err)
	}

	regexes := []string{"ssh", "worker", "swap", "scsi", "loop", "gvfs", "idle", "injection", "nvme", "jbd", "snap", "cpu", "soft", "bash", "integrity", "kcryptd", "krfcommd", "kcompactd0", "wpa_supplican", "oom_reaper", "registryd", "migration", "kblockd", "gsd-", "kdevtmpfs", "pipewire"}

	for _, r := range regexes {
		exampleConstraint := tarianpb.Constraint{Namespace: "tarian-system", Name: "nginx-" + r, Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "(.*)" + r + "(.*)"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		err := dbStore.Add(&exampleConstraint)
		if err != nil {
			logger.Fatalw("error while adding seed data: constraint", "err", err)
		}
	}

	regexes2 := []string{"sleep", "pause", "tarian-pod-agent"}

	for _, r := range regexes2 {
		exampleConstraint := tarianpb.Constraint{Namespace: "tarian-system", Name: "nginx2-" + r, Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx2"}, {Key: "app2", Value: "nginx3"}}}}
		allowedProcessRegex := "(.*)" + r + "(.*)"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		err := dbStore.Add(&exampleConstraint)
		if err != nil {
			logger.Fatalw("error while adding seed data: constraint", "err", err)
			return err
		}
	}

	logger.Infow("finished adding seed data")

	return nil
}
