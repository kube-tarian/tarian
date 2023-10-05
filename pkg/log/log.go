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
	DefaultFormat()
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

// NoTimestampFormatter is a logrus formatter that does not print the timestamp
type NoTimestampFormatter struct{}

// Format formats the log entry without the timestamp
func (f *NoTimestampFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

// DefaultFormat sets the default log format
func DefaultFormat() {
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		DisableColors: false,
		FullTimestamp: true,
	})
}

// MiniLogFormat sets the minimal log format mostly used for testing purpose
func MiniLogFormat() {
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:          true,
		DisableQuote:           true,
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
}
