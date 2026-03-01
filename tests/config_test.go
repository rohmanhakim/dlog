package dlog_test

import (
	"log/slog"
	"testing"

	"github.com/rohmanhakim/dlog"
)

func TestNewSlogLogger_DisabledReturnsNoOp_FromConfig(t *testing.T) {
	logger, err := dlog.NewSlogLogger(false, dlog.FormatJSON, "")
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}

	// Should return NoOpLogger when disabled
	if logger.Enabled() {
		t.Error("Expected NoOpLogger when enabled=false, but Enabled() returned true")
	}
}

func TestNewSlogLogger_WithOptions(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, dlog.FormatJSON, "",
		dlog.WithMinLevel(slog.LevelInfo),
		dlog.WithIncludeFields([]string{"fieldA", "fieldB"}),
		dlog.WithExcludeFields([]string{"fieldC"}),
	)
	if err != nil {
		t.Fatalf("NewSlogLogger failed: %v", err)
	}
	defer logger.Close()

	if !logger.Enabled() {
		t.Error("Expected Enabled() to return true")
	}
}
