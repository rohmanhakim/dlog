package dlog_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohmanhakim/dlog"
)

func TestNewSlogLogger_DisabledReturnsNoOp(t *testing.T) {
	logger, err := dlog.NewSlogLogger(false, dlog.FormatJSON, "")
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	// Should return NoOpLogger when disabled
	if logger.Enabled() {
		t.Error("Expected NoOpLogger when Enabled=false, but Enabled() returned true")
	}
}

func TestNewSlogLogger_EnabledReturnsSlogLogger(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	if !logger.Enabled() {
		t.Error("Expected Enabled() to return true for SlogLogger")
	}
}

func TestNewSlogLogger_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "slog-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("output file was not created: %s", outputFile)
	}
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
			name:   "text format",
			format: dlog.FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := dlog.NewSlogLogger(true, tt.format, "")
			if err != nil {
				t.Fatalf("NewSlogLogger failed: %v", err)
			}
			defer logger.Close()

			if !logger.Enabled() {
				t.Error("Expected Enabled() to return true")
			}
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
			if err != nil {
				t.Fatalf("NewSlogLogger failed: %v", err)
			}
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
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()
	testErr := &testError{msg: "test error"}

	logger.LogError(ctx, "error stage", testErr, dlog.FieldMap{"key": "value"})
	// Verify no panic and method works
}

func TestSlogLogger_LogError_NilError(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "nil-error-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	// This should not panic when err is nil
	logger.LogError(ctx, "test message", nil, dlog.FieldMap{"key": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify message is logged correctly (not hardcoded "Error occurred")
	if entry["message"] != "test message" {
		t.Errorf("expected message='test message', got %v", entry["message"])
	}

	// Verify error field is not present when err is nil
	if _, ok := entry["error"]; ok {
		t.Errorf("expected no error field when err is nil, but got: %v", entry["error"])
	}

	// Verify other fields are present
	if entry["key"] != "value" {
		t.Errorf("expected key=value, got %v", entry["key"])
	}
}

func TestSlogLogger_WithFields(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	fields := dlog.FieldMap{
		"service":  "test-service",
		"version":  "1.0.0",
		"trace_id": "abc123",
	}

	newLogger := logger.WithFields(fields)

	if newLogger == nil {
		t.Fatal("WithFields returned nil")
	}

	if !newLogger.Enabled() {
		t.Error("WithFields logger should have Enabled() = true")
	}

	// Both loggers should be independent
	newLogger2 := newLogger.WithFields(dlog.FieldMap{"extra": "field"})
	if newLogger2 == nil {
		t.Fatal("second WithFields returned nil")
	}
}

func TestSlogLogger_WithFieldsPreservesFields(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withfields-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	// Add pre-populated fields
	loggerWithFields := logger.WithFields(dlog.FieldMap{
		"service": "billing-api",
	})

	ctx := context.Background()
	loggerWithFields.LogInfo(ctx, "test message", dlog.FieldMap{"extra": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify pre-populated field is present
	if entry["service"] != "billing-api" {
		t.Errorf("expected service=billing-api, got %v", entry["service"])
	}

	// Verify additional field is present
	if entry["extra"] != "value" {
		t.Errorf("expected extra=value, got %v", entry["extra"])
	}
}

func TestSlogLogger_WithGroup(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	loggerWithGroup := logger.WithGroup("myservice")

	if loggerWithGroup == nil {
		t.Fatal("WithGroup returned nil")
	}

	if !loggerWithGroup.Enabled() {
		t.Error("WithGroup logger should have Enabled() = true")
	}
}

func TestSlogLogger_WithGroupPreservesFields(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroup-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

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
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Note: preAttrs are also affected by WithGroup since they're logged through the grouped logger
	// Verify pre-populated field is prefixed with group name
	if entry["request.service"] != "billing-api" {
		t.Errorf("expected request.service=billing-api, got %v", entry["request.service"])
	}

	// Verify grouped field is prefixed with group name
	if entry["request.id"] != "abc123" {
		t.Errorf("expected request.id=abc123, got %v", entry["request.id"])
	}
}

func TestSlogLogger_WithGroupChained(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroupchained-test.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	// Chain WithFields and WithGroup
	logger = logger.WithFields(dlog.FieldMap{"app": "myapp"}).
		WithGroup("myservice").
		WithFields(dlog.FieldMap{"component": "scheduler"})

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"operation": "retry"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Note: preAttrs are also affected by WithGroup since they're logged through the grouped logger
	// Verify all fields are prefixed with group name
	if entry["myservice.app"] != "myapp" {
		t.Errorf("expected myservice.app=myapp, got %v", entry["myservice.app"])
	}
	if entry["myservice.component"] != "scheduler" {
		t.Errorf("expected myservice.component=scheduler, got %v", entry["myservice.component"])
	}
	if entry["myservice.operation"] != "retry" {
		t.Errorf("expected myservice.operation=retry, got %v", entry["myservice.operation"])
	}
}

func TestSlogLogger_Close(t *testing.T) {
	t.Run("close with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "close-test.jsonl")

		logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
		if err != nil {
			t.Fatalf("NewSlogLogger failed: %v", err)
		}

		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("close without file", func(t *testing.T) {
		logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "")
		if err != nil {
			t.Fatalf("NewSlogLogger failed: %v", err)
		}

		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})
}

func TestSlogLogger_Integration_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-json.jsonl")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	ctx := context.Background()

	// Log multiple messages
	logger.LogDebug(ctx, "debug message", dlog.FieldMap{"key": "debug"})
	logger.LogInfo(ctx, "info message", dlog.FieldMap{"key": "info"})
	logger.LogWarn(ctx, "warn message", dlog.FieldMap{"key": "warn"})
	logger.LogError(ctx, "error stage", &testError{msg: "test error"}, dlog.FieldMap{"key": "error"})

	logger.Close()

	// Verify output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 log lines, got %d", len(lines))
	}

	// Verify each line is valid JSON with expected fields
	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d: failed to parse JSON: %v", i+1, err)
			continue
		}

		// Check required fields
		if _, ok := entry["@timestamp"]; !ok {
			t.Errorf("line %d: missing @timestamp", i+1)
		}
		if _, ok := entry["log.level"]; !ok {
			t.Errorf("line %d: missing level", i+1)
		}
		if _, ok := entry["message"]; !ok {
			t.Errorf("line %d: missing message", i+1)
		}
	}
}

