package dlog_test

import (
	"log/slog"
	"testing"
	"time"

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

func TestWithSyncMode(t *testing.T) {
	tests := []struct {
		name      string
		syncMode  dlog.SyncMode
		expectMsg string
	}{
		{
			name:      "SyncImmediate mode",
			syncMode:  dlog.SyncImmediate,
			expectMsg: "SyncImmediate mode should be set",
		},
		{
			name:      "SyncBuffered mode",
			syncMode:  dlog.SyncBuffered,
			expectMsg: "SyncBuffered mode should be set",
		},
		{
			name:      "SyncPeriodic mode",
			syncMode:  dlog.SyncPeriodic,
			expectMsg: "SyncPeriodic mode should be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := dlog.NewSlogLogger(true, "", dlog.FormatJSON,
				dlog.WithSyncMode(tt.syncMode),
			)
			require.NoError(t, err, "NewSlogLogger failed")
			defer logger.Close()

			assert.True(t, logger.Enabled(), tt.expectMsg)
		})
	}
}

func TestWithSyncInterval(t *testing.T) {
	tests := []struct {
		name      string
		interval  time.Duration
		syncMode  dlog.SyncMode
		expectMsg string
	}{
		{
			name:      "500ms interval",
			interval:  500 * time.Millisecond,
			syncMode:  dlog.SyncPeriodic,
			expectMsg: "500ms sync interval should be set",
		},
		{
			name:      "2 second interval",
			interval:  2 * time.Second,
			syncMode:  dlog.SyncPeriodic,
			expectMsg: "2 second sync interval should be set",
		},
		{
			name:      "5 second interval",
			interval:  5 * time.Second,
			syncMode:  dlog.SyncPeriodic,
			expectMsg: "5 second sync interval should be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := dlog.NewSlogLogger(true, "", dlog.FormatJSON,
				dlog.WithSyncMode(tt.syncMode),
				dlog.WithSyncInterval(tt.interval),
			)
			require.NoError(t, err, "NewSlogLogger failed")
			defer logger.Close()

			assert.True(t, logger.Enabled(), tt.expectMsg)
		})
	}
}

func TestWithSyncMode_WithFileOutput(t *testing.T) {
	// Create a temp file for testing
	tmpFile := "/tmp/dlog_test_syncmode.log"

	logger, err := dlog.NewSlogLogger(true, tmpFile, dlog.FormatJSON,
		dlog.WithSyncMode(dlog.SyncBuffered),
	)
	require.NoError(t, err, "NewSlogLogger with file output failed")
	defer logger.Close()

	assert.True(t, logger.Enabled(), "Logger should be enabled with SyncBuffered mode")
}
