package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level    string
	FilePath string
}

type nopSyncer struct {
	mu     sync.RWMutex
	writer io.Writer
}

func (n *nopSyncer) Write(p []byte) (int, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.writer == nil {
		return 0, nil
	}
	return n.writer.Write(p)
}

func (n *nopSyncer) Sync() error {
	return nil // no-op
}

type StatusWriter struct {
	Status chan string
}

func (sw *StatusWriter) Write(p []byte) (int, error) {
	sw.Status <- string(p)
	return len(p), nil
}

var (
	sugarLogger   *zap.SugaredLogger
	baseLogger    *zap.Logger
	atomicLevel   zap.AtomicLevel // This allows dynamic level changes
	once          sync.Once
	mu            sync.RWMutex
	logFile       *os.File
	currentConfig Config
	stderrSyncer  = &nopSyncer{writer: os.Stderr}
)

func initLogger() {
	if err := applyConfig(Config{Level: "info"}); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %v", err))
	}
}

// initLoggerWithLevel initializes the logger with a specific level
func initLoggerWithLevel(level string) {
	if err := applyConfig(Config{Level: level}); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %v", err))
	}
}

func applyConfig(cfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	level := parseLevel(cfg.Level)

	if atomicLevel == (zap.AtomicLevel{}) {
		atomicLevel = zap.NewAtomicLevelAt(level)
	} else {
		atomicLevel.SetLevel(level)
	}

	encoderCfg := zap.NewDevelopmentConfig().EncoderConfig
	encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(stderrSyncer), atomicLevel)
	cores := []zapcore.Core{consoleCore}

	filePath := strings.TrimSpace(cfg.FilePath)
	if filePath != "" {
		fileCore, handle, err := buildFileCore(encoderCfg, filePath)
		if err != nil {
			return err
		}

		if logFile != nil && logFile != handle {
			_ = logFile.Close()
		}
		logFile = handle
		cores = append(cores, fileCore)
	} else if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	core := zapcore.NewTee(cores...)

	options := []zap.Option{
		zap.AddCaller(),
		zap.Development(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	newLogger := zap.New(core, options...)
	sugar := newLogger.Sugar()

	baseLogger = newLogger
	sugarLogger = sugar

	zap.ReplaceGlobals(baseLogger)

	currentConfig = Config{Level: level.String(), FilePath: filePath}

	return nil
}

func buildFileCore(encoderCfg zapcore.EncoderConfig, path string) (zapcore.Core, *os.File, error) {
	cleanedPath := filepath.Clean(path)
	dir := filepath.Dir(cleanedPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("creating log directory %q: %w", dir, err)
		}
	}

	file, err := os.OpenFile(cleanedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, nil, fmt.Errorf("opening log file %q: %w", cleanedPath, err)
	}

	fileEncoderCfg := encoderCfg
	fileEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderCfg)
	core := zapcore.NewCore(fileEncoder, zapcore.AddSync(file), atomicLevel)

	return core, file, nil
}

func InitWithConfig(cfg Config) (*zap.SugaredLogger, func(), error) {
	initializedHere := false
	var initErr error
	requested := Config{Level: parseLevel(cfg.Level).String(), FilePath: strings.TrimSpace(cfg.FilePath)}

	once.Do(func() {
		initErr = applyConfig(cfg)
		initializedHere = true
	})

	if initErr != nil {
		return nil, nil, fmt.Errorf("logger initialization failed: %w", initErr)
	}

	if !initializedHere {
		mu.RLock()
		sameConfig := currentConfig == requested
		mu.RUnlock()

		if !sameConfig {
			if err := applyConfig(cfg); err != nil {
				return nil, nil, fmt.Errorf("logger reconfiguration failed: %w", err)
			}
		}
	}

	mu.RLock()
	if baseLogger == nil {
		mu.RUnlock()
		return nil, nil, fmt.Errorf("logger initialization failed: baseLogger is nil")
	}
	sugar := sugarLogger
	mu.RUnlock()

	return sugar, createCleanupFunc(), nil
}

// Init sets up the global zap logger and installs it as the zap global logger.
// It returns the sugared logger and a cleanup function that must be deferred.
func Init() (*zap.SugaredLogger, func()) {
	sugar, cleanup, err := InitWithConfig(Config{Level: "info"})
	if err != nil {
		panic(fmt.Sprintf("logger initialization failed: %v", err))
	}
	return sugar, cleanup
}

// InitWithLevel sets up the global zap logger with a specific log level
func InitWithLevel(level string) (*zap.SugaredLogger, func()) {
	sugar, cleanup, err := InitWithConfig(Config{Level: level})
	if err != nil {
		panic(fmt.Sprintf("logger initialization failed: %v", err))
	}
	return sugar, cleanup
}

func Logger() *zap.SugaredLogger {
	once.Do(initLogger)

	mu.RLock()
	defer mu.RUnlock()

	if sugarLogger == nil {
		panic("logger initialization failed: sugarLogger is nil")
	}

	return sugarLogger
}

func With(args ...interface{}) *zap.SugaredLogger {
	return Logger().With(args...)
}

func createCleanupFunc() func() {
	mu.RLock()
	currentFile := logFile
	mu.RUnlock()

	return func() {
		mu.Lock()
		defer mu.Unlock()

		if baseLogger != nil {
			if err := baseLogger.Sync(); err != nil {
				fmt.Fprintf(os.Stderr, "error syncing logger: %v\n", err)
			}
		}

		if currentFile != nil {
			if err := currentFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing log file: %v\n", err)
			}
			if logFile == currentFile {
				logFile = nil
			}
		}
	}
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// SetLogLevel dynamically changes the log level without re-initializing the logger
func SetLogLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	if atomicLevel == (zap.AtomicLevel{}) {
		return // Not initialized yet
	}

	newLevel := parseLevel(level)
	atomicLevel.SetLevel(newLevel)
	currentConfig.Level = newLevel.String()
}

// ReplaceStderrWriter swaps the current stderr writer used by the logger.
// It returns the previous writer (never nil; defaults to os.Stderr).
func ReplaceStderrWriter(newOut io.Writer) (oldOut io.Writer) {
	if newOut == nil {
		newOut = os.Stderr
	}

	stderrSyncer.mu.Lock()
	defer stderrSyncer.mu.Unlock()

	oldOut = stderrSyncer.writer
	if oldOut == nil {
		oldOut = os.Stderr
	}
	stderrSyncer.writer = newOut
	return
}
