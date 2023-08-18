// Package util provides helpers for tarianctl
package util

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/kube-tarian/tarian/cmd/tarianctl/cmd/flags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewProduction()

	if err != nil {
		panic("Can not create logger")
	}

	logger = l.Sugar()
}

func SetLogger(l *zap.SugaredLogger) {
	logger = l
}

func ClientOptionsFromCliContext(globalFlags *flags.GlobalFlags) []grpc.DialOption {
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
				logger.Fatalw("failed to read server tls ca files", "filename", serverCAFile, "err", err)
			}

			if ok := certPool.AppendCertsFromPEM(serverCACert); !ok {
				logger.Errorw("failed to append server ca file")
			}
		}
		tlsConfig := &tls.Config{ServerName: "", RootCAs: certPool}

		tlsConfig.InsecureSkipVerify = globalFlags.ServerTLSInsecureSkipVerify
		o = append(o, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		o = append(o, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return o
}
