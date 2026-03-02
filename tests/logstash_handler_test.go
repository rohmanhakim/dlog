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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to convert slog.Level to dlog.Level for tests
func slogToDlogLevelLogstash(level slog.Level) dlog.Level {
	return dlog.Level(level)
}

func TestNewLogstashHandler_NilOptions(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, nil)

	require.NotNil(t, handler, "NewLogstashHandler returned nil")
}

func TestNewLogstashHandler_WithLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelWarn,
	})

	ctx := context.Background()

	// Info should be disabled when handler at Warn
	assert.False(t, handler.Enabled(ctx, slog.LevelInfo), "Expected Info to be disabled with Warn level")

	// Warn should be enabled
	assert.True(t, handler.Enabled(ctx, slog.LevelWarn), "Expected Warn to be enabled")
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
				Level: slogToDlogLevelLogstash(tt.handlerLevel),
			})

			ctx := context.Background()
			result := handler.Enabled(ctx, tt.checkLevel)

			assert.Equal(t, tt.expected, result, "Enabled(%v)", tt.checkLevel)
		})
	}
}

func TestLogstashHandler_Handle_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	output := buf.String()

	// Verify it's valid JSON
	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	require.NoError(t, err, "output is not valid JSON")
}

func TestLogstashHandler_Handle_RequiredFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Check required Logstash fields
	assert.Contains(t, entry, "@timestamp", "missing @timestamp field")
	assert.Equal(t, "INFO", entry["log.level"], "level should be 'INFO'")
	assert.Equal(t, "test message", entry["message"], "message should be 'test message'")
}

func TestLogstashHandler_Handle_WithFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
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

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Check custom fields
	assert.Equal(t, "billing-api", entry["service"], "service should be 'billing-api'")
	assert.Equal(t, float64(42), entry["count"], "count should be 42") // JSON numbers are float64
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
				Level:         dlog.LevelDebug,
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

			var entry map[string]any
			err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
			require.NoError(t, err, "failed to parse JSON")

			// Check expected fields
			for key, value := range tt.expected {
				assert.Equal(t, value, entry[key], "entry[%q]", key)
			}

			// Check fields that should not be present
			for _, key := range tt.notExpected {
				assert.NotContains(t, entry, key, "entry should not contain %q", key)
			}
		})
	}
}

func TestLogstashHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	// Add attributes
	newHandler := handler.WithAttrs([]slog.Attr{
		slog.String("service", "test-service"),
		slog.String("version", "1.0.0"),
	})

	require.NotNil(t, newHandler, "WithAttrs returned nil")

	// The new handler should be different from the original
	assert.NotEqual(t, handler, newHandler, "WithAttrs should return a new handler")

	// Verify the new handler works with pre-populated fields
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	var newBuf bytes.Buffer
	logstashHandler := dlog.NewLogstashHandler(&newBuf, &dlog.HandlerOptions{Level: dlog.LevelDebug})
	handlerWithAttrs := logstashHandler.WithAttrs([]slog.Attr{
		slog.String("pre_field", "pre_value"),
	})

	err := handlerWithAttrs.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(newBuf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	assert.Equal(t, "pre_value", entry["pre_field"], "entry should contain pre_field=pre_value")
}

func TestLogstashHandler_WithAttrs_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	// With empty attrs should return same handler
	newHandler := handler.WithAttrs([]slog.Attr{})

	assert.Same(t, handler, newHandler, "WithAttrs with empty attrs should return same handler")
}

func TestLogstashHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	// WithGroup should return a new handler with group prefix
	newHandler := handler.WithGroup("request")

	require.NotNil(t, newHandler, "WithGroup returned nil")

	// The new handler should be different from the original
	assert.NotEqual(t, handler, newHandler, "WithGroup should return a new handler")
}

func TestLogstashHandler_WithGroup_Empty(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	// WithGroup with empty name should return same handler
	newHandler := handler.WithGroup("")

	assert.Same(t, handler, newHandler, "WithGroup with empty name should return same handler")
}

