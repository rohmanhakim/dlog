package dlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// TextHandler creates a human-readable text output.
//
// Implements [slog.Handler] interface.
type TextHandler struct {
	mu          sync.Mutex
	w           io.Writer
	level       slog.Level
	attrs       []slog.Attr
	fieldFilter *fieldFilter
}

// NewTextHandler creates a new TextHandler writing to the specified writer.
func NewTextHandler(w io.Writer, opts *HandlerOptions) *TextHandler {
	level := slog.LevelDebug // default level
	var ff *fieldFilter

	// Only override default if opts is provided
	if opts != nil {
		level = opts.Level.toSlog()
		ff = newFieldFilter(opts.IncludeFields, opts.ExcludeFields)
	}

	return &TextHandler{
		w:           w,
		level:       level,
		fieldFilter: ff,
	}
}

// Enabled returns true if the handler should log at the given level.
func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes the log record and writes it in human-readable text format.
func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	// Format: 2026-02-21T13:50:00.123Z [DEBUG] [logger_name] message
	var sb strings.Builder

	// Timestamp
	sb.WriteString(r.Time.Format("2006-01-02T15:04:05.000Z"))

	// Level
	sb.WriteString(" [")
	sb.WriteString(r.Level.String())
	sb.WriteString("]")

	// Message
	sb.WriteString(" ")
	sb.WriteString(r.Message)

	// Collect fields for output
	fields := make([]string, 0)

	// Add attrs from handler context
	for _, attr := range h.attrs {
		fields = append(fields, formatField(attr.Key, attr.Value))
	}

	// Add attrs from the record
	r.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, formatField(attr.Key, attr.Value))
		return true
	})

	// Apply field filtering
	fields = h.filterFieldList(fields)

	// Write fields
	if len(fields) > 0 {
		sb.WriteString(" ")
		sb.WriteString(strings.Join(fields, " "))
	}

	sb.WriteString("\n")

	// Build the output bytes first, then lock only for the write
	output := []byte(sb.String())

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(output)
	return err
}

// WithAttrs returns a new handler with the given attributes added.
func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := &TextHandler{
		w:           h.w,
		level:       h.level,
		fieldFilter: h.fieldFilter,
	}
	newHandler.attrs = append(newHandler.attrs, h.attrs...)
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return newHandler
}

// WithGroup returns a new handler with the given group name.
// Text handler doesn't support grouping, so it's a no-op.
func (h *TextHandler) WithGroup(name string) slog.Handler {
	return h
}

// formatField formats a key-value pair for text output.
func formatField(key string, value slog.Value) string {
	switch value.Kind() {
	case slog.KindGroup:
		// For groups, format as nested key=value
		parts := make([]string, 0)
		for _, attr := range value.Group() {
			parts = append(parts, formatField(attr.Key, attr.Value))
		}
		return fmt.Sprintf("%s={%s}", key, strings.Join(parts, " "))
	default:
		return fmt.Sprintf("%s=%v", key, value.Any())
	}
}

// filterFieldList applies include/exclude field filtering to a list of key=value strings.
func (h *TextHandler) filterFieldList(fields []string) []string {
	if h.fieldFilter == nil {
		return fields
	}

	result := make([]string, 0, len(fields))
	for _, field := range fields {
		// Extract key from "key=value" format
		idx := strings.Index(field, "=")
		if idx == -1 {
			result = append(result, field)
			continue
		}
		key := field[:idx]

		if h.fieldFilter.shouldInclude(key) {
			result = append(result, field)
		}
	}

	return result
}
