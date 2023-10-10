package flags

import (
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
}

func TestValidateGlobalFlags(t *testing.T) {
	tests := []struct {
		name          string
		globalFlags   *GlobalFlags
		expectedError string
	}{
		{
			name:          "Valid Flags",
			globalFlags:   &GlobalFlags{LogLevel: "info", LogFormatter: "text"},
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
