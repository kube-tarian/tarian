package server

import (
	"fmt"

	"go.uber.org/zap"
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
