package dlog

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// MultiWriter writes to multiple outputs simultaneously.
// It supports writing to stdout and/or a file.
type MultiWriter struct {
	mu     sync.Mutex
	writer io.Writer
	file   *os.File
	closer io.Closer
}

// NewMultiWriter creates a writer that outputs to stdout and/or file.
// If outputFile is empty, only stdout is used.
func NewMultiWriter(outputFile string, syncMode SyncMode, syncInterval time.Duration) (*MultiWriter, error) {
	var writers []io.Writer
	var file *os.File
	var closer io.Closer

	// Always include stdout
	writers = append(writers, os.Stdout)

	// Optionally include file
	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open debug log file: %w", err)
		}
		file = f

		// Create file output with specified sync mode
		fo := newFileOutput(f, syncMode, syncInterval)
		writers = append(writers, fo)
		closer = fo
	}

	return &MultiWriter{
		writer: io.MultiWriter(writers...),
		file:   file,
		closer: closer,
	}, nil
}

// Write implements io.Writer.
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writer.Write(p)
}

// Close closes the file handle if one was opened.
func (m *MultiWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closer != nil {
		return m.closer.Close()
	}
	return nil
}

// fileOutput wraps a file with configurable sync behavior.
type fileOutput struct {
	mu           sync.Mutex
	file         *os.File
	writer       *bufio.Writer
	syncMode     SyncMode
	syncInterval time.Duration
	stopCtx      context.Context
	stopCancel   context.CancelFunc
	wg           sync.WaitGroup
}

// newFileOutput creates a new file output with the specified sync mode.
func newFileOutput(file *os.File, syncMode SyncMode, syncInterval time.Duration) *fileOutput {
	// Default sync interval if not specified for periodic mode
	if syncInterval <= 0 {
		syncInterval = time.Second
	}

	fo := &fileOutput{
		file:         file,
		writer:       bufio.NewWriter(file),
		syncMode:     syncMode,
		syncInterval: syncInterval,
	}

	// Start periodic flush goroutine if needed
	if syncMode == SyncPeriodic {
		fo.stopCtx, fo.stopCancel = context.WithCancel(context.Background())
		fo.wg.Add(1)
		go fo.periodicFlush()
	}

	return fo
}

// periodicFlush periodically flushes the buffer.
func (f *fileOutput) periodicFlush() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopCtx.Done():
			return
		case <-ticker.C:
			f.mu.Lock()
			f.writer.Flush()
			f.mu.Unlock()
		}
	}
}

// Write writes to the file with the configured sync behavior.
func (f *fileOutput) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	n, err = f.writer.Write(p)
	if err != nil {
		return n, err
	}

	// Handle sync modes
	switch f.syncMode {
	case SyncImmediate:
		if err := f.writer.Flush(); err != nil {
			return n, err
		}
	case SyncBuffered:
		// Don't flush, let it accumulate until Close()
	case SyncPeriodic:
		// Periodic flush is handled by goroutine
	}

	return n, nil
}

// Close flushes any buffered data and closes the file.
func (f *fileOutput) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Stop periodic flush goroutine if running
	if f.stopCancel != nil {
		f.stopCancel()
		f.wg.Wait()
	}

	// Flush any remaining buffered data
	if err := f.writer.Flush(); err != nil {
		return err
	}

	return f.file.Close()
}
