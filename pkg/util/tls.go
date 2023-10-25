package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GetDialOptions returns grpc dial options
func GetDialOptions(logger *logrus.Logger, serverTLSEnabled, serverTLSInsecureSkipVerify bool, serverCAFile string) ([]grpc.DialOption, error) {
	o := []grpc.DialOption{}

	if serverTLSEnabled {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

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

		tlsConfig.InsecureSkipVerify = serverTLSInsecureSkipVerify
		o = append(o, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		o = append(o, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return o, nil
}
