package dlog_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlogLogger_DisabledReturnsNoOp(t *testing.T) {
	logger, err := dlog.NewSlogLogger(false, dlog.FormatJSON, "")
	require.NoError(t, err, "NewSlogLogger failed")

	// Should return NoOpLogger when disabled
	assert.False(t, logger.Enabled(), "Expected NoOpLogger when Enabled=false, but Enabled() returned true")
}

func TestNewSlogLogger_EnabledReturnsSlogLogger(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	assert.True(t, logger.Enabled(), "Expected Enabled() to return true for SlogLogger")
}

func TestNewSlogLogger_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "slog-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	// Verify file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err, "output file was not created: %s", outputFile)
}

func TestNewSlogLogger_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format dlog.Format
	}{
		{
			name:   "json format",
			format: dlog.FormatJSON,
		},
		{
			name:   "logstash format",
			format: dlog.FormatLogstash,
		},
		{
			name:   "text format",
			format: dlog.FormatText,
		},
		{
			name:   "logfmt format",
			format: dlog.FormatLogfmt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := dlog.NewSlogLogger(true, tt.format, "")
			require.NoError(t, err, "NewSlogLogger failed")
			defer logger.Close()

			assert.True(t, logger.Enabled(), "Expected Enabled() to return true")
		})
	}
}

func TestSlogLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(dlog.DebugLogger, context.Context, string, dlog.FieldMap)
		level   string
	}{
		{
			name: "LogDebug",
			logFunc: func(l dlog.DebugLogger, ctx context.Context, msg string, fm dlog.FieldMap) {
				l.LogDebug(ctx, msg, fm)
			},
			level: "DEBUG",
		},
		{
			name: "LogInfo",
			logFunc: func(l dlog.DebugLogger, ctx context.Context, msg string, fm dlog.FieldMap) {
				l.LogInfo(ctx, msg, fm)
			},
			level: "INFO",
		},
		{
			name: "LogWarn",
			logFunc: func(l dlog.DebugLogger, ctx context.Context, msg string, fm dlog.FieldMap) {
				l.LogWarn(ctx, msg, fm)
			},
			level: "WARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := dlog.NewSlogLogger(true, dlog.FormatText, "")
			require.NoError(t, err, "NewSlogLogger failed")
			defer logger.Close()

			ctx := context.Background()
			tt.logFunc(logger, ctx, "test message", dlog.FieldMap{"key": "value"})

			// Note: The actual output goes to stdout, so we can't easily capture it here
			// This test verifies the methods don't panic and work correctly
		})
	}
}

func TestSlogLogger_LogError(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	ctx := context.Background()
	testErr := &testError{msg: "test error"}

	logger.LogError(ctx, "error stage", testErr, dlog.FieldMap{"key": "value"})
	// Verify no panic and method works
}

