package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// resetLogger resets the global logger state for testing
func resetLogger() {
	mu.Lock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
	sugarLogger = nil
	baseLogger = nil
	atomicLevel = zap.AtomicLevel{}
	mu.Unlock()
	once = sync.Once{}
}

func TestNopSyncer(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_nopsyncer")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	syncer := nopSyncer{writer: tmpFile}

	// Test Write
	testData := []byte("test data")
	n, err := syncer.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, got %d", len(testData), n)
	}

	// Test Sync (should be no-op)
	err = syncer.Sync()
	if err != nil {
		t.Errorf("Sync should be no-op but returned error: %v", err)
	}
}

func TestInitLoggerWithLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zapcore.Level
	}{
		{"debug level", "debug", zapcore.DebugLevel},
		{"info level", "info", zapcore.InfoLevel},
		{"warn level", "warn", zapcore.WarnLevel},
		{"warning level", "warning", zapcore.WarnLevel},
		{"error level", "error", zapcore.ErrorLevel},
		{"invalid level defaults to info", "invalid", zapcore.InfoLevel},
		{"case insensitive", "DEBUG", zapcore.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLogger()

			initLoggerWithLevel(tt.level)

			if baseLogger == nil {
				t.Fatal("baseLogger should not be nil after initialization")
			}
			if sugarLogger == nil {
				t.Fatal("sugarLogger should not be nil after initialization")
			}

			// Check that the atomic level is set correctly
			if !atomicLevel.Level().Enabled(tt.expectedLevel) && atomicLevel.Level() != tt.expectedLevel {
				t.Errorf("Expected level %v, got %v", tt.expectedLevel, atomicLevel.Level())
			}
		})
	}
}

func TestInit(t *testing.T) {
	resetLogger()

	sugar, cleanup := Init()
	defer cleanup()

	if sugar == nil {
		t.Fatal("Init should return a non-nil SugaredLogger")
	}

	if baseLogger == nil {
		t.Fatal("baseLogger should not be nil after Init")
	}

	if sugarLogger == nil {
		t.Fatal("sugarLogger should not be nil after Init")
	}

	// Test that calling Init multiple times doesn't panic (due to sync.Once)
	sugar2, cleanup2 := Init()
	defer cleanup2()

	if sugar != sugar2 {
		t.Error("Multiple calls to Init should return the same logger instance")
	}
}

func TestInitWithLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLogger()

			sugar, cleanup := InitWithLevel(tt.level)
			defer cleanup()

			if sugar == nil {
				t.Fatal("InitWithLevel should return a non-nil SugaredLogger")
			}

			if baseLogger == nil {
				t.Fatal("baseLogger should not be nil after InitWithLevel")
			}

			if sugarLogger == nil {
				t.Fatal("sugarLogger should not be nil after InitWithLevel")
			}
		})
	}
}

func TestInitWithLevelMultipleCalls(t *testing.T) {
	resetLogger()

	// First call
	sugar1, cleanup1 := InitWithLevel("debug")
	defer cleanup1()

	// Second call with different level (should use SetLogLevel)
	sugar2, cleanup2 := InitWithLevel("error")
	defer cleanup2()

	if sugar1 == nil {
		t.Fatal("First InitWithLevel call returned nil logger")
	}

	if sugar2 == nil {
		t.Fatal("Second InitWithLevel call returned nil logger")
	}

	if sugar2 != Logger() {
		t.Error("Latest InitWithLevel call did not update the global logger instance")
	}

	if atomicLevel.Level() != zapcore.ErrorLevel {
		t.Errorf("Expected log level to be error, got %v", atomicLevel.Level())
	}
}

func TestInitWithConfigFile(t *testing.T) {
	resetLogger()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	sugar, cleanup, err := InitWithConfig(Config{Level: "info", FilePath: logPath})
	if err != nil {
		t.Fatalf("InitWithConfig returned error: %v", err)
	}

	sugar.Info("file logging test")
	cleanup()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "file logging test") {
		t.Errorf("log file does not contain expected message: %s", data)
	}
}

func TestLogger(t *testing.T) {
	resetLogger()

	logger := Logger()

	if logger == nil {
		t.Fatal("Logger should return a non-nil SugaredLogger")
	}

	// Test that multiple calls return the same instance
	logger2 := Logger()
	if logger != logger2 {
		t.Error("Multiple calls to Logger should return the same instance")
	}
}

