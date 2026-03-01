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
// If enabled is false, a NoOpLogger is returned instead.
// Optional parameters can be provided using WithMinLevel, WithIncludeFields, and WithExcludeFields.
func NewSlogLogger(enabled bool, format Format, outputFile string, opts ...Option) (DebugLogger, error) {
	if !enabled {
		return NewNoOpLogger(), nil
	}

	// Apply defaults
	cfg := &config{
		minLevel:      slog.LevelDebug,
		includeFields: []string{},
		excludeFields: []string{},
		preAttrs:      nil,
		groupName:     "",
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create the writer (stdout + optional file)
	writer, err := NewMultiWriter(outputFile)
	if err != nil {
		return nil, err
	}

	// Create the appropriate handler based on format
	var handler slog.Handler
	switch format {
	case FormatText:
		handler = NewTextHandler(
			writer,
			&TextHandlerOptions{
				Level: cfg.minLevel,
			},
		)
	case FormatLogfmt:
		handler = NewLogfmtHandler(
			writer,
			&LogfmtHandlerOptions{
				Level: cfg.minLevel,
			},
		)
	default:
		handler = NewLogstashHandler(
			writer,
			&LogstashHandlerOptions{
				Level:         cfg.minLevel,
				IncludeFields: cfg.includeFields,
				ExcludeFields: cfg.excludeFields,
			},
		)
	}

	// Create logger and apply group if specified
	logger := slog.New(handler)
	if cfg.groupName != "" {
		logger = logger.WithGroup(cfg.groupName)
	}

	return &SlogLogger{
		logger:   logger,
		enabled:  true,
		preAttrs: cfg.preAttrs,
		closer:   writer.Close,
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

// WithGroup returns a logger with all subsequent attributes grouped under the given name.
func (s *SlogLogger) WithGroup(name string) DebugLogger {
	return &SlogLogger{
		logger:   s.logger.WithGroup(name),
		enabled:  s.enabled,
		preAttrs: s.preAttrs,
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
