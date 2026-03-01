package dlog_test

import (
	"context"
	"testing"

	"github.com/rohmanhakim/dlog"
)

func TestNewNoOpLogger(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	if logger == nil {
		t.Fatal("NewNoOpLogger returned nil")
	}
}

func TestNoOpLogger_Enabled(t *testing.T) {
	logger := dlog.NewNoOpLogger()
	if logger.Enabled() {
		t.Error("Enabled() should always return false for NoOpLogger")
	}
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
	if result != logger {
		t.Error("WithFields should return the same NoOpLogger instance")
	}

	// The returned logger should also be a NoOpLogger
	if result.Enabled() {
		t.Error("Returned logger's Enabled() should return false")
	}
}

func TestNoOpLogger_Close(t *testing.T) {
	logger := dlog.NewNoOpLogger()

	err := logger.Close()
	if err != nil {
		t.Errorf("Close() should return nil, got: %v", err)
	}
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
