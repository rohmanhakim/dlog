package dlog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/go-logfmt/logfmt"
)

// LogfmtHandler creates logfmt-formatted output.
// Logfmt is a key=value format that is both human-readable and machine-parseable.
//
// Implements [slog.Handler] interface.
type LogfmtHandler struct {
	mu          sync.Mutex
	w           io.Writer
	level       slog.Level
	attrs       []slog.Attr
	fieldFilter *fieldFilter
}

// NewLogfmtHandler creates a new LogfmtHandler writing to the specified writer.
func NewLogfmtHandler(w io.Writer, opts *HandlerOptions) *LogfmtHandler {
	level := slog.LevelDebug // default level
	var ff *fieldFilter

	// Only override default if opts is provided
	if opts != nil {
		level = opts.Level.toSlog()
		ff = newFieldFilter(opts.IncludeFields, opts.ExcludeFields)
	}

	return &LogfmtHandler{
		w:           w,
		level:       level,
		fieldFilter: ff,
	}
}

// Enabled returns true if the handler should log at the given level.
func (h *LogfmtHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes the log record and writes it in logfmt format.
func (h *LogfmtHandler) Handle(_ context.Context, r slog.Record) error {
	// Encode to a buffer first, then lock only for the write
	var buf bytes.Buffer
	enc := logfmt.NewEncoder(&buf)

	// Collect all key-value pairs for filtering
	fields := make([]struct {
		key   string
		value any
	}, 0)

	// Write timestamp
	fields = append(fields, struct {
		key   string
		value any
	}{"time", r.Time.Format("2006-01-02T15:04:05.000Z")})

	// Write level
	fields = append(fields, struct {
		key   string
		value any
	}{"level", r.Level.String()})

	// Write message
	fields = append(fields, struct {
		key   string
		value any
	}{"msg", r.Message})

	// Collect attrs from handler context
	for _, attr := range h.attrs {
		fields = appendField(fields, attr)
	}

	// Collect attrs from the record
	r.Attrs(func(attr slog.Attr) bool {
		fields = appendField(fields, attr)
		return true
	})

	// Apply field filtering and encode
	for _, field := range fields {
		if h.shouldIncludeField(field.key) {
			enc.EncodeKeyval(field.key, field.value)
		}
	}

	if err := enc.EndRecord(); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

// shouldIncludeField checks if a field should be included based on include/exclude lists.
func (h *LogfmtHandler) shouldIncludeField(key string) bool {
	if h.fieldFilter == nil {
		return true
	}
	return h.fieldFilter.shouldInclude(key)
}

// WithAttrs returns a new handler with the given attributes added.
func (h *LogfmtHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := &LogfmtHandler{
		w:           h.w,
		level:       h.level,
		fieldFilter: h.fieldFilter,
	}
	newHandler.attrs = append(newHandler.attrs, h.attrs...)
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return newHandler
}

// WithGroup returns a new handler with the given group name.
// Logfmt handler doesn't support grouping, so it's a no-op.
func (h *LogfmtHandler) WithGroup(name string) slog.Handler {
	return h
}

// appendField appends a field to the fields slice, handling groups.
func appendField(fields []struct {
	key   string
	value any
}, attr slog.Attr) []struct {
	key   string
	value any
} {
	if attr.Value.Kind() == slog.KindGroup {
		// For groups, flatten with prefixed keys
		for _, groupAttr := range attr.Value.Group() {
			fields = append(fields, struct {
				key   string
				value any
			}{attr.Key + "." + groupAttr.Key, groupAttr.Value.Any()})
		}
	} else {
		fields = append(fields, struct {
			key   string
			value any
		}{attr.Key, attr.Value.Any()})
	}
	return fields
}
