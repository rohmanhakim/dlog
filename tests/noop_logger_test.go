package dlog_test

import (
	"context"
	"testing"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpLogger(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	require.NotNil(t, logger)
}

func TestNoOpLogger_Enabled(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	assert.False(t, logger.Enabled())
}

func TestNoOpLogger_LogMethods(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	ctx := context.Background()
	fieldMap := dlog.FieldMap{"key": "value"}

	// These should not panic and should be no-ops
	logger.LogDebug(ctx, "debug message", fieldMap)
	logger.LogInfo(ctx, "info message", fieldMap)
	logger.LogWarn(ctx, "warn message", fieldMap)
	logger.LogError(ctx, "error message", nil, fieldMap)

	// Test with nil fieldMap
	logger.LogDebug(ctx, "debug message")
	logger.LogInfo(ctx, "info message")
	logger.LogWarn(ctx, "warn message")
	logger.LogError(ctx, "error message", nil)

	// Test with multiple fieldMaps
	logger.LogDebug(ctx, "debug message", fieldMap, dlog.FieldMap{"key2": "value2"})
}

func TestNoOpLogger_WithFields(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	fields := dlog.FieldMap{"key": "value"}

	result := logger.WithFields(fields)

	// WithFields should return the same instance for NoOpLogger
	assert.Same(t, logger, result, "WithFields should return the same NoOpLogger instance")

	// The returned logger should also be a NoOpLogger
	assert.False(t, result.Enabled())
}

func TestNoOpLogger_WithGroup(t *testing.T) {
	logger := dlog.NewNoOpLogger()

	result := logger.WithGroup("myservice")

	// WithGroup should return the same instance for NoOpLogger
	assert.Same(t, logger, result, "WithGroup should return the same NoOpLogger instance")

	// The returned logger should also be a NoOpLogger
	assert.False(t, result.Enabled())
}

func TestNoOpLogger_Close(t *testing.T) {
	logger := dlog.NewNoOpLogger()

	err := logger.Close()
	assert.NoError(t, err)
}

func TestNoOpLogger_ImplementsDebugLogger(t *testing.T) {
	// Compile-time check that NoOpLogger implements DebugLogger
	var _ dlog.DebugLogger = dlog.NewNoOpLogger()
}

func TestNoOpLogger_TableDriven(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	ctx := context.Background()

	tests := []struct {
		name   string
		method func()
	}{
		{
			name: "LogDebug",
			method: func() {
				logger.LogDebug(ctx, "test")
			},
		},
		{
			name: "LogInfo",
			method: func() {
				logger.LogInfo(ctx, "test")
			},
		},
		{
			name: "LogWarn",
			method: func() {
				logger.LogWarn(ctx, "test")
			},
		},
		{
			name: "LogError",
			method: func() {
				logger.LogError(ctx, "test", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.method()
		})
	}
}
