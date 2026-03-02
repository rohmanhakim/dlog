package main

import (
	"context"
	"log/slog"

	"github.com/rohmanhakim/dlog"
)

func main() {
	ctx := context.Background()

	// =========================================================================
	// FormatJSON vs FormatLogstash - Understanding the difference
	// =========================================================================

	// FormatJSON uses standard slog.JSONHandler with NESTED groups:
	// Output: {"time":"...","level":"INFO","msg":"...","request":{"id":"abc","method":"GET"}}
	jsonLogger, _ := dlog.NewSlogLogger(true, dlog.FormatJSON)
	defer jsonLogger.Close()

	jsonGrouped := jsonLogger.WithGroup("request")
	jsonGrouped.LogInfo(ctx, "JSON format - nested group", dlog.FieldMap{
		"id":     "abc123",
		"method": "GET",
	})

	// FormatLogstash uses FLATTENED groups with dot notation:
	// Output: {"@timestamp":"...","log.level":"INFO","message":"...","request.id":"abc","request.method":"GET"}
	logstashLogger, _ := dlog.NewSlogLogger(true, dlog.FormatLogstash)
	defer logstashLogger.Close()

	logstashGrouped := logstashLogger.WithGroup("request")
	logstashGrouped.LogInfo(ctx, "Logstash format - flattened group", dlog.FieldMap{
		"id":     "abc123",
		"method": "GET",
	})

	// =========================================================================
	// Pattern 1: WithFields - Pre-populate fields on the logger
	// =========================================================================
	logger1, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatLogstash,
		dlog.WithFields(dlog.FieldMap{
			"service.name":    "billing-api",
			"service.version": "1.0.0",
		}),
	)
	defer logger1.Close()

	logger1.LogInfo(ctx, "Service started", dlog.FieldMap{
		"port": 8080,
	})

	// =========================================================================
	// Pattern 2: WithGroup - Group fields under a namespace
	// =========================================================================
	logger2, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatLogstash,
	)
	defer logger2.Close()

	groupedLogger := logger2.WithGroup("request")
	groupedLogger.LogInfo(ctx, "Processing request", dlog.FieldMap{
		"method": "POST",
		"path":   "/api/v1/orders",
	})

	// =========================================================================
	// Pattern 3: Field Filtering - Include/exclude specific fields
	// =========================================================================

	// 3a: Exclude sensitive fields
	logger3a, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatLogstash,
		dlog.WithExcludeFields([]string{"password", "token", "secret"}),
	)
	defer logger3a.Close()

	logger3a.LogInfo(ctx, "User login attempt", dlog.FieldMap{
		"username": "john.doe",
		"password": "super-secret-123", // This will be excluded from output
		"success":  true,
	})

	// 3b: Include only specific fields
	logger3b, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatLogstash,
		dlog.WithIncludeFields([]string{
			"service.name", "service.version", "request.id", "user.id",
		}),
	)
	defer logger3b.Close()

	logger3b.LogInfo(ctx, "Processing order", dlog.FieldMap{
		"service.name":    "order-service",
		"service.version": "2.1.0",
		"request.id":      "req-abc-123",
		"user.id":         "user-456",
		"debug":           "verbose debug info", // This will be excluded (not in include list)
	})

	// 3c: Combine WithGroup and WithFields
	logger3c, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatLogstash,
		dlog.WithFields(dlog.FieldMap{
			"service": "api-gateway",
		}),
		dlog.WithGroup("http"),
	)
	defer logger3c.Close()

	logger3c.LogInfo(ctx, "Request received", dlog.FieldMap{
		"method": "GET",
		"path":   "/health",
	})

	// =========================================================================
	// Pattern 4: Minimum level filtering
	// =========================================================================
	logger4, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatText,
		dlog.WithMinLevel(slog.LevelWarn),
	)
	defer logger4.Close()

	logger4.LogDebug(ctx, "This won't be logged") // Below minimum level
	logger4.LogInfo(ctx, "This won't be logged")  // Below minimum level
	logger4.LogWarn(ctx, "This will be logged")   // At or above minimum level
	logger4.LogError(ctx, "Error Occurred: ", context.Canceled)
}
