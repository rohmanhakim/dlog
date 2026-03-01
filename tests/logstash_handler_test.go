package dlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rohmanhakim/dlog"
)

func TestNewLogstashHandler_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, nil)

	if handler == nil {
		t.Fatal("NewLogstashHandler returned nil")
	}
}

func TestNewLogstashHandler_WithLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	ctx := context.Background()

	// Info should be disabled when handler at Warn
	if handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("Expected Info to be disabled with Warn level")
	}

	// Warn should be enabled
	if !handler.Enabled(ctx, slog.LevelWarn) {
		t.Error("Expected Warn to be enabled")
	}
}

func TestLogstashHandler_Enabled(t *testing.T) {
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
			name:         "warn disabled when handler at error",
			handlerLevel: slog.LevelError,
			checkLevel:   slog.LevelWarn,
			expected:     false,
		},
		{
			name:         "error enabled when handler at warn",
			handlerLevel: slog.LevelWarn,
			checkLevel:   slog.LevelError,
			expected:     true,
		},
		{
			name:         "info disabled when handler at warn",
			handlerLevel: slog.LevelWarn,
			checkLevel:   slog.LevelInfo,
			expected:     false,
		},
		{
			name:         "warn enabled when handler at warn",
			handlerLevel: slog.LevelWarn,
			checkLevel:   slog.LevelWarn,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
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

func TestLogstashHandler_Handle_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
}

func TestLogstashHandler_Handle_RequiredFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	if err := handler.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check required Logstash fields
	if _, ok := entry["@timestamp"]; !ok {
		t.Error("missing @timestamp field")
	}
	if entry["@version"] != "1" {
		t.Errorf("@version should be '1', got: %v", entry["@version"])
	}
	if entry["level"] != "INFO" {
		t.Errorf("level should be 'INFO', got: %v", entry["level"])
	}
	if entry["message"] != "test message" {
		t.Errorf("message should be 'test message', got: %v", entry["message"])
	}
	if entry["thread_name"] != "main" {
		t.Errorf("thread_name should be 'main', got: %v", entry["thread_name"])
	}
}

func TestLogstashHandler_Handle_WithFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
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

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check custom fields
	if entry["service"] != "billing-api" {
		t.Errorf("service should be 'billing-api', got: %v", entry["service"])
	}
	if entry["count"] != float64(42) { // JSON numbers are float64
		t.Errorf("count should be 42, got: %v", entry["count"])
	}
}

func TestLogstashHandler_FieldFiltering(t *testing.T) {
	tests := []struct {
		name          string
		includeFields []string
		excludeFields []string
		attrs         []slog.Attr
		expected      map[string]any
		notExpected   []string
	}{
		{
			name:          "no filtering",
			includeFields: nil,
			excludeFields: nil,
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			expected: map[string]any{
				"service": "api",
				"version": "1.0",
			},
			notExpected: nil,
		},
		{
			name:          "include fields only",
			includeFields: []string{"@timestamp", "@version", "level", "message", "thread_name", "service"},
			excludeFields: nil,
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			expected: map[string]any{
				"service": "api",
			},
			notExpected: []string{"version"},
		},
		{
			name:          "exclude fields",
			includeFields: nil,
			excludeFields: []string{"version"},
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			expected: map[string]any{
				"service": "api",
			},
			notExpected: []string{"version"},
		},
		{
			name:          "include and exclude combined",
			includeFields: []string{"@timestamp", "@version", "level", "message", "thread_name", "service", "version"},
			excludeFields: []string{"version"},
			attrs: []slog.Attr{
				slog.String("service", "api"),
				slog.String("version", "1.0"),
			},
			expected: map[string]any{
				"service": "api",
			},
			notExpected: []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
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

			var entry map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			// Check expected fields
			for key, value := range tt.expected {
				if entry[key] != value {
					t.Errorf("entry[%q] = %v, want %v", key, entry[key], value)
				}
			}

			// Check fields that should not be present
			for _, key := range tt.notExpected {
				if _, ok := entry[key]; ok {
					t.Errorf("entry should not contain %q", key)
				}
			}
		})
	}
}

func TestLogstashHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
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

	// Verify the new handler works with pre-populated fields
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	var newBuf bytes.Buffer
	logstashHandler := dlog.NewLogstashHandler(&newBuf, &dlog.HandlerOptions{Level: slog.LevelDebug})
	handlerWithAttrs := logstashHandler.WithAttrs([]slog.Attr{
		slog.String("pre_field", "pre_value"),
	})

	if err := handlerWithAttrs.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(newBuf.String())), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["pre_field"] != "pre_value" {
		t.Errorf("entry should contain pre_field=pre_value, got: %v", entry["pre_field"])
	}
}

func TestLogstashHandler_WithAttrs_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// With empty attrs should return same handler
	newHandler := handler.WithAttrs([]slog.Attr{})

	if newHandler != handler {
		t.Error("WithAttrs with empty attrs should return same handler")
	}
}

func TestLogstashHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// WithGroup should return a new handler with group prefix
	newHandler := handler.WithGroup("request")

	if newHandler == nil {
		t.Fatal("WithGroup returned nil")
	}

	// The new handler should be different from the original
	if newHandler == handler {
		t.Error("WithGroup should return a new handler")
	}
}

func TestLogstashHandler_WithGroup_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// WithGroup with empty name should return same handler
	newHandler := handler.WithGroup("")

	if newHandler != handler {
		t.Error("WithGroup with empty name should return same handler")
	}
}

func TestLogstashHandler_WithGroup_PrefixFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	handlerWithGroup := handler.WithGroup("request")
	handlerWithGroupAndAttrs := handlerWithGroup.WithAttrs([]slog.Attr{
		slog.String("id", "abc123"),
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	if err := handlerWithGroupAndAttrs.Handle(ctx, record); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Field should be prefixed with group name
	if entry["request.id"] != "abc123" {
		t.Errorf("entry should contain request.id=abc123, got: %v", entry["request.id"])
	}
}

func TestLogstashHandler_ImplementsSlogHandler(t *testing.T) {
	// Compile-time check that LogstashHandler implements slog.Handler
	var _ slog.Handler = dlog.NewLogstashHandler(nil, nil)
}

func TestLogstashHandler_LevelNames(t *testing.T) {
	tests := []struct {
		name          string
		level         slog.Level
		expectedLevel string
	}{
		{
			name:          "debug level",
			level:         slog.LevelDebug,
			expectedLevel: "DEBUG",
		},
		{
			name:          "info level",
			level:         slog.LevelInfo,
			expectedLevel: "INFO",
		},
		{
			name:          "warn level",
			level:         slog.LevelWarn,
			expectedLevel: "WARN",
		},
		{
			name:          "error level",
			level:         slog.LevelError,
			expectedLevel: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
				Level: slog.LevelDebug,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, tt.level, "test message", 0)

			if err := handler.Handle(ctx, record); err != nil {
				t.Fatalf("Handle failed: %v", err)
			}

			var entry map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if entry["level"] != tt.expectedLevel {
				t.Errorf("level = %v, want %v", entry["level"], tt.expectedLevel)
			}
		})
	}
}

func TestLogstashHandler_Integration(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)

	// Log multiple records
	records := []slog.Record{
		slog.NewRecord(now, slog.LevelDebug, "debug message", 0),
		slog.NewRecord(now, slog.LevelInfo, "info message", 0),
		slog.NewRecord(now, slog.LevelWarn, "warn message", 0),
		slog.NewRecord(now, slog.LevelError, "error message", 0),
	}

	for _, record := range records {
		if err := handler.Handle(ctx, record); err != nil {
			t.Fatalf("Handle failed: %v", err)
		}
	}

	// Each record should be on its own line (JSONL format)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// Each line should be valid JSON
	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d: failed to parse JSON: %v", i+1, err)
		}
	}
}
