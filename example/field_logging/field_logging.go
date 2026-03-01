package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rohmanhakim/dlog"
)

func main() {
	// Initialize debug logger based on config
	var logstashLogger dlog.DebugLogger = dlog.NewNoOpLogger()
	var textLogger dlog.DebugLogger = dlog.NewNoOpLogger()
	var logfmtLogger dlog.DebugLogger = dlog.NewNoOpLogger()

	// =============================================================================
	// Pattern 1: Functional Options (at initialization)
	// =============================================================================
	// Use functional options to set fields and group during logger creation.
	// This is cleaner for initial setup and base configuration.

	fieldMap := dlog.FieldMap{
		"service.name":    "billing-api",
		"service.version": "1.4.2",
		"component":       "scheduler",
		"operation":       "retry_job",
		"trace.id":        "abc123",
	}

	logstashLogger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "test-output.jsonl",
		dlog.WithFields(fieldMap),
		dlog.WithGroup("myservice"),
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logstashLogger = dlog.NewNoOpLogger()
	}

	textLogger, err = dlog.NewSlogLogger(true, dlog.FormatText, "test-output.txt",
		dlog.WithFields(fieldMap),
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		textLogger = dlog.NewNoOpLogger()
	}

	logfmtLogger, err = dlog.NewSlogLogger(true, dlog.FormatLogfmt, "test-output.logfmt",
		dlog.WithFields(fieldMap),
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logfmtLogger = dlog.NewNoOpLogger()
	}

	// =============================================================================
	// Pattern 2: Chaining (for derived loggers)
	// =============================================================================
	// Use method chaining to create specialized loggers from a base logger.
	// This is useful for creating component-specific loggers that share base config.

	// Create a base logger with common service fields
	baseLogger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "derived-output.jsonl",
		dlog.WithFields(dlog.FieldMap{
			"service.name":    "billing-api",
			"service.version": "1.4.2",
		}),
	)
	if err != nil {
		log.Printf("failed to create base logger: %v", err)
	} else {
		// Derive specialized loggers for different components
		schedulerLogger := baseLogger.WithGroup("scheduler").
			WithFields(dlog.FieldMap{"component": "job-retry"})

		dbLogger := baseLogger.WithGroup("database").
			WithFields(dlog.FieldMap{"component": "postgres"})

		// Use derived loggers
		ctx := context.Background()
		schedulerLogger.LogInfo(ctx, "Job retry started", dlog.FieldMap{"job_id": "job-123"})
		dbLogger.LogInfo(ctx, "Database connection established", dlog.FieldMap{"host": "localhost"})
		baseLogger.Close()
	}

	// =============================================================================
	// Log messages using the loggers created with functional options
	// =============================================================================
	ctx := context.Background()
	logstashLogger.LogDebug(ctx, "New JSON-formatted Message with Debug-level", dlog.FieldMap{"my_debug_key": "my_debug_val"})
	logstashLogger.LogWarn(ctx, "New JSON-formatted Message with Warn-level", dlog.FieldMap{"my_warn_key": "my_warn_val"})

	textLogger.LogInfo(ctx, "New Text-formatted Message with Info-level", dlog.FieldMap{"my_info_key": "my_info_val"})
	textLogger.LogError(ctx, "New Text-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"), dlog.FieldMap{"my_error_key": "my_error_val"})

	logfmtLogger.LogInfo(ctx, "New Logfmt-formatted Message with Info-level", dlog.FieldMap{"my_info_key": "my_info_val"})
	logfmtLogger.LogError(ctx, "New Logfmt-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"), dlog.FieldMap{"my_error_key": "my_error_val"})

	// Close loggers
	logstashLogger.Close()
	textLogger.Close()
	logfmtLogger.Close()
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
