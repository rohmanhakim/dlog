package dlog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rohmanhakim/dlog"
)

// BenchmarkSyncImmediate measures performance with immediate flush on every write
func BenchmarkSyncImmediate(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-immediate.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

// BenchmarkSyncBuffered measures performance with buffered writes (flush only on Close)
func BenchmarkSyncBuffered(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-buffered.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

// BenchmarkSyncPeriodic measures performance with periodic flush
func BenchmarkSyncPeriodic(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-periodic.log")

	// Use a long interval so flush doesn't interfere during benchmark
	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncPeriodic, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

// BenchmarkSyncImmediate_Parallel measures parallel performance with immediate flush
func BenchmarkSyncImmediate_Parallel(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-immediate-parallel.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mw.Write(data)
		}
	})
	b.StopTimer()
	mw.Close()
}

// BenchmarkSyncBuffered_Parallel measures parallel performance with buffered writes
func BenchmarkSyncBuffered_Parallel(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-buffered-parallel.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mw.Write(data)
		}
	})
	b.StopTimer()
	mw.Close()
}

// Benchmark with different message sizes
func BenchmarkSyncImmediate_SmallMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncImmediate, 32)
}

func BenchmarkSyncBuffered_SmallMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncBuffered, 32)
}

func BenchmarkSyncImmediate_MediumMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncImmediate, 256)
}

func BenchmarkSyncBuffered_MediumMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncBuffered, 256)
}

func BenchmarkSyncImmediate_LargeMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncImmediate, 4096)
}

func BenchmarkSyncBuffered_LargeMessage(b *testing.B) {
	benchmarkWithMessageSize(b, dlog.SyncBuffered, 4096)
}

func benchmarkWithMessageSize(b *testing.B, syncMode dlog.SyncMode, size int) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "benchmark-size.log")

	mw, err := dlog.NewMultiWriter(outputFile, syncMode, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	// Create message of specified size
	data := make([]byte, size)
	for i := range data {
		data[i] = 'x'
	}
	data[size-1] = '\n'

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

// Benchmark comparing real-world logging scenario
func BenchmarkRealWorld_SyncImmediate(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "realworld-immediate.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncImmediate, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	// Simulate a realistic log message with JSON-like structure
	data := []byte(`{"level":"info","timestamp":"2024-01-15T10:30:00Z","message":"Request processed","request_id":"abc-123","duration_ms":42,"status":200}` + "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

func BenchmarkRealWorld_SyncBuffered(b *testing.B) {
	tmpDir := b.TempDir()
	outputFile := filepath.Join(tmpDir, "realworld-buffered.log")

	mw, err := dlog.NewMultiWriter(outputFile, dlog.SyncBuffered, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	// Simulate a realistic log message with JSON-like structure
	data := []byte(`{"level":"info","timestamp":"2024-01-15T10:30:00Z","message":"Request processed","request_id":"abc-123","duration_ms":42,"status":200}` + "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
	b.StopTimer()
	mw.Close()
}

// Benchmark stdout only (no file I/O) for baseline comparison
func BenchmarkStdoutOnly(b *testing.B) {
	// Redirect stdout to dev null to avoid polluting test output
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("Failed to open /dev/null: %v", err)
	}
	defer devNull.Close()

	// Save original stdout
	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	mw, err := dlog.NewMultiWriter("", dlog.SyncImmediate, 0)
	if err != nil {
		b.Fatalf("NewMultiWriter failed: %v", err)
	}
	defer mw.Close()

	data := []byte("benchmark log message with some data\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
}
