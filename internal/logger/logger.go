package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	disableLogFile = "none"
	maxLogBytes    = 10 << 20 // 10 MiB; rotated to *.old
)

var outFile *os.File

// DefaultLogFile is the platform-specific path used when --log-file is unset.
func DefaultLogFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "dockerctl", "dockerctl.log")
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "dockerctl", "dockerctl.log")
	default:
		base := os.Getenv("XDG_STATE_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(base, "dockerctl", "dockerctl.log")
	}
}

// Init installs the global zap logger used through zap.L() and zap.S().
// Surface logs go to stderr so stdout stays reserved for command output.
// Internal logs are also appended as JSON to logFile unless it is empty, "-", or "none".
func Init(level, format, logFile string) error {
	if outFile != nil {
		_ = outFile.Close()
		outFile = nil
	}

	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", level, err)
	}

	var encoder zapcore.Encoder
	switch format {
	case "console":
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		if isTerminal(os.Stderr) {
			cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(cfg)
	case "json":
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	default:
		return fmt.Errorf("invalid log format %q: use \"console\" or \"json\"", format)
	}

	stderr := zapcore.Lock(os.Stderr)
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, stderr, lvl),
	}

	if path, ok := resolvedLogFile(logFile); ok {
		f, err := openLogFile(path)
		if err != nil {
			return fmt.Errorf("open log file %q: %w", path, err)
		}
		outFile = f
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
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
	zap.ReplaceGlobals(logger)

	if outFile != nil {
		zap.L().Debug("internal log file", zap.String("path", outFile.Name()))
	}

	return nil
}

// Sync flushes buffered log entries and closes the log file. Called before the process exits.
func Sync() {
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

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