func TestSlogLogger_Integration_TextFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-text.txt")

	logger, err := dlog.NewSlogLogger(true, dlog.FormatText, outputFile)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"key": "value"})

	logger.Close()

	// Verify output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	output := string(content)

	// Check for expected patterns
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("output should contain [INFO], got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("output should contain 'key=value', got: %s", output)
	}
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
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"extra": "value"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify pre-populated fields are present
	if entry["service"] != "billing-api" {
		t.Errorf("expected service=billing-api, got %v", entry["service"])
	}
	if entry["version"] != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %v", entry["version"])
	}

	// Verify additional field is present
	if entry["extra"] != "value" {
		t.Errorf("expected extra=value, got %v", entry["extra"])
	}
}

func TestSlogLogger_WithGroupOption(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withgroup-option-test.jsonl")

	// Create logger with WithGroup functional option
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile,
		dlog.WithGroup("myservice"),
	)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify grouped field is prefixed with group name
	if entry["myservice.id"] != "abc123" {
		t.Errorf("expected myservice.id=abc123, got %v", entry["myservice.id"])
	}
}

func TestSlogLogger_WithFieldsAndGroupOptions(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "withfields-group-option-test.jsonl")

	// Create logger with both WithFields and WithGroup functional options
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile,
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
		}),
		dlog.WithGroup("myservice"),
	)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	ctx := context.Background()
	logger.LogInfo(ctx, "test message", dlog.FieldMap{"id": "abc123"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify pre-populated field is prefixed with group name
	if entry["myservice.service"] != "billing-api" {
		t.Errorf("expected myservice.service=billing-api, got %v", entry["myservice.service"])
	}

	// Verify additional field is prefixed with group name
	if entry["myservice.id"] != "abc123" {
		t.Errorf("expected myservice.id=abc123, got %v", entry["myservice.id"])
	}
}

func TestSlogLogger_FunctionalOptionsCombinedWithChaining(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "combined-test.jsonl")

	// Create logger with functional options, then chain method calls
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, outputFile,
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
		}),
		dlog.WithGroup("myservice"),
	)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	// Chain additional WithFields on the created logger
	derivedLogger := logger.WithFields(dlog.FieldMap{"component": "scheduler"})

	ctx := context.Background()
	derivedLogger.LogInfo(ctx, "test message", dlog.FieldMap{"operation": "retry"})

	logger.Close()

	// Read and parse the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify all fields are prefixed with group name
	if entry["myservice.service"] != "billing-api" {
		t.Errorf("expected myservice.service=billing-api, got %v", entry["myservice.service"])
	}
	if entry["myservice.component"] != "scheduler" {
		t.Errorf("expected myservice.component=scheduler, got %v", entry["myservice.component"])
	}
	if entry["myservice.operation"] != "retry" {
		t.Errorf("expected myservice.operation=retry, got %v", entry["myservice.operation"])
	}
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
