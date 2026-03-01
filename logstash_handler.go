package dlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

// encodeJSON marshals a map to JSON bytes.
func encodeJSON(m map[string]any) ([]byte, error) {
	return json.Marshal(m)
}

// LogstashHandler is a custom slog.Handler that outputs logs in Logstash/Elasticsearch
// compatible JSON format as specified in the design document.
//
// Implements [slog.Handler] interface.
type LogstashHandler struct {
	mu            sync.Mutex
	w             io.Writer
	level         slog.Level
	attrs         []slog.Attr
	groups        []string
	includeFields []string
	excludeFields []string
}

// NewLogstashHandler creates a new LogstashHandler writing to the specified writer.
func NewLogstashHandler(w io.Writer, opts *HandlerOptions) *LogstashHandler {
	level := slog.LevelDebug // default level
	var includeFields, excludeFields []string

	// Only override default if opts is provided
	if opts != nil {
		level = opts.Level
		includeFields = opts.IncludeFields
		excludeFields = opts.ExcludeFields
	}

	return &LogstashHandler{
		w:             w,
		level:         level,
		includeFields: includeFields,
		excludeFields: excludeFields,
	}
}

// Enabled returns true if the handler should log at the given level.
func (h *LogstashHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes the log record and writes it in Logstash JSON format.
func (h *LogstashHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Build the log entry following the design document structure
	entry := map[string]any{
		"@timestamp":  r.Time.Format(time.RFC3339Nano),
		"@version":    "1",
		"level":       r.Level.String(),
		"message":     r.Message,
		"thread_name": "main",
	}

	// Add attrs from handler context
	for _, attr := range h.attrs {
		h.addField(entry, attr.Key, attr.Value)
	}

	// Add attrs from the record
	r.Attrs(func(attr slog.Attr) bool {
		h.addField(entry, attr.Key, attr.Value)
		return true
	})

	// Apply field filtering using the shared function
	entry = FilterFields(entry, h.includeFields, h.excludeFields)

	// Write JSON line
	jsonData, err := encodeJSON(entry)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(h.w, string(jsonData))
	return err
}

// WithAttrs returns a new handler with the given attributes added.
func (h *LogstashHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := h.clone()
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return newHandler
}

// WithGroup returns a new handler with the given group name prepended to attribute keys.
func (h *LogstashHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandler := h.clone()
	newHandler.groups = append(newHandler.groups, name)
	return newHandler
}

// clone creates a copy of the handler.
func (h *LogstashHandler) clone() *LogstashHandler {
	return &LogstashHandler{
		w:             h.w,
		level:         h.level,
		attrs:         slices.Clone(h.attrs),
		groups:        slices.Clone(h.groups),
		includeFields: h.includeFields,
		excludeFields: h.excludeFields,
	}
}

// addField adds a field to the entry map, respecting group prefixes.
func (h *LogstashHandler) addField(entry map[string]any, key string, value slog.Value) {
	// Build the full key with group prefix
	fullKey := key
	if len(h.groups) > 0 {
		fullKey = strings.Join(h.groups, ".") + "." + key
	}

	// Handle different value kinds
	switch value.Kind() {
	case slog.KindGroup:
		// For group values, recursively add fields
		groupAttrs := value.Group()
		for _, attr := range groupAttrs {
			h.addField(entry, fullKey+"."+attr.Key, attr.Value)
		}
	case slog.KindLogValuer:
		entry[fullKey] = value.Resolve()
	default:
		entry[fullKey] = value.Any()
	}
}