func TestWith(t *testing.T) {
	resetLogger()

	logger := With("key", "value")

	if logger == nil {
		t.Fatal("With should return a non-nil SugaredLogger")
	}

	// Test with multiple key-value pairs
	logger2 := With("key1", "value1", "key2", "value2")

	if logger2 == nil {
		t.Fatal("With should return a non-nil SugaredLogger with multiple args")
	}
}

func TestSetLogLevel(t *testing.T) {
	resetLogger()

	// Initialize logger first
	initLoggerWithLevel("info")

	tests := []struct {
		name          string
		level         string
		expectedLevel zapcore.Level
	}{
		{"set debug", "debug", zapcore.DebugLevel},
		{"set info", "info", zapcore.InfoLevel},
		{"set warn", "warn", zapcore.WarnLevel},
		{"set warning", "warning", zapcore.WarnLevel},
		{"set error", "error", zapcore.ErrorLevel},
		{"set invalid defaults to info", "invalid", zapcore.InfoLevel},
		{"case insensitive", "ERROR", zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.level)

			if atomicLevel.Level() != tt.expectedLevel {
				t.Errorf("Expected level %v, got %v", tt.expectedLevel, atomicLevel.Level())
			}
		})
	}
}

func TestSetLogLevelBeforeInit(t *testing.T) {
	resetLogger()

	// Call SetLogLevel before initialization - should not panic
	SetLogLevel("debug")

	// The level should remain uninitialized
	if atomicLevel != (zap.AtomicLevel{}) {
		t.Error("SetLogLevel before initialization should not modify atomicLevel")
	}
}

func TestInitWithConfigReturnsError(t *testing.T) {
	resetLogger()

	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	// Using a file path nested under an existing file should cause directory creation to fail
	_, _, err := InitWithConfig(Config{Level: "info", FilePath: filepath.Join(blockingFile, "app.log")})
	if err == nil {
		t.Fatal("InitWithConfig should return an error when log file cannot be created")
	}
}

func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"Debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"WARN", zapcore.WarnLevel},
		{"warning", zapcore.WarnLevel},
		{"WARNING", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"invalid", zapcore.InfoLevel},
		{"", zapcore.InfoLevel},
		{"unknown", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resetLogger()
			initLoggerWithLevel(tt.input)

			if atomicLevel.Level() != tt.expected {
				t.Errorf("For input %q, expected level %v, got %v",
					tt.input, tt.expected, atomicLevel.Level())
			}
		})
	}
}

func TestLoggerConfiguration(t *testing.T) {
	resetLogger()

	initLoggerWithLevel("debug")

	// Test that the logger is properly configured
	if baseLogger == nil {
		t.Fatal("baseLogger should not be nil")
	}

	if sugarLogger == nil {
		t.Fatal("sugarLogger should not be nil")
	}

	// Test that we can actually log without panicking
	sugarLogger.Debug("test debug message")
	sugarLogger.Info("test info message")
	sugarLogger.Warn("test warn message")
	sugarLogger.Error("test error message")
}

func TestCleanupFunction(t *testing.T) {
	resetLogger()

	// Capture stderr to check for error messages
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	r, w, _ := os.Pipe()
	os.Stderr = w

	_, cleanup := Init()

	// Call cleanup - should not panic
	cleanup()

	w.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	// The cleanup should not produce any error output for our test logger
	// (since we're using nopSyncer which always returns nil for Sync)
	output := buf.String()
	if strings.Contains(output, "error syncing logger") {
		t.Errorf("Cleanup produced unexpected error: %s", output)
	}
}

// TestConcurrentAccess tests that the logger can be safely accessed from multiple goroutines
func TestConcurrentAccess(t *testing.T) {
	resetLogger()

	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				logger := Logger()
				if logger == nil {
					t.Errorf("Logger returned nil in goroutine")
					return
				}

				// Test With function
				withLogger := With("iteration", j)
				if withLogger == nil {
					t.Errorf("With returned nil in goroutine")
					return
				}

				// Test SetLogLevel
				levels := []string{"debug", "info", "warn", "error"}
				SetLogLevel(levels[j%len(levels)])
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
