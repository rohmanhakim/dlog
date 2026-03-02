package dlog_test

import (
	"log/slog"
	"testing"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlogLogger_DisabledReturnsNoOp_FromConfig(t *testing.T) {
	logger, err := dlog.NewSlogLogger(false, "", dlog.FormatJSON)
	require.NoError(t, err, "NewSlogLogger failed")

	// Should return NoOpLogger when disabled
	assert.False(t, logger.Enabled(), "Expected NoOpLogger when enabled=false, but Enabled() returned true")
}

func TestNewSlogLogger_WithOptions(t *testing.T) {
	logger, err := dlog.NewSlogLogger(true, "", dlog.FormatJSON,
		dlog.WithMinLevel(slog.LevelInfo),
		dlog.WithIncludeFields([]string{"fieldA", "fieldB"}),
		dlog.WithExcludeFields([]string{"fieldC"}),
	)
	require.NoError(t, err, "NewSlogLogger failed")
	defer logger.Close()

	assert.True(t, logger.Enabled(), "Expected Enabled() to return true")
}
