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

	logstashConfig, err := dlog.NewDebugConfig(true, "MyJSONLogger", "test-output.jsonl", dlog.FormatJSON)
	if err != nil {
		log.Printf("failed to create debug config: %v, using NoOpLogger", err)
	} else {
		fieldMap := dlog.FieldMap{
			"service.name":    "billing-api",
			"service.version": "1.4.2",
			"component":       "scheduler",
			"operation":       "retry_job",
			"trace.id":        "abc123",
		}
		logstashLogger, err = dlog.NewSlogLogger(logstashConfig)
		logstashLogger = logstashLogger.WithFields(fieldMap)
		logstashLogger = logstashLogger.WithGroup("myservice")
		if err != nil {
			log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
			logstashLogger = dlog.NewNoOpLogger()
		}
	}

	textConfig, err := dlog.NewDebugConfig(true, "MyTextLogger", "test-output.txt", dlog.FormatText)
	if err != nil {
		log.Printf("failed to create debug config: %v, using NoOpLogger", err)
	} else {
		textLogger, err = dlog.NewSlogLogger(textConfig)
		fieldMap := dlog.FieldMap{
			"service.name":    "billing-api",
			"service.version": "1.4.2",
			"component":       "scheduler",
			"operation":       "retry_job",
			"trace.id":        "abc123",
		}
		textLogger = textLogger.WithFields(fieldMap)
		if err != nil {
			log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
			textLogger = dlog.NewNoOpLogger()
		}
	}

	logfmtConfig, err := dlog.NewDebugConfig(true, "MyLogfmtLogger", "test-output.logfmt", dlog.FormatLogfmt)
	if err != nil {
		log.Printf("failed to create debug config: %v, using NoOpLogger", err)
	} else {
		logfmtLogger, err = dlog.NewSlogLogger(logfmtConfig)
		fieldMap := dlog.FieldMap{
			"service.name":    "billing-api",
			"service.version": "1.4.2",
			"component":       "scheduler",
			"operation":       "retry_job",
			"trace.id":        "abc123",
		}
		logfmtLogger = logfmtLogger.WithFields(fieldMap)
		if err != nil {
			log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
			logfmtLogger = dlog.NewNoOpLogger()
		}
	}

	ctx := context.Background()
	logstashLogger.LogDebug(ctx, "New JSON-formatted Message with Debug-level", dlog.FieldMap{"my_debug_key": "my_debug_val"})
	logstashLogger.LogWarn(ctx, "New JSON-formatted Message with Warn-level", dlog.FieldMap{"my_warn_key": "my_warn_val"})

	textLogger.LogInfo(ctx, "New Text-formatted Message with Info-level", dlog.FieldMap{"my_info_key": "my_info_val"})
	textLogger.LogError(ctx, "New Text-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"), dlog.FieldMap{"my_error_key": "my_error_val"})

	logfmtLogger.LogInfo(ctx, "New Logfmt-formatted Message with Info-level", dlog.FieldMap{"my_info_key": "my_info_val"})
	logfmtLogger.LogError(ctx, "New Logfmt-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"), dlog.FieldMap{"my_error_key": "my_error_val"})

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
