package main

import (
	"context"
	"log/slog"

	"github.com/rohmanhakim/dlog"
)

func main() {
	ctx := context.Background()

	// Pattern 1: WithFields - Pre-populate fields on the logger
	// Use this when you want to create a logger with persistent fields
	// that will be included in all log messages from that logger
	logger1, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatJSON,
		"",
		dlog.WithFields(dlog.FieldMap{
			"service": "billing-api",
			"version": "1.0.0",
		}),
	)
	defer logger1.Close()

	logger1.LogInfo(ctx, "Service started", dlog.FieldMap{
		"port": 8080,
	})

	// Pattern 2: WithGroup - Group fields under a namespace
	// Use this to organize related fields under a common prefix
	logger2, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatJSON,
		"",
	)
	defer logger2.Close()

	groupedLogger := logger2.WithGroup("request")
	groupedLogger.LogInfo(ctx, "Processing request", dlog.FieldMap{
		"method": "POST",
		"path":   "/api/v1/orders",
	})

	// Pattern 3: Field Filtering - Include/exclude specific fields
	// Use this to control which fields appear in the output
	// This is useful for:
	// - Security: excluding sensitive fields like passwords, tokens
	// - Performance: including only essential fields in high-volume logs
	// - Compliance: ensuring PII is never logged

	// 3a: Exclude sensitive fields
	logger3a, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatJSON,
		"",
		dlog.WithExcludeFields([]string{"password", "token", "secret"}),
	)
	defer logger3a.Close()

	logger3a.LogInfo(ctx, "User login attempt", dlog.FieldMap{
		"username": "john.doe",
		"password": "super-secret-123", // This will be excluded from output
		"success":  true,
	})

	// 3b: Include only specific fields (plus core fields)
	logger3b, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatJSON,
		"",
		dlog.WithIncludeFields([]string{
			"service", "version", "request_id", "user_id",
		}),
	)
	defer logger3b.Close()

	logger3b.LogInfo(ctx, "Processing order", dlog.FieldMap{
		"service":    "order-service",
		"version":    "2.1.0",
		"request_id": "req-abc-123",
		"user_id":    "user-456",
		"debug":      "verbose debug info", // This will be excluded (not in include list)
	})

	// 3c: Combine WithGroup and WithFields
	logger3c, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatJSON,
		"",
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

	// Pattern 4: Minimum level filtering
	logger4, _ := dlog.NewSlogLogger(
		true,
		dlog.FormatText,
		"",
		dlog.WithMinLevel(slog.LevelWarn),
	)
	defer logger4.Close()

	logger4.LogDebug(ctx, "This won't be logged") // Below minimum level
	logger4.LogInfo(ctx, "This won't be logged")  // Below minimum level
	logger4.LogWarn(ctx, "This will be logged")   // At or above minimum level
	logger4.LogError(ctx, "Error Occured: ", context.Canceled)
}
