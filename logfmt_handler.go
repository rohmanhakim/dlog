package dlog

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"

	"github.com/go-logfmt/logfmt"
)

// LogfmtHandler creates logfmt-formatted output.
// Logfmt is a key=value format that is both human-readable and machine-parseable.
//
// Implements [slog.Handler] interface.
type LogfmtHandler struct {
	mu    sync.Mutex
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

// NewLogfmtHandler creates a new LogfmtHandler writing to the specified writer.
func NewLogfmtHandler(w io.Writer, opts *LogfmtHandlerOptions) *LogfmtHandler {
	level := slog.LevelDebug // default level

	// Only override default if opts is provided and has a non-zero level
	// Note: slog.LevelInfo = 0, so we check if opts is non-nil first
	if opts != nil {
		level = opts.Level
	}

	return &LogfmtHandler{
		w:     w,
		level: level,
	}
}

// LogfmtHandlerOptions configures the LogfmtHandler.
type LogfmtHandlerOptions struct {
	// Level is the minimum log level to output.
	Level slog.Level
}

// Enabled returns true if the handler should log at the given level.
func (h *LogfmtHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes the log record and writes it in logfmt format.
func (h *LogfmtHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	enc := logfmt.NewEncoder(h.w)

	// Write timestamp
	enc.EncodeKeyval("time", r.Time.Format("2006-01-02T15:04:05.000Z"))

	// Write level
	enc.EncodeKeyval("level", r.Level.String())

	// Write message
	enc.EncodeKeyval("msg", r.Message)

	// Add attrs from handler context
	for _, attr := range h.attrs {
		encodeAttr(enc, attr)
	}

	// Add attrs from the record
	r.Attrs(func(attr slog.Attr) bool {
		encodeAttr(enc, attr)
		return true
	})

	return enc.EndRecord()
}

// WithAttrs returns a new handler with the given attributes added.
func (h *LogfmtHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := &LogfmtHandler{
		w:     h.w,
		level: h.level,
		attrs: slices.Clone(h.attrs),
	}
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return newHandler
}

// WithGroup returns a new handler with the given group name.
// Logfmt handler doesn't support grouping, so it's a no-op.
func (h *LogfmtHandler) WithGroup(name string) slog.Handler {
	return h
}

// encodeAttr encodes a single attribute to the logfmt encoder.
func encodeAttr(enc *logfmt.Encoder, attr slog.Attr) {
	if attr.Value.Kind() == slog.KindGroup {
		// For groups, flatten with prefixed keys
		for _, groupAttr := range attr.Value.Group() {
			enc.EncodeKeyval(attr.Key+"."+groupAttr.Key, groupAttr.Value.Any())
		}
	} else {
		enc.EncodeKeyval(attr.Key, attr.Value.Any())
	}
}
