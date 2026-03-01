package dlog

import (
	"log/slog"
	"slices"
)

// HandlerOptions configures handler behavior for all output formats.
// It provides a unified configuration interface for TextHandler, LogfmtHandler,
// and LogstashHandler.
type HandlerOptions struct {
	// Level is the minimum log level to output.
	Level slog.Level

	// IncludeFields filters fields to include (empty = all).
	// When specified, only fields in this list will be included in the output.
	// Built-in fields like @timestamp, @version, level, message, thread_name
	// should be included explicitly if you want them in the output.
	IncludeFields []string

	// ExcludeFields filters fields to exclude from the output.
	// Useful for removing sensitive data like passwords or API keys.
	ExcludeFields []string
}

// FilterFields applies include/exclude field filtering to a log entry.
// If both IncludeFields and ExcludeFields are specified, exclude takes precedence.
func FilterFields(entry map[string]any, includeFields, excludeFields []string) map[string]any {
	if len(includeFields) == 0 && len(excludeFields) == 0 {
		return entry
	}

	result := make(map[string]any)
	for key, value := range entry {
		// Check exclude list first
		if slices.Contains(excludeFields, key) {
			continue
		}

		// Check include list (if specified)
		if len(includeFields) > 0 && !slices.Contains(includeFields, key) {
			continue
		}

		result[key] = value
	}

	return result
}
