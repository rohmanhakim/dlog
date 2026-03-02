package dlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogfmtHandler_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, nil)

	require.NotNil(t, handler, "NewLogfmtHandler returned nil")
}

func TestNewLogfmtHandler_WithLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	ctx := context.Background()

	// Debug should be disabled
	assert.False(t, handler.Enabled(ctx, slog.LevelDebug), "Expected Debug to be disabled with Info level")

	// Info should be enabled
	assert.True(t, handler.Enabled(ctx, slog.LevelInfo), "Expected Info to be enabled")
}

func TestLogfmtHandler_Enabled(t *testing.T) {
	tests := []struct {
		name         string
		handlerLevel slog.Level
		checkLevel   slog.Level
		expected     bool
	}{
		{
			name:         "debug enabled when handler at debug",
			handlerLevel: slog.LevelDebug,
			checkLevel:   slog.LevelDebug,
			expected:     true,
		},
		{
			name:         "debug disabled when handler at info",
			handlerLevel: slog.LevelInfo,
			checkLevel:   slog.LevelDebug,
			expected:     false,
		},
		{
			name:         "info enabled when handler at info",
			handlerLevel: slog.LevelInfo,
			checkLevel:   slog.LevelInfo,
			expected:     true,
		},
		{
			name:         "warn enabled when handler at info",
			handlerLevel: slog.LevelInfo,
			checkLevel:   slog.LevelWarn,
			expected:     true,
		},
		{
			name:         "error enabled when handler at warn",
			handlerLevel: slog.LevelWarn,
			checkLevel:   slog.LevelError,
			expected:     true,
		},
		{
			name:         "warn disabled when handler at error",
			handlerLevel: slog.LevelError,
			checkLevel:   slog.LevelWarn,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
				Level: tt.handlerLevel,
			})

			ctx := context.Background()
			result := handler.Enabled(ctx, tt.checkLevel)

			assert.Equal(t, tt.expected, result, "Enabled(%v)", tt.checkLevel)
		})
	}
}

func TestLogfmtHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	output := buf.String()

	// Check output contains timestamp (logfmt format: key=value)
	assert.Contains(t, output, "time=2026-03-01T10:30:00.000Z")

	// Check output contains level
	assert.Contains(t, output, "level=INFO")

	// Check output contains message
	assert.Contains(t, output, `msg="test message"`)
}

func TestLogfmtHandler_HandleWithFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("service", "billing-api"),
		slog.Int("count", 42),
	)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	output := buf.String()

	// Check fields are present (logfmt format: key=value)
	assert.Contains(t, output, "service=billing-api")
	assert.Contains(t, output, "count=42")
}

func TestLogfmtHandler_FieldFiltering(t *testing.T) {
	tests := []struct {
		name          string
		includeFields []string
		excludeFields []string
		attrs         []slog.Attr
		contains      []string
		notContains   []string
	}{
		{
			name:          "no filtering",
			includeFields: nil,
			excludeFields: nil,
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			contains:    []string{"service=api", "version=1.0"},
			notContains: nil,
		},
		{
			name:          "exclude fields",
			includeFields: nil,
			excludeFields: []string{"version"},
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			contains:    []string{"service=api"},
			notContains: []string{"version=1.0"},
		},
		{
			name:          "include fields only",
			includeFields: []string{"time", "level", "msg", "service"},
			excludeFields: nil,
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			contains:    []string{"service=api"},
			notContains: []string{"version=1.0"},
		},
		{
			name:          "include and exclude combined",
			includeFields: []string{"time", "level", "msg", "service", "version"},
			excludeFields: []string{"version"},
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			contains:    []string{"service=api"},
			notContains: []string{"version=1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
				Level:         slog.LevelDebug,
				IncludeFields: tt.includeFields,
				ExcludeFields: tt.excludeFields,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)
			if len(tt.attrs) > 0 {
				record.AddAttrs(tt.attrs...)
			}

			err := handler.Handle(ctx, record)
			require.NoError(t, err, "Handle failed")

			output := buf.String()

			// Check expected content
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}

			// Check content that should not be present
			for _, s := range tt.notContains {
				assert.NotContains(t, output, s)
			}
		})
	}
}

func TestLogfmtHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Add attributes
	newHandler := handler.WithAttrs([]slog.Attr{
		slog.String("service", "test-service"),
		slog.String("version", "1.0.0"),
	})

	require.NotNil(t, newHandler, "WithAttrs returned nil")

	// The new handler should be different from the original
	assert.NotEqual(t, handler, newHandler, "WithAttrs should return a new handler")

	// Verify the new handler works
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	var newBuf bytes.Buffer
	logfmtHandler := dlog.NewLogfmtHandler(&newBuf, &dlog.HandlerOptions{Level: slog.LevelDebug})
	handlerWithAttrs := logfmtHandler.WithAttrs([]slog.Attr{
		slog.String("pre_field", "pre_value"),
	})

	err := handlerWithAttrs.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	output := newBuf.String()
	assert.Contains(t, output, "pre_field=pre_value")
}

func TestLogfmtHandler_WithAttrs_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// With empty attrs should return same handler
	newHandler := handler.WithAttrs([]slog.Attr{})

	assert.Same(t, handler, newHandler, "WithAttrs with empty attrs should return same handler")
}

func TestLogfmtHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// WithGroup is a no-op for LogfmtHandler, should return same handler
	newHandler := handler.WithGroup("test-group")

	assert.Same(t, handler, newHandler, "WithGroup should return same handler (no-op for LogfmtHandler)")
}

func TestLogfmtHandler_ImplementsSlogHandler(t *testing.T) {
	// Compile-time check that LogfmtHandler implements slog.Handler
	var _ slog.Handler = dlog.NewLogfmtHandler(nil, nil)
}

func TestLogfmtHandler_Format(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		message  string
		attrs    []slog.Attr
		contains []string
	}{
		{
			name:     "debug level",
			level:    slog.LevelDebug,
			message:  "debug msg",
			contains: []string{"level=DEBUG", `msg="debug msg"`},
		},
		{
			name:     "info level",
			level:    slog.LevelInfo,
			message:  "info msg",
			contains: []string{"level=INFO", `msg="info msg"`},
		},
		{
			name:     "warn level",
			level:    slog.LevelWarn,
			message:  "warn msg",
			contains: []string{"level=WARN", `msg="warn msg"`},
		},
		{
			name:     "error level",
			level:    slog.LevelError,
			message:  "error msg",
			contains: []string{"level=ERROR", `msg="error msg"`},
		},
		{
			name:    "with string attr",
			level:   slog.LevelInfo,
			message: "msg",
			attrs: []slog.Attr{
				slog.String("key", "value"),
			},
			contains: []string{"key=value"},
		},
		{
			name:    "with int attr",
			level:   slog.LevelInfo,
			message: "msg",
			attrs: []slog.Attr{
				slog.Int("count", 100),
			},
			contains: []string{"count=100"},
		},
		{
			name:     "with message containing spaces",
			level:    slog.LevelInfo,
			message:  "hello world",
			attrs:    []slog.Attr{},
			contains: []string{`msg="hello world"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogfmtHandler(&buf, &dlog.HandlerOptions{
				Level: slog.LevelDebug,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, tt.level, tt.message, 0)
			if len(tt.attrs) > 0 {
				record.AddAttrs(tt.attrs...)
			}

			err := handler.Handle(ctx, record)
			require.NoError(t, err, "Handle failed")

			output := buf.String()
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}
