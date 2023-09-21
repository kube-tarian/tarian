package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultPort = "50052"
	defaultHost = ""

	defaultServerAddress = "localhost:50051"

	defaultSidekickListenerHTTPPort = "8088"
)

type runCommand struct {
	globalFlags *flags.GlobalFlags
	logger      *logrus.Logger

	host string
	port string

	serverAddress               string
	serverTLSEnabled            bool
	serverTLSCAFile             string
	serverTLSInsecureSkipVerify bool

	enableAddConstraint bool

	falcoListenerHTTPPort string
}

func newRunCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &runCommand{
		globalFlags: globalFlags,
		logger:      log.GetLogger(),
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the cluster agent",
		RunE:  cmd.run,
	}

	// Add flags
	runCmd.Flags().StringVar(&cmd.host, "host", defaultHost, "host to listen on")
	runCmd.Flags().StringVar(&cmd.port, "port", defaultPort, "port to listen on")
	runCmd.Flags().StringVar(&cmd.serverAddress, "server-address", defaultServerAddress, "address of the Tarian Server")
	runCmd.Flags().BoolVar(&cmd.serverTLSEnabled, "server-tls-enabled", false, "enable TLS for the Tarian Server")
	runCmd.Flags().StringVar(&cmd.serverTLSCAFile, "server-tls-ca-file", "", "CA certificate file for the Tarian Server")
	runCmd.Flags().BoolVar(&cmd.serverTLSInsecureSkipVerify, "server-tls-insecure-skip-verify", true, "skip TLS verification for the Tarian Server")
	runCmd.Flags().BoolVar(&cmd.enableAddConstraint, "enable-add-constraint", false, "enable to support pod agent running in register mode")
	runCmd.Flags().StringVar(&cmd.falcoListenerHTTPPort, "falco-listener-http-port", defaultSidekickListenerHTTPPort, "falco listener http port")

	return runCmd
}

func (c *runCommand) run(_ *cobra.Command, args []string) error {
	addr := c.host + ":" + c.port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		c.logger.WithError(err).Error("failed to listen")
		return fmt.Errorf("run: failed to listen: %w", err)
	}

	clusterAgentConfig, err := c.newClusterAgentConfigFromCliContext()
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	clusterAgent, err := clusteragent.NewClusterAgent(c.logger, clusterAgentConfig)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	defer clusterAgent.Close()

	grpcServer := clusterAgent.GetGrpcServer()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		c.logger.WithField("signal", sig).Info("got sigterm signal, attempting graceful shutdown")
		grpcServer.GracefulStop()
	}()

	go clusterAgent.Run()

	c.logger.WithField("address", listener.Addr()).Info("tarian-cluster-agent is listening at")

	if err := grpcServer.Serve(listener); err != nil {
		c.logger.WithError(err).Error("failed to serve")
		return fmt.Errorf("run: failed to serve: %w", err)
	}

	c.logger.Info("tarian-cluster-agent shutdown gracefully")
	return nil
}

func (c *runCommand) newClusterAgentConfigFromCliContext() (*clusteragent.Config, error) {
	dialOpts := []grpc.DialOption{}

	if c.serverTLSEnabled {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		serverCAFile := c.serverTLSCAFile

		if serverCAFile != "" {
			serverCACert, err := os.ReadFile(serverCAFile)
			if err != nil {
				c.logger.WithError(err).WithField("filename", serverCAFile).Error("failed to read server tls ca files")
				return nil, fmt.Errorf("newClusterAgentConfigFromCliContext: %w", err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				c.logger.WithError(err).Error("failed to append server ca file")
			}
		}

		tlsConfig := &tls.Config{ServerName: "", RootCAs: certPool}
		tlsConfig.InsecureSkipVerify = c.serverTLSInsecureSkipVerify

		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	config := &clusteragent.Config{
		ServerAddress:         c.serverAddress,
		ServerGrpcDialOptions: dialOpts,
		EnableAddConstraint:   c.enableAddConstraint,
	}

	return config, nil
}
