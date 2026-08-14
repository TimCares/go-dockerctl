package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init installs the global zap logger used through zap.L() and zap.S().
// Logs are written to stderr so stdout stays reserved for command output.
func Init(level, format string) error {
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
	logger := zap.New(
		zapcore.NewCore(encoder, stderr, lvl),
		zap.ErrorOutput(stderr),
		zap.AddCaller(),
	)
	zap.ReplaceGlobals(logger)

	return nil
}

// Sync flushes buffered log entries. Called before the process exits.
func Sync() {
	_ = zap.L().Sync()
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
