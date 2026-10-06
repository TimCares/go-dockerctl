package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap/zapcore"
)

// LoggingConfig controls what is logged, and where to.
//
// ExtraFields are attached to every record under "labels".
type LoggingConfig struct {
	MessageKey  string            `yaml:"messageKey"`
	ExtraFields map[string]string `yaml:"extra_fields"`
	LogLevel    string            `yaml:"logLevel"`
	LogFormat   string            `yaml:"logFormat"`
	LogFile     string            `yaml:"logFile"`
}

// DefaultLoggingConfig sets MessageKey to "msg", LogLevel to "info",
// LogFormat to "console", and LogFile to [DefaultLogFile].
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		MessageKey: "msg",
		LogLevel:   "info",
		LogFormat:  "console",
		LogFile:    DefaultLogFile(),
	}
}

// DefaultLogFile is the platform-specific path used when logFile is unset.
// An empty home directory disables the log file.
func DefaultLogFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	const name = "dockerctl.log"
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "dockerctl", name)
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "dockerctl", name)
	default:
		base := os.Getenv("XDG_STATE_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(base, "dockerctl", name)
	}
}

// OtelConfig points the OTLP metrics exporter at a collector.
//
// SecretFile holds the bearer token sent to Endpoint.
type OtelConfig struct {
	Endpoint   string `yaml:"endpoint"`
	SecretFile string `yaml:"secretFile"`
	Enabled    bool   `yaml:"enabled"`
}

// ObservabilityConfig is logging and optional OTLP metrics.
// Otel.Enabled defaults to false, which leaves metrics off.
type ObservabilityConfig struct {
	Logging LoggingConfig `yaml:"logging"`
	Otel    OtelConfig    `yaml:"otel,omitempty"`
}

// DefaultObservabilityConfig adds the default logging config and disables otel.
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Logging: DefaultLoggingConfig(),
	}
}

// applyDefaults fills empty logging fields. Explicit values, including "none" for LogFile, stay.
func (c *ObservabilityConfig) applyDefaults() {
	defaults := DefaultLoggingConfig()
	if c.Logging.MessageKey == "" {
		c.Logging.MessageKey = defaults.MessageKey
	}
	if c.Logging.LogLevel == "" {
		c.Logging.LogLevel = defaults.LogLevel
	}
	if c.Logging.LogFormat == "" {
		c.Logging.LogFormat = defaults.LogFormat
	}
	if strings.TrimSpace(c.Logging.LogFile) == "" {
		c.Logging.LogFile = defaults.LogFile
	}
}

func validateObservabilityConfig(cfg *Config) error {
	logging := &cfg.Observability.Logging
	logging.LogLevel = strings.TrimSpace(logging.LogLevel)
	level, err := zapcore.ParseLevel(logging.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", logging.LogLevel, err)
	}
	logging.LogLevel = level.String()

	logging.LogFormat = strings.ToLower(strings.TrimSpace(logging.LogFormat))
	switch logging.LogFormat {
	case "console", "json":
	default:
		return fmt.Errorf("invalid log format %q: use \"console\" or \"json\"", logging.LogFormat)
	}

	logging.LogFile = strings.TrimSpace(logging.LogFile)

	if !cfg.Observability.Otel.Enabled {
		return nil
	}

	otelCfg := &cfg.Observability.Otel
	otelCfg.Endpoint = strings.TrimSpace(otelCfg.Endpoint)
	if otelCfg.Endpoint == "" {
		return errors.New("otel is enabled but endpoint is empty")
	}
	otelCfg.SecretFile = strings.TrimSpace(otelCfg.SecretFile)
	if otelCfg.SecretFile == "" {
		return errors.New("otel is enabled but secretFile is empty")
	}
	if !filepath.IsAbs(otelCfg.SecretFile) {
		otelCfg.SecretFile = filepath.Join(cfg.Runtime.ProjectDir, otelCfg.SecretFile)
	}
	return nil
}
