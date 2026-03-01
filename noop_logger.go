package dlog

import (
	"context"
)

// NoOpLogger is a no-operation implementation of DebugLogger.
// It provides zero overhead when debug mode is disabled.
// All methods are empty and Enabled() always returns false.
type NoOpLogger struct{}

// NewNoOpLogger creates a new NoOpLogger instance.
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Enabled returns false - debug logging is disabled.
func (n *NoOpLogger) Enabled() bool { return false }

// LogDebug logs string message with slog.Level = LevelDebug.
func (n *NoOpLogger) LogDebug(ctx context.Context, message string, fieldMap ...FieldMap) {}

// LogDebug logs string message with slog.Level = LevelInfo.
func (n *NoOpLogger) LogInfo(ctx context.Context, message string, fieldMap ...FieldMap) {}

// LogDebug logs string message with slog.Level = LevelWarn.
func (n *NoOpLogger) LogWarn(ctx context.Context, message string, fieldMap ...FieldMap) {}

// LogError is a no-op.
func (n *NoOpLogger) LogError(_ context.Context, _ string, _ error, _ ...FieldMap) {}

// WithFields returns the same NoOpLogger instance.
func (n *NoOpLogger) WithFields(_ FieldMap) DebugLogger { return n }

// WithGroup returns the same NoOpLogger instance.
func (n *NoOpLogger) WithGroup(_ string) DebugLogger { return n }

// Close returns nil - no resources to release.
func (n *NoOpLogger) Close() error { return nil }
