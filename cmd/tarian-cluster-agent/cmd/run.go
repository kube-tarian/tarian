package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd/flags"
	"github.com/kube-tarian/tarian/pkg/clusteragent"
	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultPort                     = "50052"
	defaultHost                     = ""
	defaultServerAddress            = "localhost:50051"
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

	clusterAgent clusteragent.Agent
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

	if c.clusterAgent == nil {
		clusterAgentConfig, err := c.newClusterAgentConfigFromCliContext()
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}

		c.clusterAgent, err = clusteragent.NewClusterAgent(c.logger, clusterAgentConfig)
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}
		defer c.clusterAgent.Close()
	}

	grpcServer := c.clusterAgent.GetGrpcServer()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		c.logger.WithField("signal", sig).Info("got sigterm signal, attempting graceful shutdown")
		grpcServer.GracefulStop()
	}()

	go c.clusterAgent.Run()

	c.logger.WithField("address", listener.Addr()).Info("tarian-cluster-agent is listening at")

	if err := grpcServer.Serve(listener); err != nil {
		c.logger.WithError(err).Error("failed to serve")
		return fmt.Errorf("run: failed to serve: %w", err)
	}

	c.logger.Info("tarian-cluster-agent shutdown gracefully")
	return nil
}

func (c *runCommand) newClusterAgentConfigFromCliContext() (*clusteragent.Config, error) {
	dialOpts, err := util.GetDialOptions(c.logger, c.serverTLSEnabled, c.serverTLSInsecureSkipVerify, c.serverTLSCAFile)
	if err != nil {
		return nil, fmt.Errorf("new cluster agent config: %w", err)
	}
	config := &clusteragent.Config{
		ServerAddress:         c.serverAddress,
		ServerGrpcDialOptions: dialOpts,
		EnableAddConstraint:   c.enableAddConstraint,
	}

	return config, nil
}
