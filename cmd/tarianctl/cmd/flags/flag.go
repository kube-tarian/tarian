// Package flags provides a way to manage global flags for the application.
package flags

import (
	"os"

	"github.com/spf13/pflag"
)

const (
	defaultServerAddress = "localhost:50051"
)

// GlobalFlags holds the global flag values for the application.
type GlobalFlags struct {
	LogLevel                    string
	LogEncoding                 string
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
	flags.StringVarP(&globalFlags.LogLevel, "log-level", "l", "info", "Log level. Valid values: debug, info, warn, error, fatal, panic")

	// Define and bind the "log-encoding" persistent flag
	flags.StringVarP(&globalFlags.LogEncoding, "log-encoding", "e", "console", "Log encoding. Valid values: json, console")

	// Read environment variable for "server-address" flag
	if serverAddressEnv := os.Getenv("TARIAN_SERVER_ADDRESS"); serverAddressEnv != "" {
		globalFlags.ServerAddr = serverAddressEnv
	}

	// Define and bind the "server-address" persistent flag
	flags.StringVarP(&globalFlags.ServerAddr, "server-address", "s", defaultServerAddress, "Tarian server address to communicate with")

	// Read environment variable for "server-tls-enabled" flag
	if serverTLSEnabledEnv := os.Getenv("TARIAN_TLS_ENABLED"); serverTLSEnabledEnv != "" {
		globalFlags.ServerTLSEnabled = true
	}

	// Define and bind the "server-tls-enabled" persistent flag
	flags.BoolVarP(&globalFlags.ServerTLSEnabled, "server-tls-enabled", "t", false, "If enabled, it will communicate with the server using TLS")

	// Read environment variable for "server-tls-ca-file" flag
	if serverTLSCAFileEnv := os.Getenv("TARIAN_TLS_CA_FILE"); serverTLSCAFileEnv != "" {
		globalFlags.ServerTLSCAFile = serverTLSCAFileEnv
	}

	// Define and bind the "server-tls-ca-file" persistent flag
	flags.StringVarP(&globalFlags.ServerTLSCAFile, "server-tls-ca-file", "c", "", "The CA the server uses for TLS connection.")

	// Read environment variable for "server-tls-insecure-skip-verify" flag
	if serverTLSInsecureSkipVerifyEnv := os.Getenv("TARIAN_TLS_INSECURE_SKIP_VERIFY"); serverTLSInsecureSkipVerifyEnv != "" {
		globalFlags.ServerTLSInsecureSkipVerify = true
	}

	// Define and bind the "server-tls-insecure-skip-verify" persistent flag
	flags.BoolVarP(&globalFlags.ServerTLSInsecureSkipVerify, "server-tls-insecure-skip-verify", "i", true, "If set to true, it will skip server's certificate chain and hostname verification")

	return globalFlags
}
