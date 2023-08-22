// Package log provides configured logger that is ready to use
package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// ConfigureLogger initializes the logger based on the provided log level.
func ConfigureLogger(logLevel string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel // Default to InfoLevel if parsing fails
	}

	logger = logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
	})
	logger.SetOutput(os.Stdout)
}

// GetLogger returns the configured logger instance.
func GetLogger() *logrus.Logger {
	if logger == nil {
		// Fallback to a default logger if not configured
		ConfigureLogger("info")
	}
	return logger
}