func TestSlogLogger_LogError_NilError(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "nil-error-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	ctx := context.Background()

	// This should not panic when err is nil
	logger.LogError(ctx, "test message", nil, dlog.FieldMap{"key": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify message is logged correctly (not hardcoded "Error occurred")
	assert.Equal(t, "test message", entry["message"], "expected message='test message'")

	// Verify error field is not present when err is nil
	assert.NotContains(t, entry, "error", "expected no error field when err is nil")

	// Verify other fields are present
	assert.Equal(t, "value", entry["key"], "expected key=value")
}

func TestSlogLogger_WithFields(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	fields := dlog.FieldMap{
		"service":  "test-service",
		"version":  "1.0.0",
		"trace_id": "abc123",
	}

	newLogger := logger.WithFields(fields)

	require.NotNil(t, newLogger, "WithFields returned nil")
	assert.True(t, newLogger.Enabled(), "WithFields logger should have Enabled() = true")

	// Both loggers should be independent
	newLogger2 := newLogger.WithFields(dlog.FieldMap{"extra": "field"})
	require.NotNil(t, newLogger2, "second WithFields returned nil")
}

func TestSlogLogger_WithFieldsPreservesFields(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withfields-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")

	// Add pre-populated fields
	loggerWithFields := logger.WithFields(dlog.FieldMap{
		"service": "billing-api",
	})

	ctx := context.Background()
	loggerWithFields.LogInfo(ctx, "test message", dlog.FieldMap{"extra": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify pre-populated field is present
	assert.Equal(t, "billing-api", entry["service"], "expected service=billing-api")

	// Verify additional field is present
	assert.Equal(t, "value", entry["extra"], "expected extra=value")
}

func TestSlogLogger_WithGroup(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	loggerWithGroup := logger.WithGroup("myservice")

	require.NotNil(t, loggerWithGroup, "WithGroup returned nil")
	assert.True(t, loggerWithGroup.Enabled(), "WithGroup logger should have Enabled() = true")
}

func TestSlogLogger_WithGroupPreservesFields(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroup-test.jsonl")

	// Use FormatLogstash for flattened group keys
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")

	// Add pre-populated fields and group
	loggerWithFields := logger.WithFields(dlog.FieldMap{
		"service": "billing-api",
	})
	loggerWithGroup := loggerWithFields.WithGroup("request")

	ctx := context.Background()
	loggerWithGroup.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Note: preAttrs are also affected by WithGroup since they're logged through the grouped logger
	// Verify pre-populated field is prefixed with group name
	assert.Equal(t, "billing-api", entry["request.service"], "expected request.service=billing-api")

	// Verify grouped field is prefixed with group name
	assert.Equal(t, "abc123", entry["request.id"], "expected request.id=abc123")
}

func TestSlogLogger_WithGroupChained(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroupchained-test.jsonl")

	// Use FormatLogstash for flattened group keys
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")

	// Chain WithFields and WithGroup
	logger = logger.WithFields(dlog.FieldMap{"app": "myapp"}).
		WithGroup("myservice").
		WithFields(dlog.FieldMap{"component": "scheduler"})

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"operation": "retry"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Note: preAttrs are also affected by WithGroup since they're logged through the grouped logger
	// Verify all fields are prefixed with group name
	assert.Equal(t, "myapp", entry["myservice.app"], "expected myservice.app=myapp")
	assert.Equal(t, "scheduler", entry["myservice.component"], "expected myservice.component=scheduler")
	assert.Equal(t, "retry", entry["myservice.operation"], "expected myservice.operation=retry")
}

func TestSlogLogger_Close(t *testing.T) {
	t.Run("close with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "close-test.jsonl")

		logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
		require.NoError(t, err, "NewSlogLogger failed")

		err = logger.Close()
		assert.NoError(t, err, "Close failed")
	})

	t.Run("close without file", func(t *testing.T) {
		logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
		require.NoError(t, err, "NewSlogLogger failed")

		err = logger.Close()
		assert.NoError(t, err, "Close failed")
	})
}

func TestSlogLogger_Integration_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-json.jsonl")

	// Use FormatLogstash for @timestamp, log.level, message fields
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()

	// Log multiple messages
	logger.LogDebug(ctx, "debug message", dlog.FieldMap{"key": "debug"})
	logger.LogInfo(ctx, "info message", dlog.FieldMap{"key": "info"})
	logger.LogWarn(ctx, "warn message", dlog.FieldMap{"key": "warn"})
	logger.LogError(ctx, "error stage", &testError{msg: "test error"}, dlog.FieldMap{"key": "error"})

	logger.Close()

	// Verify output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Equal(t, 4, len(lines), "expected 4 log lines")

	// Verify each line is valid JSON with expected fields
	for i, line := range lines {
		var entry map[string]any
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err, "line %d: failed to parse JSON", i+1)

		// Check required fields
		assert.Contains(t, entry, "@timestamp", "line %d: missing @timestamp", i+1)
		assert.Contains(t, entry, "log.level", "line %d: missing level", i+1)
		assert.Contains(t, entry, "message", "line %d: missing message", i+1)
	}
}

func TestSlogLogger_Integration_TextFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-text.txt")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatText, outputFile)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"key": "value"})

	logger.Close()

	// Verify output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	output := string(content)

	// Check for expected patterns
	assert.Contains(t, output, "[INFO]", "output should contain [INFO]")
	assert.Contains(t, output, "test message", "output should contain 'test message'")
	assert.Contains(t, output, "key=value", "output should contain 'key=value'")
}

