package dlog

import (
	"context"
	"log/slog"
)

// SlogLogger wraps slog.Logger and implements the DebugLogger interface.
// It provides structured logging with domain-specific methods for pipeline stages,
// retry attempts, and rate limiting decisions.
type SlogLogger struct {
	logger   *slog.Logger
	enabled  bool
	preAttrs FieldMap
	closer   func() error
}

// NewSlogLogger creates a new SlogLogger with the given configuration.
// If config.Enabled is false, a NoOpLogger is returned instead.
func NewSlogLogger(config DebugConfig) (DebugLogger, error) {
	if !config.Enabled {
		return NewNoOpLogger(), nil
	}

	// Create the writer (stdout + optional file)
	writer, err := NewMultiWriter(config.OutputFile)
	if err != nil {
		return nil, err
	}

	// Create the appropriate handler based on format
	var handler slog.Handler
	switch config.Format {
	case FormatText:
		handler = NewTextHandler(
			writer,
			&TextHandlerOptions{
				Level: config.MinLevel,
			},
		)
	case FormatLogfmt:
		handler = NewLogfmtHandler(
			writer,
			&LogfmtHandlerOptions{
				Level: config.MinLevel,
			},
		)
	default:
		handler = NewLogstashHandler(
			writer,
			&LogstashHandlerOptions{
				Level:         config.MinLevel,
				IncludeFields: config.IncludeFields,
				ExcludeFields: config.ExcludeFields,
			},
		)
	}

	return &SlogLogger{
		logger:  slog.New(handler),
		enabled: true,
		closer:  writer.Close,
	}, nil
}

// Enabled returns true if debug logging is enabled.
func (s *SlogLogger) Enabled() bool { return s.enabled }

// LogError logs a debug-level message with context.
func (s *SlogLogger) LogDebug(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := []slog.Attr{}

	// Add pre-populated fields
	for k, v := range s.preAttrs {
		attrs = append(attrs, slog.Any(k, v))
	}

	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// LogInfo logs an info-level message with context.
func (s *SlogLogger) LogInfo(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := []slog.Attr{}

	// Add pre-populated fields
	for k, v := range s.preAttrs {
		attrs = append(attrs, slog.Any(k, v))
	}

	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogError logs a warn-level message with context.
func (s *SlogLogger) LogWarn(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := []slog.Attr{}

	// Add pre-populated fields
	for k, v := range s.preAttrs {
		attrs = append(attrs, slog.Any(k, v))
	}

	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// LogError logs a debug-level error with context.
func (s *SlogLogger) LogError(ctx context.Context, stage string, err error, fieldMap ...FieldMap) {
	attrs := []slog.Attr{
		slog.String("error", err.Error()),
	}

	// Add pre-populated fields
	for k, v := range s.preAttrs {
		attrs = append(attrs, slog.Any(k, v))
	}

	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelError, "Error occurred", attrs...)
}

// WithFields returns a logger with pre-populated fields.
func (s *SlogLogger) WithFields(fields FieldMap) DebugLogger {
	merged := make(FieldMap)
	for k, v := range s.preAttrs {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}

	return &SlogLogger{
		logger:   s.logger,
		enabled:  s.enabled,
		preAttrs: merged,
		closer:   s.closer,
	}
}

// Close flushes any buffered output and closes file handles.
func (s *SlogLogger) Close() error {
	if s.closer != nil {
		return s.closer()
	}
	return nil
}
