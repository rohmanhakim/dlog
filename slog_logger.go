package dlog

import (
	"context"
	"log/slog"
)

// SlogLogger wraps slog.Logger and implements the DebugLogger interface.
// It provides structured logging with domain-specific methods for pipeline stages,
// retry attempts, and rate limiting decisions.
type SlogLogger struct {
	logger  *slog.Logger
	enabled bool
	closer  func() error
}

// NewSlogLogger creates a new SlogLogger with the given configuration.
// If enabled is false, a NoOpLogger is returned instead.
// Optional parameters can be provided using WithMinLevel, WithIncludeFields, and WithExcludeFields.
func NewSlogLogger(enabled bool, outputFile string, format Format, opts ...Option) (DebugLogger, error) {
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
		syncMode:      SyncImmediate, // Default to maximum durability
		syncInterval:  0,             // Will use default in NewMultiWriter
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create the writer (stdout + optional file)
	writer, err := NewMultiWriter(outputFile, cfg.syncMode, cfg.syncInterval)
	if err != nil {
		return nil, err
	}

	// Create the appropriate handler based on format
	var handler slog.Handler
	handlerOpts := &HandlerOptions{
		Level:         cfg.minLevel,
		IncludeFields: cfg.includeFields,
		ExcludeFields: cfg.excludeFields,
	}

	switch format {
	case FormatJSON:
		// Standard slog.JSONHandler (nested groups)
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: cfg.minLevel,
		})
	case FormatText:
		handler = NewTextHandler(writer, handlerOpts)
	case FormatLogfmt:
		handler = NewLogfmtHandler(writer, handlerOpts)
	case FormatLogstash:
		// Logstash-compatible format (flattened groups, renamed fields)
		handler = NewLogstashHandler(writer, handlerOpts)
	}

	// Create logger and apply group if specified
	logger := slog.New(handler)
	if cfg.groupName != "" {
		logger = logger.WithGroup(cfg.groupName)
	}

	// Apply pre-populated fields using slog's built-in With() for optimal performance
	// Convert FieldMap to alternating key-value pairs for logger.With()
	if len(cfg.preAttrs) > 0 {
		args := make([]any, 0, len(cfg.preAttrs)*2)
		for k, v := range cfg.preAttrs {
			args = append(args, k, v)
		}
		logger = logger.With(args...)
	}

	return &SlogLogger{
		logger:  logger,
		enabled: true,
		closer:  writer.Close,
	}, nil
}

// Enabled returns true if debug logging is enabled.
func (s *SlogLogger) Enabled() bool { return s.enabled }

// LogDebug logs a debug-level message with context.
// Pre-populated fields from WithFields are already baked into the logger.
func (s *SlogLogger) LogDebug(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := make([]slog.Attr, 0, len(fieldMap))
	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	s.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// LogInfo logs an info-level message with context.
// Pre-populated fields from WithFields are already baked into the logger.
func (s *SlogLogger) LogInfo(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := make([]slog.Attr, 0, len(fieldMap))
	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	s.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogWarn logs a warn-level message with context.
// Pre-populated fields from WithFields are already baked into the logger.
func (s *SlogLogger) LogWarn(ctx context.Context, message string, fieldMap ...FieldMap) {
	attrs := make([]slog.Attr, 0, len(fieldMap))
	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	s.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// LogError logs an error-level message with context.
// Pre-populated fields from WithFields are already baked into the logger.
func (s *SlogLogger) LogError(ctx context.Context, message string, err error, fieldMap ...FieldMap) {
	attrs := make([]slog.Attr, 0, 1+len(fieldMap))

	// Add error attribute with nil check
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	for _, fm := range fieldMap {
		for k, v := range fm {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	s.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// WithFields returns a logger with pre-populated fields.
// Fields are baked into the slog.Logger for optimal performance.
func (s *SlogLogger) WithFields(fields FieldMap) DebugLogger {
	// Convert FieldMap to alternating key-value pairs for logger.With()
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	return &SlogLogger{
		logger:  s.logger.With(args...),
		enabled: s.enabled,
		closer:  s.closer,
	}
}

// WithGroup returns a logger with all subsequent attributes grouped under the given name.
func (s *SlogLogger) WithGroup(name string) DebugLogger {
	return &SlogLogger{
		logger:  s.logger.WithGroup(name),
		enabled: s.enabled,
		closer:  s.closer,
	}
}

// Close flushes any buffered output and closes file handles.
func (s *SlogLogger) Close() error {
	if s.closer != nil {
		return s.closer()
	}
	return nil
}
