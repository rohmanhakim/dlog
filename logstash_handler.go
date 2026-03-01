package dlog

import (
	"context"
	"io"
	"log/slog"
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
	jsonHandler *slog.JSONHandler
	fieldFilter *fieldFilter
	groups      []string
	attrs       []slog.Attr
}

// NewLogstashHandler creates a new LogstashHandler writing to the specified writer.
func NewLogstashHandler(w io.Writer, opts *HandlerOptions) *LogstashHandler {
	level := slog.LevelDebug // default level
	var ff *fieldFilter

	if opts != nil {
		level = opts.Level
		ff = newFieldFilter(opts.IncludeFields, opts.ExcludeFields)
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
		jsonHandler: jsonHandler,
		fieldFilter: ff,
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

// builtinFields are fields that are always included (handled by ReplaceAttr)
var builtinFields = map[string]struct{}{
	"time":       {},
	"level":      {},
	"msg":        {},
	"@timestamp": {},
	"log.level":  {},
	"message":    {},
}

// shouldIncludeField checks if a field should be included based on include/exclude lists.
func (h *LogstashHandler) shouldIncludeField(key string) bool {
	// Built-in fields are always included
	if _, isBuiltin := builtinFields[key]; isBuiltin {
		return true
	}

	if h.fieldFilter == nil {
		return true
	}
	return h.fieldFilter.shouldInclude(key)
}

// WithAttrs returns a new handler with the given attributes added.
func (h *LogstashHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &LogstashHandler{
		jsonHandler: h.jsonHandler,
		fieldFilter: h.fieldFilter,
		groups:      h.groups,
		attrs:       newAttrs,
	}
}

// WithGroup returns a new handler with the given group name prepended to attribute keys.
func (h *LogstashHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &LogstashHandler{
		jsonHandler: h.jsonHandler,
		fieldFilter: h.fieldFilter,
		groups:      newGroups,
		attrs:       h.attrs,
	}
}
