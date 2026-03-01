package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rohmanhakim/dlog"
)

func main() {
	logstashLogger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "test-output.jsonl")
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logstashLogger = dlog.NewNoOpLogger()
	}

	textLogger, err := dlog.NewSlogLogger(true, dlog.FormatText, "test-output.txt")
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		textLogger = dlog.NewNoOpLogger()
	}

	ctx := context.Background()
	logstashLogger.LogDebug(ctx, "New JSON-formatted Message with Debug-level")
	textLogger.LogInfo(ctx, "New Text-formatted Message with Info-level")
	textLogger.LogError(ctx, "New Text-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"))
	logstashLogger.LogWarn(ctx, "New JSON-formatted Message with Warn-level")
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
