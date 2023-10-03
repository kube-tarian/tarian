package flags

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestSetGlobalFlags(t *testing.T) {
	// Create a FlagSet for testing
	fs := pflag.NewFlagSet("test", pflag.ExitOnError)

	// Initialize global flags
	globalFlags := SetGlobalFlags(fs)

	// Test default values
	assert.Equal(t, "info", globalFlags.LogLevel)
	assert.Equal(t, "text", globalFlags.LogFormatter)
	assert.Equal(t, defaultServerAddress, globalFlags.ServerAddr)
	assert.False(t, globalFlags.ServerTLSEnabled)
	assert.Equal(t, "", globalFlags.ServerTLSCAFile)
	assert.True(t, globalFlags.ServerTLSInsecureSkipVerify)
}

func TestValidateGlobalFlags(t *testing.T) {
	tests := []struct {
		name          string
		globalFlags   *GlobalFlags
		expectedError string
	}{
		{
			name:          "Valid Flags",
			globalFlags:   &GlobalFlags{LogLevel: "info", LogFormatter: "text", ServerTLSCAFile: "ca.pem"},
			expectedError: "",
		},
		{
			name:          "Invalid LogLevel",
			globalFlags:   &GlobalFlags{LogLevel: "invalid", LogFormatter: "text"},
			expectedError: "invalid log level: invalid",
		},
		{
			name:          "Invalid LogFormatter",
			globalFlags:   &GlobalFlags{LogLevel: "info", LogFormatter: "invalid"},
			expectedError: "invalid log formatter: invalid",
		},
		{
			name:          "ServerTLSWithoutCAFile",
			globalFlags:   &GlobalFlags{LogLevel: "info", LogFormatter: "text", ServerTLSEnabled: true},
			expectedError: "server TLS enabled but CA file is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.globalFlags.ValidateGlobalFlags()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

func TestGetFlagValuesFromEnvVar(t *testing.T) {
	// Set environment variables for testing
	tarianServerEnvVar := "TARIAN_SERVER_ADDRESS"
	tarianServerEnvVarValue := "test-server:1234"
	if err := os.Setenv(tarianServerEnvVar, tarianServerEnvVarValue); !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	defer os.Unsetenv(tarianServerEnvVar)

	// Set more environment variables for testing
	TLSEnabledEnvVar := "TARIAN_TLS_ENABLED"
	TLSEnabledEnvVarValue := "true"
	if err := os.Setenv(TLSEnabledEnvVar, TLSEnabledEnvVarValue); !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	defer os.Unsetenv(TLSEnabledEnvVar)

	TLSCAFilEnvVar := "TARIAN_TLS_CA_FILE"
	TLSCAFilEnvVarValue := "/path/to/ca.pem"
	if err := os.Setenv(TLSCAFilEnvVar, TLSCAFilEnvVarValue); !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	defer os.Unsetenv(TLSCAFilEnvVar)

	TLSInsecureEnvVar := "TARIAN_TLS_INSECURE_SKIP_VERIFY"
	TLSInsecureEnvVarValue := "false"
	if err := os.Setenv(TLSInsecureEnvVar, TLSInsecureEnvVarValue); !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	defer os.Unsetenv(TLSInsecureEnvVar)

	// Create global flags and load values from environment variables
	globalFlags := &GlobalFlags{}
	globalFlags.GetFlagValuesFromEnvVar()

	// Check if the value was correctly loaded from the environment variable
	assert.Equal(t, tarianServerEnvVarValue, globalFlags.ServerAddr)
	assert.True(t, globalFlags.ServerTLSEnabled)
	assert.False(t, globalFlags.ServerTLSInsecureSkipVerify)
	assert.Equal(t, "/path/to/ca.pem", globalFlags.ServerTLSCAFile)
}
