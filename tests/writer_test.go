package dlog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiWriter_StdoutOnly(t *testing.T) {
	mw, err := dlog.NewMultiWriter("", dlog.SyncImmediate, 0)
	require.NoError(t, err, "NewMultiWriter('') failed")
	require.NotNil(t, mw, "NewMultiWriter('') returned nil")
	defer mw.Close()
}

func TestNewMultiWriter_WithFile(t *testing.T) {
	// Create temp file path
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	require.NoError(t, err, "NewMultiWriter(%q) failed", outputFile)
	require.NotNil(t, mw, "NewMultiWriter returned nil")
	defer mw.Close()

	// Verify file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err, "output file was not created: %s", outputFile)
}

func TestNewMultiWriter_InvalidFilePath(t *testing.T) {
	// Use an invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/test.log"

	mw, err := dlog.NewMultiWriter(invalidPath, dlog.SyncImmediate, 0)
	require.Error(t, err, "expected error for invalid file path, got nil")
	if mw != nil {
		mw.Close()
	}
	assert.Contains(t, err.Error(), "failed to open debug log file")
}

func TestMultiWriter_Write(t *testing.T) {
	tests := []struct {
		name       string
		outputFile string
		data       string
	}{
		{
			name:       "write to stdout only",
			outputFile: "",
			data:       "test message to stdout\n",
		},
		{
			name:       "write to stdout and file",
			outputFile: filepath.Join(t.TempDir(), "write-test.log"),
			data:       "test message to file\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := dlog.NewMultiWriter(tt.outputFile, dlog.SyncImmediate, 0)
			require.NoError(t, err, "NewMultiWriter failed")
			defer mw.Close()

			n, err := mw.Write([]byte(tt.data))
			require.NoError(t, err, "Write failed")
			assert.Equal(t, len(tt.data), n, "Write returned wrong byte count")

			// If file output, verify content
			if tt.outputFile != "" {
				content, err := os.ReadFile(tt.outputFile)
				require.NoError(t, err, "failed to read output file")
				assert.Equal(t, tt.data, string(content))
			}
		})
	}
}

func TestMultiWriter_Close(t *testing.T) {
	t.Run("close with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "close-test.log")

		mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
		require.NoError(t, err, "NewMultiWriter failed")

		err = mw.Close()
		assert.NoError(t, err, "Close failed")
	})

	t.Run("close without file", func(t *testing.T) {
		mw, err := dlog.NewMultiWriter("", dlog.SyncImmediate, 0)
		require.NoError(t, err, "NewMultiWriter failed")

		err = mw.Close()
		assert.NoError(t, err, "Close failed")
	})
}

func TestMultiWriter_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	require.NoError(t, err, "NewMultiWriter failed")
	defer mw.Close()

	messages := []string{
		"first log line\n",
		"second log line\n",
		"third log line\n",
	}

	for _, msg := range messages {
		n, err := mw.Write([]byte(msg))
		require.NoError(t, err, "Write failed")
		assert.Equal(t, len(msg), n, "Write returned wrong byte count")
	}

	mw.Close()

	// Verify file content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	expected := "first log line\nsecond log line\nthird log line\n"
	assert.Equal(t, expected, string(content))
}

// Test durability: data should be immediately visible on disk with SyncImmediate
func TestSyncMode_Immediate_Durability(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "immediate-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	require.NoError(t, err, "NewMultiWriter failed")
	defer mw.Close()

	data := "immediate write test\n"
	n, err := mw.Write([]byte(data))
	require.NoError(t, err, "Write failed")
	assert.Equal(t, len(data), n, "Write returned wrong byte count")

	// Read file immediately - data should be there without Close()
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")
	assert.Equal(t, data, string(content))
}

// Test buffering: data should NOT be visible until Close() with SyncBuffered
func TestSyncMode_Buffered_BuffersUntilClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "buffered-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	require.NoError(t, err, "NewMultiWriter failed")

	data := "buffered write test\n"
	n, err := mw.Write([]byte(data))
	require.NoError(t, err, "Write failed")
	assert.Equal(t, len(data), n, "Write returned wrong byte count")

	// Read file immediately - data should NOT be there yet (still buffered)
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")
	assert.NotEqual(t, data, string(content), "SyncBuffered: data should not be flushed yet, but was found in file")

	// Close should flush the buffer
	require.NoError(t, mw.Close(), "Close failed")

	// Now data should be visible
	content, err = os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file after close")
	assert.Equal(t, data, string(content))
}

// Test that Close() flushes buffered data
func TestSyncMode_Buffered_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "buffered-multi-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	require.NoError(t, err, "NewMultiWriter failed")

	messages := []string{
		"first buffered line\n",
		"second buffered line\n",
		"third buffered line\n",
	}

	for _, msg := range messages {
		n, err := mw.Write([]byte(msg))
		require.NoError(t, err, "Write failed")
		assert.Equal(t, len(msg), n, "Write returned wrong byte count")
	}

	// Close should flush all buffered data
	require.NoError(t, mw.Close(), "Close failed")

	// Verify all data was flushed
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")

	expected := "first buffered line\nsecond buffered line\nthird buffered line\n"
	assert.Equal(t, expected, string(content))
}

// Test periodic flush: data should be flushed at intervals
func TestSyncMode_Periodic_FlushesAtIntervals(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "periodic-test.log")

	// Use a short interval for testing
	interval := 50 * time.Millisecond
	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncPeriodic, interval)
	require.NoError(t, err, "NewMultiWriter failed")
	defer mw.Close()

	data := "periodic write test\n"
	n, err := mw.Write([]byte(data))
	require.NoError(t, err, "Write failed")
	assert.Equal(t, len(data), n, "Write returned wrong byte count")

	// Data should not be immediately visible
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file")
	assert.NotEqual(t, data, string(content), "SyncPeriodic: data should not be flushed immediately")

	// Wait for periodic flush to occur (interval + buffer)
	time.Sleep(interval + 25*time.Millisecond)

	// Now data should be visible
	content, err = os.ReadFile(outputFile)
	require.NoError(t, err, "failed to read output file after interval")
	assert.Equal(t, data, string(content))
}

// Test that SyncPeriodic stops goroutine on Close
func TestSyncMode_Periodic_StopsOnClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "periodic-close-test.log")

	interval := 10 * time.Millisecond
	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncPeriodic, interval)
	require.NoError(t, err, "NewMultiWriter failed")

	// Write some data
	mw.Write([]byte("test data\n"))

	// Close should stop the periodic flush goroutine cleanly
	done := make(chan error, 1)
	go func() {
		done <- mw.Close()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "Close failed")
	case <-time.After(time.Second):
		t.Error("Close took too long - goroutine may not have stopped")
	}
}

// Helper for capturing stdout in tests (used by other test files)
type captureWriter struct {
	buf bytes.Buffer
}

func (c *captureWriter) Write(p []byte) (n int, err error) {
	return c.buf.Write(p)
}

func (c *captureWriter) String() string {
	return c.buf.String()
}
