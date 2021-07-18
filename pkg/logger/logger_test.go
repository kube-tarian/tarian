package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger("debug", "json")

	entry := logger.Desugar().Check(zapcore.DebugLevel, "log message")

	require.NotNil(t, entry)
	assert.Equal(t, zapcore.DebugLevel, entry.Level)
	assert.Equal(t, "log message", entry.Message)
}