func TestLogstashHandler_WithGroup_PrefixFields(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	handlerWithGroup := handler.WithGroup("request")
	handlerWithGroupAndAttrs := handlerWithGroup.WithAttrs([]slog.Attr{
		slog.String("id", "abc123"),
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	err := handlerWithGroupAndAttrs.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Field should be prefixed with group name
	assert.Equal(t, "abc123", entry["request.id"], "entry should contain request.id=abc123")
}

func TestLogstashHandler_ImplementsSlogHandler(t *testing.T) {
	// Compile-time check that LogstashHandler implements slog.Handler
	var _ slog.Handler = dlog.NewLogstashHandler(nil, nil)
}

func TestLogstashHandler_FlattenAttrs_GroupKind(t *testing.T) {
	// Test flattenAndFilterAttrs when attr.Value.Kind() == slog.KindGroup
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	// Add group attributes - tests flattenAndFilterAttrs with slog.KindGroup
	record.AddAttrs(
		slog.Group("request",
			slog.String("id", "abc123"),
			slog.String("method", "GET"),
			slog.Group("headers",
				slog.String("content-type", "application/json"),
				slog.String("authorization", "Bearer token"),
			),
		),
	)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Check that groups are flattened with dot notation
	assert.Equal(t, "abc123", entry["request.id"], "request.id should be flattened")
	assert.Equal(t, "GET", entry["request.method"], "request.method should be flattened")
	assert.Equal(t, "application/json", entry["request.headers.content-type"], "nested group should be flattened")
	assert.Equal(t, "Bearer token", entry["request.headers.authorization"], "nested group should be flattened")
}

// testLogValuer is a custom type that implements slog.LogValuer
type testLogValuer struct {
	value string
}

// LogValue implements slog.LogValuer interface
func (t testLogValuer) LogValue() slog.Value {
	return slog.StringValue("resolved_" + t.value)
}

func TestLogstashHandler_FlattenAttrs_LogValuerKind(t *testing.T) {
	// Test flattenAndFilterAttrs when attr.Value.Kind() == slog.KindLogValuer
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
	})

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

	// Add an attribute that implements slog.LogValuer
	record.AddAttrs(
		slog.Any("custom_value", testLogValuer{value: "test"}),
	)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Check that LogValuer was resolved
	assert.Equal(t, "resolved_test", entry["custom_value"], "LogValuer should be resolved")
}

func TestLogstashHandler_ShouldIncludeField_BuiltinFields(t *testing.T) {
	// Test shouldIncludeField returns true for builtin fields even when not in include list
	tests := []struct {
		name          string
		includeFields []string
		builtinField  string
	}{
		{
			name:          "time field is always included",
			includeFields: []string{"custom_field"}, // only include custom field
			builtinField:  "time",
		},
		{
			name:          "level field is always included",
			includeFields: []string{"custom_field"},
			builtinField:  "level",
		},
		{
			name:          "msg field is always included",
			includeFields: []string{"custom_field"},
			builtinField:  "msg",
		},
		{
			name:          "@timestamp field is always included",
			includeFields: []string{"custom_field"},
			builtinField:  "@timestamp",
		},
		{
			name:          "log.level field is always included",
			includeFields: []string{"custom_field"},
			builtinField:  "log.level",
		},
		{
			name:          "message field is always included",
			includeFields: []string{"custom_field"},
			builtinField:  "message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
				Level:         dlog.LevelDebug,
				IncludeFields: tt.includeFields,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)

			err := handler.Handle(ctx, record)
			require.NoError(t, err, "Handle failed")

			var entry map[string]any
			err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
			require.NoError(t, err, "failed to parse JSON")

			// Builtin fields should still be present
			if tt.builtinField == "time" || tt.builtinField == "@timestamp" {
				assert.Contains(t, entry, "@timestamp", "@timestamp should always be included")
			} else if tt.builtinField == "level" || tt.builtinField == "log.level" {
				assert.Contains(t, entry, "log.level", "log.level should always be included")
			} else if tt.builtinField == "msg" || tt.builtinField == "message" {
				assert.Contains(t, entry, "message", "message should always be included")
			}
		})
	}
}

func TestLogstashHandler_ShouldIncludeField_NilFieldFilter(t *testing.T) {
	// Test shouldIncludeField returns true when fieldFilter == nil
	// This happens when HandlerOptions is nil
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, nil)

	ctx := context.Background()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	record := slog.NewRecord(now, slog.LevelInfo, "test message", 0)
	record.AddAttrs(
		slog.String("field1", "value1"),
		slog.String("field2", "value2"),
		slog.String("field3", "value3"),
	)

	err := handler.Handle(ctx, record)
	require.NoError(t, err, "Handle failed")

	var entry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
	require.NoError(t, err, "failed to parse JSON")

	// All fields should be present when fieldFilter is nil
	assert.Equal(t, "value1", entry["field1"])
	assert.Equal(t, "value2", entry["field2"])
	assert.Equal(t, "value3", entry["field3"])
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
				Level: dlog.LevelDebug,
			})

			ctx := context.Background()
			now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
			record := slog.NewRecord(now, tt.level, "test message", 0)

			err := handler.Handle(ctx, record)
			require.NoError(t, err, "Handle failed")

			var entry map[string]any
			err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry)
			require.NoError(t, err, "failed to parse JSON")

			assert.Equal(t, tt.expectedLevel, entry["log.level"], "log.level")
		})
	}
}

func TestLogstashHandler_Integration(t *testing.T) {
	var buf bytes.Buffer
	handler := dlog.NewLogstashHandler(&buf, &dlog.HandlerOptions{
		Level: dlog.LevelDebug,
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
		err := handler.Handle(ctx, record)
		require.NoError(t, err, "Handle failed")
	}

	// Each record should be on its own line (JSONL format)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Equal(t, 4, len(lines), "expected 4 lines")

	// Each line should be valid JSON
	for i, line := range lines {
		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err, "line %d: failed to parse JSON", i+1)
	}
}
