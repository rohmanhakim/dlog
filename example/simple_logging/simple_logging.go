package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohmanhakim/dlog"
)

func main() {
	// Default: SyncImmediate (maximum durability, flushes on every write)
	logstashLogger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "test-output.jsonl")
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		logstashLogger = dlog.NewNoOpLogger()
	}

	// With SyncBuffered for better performance (flushes only on Close())
	textLogger, err := dlog.NewSlogLogger(true, dlog.FormatText, "test-output.txt",
		dlog.WithSyncMode(dlog.SyncBuffered),
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		textLogger = dlog.NewNoOpLogger()
	}

	// With SyncPeriodic for balanced durability/performance (flushes every second by default)
	periodicLogger, err := dlog.NewSlogLogger(true, dlog.FormatLogfmt, "test-output.logfmt",
		dlog.WithSyncMode(dlog.SyncPeriodic),
		dlog.WithSyncInterval(500*time.Millisecond), // custom flush interval
	)
	if err != nil {
		log.Printf("failed to create debug logger: %v, using NoOpLogger", err)
		periodicLogger = dlog.NewNoOpLogger()
	}
	defer periodicLogger.Close()

	ctx := context.Background()
	logstashLogger.LogDebug(ctx, "New JSON-formatted Message with Debug-level")
	textLogger.LogInfo(ctx, "New Text-formatted Message with Info-level")
	textLogger.LogError(ctx, "New Text-formatted Message with Error-level", NewSimpleError(ErrCauseUnknown, "unknown error occured!"))
	logstashLogger.LogWarn(ctx, "New JSON-formatted Message with Warn-level")
	periodicLogger.LogInfo(ctx, "New Logfmt-formatted Message with periodic flush")

	// Close loggers to flush any buffered data
	logstashLogger.Close()
	textLogger.Close()
	// periodicLogger is closed via defer
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
