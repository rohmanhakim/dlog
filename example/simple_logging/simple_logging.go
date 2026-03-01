package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohmanhakim/dlog"
)

func main() {
	// FormatJSON - Standard slog.JSONHandler (nested groups)
	// Output: {"time":"...","level":"INFO","msg":"...","request":{"id":"..."}}
	jsonLogger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "test-output.json")
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		jsonLogger = dlog.NewNoOpLogger()
	}
	defer jsonLogger.Close()

	// FormatLogstash - Logstash-compatible format (flattened groups, renamed fields)
	// Output: {"@timestamp":"...","log.level":"INFO","message":"...","request.id":"..."}
	logstashLogger, err := dlog.NewSlogLogger(true, dlog.FormatLogstash, "test-output-logstash.json")
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logstashLogger = dlog.NewNoOpLogger()
	}
	defer logstashLogger.Close()

	// FormatText - Human-readable text format
	// With SyncBuffered for better performance (flushes only on Close())
	textLogger, err := dlog.NewSlogLogger(true, dlog.FormatText, "test-output.txt",
		dlog.WithSyncMode(dlog.SyncBuffered),
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		textLogger = dlog.NewNoOpLogger()
	}
	defer textLogger.Close()

	// FormatLogfmt - Logfmt format (key=value pairs)
	// With SyncPeriodic for balanced durability/performance (flushes every second by default)
	logfmtLogger, err := dlog.NewSlogLogger(true, dlog.FormatLogfmt, "test-output.logfmt",
		dlog.WithSyncMode(dlog.SyncPeriodic),
		dlog.WithSyncInterval(500*time.Millisecond), // custom flush interval
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logfmtLogger = dlog.NewNoOpLogger()
	}
	defer logfmtLogger.Close()

	ctx := context.Background()

	// Log with different formats
	jsonLogger.LogDebug(ctx, "JSON format - nested groups", dlog.FieldMap{"service": "api"})
	logstashLogger.LogInfo(ctx, "Logstash format - flattened groups", dlog.FieldMap{"service": "api"})
	textLogger.LogInfo(ctx, "Text format - human readable")
	textLogger.LogError(ctx, "Text format with error", NewSimpleError(ErrCauseUnknown, "unknown error occurred!"))
	logstashLogger.LogWarn(ctx, "Logstash format warning", dlog.FieldMap{"count": 42})
	logfmtLogger.LogInfo(ctx, "Logfmt format - key=value pairs")
}

// Example custom error
type ClassifiedError interface {
	error
	Severity() Severity
}

type Severity string

const (
	SeverityRecoverable Severity = "recoverable"
	SeverityFatal       Severity = "fatal"
)

type SimpleError struct {
	Message string
	Cause   SimpleErrorCause
}

type SimpleErrorCause string

const (
	ErrCauseUnknown SimpleErrorCause = "unknown"
)

func NewSimpleError(cause SimpleErrorCause, message string) *SimpleError {
	return &SimpleError{
		Message: message,
		Cause:   cause,
	}
}

func (e *SimpleError) Error() string {
	return fmt.Sprintf("SimpleError: %s", e.Cause)
}

func (e *SimpleError) Severity() Severity {
	if e.Cause == ErrCauseUnknown {
		return SeverityFatal
	} else {
		return SeverityRecoverable
	}
}
