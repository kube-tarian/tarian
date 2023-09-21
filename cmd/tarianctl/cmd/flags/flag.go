// Package flags provides a way to manage global flags for the application.
package flags

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

const (
	defaultServerAddress = "localhost:50051"
)

// GlobalFlags holds the global flag values for the application.
type GlobalFlags struct {
	LogLevel                    string
	LogFormatter                string
	ServerAddr                  string
	ServerTLSEnabled            bool
	ServerTLSCAFile             string
	ServerTLSInsecureSkipVerify bool
}

// SetGlobalFlags initializes and binds global flags using the provided FlagSet.
// It returns a pointer to the initialized GlobalFlags struct.
func SetGlobalFlags(flags *pflag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	// Define and bind the "log-level" persistent flag
	flags.StringVarP(&globalFlags.LogLevel, "log-level", "l", "info", "valid log levels: debug, info(default), warn/warning, error, fatal")

	// Define and bind the "log-encoding" persistent flag
	flags.StringVarP(&globalFlags.LogFormatter, "log-formatter", "e", "text", "valid log formatters: json, text(default)")

	// Define and bind the "server-address" persistent flag
	flags.StringVarP(&globalFlags.ServerAddr, "server-address", "s", defaultServerAddress, "tarian server address to communicate with")

	// Define and bind the "server-tls-enabled" persistent flag
	flags.BoolVarP(&globalFlags.ServerTLSEnabled, "server-tls-enabled", "t", false, "if enabled, it will communicate with the server using TLS")

	// Define and bind the "server-tls-ca-file" persistent flag
	flags.StringVarP(&globalFlags.ServerTLSCAFile, "server-tls-ca-file", "c", "", "ca file that server uses for TLS connection")

	// Define and bind the "server-tls-insecure-skip-verify" persistent flag
	flags.BoolVarP(&globalFlags.ServerTLSInsecureSkipVerify, "server-tls-insecure-skip-verify", "i", true, "if set to true, it will skip server's certificate chain and hostname verification")

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

	// Check if server TLS is enabled but CA file is not provided.
	if globalFlags.ServerTLSEnabled && globalFlags.ServerTLSCAFile == "" {
		return fmt.Errorf("server TLS enabled but CA file is not provided")
	}

	// All checks passed, return nil (no error).
	return nil
}

// GetFlagValuesFromEnvVar reads the environment variables for the global flags.
func (globalFlags *GlobalFlags) GetFlagValuesFromEnvVar() {
	// Read environment variable for "server-address" flag
	if serverAddressEnv := os.Getenv("TARIAN_SERVER_ADDRESS"); serverAddressEnv != "" {
		globalFlags.ServerAddr = serverAddressEnv
	}

	// Read environment variable for "server-tls-enabled" flag
	if serverTLSEnabledEnv := os.Getenv("TARIAN_TLS_ENABLED"); serverTLSEnabledEnv != "" {
		globalFlags.ServerTLSEnabled = true
	}

	// Read environment variable for "server-tls-ca-file" flag
	if serverTLSCAFileEnv := os.Getenv("TARIAN_TLS_CA_FILE"); serverTLSCAFileEnv != "" {
		globalFlags.ServerTLSCAFile = serverTLSCAFileEnv
	}

	// Read environment variable for "server-tls-insecure-skip-verify" flag
	if serverTLSInsecureSkipVerifyEnv := os.Getenv("TARIAN_TLS_INSECURE_SKIP_VERIFY"); serverTLSInsecureSkipVerifyEnv != "" {
		globalFlags.ServerTLSInsecureSkipVerify = true
	}
}