func TestSlogLogger_WithFieldsOption(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withfields-option-test.jsonl")

	// Create logger with WithFields functional option
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile,
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
			"version": "1.0.0",
		}),
	)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"extra": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify pre-populated fields are present
	assert.Equal(t, "billing-api", entry["service"], "expected service=billing-api")
	assert.Equal(t, "1.0.0", entry["version"], "expected version=1.0.0")

	// Verify additional field is present
	assert.Equal(t, "value", entry["extra"], "expected extra=value")
}

func TestSlogLogger_WithGroupOption(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroup-option-test.jsonl")

	// Use FormatLogstash for flattened group keys
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile,
		dlog.WithGroup("myservice"),
	)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify grouped field is prefixed with group name
	assert.Equal(t, "abc123", entry["myservice.id"], "expected myservice.id=abc123")
}

func TestSlogLogger_WithFieldsAndGroupOptions(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withfields-group-option-test.jsonl")

	// Use FormatLogstash for flattened group keys
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile,
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
		}),
		dlog.WithGroup("myservice"),
	)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify pre-populated field is prefixed with group name
	assert.Equal(t, "billing-api", entry["myservice.service"], "expected myservice.service=billing-api")

	// Verify additional field is prefixed with group name
	assert.Equal(t, "abc123", entry["myservice.id"], "expected myservice.id=abc123")
}

func TestSlogLogger_FunctionalOptionsCombinedWithChaining(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "combined-test.jsonl")

	// Use FormatLogstash for flattened group keys
	logger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, outputFile,
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
		}),
		dlog.WithGroup("myservice"),
	)
	require.NoError(t, err, "NewSlogLogger failed")

	// Chain additional WithFields on the created logger
	derivedLogger := logger.WithFields(dlog.FieldMap{"component": "scheduler"})

	ctx := context.Background()
	derivedLogger.LogInfo(ctx, "test message", dlog.FieldMap{"operation": "retry"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// Verify all fields are prefixed with group name
	assert.Equal(t, "billing-api", entry["myservice.service"], "expected myservice.service=billing-api")
	assert.Equal(t, "scheduler", entry["myservice.component"], "expected myservice.component=scheduler")
	assert.Equal(t, "retry", entry["myservice.operation"], "expected myservice.operation=retry")
}

func TestSlogLogger_JSONFormat_NestedGroups(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "json-nested-test.jsonl")

	// FormatJSON produces nested structures (standard slog.JSONHandler behavior)
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile,
		dlog.WithGroup("myservice"),
	)
	require.NoError(t, err, "NewSlogLogger failed")

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	var entry map[string]any
	err = json.Unmarshal(content, &entry)
	require.NoError(t, err, "failed to parse JSON")

	// FormatJSON uses nested objects, not flattened keys
	myservice, ok := entry["myservice"].(map[string]any)
	require.True(t, ok, "expected myservice to be a nested object")

	assert.Equal(t, "abc123", myservice["id"], "expected myservice.id=abc123")
}

func TestSlogLogger_ImplementsDebugLogger(t *testing.T) {
	// Compile-time check that SlogLogger implements DebugLogger
	var _ dlog.DebugLogger = &dlog.SlogLogger{}

	// Also verify through NewSlogLogger
	logger, _ := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	defer logger.Close()
	var _ dlog.DebugLogger = logger
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
