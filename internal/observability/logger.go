package observability

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/TimCares/go-dockerctl/internal/config"
)

const (
	disableLogFile = "none"
	maxLogBytes    = 10 << 20 // 10 MiB, rotated to *.old
)

var outFile *os.File

// InitLogging installs the global zap logger used through zap.L() and zap.S().
//
// What reaches stderr depends on LogFormat, filtered by LogLevel:
//   - "console": only the plain-text messages of events implementing [see.FriendlyMessenger].
//   - "json": every record as JSON, including all context fields.
//
// Stdout stays reserved for command output.
// Every record is also appended as JSON to LogFile unless it is empty, "-", or "none".
func InitLogging(_ context.Context, logging config.LoggingConfig) (*zap.Logger, error) {
	if outFile != nil {
		_ = outFile.Close()
		outFile = nil
	}

	lvl, err := zapcore.ParseLevel(logging.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", logging.LogLevel, err)
	}

	jsonConfig := zap.NewProductionEncoderConfig()
	jsonConfig.MessageKey = logging.MessageKey

	stderr := zapcore.Lock(os.Stderr)
	var cores []zapcore.Core

	switch logging.LogFormat {
	case "console":
		cores = append(cores, newConsoleCore(stderr, lvl))
	case "json":
		cores = append(cores, zapcore.NewCore(zapcore.NewJSONEncoder(jsonConfig), stderr, lvl))
	default:
		return nil, fmt.Errorf("invalid log format %q: use \"console\" or \"json\"", logging.LogFormat)
	}

	if path, ok := resolvedLogFile(logging.LogFile); ok {
		f, err := openLogFile(path)
		if err != nil {
			return nil, fmt.Errorf("open log file %q: %w", path, err)
		}
		outFile = f
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(jsonConfig),
			zapcore.AddSync(f),
			zapcore.DebugLevel,
		))
	}

	logger := zap.New(
		zapcore.NewTee(cores...),
		zap.ErrorOutput(stderr),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// We could access the logger via see.L(), not zap.L().
	// In case we ever forget this, this acts as our "safeguard":
	zap.ReplaceGlobals(logger)

	if outFile != nil {
		zap.L().Debug("internal log file", zap.String("path", outFile.Name()))
	}

	return logger, nil
}

// FlushAndSyncLogFile flushes buffered log entries and closes the log file. Called before the process exits.
func FlushAndSyncLogFile() {
	_ = zap.L().Sync()
	if outFile != nil {
		_ = outFile.Close()
		outFile = nil
	}
}

func resolvedLogFile(path string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(path)) {
	case "", "-", disableLogFile:
		return "", false
	default:
		return path, true
	}
}

func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	if info, err := os.Stat(path); err == nil && info.Size() >= maxLogBytes {
		_ = os.Rename(path, path+".old")
	}

	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}
