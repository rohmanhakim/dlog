package dlog

import (
	"log/slog"
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

// fieldFilter provides O(1) field inclusion/exclusion lookups using pre-computed maps.
type fieldFilter struct {
	include map[string]struct{}
	exclude map[string]struct{}
}

// newFieldFilter creates a fieldFilter from include and exclude field lists.
// The maps are pre-computed once for efficient O(1) lookups during filtering.
func newFieldFilter(include, exclude []string) *fieldFilter {
	ff := &fieldFilter{}

	if len(include) > 0 {
		ff.include = make(map[string]struct{}, len(include))
		for _, f := range include {
			ff.include[f] = struct{}{}
		}
	}

	if len(exclude) > 0 {
		ff.exclude = make(map[string]struct{}, len(exclude))
		for _, f := range exclude {
			ff.exclude[f] = struct{}{}
		}
	}

	return ff
}

// shouldInclude checks if a field should be included based on include/exclude lists.
// Returns true if the field should be included, false otherwise.
// Exclude takes precedence over include.
func (ff *fieldFilter) shouldInclude(key string) bool {
	// Check exclude list first (takes precedence)
	if ff.exclude != nil {
		if _, excluded := ff.exclude[key]; excluded {
			return false
		}
	}

	// Check include list (if specified)
	if ff.include != nil {
		if _, included := ff.include[key]; !included {
			return false
		}
	}

	return true
}

// FilterFields applies include/exclude field filtering to a log entry.
// If both IncludeFields and ExcludeFields are specified, exclude takes precedence.
func FilterFields(entry map[string]any, includeFields, excludeFields []string) map[string]any {
	if len(includeFields) == 0 && len(excludeFields) == 0 {
		return entry
	}

	ff := newFieldFilter(includeFields, excludeFields)
	result := make(map[string]any)

	for key, value := range entry {
		if ff.shouldInclude(key) {
			result[key] = value
		}
	}

	return result
}
