package dlog

import (
	"fmt"
	"log/slog"
)

// Format represents the output format for debug logging.
type Format string

const (
	// FormatJSON outputs logs in JSON format (Logstash/Elasticsearch compatible).
	FormatJSON Format = "json"
	// FormatText outputs logs in human-readable text format.
	FormatText Format = "text"
	// FormatLogfmt outputs logs in logfmt format (key=value pairs).
	FormatLogfmt Format = "logfmt"
)

// // The importance or severity of a log event.
// // Follows slog's Level API.
// type Level int

// // Names for common levels.
// // Follows slog's Level API.
// const (
// 	LevelDebug Level = -4
// 	LevelInfo  Level = 0
// 	LevelWarn  Level = 4
// 	LevelError Level = 8
// )

// DebugConfig holds configuration for debug logging.
type DebugConfig struct {
	// Enabled controls whether debug logging is active.
	Enabled bool

	MinLevel slog.Level

	// OutputFile is the path to write debug logs.
	// Empty means stdout only.
	OutputFile string

	// Format controls output format: "json" or "text".
	Format Format

	// IncludeFields filters fields to include (empty = all).
	IncludeFields []string

	// ExcludeFields filters fields to exclude.
	ExcludeFields []string
}

// NewDebugConfig creates a DebugConfig.
// Default will use LevelDebug for all downstream loggings.
func NewDebugConfig(enabled bool, loggerName string, outputFile string, format Format) (DebugConfig, error) {
	return DebugConfig{
		Enabled:       enabled,
		MinLevel:      slog.LevelDebug,
		OutputFile:    outputFile,
		Format:        format,
		IncludeFields: []string{},
		ExcludeFields: []string{},
	}, nil
}

// parseFormat parses a format string and returns the corresponding Format.
func parseFormat(format string) (Format, error) {
	if format == "" {
		return FormatJSON, nil
	}

	switch Format(format) {
	case FormatJSON, FormatText, FormatLogfmt:
		return Format(format), nil
	default:
		return "", fmt.Errorf("invalid debug format: %s (valid: json, text, logfmt)", format)
	}
}

// IsFileOutput returns true if file output is configured.
func (c DebugConfig) IsFileOutput() bool {
	return c.OutputFile != ""
}
