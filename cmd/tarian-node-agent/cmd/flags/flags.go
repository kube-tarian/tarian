// Package flags provides a way to manage global flags for the application.
package flags

import (
	"fmt"

	"github.com/spf13/pflag"
)

// GlobalFlags holds the global flag values for the application.
type GlobalFlags struct {
	LogLevel     string
	LogFormatter string
}

// SetGlobalFlags initializes and binds global flags using the provided FlagSet.
// It returns a pointer to the initialized GlobalFlags struct.
func SetGlobalFlags(flags *pflag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	// Define and bind the "log-level" persistent flag
	flags.StringVarP(&globalFlags.LogLevel, "log-level", "l", "info", "Valid log levels: debug, info(default), warn/warning, error, fatal")

	// Define and bind the "log-encoding" persistent flag
	flags.StringVarP(&globalFlags.LogFormatter, "log-formatter", "e", "text", "Valid log formatters: json, text(default)")

	return globalFlags
}

// ValidateGlobalFlags validates the global flags used in the application.
func (globalFlags *GlobalFlags) ValidateGlobalFlags() error {
	// Define a set of valid log levels.
	validLogLevels := map[string]bool{
		"debug":   true,
		"info":    true,
		"warn":    true,
		"warning": true,
		"error":   true,
		"fatal":   true,
	}

	// Define a set of valid log formatters.
	validLogFormatters := map[string]bool{
		"json": true,
		"text": true,
	}

	// Check if the specified log level is valid.
	if !validLogLevels[globalFlags.LogLevel] {
		return fmt.Errorf("invalid log level: %s", globalFlags.LogLevel)
	}

	// Check if the specified log formatter is valid.
	if !validLogFormatters[globalFlags.LogFormatter] {
		return fmt.Errorf("invalid log formatter: %s", globalFlags.LogFormatter)
	}

	// All checks passed, return nil (no error).
	return nil
}
