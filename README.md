# dlog

[![codecov](https://codecov.io/gh/rohmanhakim/dlog/graph/badge.svg?token=7noPDnNvgy)](https://codecov.io/gh/rohmanhakim/dlog)
[![Go Reference](https://pkg.go.dev/badge/github.com/rohmanhakim/dlog.svg)](https://pkg.go.dev/github.com/rohmanhakim/dlog)

A simple Go's `log/slog` wrapper designed to provide debug logging capable of dual output (stdout and file).

## Features
- **Generic API**: Fully wraps standard `log/slog` types while giving a simplified `DebugLogger` abstraction
- **Zero Friction**: Clean functional options without disrupting typical `slog` idioms
- **Dual Output**: Write logs simultaneously to stdout and to a file
- **Multiple Formats**: Supports `json`, `text`, `logfmt`, and an optimized `logstash` JSON format
- **Configurable Sync Modes**: Fine-tune file I/O performance via `SyncImmediate`, `SyncBuffered`, or `SyncPeriodic` background flushing
- **Field Filtering**: Include or exclude specific fields dynamically to manage data loss vs privacy
- **Zero Overhead when Disabled**: Provides a fast `NoOpLogger` fallback when disabled

## Installation
```bash
go get github.com/rohmanhakim/dlog
```

## Quick Start
```go
package main

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/rohmanhakim/dlog"
)

func main() {
    // Execute with initialized slog wrapper options
    logger, err := dlog.NewSlogLogger(
        true, // enabled
        dlog.FormatLogstash, // specify log format
        dlog.WithOutputFile("debug.log"), // optional log file, will use stdout-only if not specified
        dlog.WithMinLevel(dlog.LevelDebug),
        dlog.WithSyncMode(dlog.SyncPeriodic),
        dlog.WithSyncInterval(1*time.Second),
    )
    if err != nil {
        panic(err)
    }
    defer logger.Close()
    
    ctx := context.Background()

    // Straightforward logging
    logger.LogInfo(ctx, "App started", dlog.FieldMap{"version": "1.0.0"})

    // Persistent context injection
    reqLogger := logger.
        WithGroup("request").
        WithFields(dlog.FieldMap{"id": "req-123"})
        
    reqLogger.LogDebug(ctx, "processing payload", dlog.FieldMap{"size": 1024})
}
```

## Configuration Options

### Functional Options
The `NewSlogLogger` function accepts multiple functional options. All options have sensible defaults:

| Option | Description | Default |
|--------|-------------|---------|
| `WithMinLevel(level Level)` | Set the minimum log level | `LevelDebug` |
| `WithIncludeFields(fields []string)` | Only output explicitly listed fields | Empty list (include all) |
| `WithExcludeFields(fields []string)` | Strip sensitive fields from output | Empty list (exclude none) |
| `WithFields(fields FieldMap)` | Pre-populate the logger with persistent fields | `nil` |
| `WithGroup(name string)` | Prefix subsequent attributes with a group name | `""` |
| `WithOutputFile(path string)` | Enable file output in addition to stdout | `""` (stdout only) |
| `WithSyncMode(mode SyncMode)` | Control file buffering logic (see Sync Modes below) | `SyncImmediate` |
| `WithSyncInterval(interval time.Duration)` | Override flushing interval for `SyncPeriodic` | `1 second` |

## Logging Handlers

### Formats
`dlog` is capable of rendering logs into a variety of formats using the `Format` type enum as an argument in `NewSlogLogger`.

| Format | Description |
|--------|-------------|
| `FormatJSON` | Standard `slog` nested format (`time`, `level`, `msg`). |
| `FormatText` | Human-readable output format ideal for local debug. |
| `FormatLogfmt` | Standard `key=value` machine-readable output. |
| `FormatLogstash` | Highly optimized flatten structure substituting variables (e.g. `@timestamp`, `log.level`). |

### Field Filtering
Filter noisy attributes directly without parsing intermediate payloads. Excludes take precedence over Includes.

```go
logger, _ := dlog.NewSlogLogger(
    true, dlog.FormatJSON,
    dlog.WithExcludeFields([]string{"password", "token"}),
)
// Result: `password` and `token` attributes are quietly dropped from final output.
```

## Sync Modes

When writing to a file, I/O efficiency matters. Use `SyncMode` to control buffering:

| Mode | Description | Profile |
|------------|-------------------------------------|----------------------------|
| `SyncImmediate` | Flushes to disk synchronously after every log line. | Scripts, low volume applications |
| `SyncBuffered` | Relies on default sizes (`bufio`). Flushes on `Close()`. | Burst workloads without losing output |
| `SyncPeriodic` | Triggers a background goroutine to `Flush()` at fixed intervals. | Extremely high throughput services |

## API Reference

### Types
```go
// DebugLogger provides structured debug logging capabilities.
// All methods are no-ops when debug mode is disabled.
type DebugLogger interface {
    Enabled() bool
    LogDebug(ctx context.Context, message string, fieldMap ...FieldMap)
    LogInfo(ctx context.Context, message string, fieldMap ...FieldMap)
    LogWarn(ctx context.Context, message string, fieldMap ...FieldMap)
    LogError(ctx context.Context, message string, err error, fieldMap ...FieldMap)
    WithFields(fields FieldMap) DebugLogger
    WithGroup(name string) DebugLogger
    Close() error
}

// FieldMap is a map of structured field names to values.
type FieldMap map[string]any

// Format represents the output format for debug logging.
type Format string

// SyncMode determines when file writes are flushed to disk.
type SyncMode int
```

### Functions
```go
// Creates a new SlogLogger with the given configuration
func NewSlogLogger(enabled bool, format Format, opts ...Option) (DebugLogger, error)

// Creates a NoOpLogger
func NewNoOpLogger() DebugLogger

// Level constants
const (
    LevelDebug Level = -4
    LevelInfo  Level = 0
    LevelWarn  Level = 4
    LevelError Level = 8
)

// Functional options
func WithMinLevel(level Level) Option
func WithIncludeFields(fields []string) Option
func WithExcludeFields(fields []string) Option
func WithFields(fields FieldMap) Option
func WithGroup(name string) Option
func WithOutputFile(path string) Option
func WithSyncMode(mode SyncMode) Option
func WithSyncInterval(interval time.Duration) Option
```

## License
MIT License - see [LICENSE](LICENSE) for details.
