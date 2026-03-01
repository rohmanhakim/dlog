package dlog_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohmanhakim/dlog"
)

func TestNewMultiWriter_StdoutOnly(t *testing.T) {
	mw, err := dlog.NewMultiWriter("")
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

	mw, err := dlog.NewMultiWriter(outputFile)
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

	mw, err := dlog.NewMultiWriter(invalidPath)
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
			mw, err := dlog.NewMultiWriter(tt.outputFile)
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

		mw, err := dlog.NewMultiWriter(outputFile)
		if err != nil {
			t.Fatalf("NewMultiWriter failed: %v", err)
		}

		if err := mw.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("close without file", func(t *testing.T) {
		mw, err := dlog.NewMultiWriter("")
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

	mw, err := dlog.NewMultiWriter(outputFile)
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
