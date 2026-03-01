package dlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rohmanhakim/dlog"
)

func TestNewTextHandler_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, nil)

	if handler == nil {
		t.Fatal("NewTextHandler returned nil")
	}
}

func TestNewTextHandler_WithLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	ctx := context.Background()

	// Debug should be disabled
	if handler.Enabled(ctx, slog.LevelDebug) {
		t.Error("Expected Debug to be disabled with Info level")
	}

	// Info should be enabled
	if !handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("Expected Info to be enabled")
	}
}

func TestTextHandler_Enabled(t *testing.T) {
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
			handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
				Level: tt.handlerLevel,
			})

			ctx := context.Background()
			result := handler.Enabled(ctx, tt.checkLevel)

			if result != tt.expected {
				t.Errorf("Enabled(%v) = %v, want %v", tt.checkLevel, result, tt.expected)
			}
		})
	}
}

func TestTextHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()

	// Check output contains timestamp
	if !strings.Contains(output, "2026-03-01T10:30:00.000Z") {
		t.Errorf("output should contain timestamp, got: %s", output)
	}

	// Check output contains level
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("output should contain [INFO], got: %s", output)
	}

	// Check output contains message
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain message, got: %s", output)
	}
}

func TestTextHandler_HandleWithFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("service", "billing-api"),
		slog.Int("count", 42),
	)

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()

	// Check fields are present
	if !strings.Contains(output, "service=billing-api") {
		t.Errorf("output should contain service field, got: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("output should contain count field, got: %s", output)
	}
}

func TestTextHandler_FieldFiltering(t *testing.T) {
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
			includeFields: []string{"service"},
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
			includeFields: []string{"service", "version"},
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
			handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
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

			if err := handler.Handle(ctx, record); err != nil {
				t.Fatalf("Handle failed: %v", err)
			}

			output := buf.String()

			// Check expected content
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output should contain %q, got: %s", s, output)
				}
			}

			// Check content that should not be present
			for _, s := range tt.notContains {
				if strings.Contains(output, s) {
					t.Errorf("output should not contain %q, got: %s", s, output)
				}
			}
		})
	}
}

func TestTextHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Add attributes
	newHandler := handler.WithAttrs([]slog.Attr{
		slog.String("service", "test-service"),
		slog.String("version", "1.0.0"),
	})

	if newHandler == nil {
		t.Fatal("WithAttrs returned nil")
	}

	// The new handler should be different from the original
	if newHandler == handler {
		t.Error("WithAttrs should return a new handler")
	}

	// Verify the new handler works
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	var newBuf bytes.Buffer
	textHandler := dlog.NewTextHandler(&newBuf, &dlog.HandlerOptions{Level: slog.LevelDebug})
	handlerWithAttrs := textHandler.WithAttrs([]slog.Attr{
		slog.String("pre_field", "pre_value"),
	})

	if err := handlerWithAttrs.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := newBuf.String()
	if !strings.Contains(output, "pre_field=pre_value") {
		t.Errorf("output should contain pre-populated field, got: %s", output)
	}
}

func TestTextHandler_WithAttrs_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// With empty attrs should return same handler
	newHandler := handler.WithAttrs([]slog.Attr{})

	if newHandler != handler {
		t.Error("WithAttrs with empty attrs should return same handler")
	}
}

func TestTextHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// WithGroup is a no-op for TextHandler, should return same handler
	newHandler := handler.WithGroup("test-group")

	if newHandler != handler {
		t.Error("WithGroup should return same handler (no-op for TextHandler)")
	}
}

func TestTextHandler_ImplementsSlogHandler(t *testing.T) {
	// Compile-time check that TextHandler implements slog.Handler
	var _ slog.Handler = dlog.NewTextHandler(nil, nil)
}

func TestTextHandler_Format(t *testing.T) {
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
			contains: []string{"[DEBUG]", "debug msg"},
		},
		{
			name:     "info level",
			level:    slog.LevelInfo,
			message:  "info msg",
			contains: []string{"[INFO]", "info msg"},
		},
		{
			name:     "warn level",
			level:    slog.LevelWarn,
			message:  "warn msg",
			contains: []string{"[WARN]", "warn msg"},
		},
		{
			name:     "error level",
			level:    slog.LevelError,
			message:  "error msg",
			contains: []string{"[ERROR]", "error msg"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewTextHandler(&buf, &dlog.HandlerOptions{
				Level: slog.LevelDebug,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, tt.level, tt.message, 0)
			if len(tt.attrs) > 0 {
				record.AddAttrs(tt.attrs...)
			}

			if err := handler.Handle(ctx, record); err != nil {
				t.Fatalf("Handle failed: %v", err)
			}

			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output should contain %q, got: %s", s, output)
				}
			}
		})
	}
}
