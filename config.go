package dlog

import (
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

// config holds configuration for debug logging (internal).
type config struct {
	// minLevel is the minimum log level.
	minLevel slog.Level

	// includeFields filters fields to include (empty = all).
	includeFields []string

	// excludeFields filters fields to exclude.
	excludeFields []string
}

// Option is a functional option for configuring the logger.
type Option func(*config)

// WithMinLevel sets the minimum log level.
func WithMinLevel(level slog.Level) Option {
	return func(c *config) {
		c.minLevel = level
	}
}

// WithIncludeFields sets the fields to include in log output.
// If specified, only these fields will be included.
func WithIncludeFields(fields []string) Option {
	return func(c *config) {
		c.includeFields = fields
	}
}

// WithExcludeFields sets the fields to exclude from log output.
func WithExcludeFields(fields []string) Option {
	return func(c *config) {
		c.excludeFields = fields
	}
}
