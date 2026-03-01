package dlog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rohmanhakim/dlog"
)

func TestNewMultiWriter_StdoutOnly(t *testing.T) {
	mw, err := dlog.NewMultiWriter("", dlog.SyncImmediate, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter('') failed: %v", err)
	}
	if mw == nil {
		t.Fatal("NewMultiWriter('') returned nil")
	}
	defer mw.Close()
}

func TestNewMultiWriter_WithFile(t *testing.T) {
	// Create temp file path
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter(%q) failed: %v", outputFile, err)
	}
	if mw == nil {
		t.Fatal("NewMultiWriter returned nil")
	}
	defer mw.Close()

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("output file was not created: %s", outputFile)
	}
}

func TestNewMultiWriter_InvalidFilePath(t *testing.T) {
	// Use an invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/test.log"

	mw, err := dlog.NewMultiWriter(invalidPath, dlog.SyncImmediate, 0)
	if err == nil {
		mw.Close()
		t.Fatal("expected error for invalid file path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open debug log file") {
		t.Errorf("unexpected error message: %v", err)
	}
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
			if err != nil {
				t.Fatalf("NewMultiWriter failed: %v", err)
			}
			defer mw.Close()

			n, err := mw.Write([]byte(tt.data))
			if err != nil {
				t.Errorf("Write failed: %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Write returned %d bytes, want %d", n, len(tt.data))
			}

			// If file output, verify content
			if tt.outputFile != "" {
				content, err := os.ReadFile(tt.outputFile)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}
				if string(content) != tt.data {
					t.Errorf("file content = %q, want %q", string(content), tt.data)
				}
			}
		})
	}
}

func TestMultiWriter_Close(t *testing.T) {
	t.Run("close with file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "close-test.log")

		mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
		if err != nil {
			t.Fatalf("NewMultiWriter failed: %v", err)
		}

		if err := mw.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("close without file", func(t *testing.T) {
		mw, err := dlog.NewMultiWriter("", dlog.SyncImmediate, 0)
		if err != nil {
			t.Fatalf("NewMultiWriter failed: %v", err)
		}

		if err := mw.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})
}

func TestMultiWriter_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "integration-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	messages := []string{
		"first log line\n",
		"second log line\n",
		"third log line\n",
	}

	for _, msg := range messages {
		n, err := mw.Write([]byte(msg))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if n != len(msg) {
			t.Errorf("Write returned %d bytes, want %d", n, len(msg))
		}
	}

	mw.Close()

	// Verify file content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expected := strings.Join(messages, "")
	if string(content) != expected {
		t.Errorf("file content = %q, want %q", string(content), expected)
	}
}

// Test durability: data should be immediately visible on disk with SyncImmediate
func TestSyncMode_Immediate_Durability(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "immediate-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := "immediate write test\n"
	n, err := mw.Write([]byte(data))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d bytes, want %d", n, len(data))
	}

	// Read file immediately - data should be there without Close()
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != data {
		t.Errorf("SyncImmediate: file content = %q, want %q", string(content), data)
	}
}

// Test buffering: data should NOT be visible until Close() with SyncBuffered
func TestSyncMode_Buffered_BuffersUntilClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "buffered-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}

	data := "buffered write test\n"
	n, err := mw.Write([]byte(data))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d bytes, want %d", n, len(data))
	}

	// Read file immediately - data should NOT be there yet (still buffered)
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) == data {
		t.Errorf("SyncBuffered: data should not be flushed yet, but was found in file")
	}

	// Close should flush the buffer
	if err := mw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Now data should be visible
	content, err = os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file after close: %v", err)
	}
	if string(content) != data {
		t.Errorf("SyncBuffered after Close: file content = %q, want %q", string(content), data)
	}
}

// Test that Close() flushes buffered data
func TestSyncMode_Buffered_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "buffered-multi-test.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}

	messages := []string{
		"first buffered line\n",
		"second buffered line\n",
		"third buffered line\n",
	}

	for _, msg := range messages {
		n, err := mw.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(msg) {
			t.Errorf("Write returned %d bytes, want %d", n, len(msg))
		}
	}

	// Close should flush all buffered data
	if err := mw.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify all data was flushed
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expected := strings.Join(messages, "")
	if string(content) != expected {
		t.Errorf("SyncBuffered multiple writes: file content = %q, want %q", string(content), expected)
	}
}

// Test periodic flush: data should be flushed at intervals
func TestSyncMode_Periodic_FlushesAtIntervals(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "periodic-test.log")

	// Use a short interval for testing
	interval := 50 * time.Millisecond
	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncPeriodic, interval)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := "periodic write test\n"
	n, err := mw.Write([]byte(data))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d bytes, want %d", n, len(data))
	}

	// Data should not be immediately visible
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) == data {
		t.Errorf("SyncPeriodic: data should not be flushed immediately")
	}

	// Wait for periodic flush to occur (interval + buffer)
	time.Sleep(interval + 25*time.Millisecond)

	// Now data should be visible
	content, err = os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file after interval: %v", err)
	}
	if string(content) != data {
		t.Errorf("SyncPeriodic after interval: file content = %q, want %q", string(content), data)
	}
}

// Test that SyncPeriodic stops goroutine on Close
func TestSyncMode_Periodic_StopsOnClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "periodic-close-test.log")

	interval := 10 * time.Millisecond
	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncPeriodic, interval)
	if err != nil {
		t.Fatalf("NewMultiWriter failed: %v", err)
	}

	// Write some data
	mw.Write([]byte("test data\n"))

	// Close should stop the periodic flush goroutine cleanly
	done := make(chan error, 1)
	go func() {
		done <- mw.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
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
