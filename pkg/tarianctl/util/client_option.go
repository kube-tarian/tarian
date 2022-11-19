// Package util provides helpers for tarianctl
package util

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/urfave/cli/v2"
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

func ClientOptionsFromCliContext(ctx *cli.Context) []grpc.DialOption {
	o := []grpc.DialOption{}

	if ctx.Bool("server-tls-enabled") {
		certPool, _ := x509.SystemCertPool()
		if certPool == nil {
			certPool = x509.NewCertPool()
		}

		serverCAFile := ctx.String("server-tls-ca-file")

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

		tlsConfig.InsecureSkipVerify = ctx.Bool("server-tls-insecure-skip-verify")
		o = append(o, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		o = append(o, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return o
}
