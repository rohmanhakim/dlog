package dlog

import (
	"context"
)

// DebugLogger provides structured debug logging capabilities.
// All methods are no-ops when debug mode is disabled.
type DebugLogger interface {
	// Enabled returns true if debug logging is enabled.
	// When false, the logger will skip logging entirely for efficiency.
	Enabled() bool

	// LogDebug logs string message with slog.Level = LevelDebug.
	LogDebug(ctx context.Context, message string, fieldMap ...FieldMap)

	// LogDebug logs string message with slog.Level = LevelInfo.
	LogInfo(ctx context.Context, message string, fieldMap ...FieldMap)

	// LogDebug logs string message with slog.Level = LevelWarn.
	LogWarn(ctx context.Context, message string, fieldMap ...FieldMap)

	// LogError logs string message with slog.level = LevelError.
	LogError(ctx context.Context, message string, err error, fieldMap ...FieldMap)

	// WithFields returns a logger with pre-populated fields.
	WithFields(fields FieldMap) DebugLogger

	// Close flushes any buffered output and closes file handles.
	Close() error
}

// FieldMap is a map of structured field names to values.
type FieldMap map[string]any
