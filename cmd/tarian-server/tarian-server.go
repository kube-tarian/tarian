package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/devopstoday11/tarian/pkg/logger"
	"github.com/devopstoday11/tarian/pkg/server"
	"github.com/devopstoday11/tarian/pkg/server/dbstore"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	cli "github.com/urfave/cli/v2"
	"google.golang.org/grpc"

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

type PostgresqlConfig struct {
	User     string `default:"postgres"`
	Password string `default:"tarian"`
	Name     string `default:"tarian"`
	Host     string `default:"localhost"`
	Port     string `default:"5432"`
	SslMode  string `default:"disable"`
}

func (p *PostgresqlConfig) GetDsn() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", p.User, p.Password, p.Host, p.Port, p.Name, p.SslMode)
}

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
		Action: run,
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
	host := c.String("host")
	if host == "" {
		host = defaultHost
	}

	port := c.String("port")
	if port == "" {
		port = defaultPort
	}

	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))
	server.SetLogger(logger)

	var cfg PostgresqlConfig
	err := envconfig.Process("Postgres", &cfg)
	if err != nil {
		logger.Fatalw("database config error", "err", err)
	}

	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		logger.Fatalw("failed to listen", "err", err)
	}

	s := grpc.NewServer()
	configServer, err := server.NewConfigServer(cfg.GetDsn())
	if err != nil {
		logger.Fatalw("failed to initiate server", "err", err)
	}

	tarianpb.RegisterConfigServer(s, configServer)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)

		s.GracefulStop()
		wg.Done()
	}()

	logger.Infow("tarian-server is listening at", "address", listener.Addr())

	if err := s.Serve(listener); err != nil {
		logger.Fatalw("failed to serve", "err", err)
	}

	wg.Wait()
	logger.Info("tarian-server shutdown gracefully")

	return nil
}

func dbmigrate(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	var cfg PostgresqlConfig
	err := envconfig.Process("Postgres", &cfg)
	if err != nil {
		logger.Fatalw("database config error", "err", err)
	}

	count, err := dbstore.RunMigration(cfg.GetDsn())

	if err != nil {
		logger.Errorw("error while running database migration", "err", err)
	} else {
		logger.Infow("completed database migration", "applied", count)
	}

	return nil
}

func devSeedData(c *cli.Context) error {
	logger := logger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	var cfg PostgresqlConfig
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
		exampleConstraint := tarianpb.Constraint{Namespace: "default", Selector: &tarianpb.Selector{MatchLabels: []*tarianpb.MatchLabel{{Key: "app", Value: "nginx"}}}}
		allowedProcessRegex := "(.*)" + r + "(.*)"
		exampleConstraint.AllowedProcesses = []*tarianpb.AllowedProcessRule{{Regex: &allowedProcessRegex}}
		err := dbStore.Add(&exampleConstraint)
		if err != nil {
			logger.Errorw("error while adding seed data: constraint", "err", err)
			return err
		}
	}

	logger.Infow("finished adding seed data")

	return nil
}
