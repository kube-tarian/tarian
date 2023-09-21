// Package util provides helpers for tarianctl
package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientOptionsFromCliContext returns grpc dial options from cli context
func ClientOptionsFromCliContext(logger *logrus.Logger, globalFlags *flags.GlobalFlags) ([]grpc.DialOption, error) {
	o := []grpc.DialOption{}

	if globalFlags.ServerTLSEnabled {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		serverCAFile := globalFlags.ServerTLSCAFile

		if serverCAFile != "" {
			serverCACert, err := os.ReadFile(serverCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read server TLS CA file: serverCAFile: %s, error: %s", serverCAFile, err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				logger.Error("failed to append server ca file")
			}
		}
		tlsConfig := &tls.Config{ServerName: "", RootCAs: certPool}

		tlsConfig.InsecureSkipVerify = globalFlags.ServerTLSInsecureSkipVerify
		o = append(o, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		o = append(o, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return o, nil
}
