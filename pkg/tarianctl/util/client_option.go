package util

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func ClientOptionsFromCliContext(ctx *cli.Context) []grpc.DialOption {
	o := []grpc.DialOption{}

	if ctx.Bool("server-tls-enabled") {
		// TODO: handle err
		certPool, _ := x509.SystemCertPool()
		tlsConfig := &tls.Config{ServerName: "", RootCAs: certPool}

		tlsConfig.InsecureSkipVerify = ctx.Bool("server-tls-insecure-skip-verify")
		o = append(o, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		o = append(o, grpc.WithInsecure())
	}

	return o
}
