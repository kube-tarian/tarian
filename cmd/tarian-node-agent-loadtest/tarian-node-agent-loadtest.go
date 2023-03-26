package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkglogger "github.com/kube-tarian/tarian/pkg/logger"
	cli "github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "main"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	app := getCliApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getCliApp() *cli.App {
	return &cli.App{
		Name:    "Tarian Node Agent Load Test",
		Usage:   "The Tarian Node Agent Lod Test is a component which runs as a daemonset, generating load to tarian server similar to events.",
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
				Name:   "run",
				Usage:  "Run the node agent load test",
				Action: run,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "cluster-agent-host",
						Usage: "Host address of the cluster agent to communicate with",
						Value: "tarian-cluster-agent.tarian-system.svc",
					},
					&cli.StringFlag{
						Name:  "cluster-agent-port",
						Usage: "Host port of the cluster agent to communicate with",
						Value: "80",
					},
					&cli.StringFlag{
						Name:  "node-name",
						Usage: "Node name where it is running. This is intended to be set from Downward API",
						Value: "",
					},
				},
			},
		},
	}
}

func run(c *cli.Context) error {
	logger := pkglogger.GetLogger(c.String("log-level"), c.String("log-encoding"))

	tp, err := initOtelTraceProvider()
	if err != nil {
		logger.Fatalw("error initializing otel trace provider", "err", err)
	}

	defer tp.Shutdown(c.Context)

	loadtester := newLoadTester(c.String("cluster-agent-host") + ":" + c.String("cluster-agent-port"))
	loadtester.SetNodeName(c.String("node-name"))

	logger.Infow("tarian-node-agent-loadtest is running", "node-name", c.String("node-name"))
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Infow("got sigterm signal, attempting graceful shutdown", "signal", sig)
		loadtester.GracefulStop()
	}()

	loadtester.Run()
	logger.Info("tarian-node-agent shutdown gracefully")

	return nil
}

func initOtelTraceProvider() (*trace.TracerProvider, error) {
	ctx := context.Background()

	client := otlptracegrpc.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceVersionKey.String(version+" ("+commit+")"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}
