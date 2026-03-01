package dlog

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
)

// LogstashHandler is a custom slog.Handler that outputs logs in Logstash/Elasticsearch
// compatible JSON format. It wraps slog.JSONHandler for optimal performance.
//
// Key differences from standard JSONHandler:
//   - Field names: @timestamp, log.level, message (instead of time, level, msg)
//   - Groups are flattened with dot notation (e.g., request.id instead of request: {id: ...})
//   - Supports field filtering via include/exclude lists
//
// Implements [slog.Handler] interface.
type LogstashHandler struct {
	jsonHandler   *slog.JSONHandler
	includeFields []string
	excludeFields []string
	groups        []string
	attrs         []slog.Attr
}

// NewLogstashHandler creates a new LogstashHandler writing to the specified writer.
func NewLogstashHandler(w io.Writer, opts *HandlerOptions) *LogstashHandler {
	level := slog.LevelDebug // default level
	var includeFields, excludeFields []string

	if opts != nil {
		level = opts.Level
		includeFields = opts.IncludeFields
		excludeFields = opts.ExcludeFields
	}

	// Use a JSONHandler with ReplaceAttr for Logstash field naming
	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename time -> @timestamp
			if a.Key == slog.TimeKey {
				a.Key = "@timestamp"
			}
			// Rename level -> log.level
			if a.Key == slog.LevelKey {
				a.Key = "log.level"
			}
			// Rename msg -> message
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})

	return &LogstashHandler{
		jsonHandler:   jsonHandler,
		includeFields: includeFields,
		excludeFields: excludeFields,
	}
}

// Enabled returns true if the handler should log at the given level.
func (h *LogstashHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.jsonHandler.Enabled(ctx, level)
}

// Handle processes the log record and writes it in Logstash JSON format.
// It flattens groups and renames standard fields before delegating to JSONHandler.
func (h *LogstashHandler) Handle(ctx context.Context, r slog.Record) error {
	// Build a new record with flattened attrs
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	// Collect and flatten all attrs
	var flattenedAttrs []slog.Attr

	// Add handler attrs with group prefix
	prefix := ""
	if len(h.groups) > 0 {
		prefix = strings.Join(h.groups, ".")
	}

	for _, attr := range h.attrs {
		flattenedAttrs = h.flattenAndFilterAttrs(flattenedAttrs, attr, prefix)
	}

	// Add record attrs
	r.Attrs(func(attr slog.Attr) bool {
		flattenedAttrs = h.flattenAndFilterAttrs(flattenedAttrs, attr, prefix)
		return true
	})

	// Add all flattened attrs to new record
	newRecord.AddAttrs(flattenedAttrs...)

	return h.jsonHandler.Handle(ctx, newRecord)
}

// flattenAndFilterAttrs recursively flattens group attrs and applies field filtering.
func (h *LogstashHandler) flattenAndFilterAttrs(attrs []slog.Attr, attr slog.Attr, prefix string) []slog.Attr {
	// Build full key
	fullKey := attr.Key
	if prefix != "" {
		fullKey = prefix + "." + attr.Key
	}

	// Handle groups by flattening
	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		for _, ga := range groupAttrs {
			attrs = h.flattenAndFilterAttrs(attrs, ga, fullKey)
		}
		return attrs
	}

	// Handle LogValuer by resolving the value
	if attr.Value.Kind() == slog.KindLogValuer {
		attr.Value = attr.Value.Resolve()
	}

	// Apply field filtering
	if !h.shouldIncludeField(fullKey) {
		return attrs
	}

	// Add with flattened key
	return append(attrs, slog.Attr{Key: fullKey, Value: attr.Value})
}

// shouldIncludeField checks if a field should be included based on include/exclude lists.
func (h *LogstashHandler) shouldIncludeField(key string) bool {
	// Built-in fields (these are handled by ReplaceAttr)
	builtins := []string{"time", "level", "msg", "@timestamp", "log.level", "message"}
	if slices.Contains(builtins, key) {
		return true
	}

	// Check exclude list first
	if slices.Contains(h.excludeFields, key) {
		return false
	}

	// Check include list (if specified)
	if len(h.includeFields) > 0 && !slices.Contains(h.includeFields, key) {
		return false
	}

	return true
}

// WithAttrs returns a new handler with the given attributes added.
func (h *LogstashHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return &LogstashHandler{
		jsonHandler:   h.jsonHandler,
		includeFields: h.includeFields,
		excludeFields: h.excludeFields,
		groups:        h.groups,
		attrs:         append(slices.Clone(h.attrs), attrs...),
	}
}

// WithGroup returns a new handler with the given group name prepended to attribute keys.
func (h *LogstashHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &LogstashHandler{
		jsonHandler:   h.jsonHandler,
		includeFields: h.includeFields,
		excludeFields: h.excludeFields,
		groups:        append(slices.Clone(h.groups), name),
		attrs:         h.attrs,
	}
}
