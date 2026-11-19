package observability

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TimCares/go-dockerctl/internal/definitions"
	"github.com/TimCares/go-dockerctl/internal/utils"
	"go.uber.org/zap/zapcore"
)

const dockerctlAppLogFileName string = "dockerctl.log"

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
// Without a home directory it is empty, which disables the log file.
func DefaultLogFile() string {
	basePath, err := utils.UserLogDir()
	if err != nil {
		return ""
	}
	return filepath.Join(basePath, definitions.DockerctlAppDirName, dockerctlAppLogFileName)
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

// ApplyDefaults fills empty logging fields. Explicit values, including "none" for LogFile, stay.
func (cfg *ObservabilityConfig) ApplyDefaults() {
	defaults := DefaultLoggingConfig()
	if cfg.Logging.MessageKey == "" {
		cfg.Logging.MessageKey = defaults.MessageKey
	}
	if cfg.Logging.LogLevel == "" {
		cfg.Logging.LogLevel = defaults.LogLevel
	}
	if cfg.Logging.LogFormat == "" {
		cfg.Logging.LogFormat = defaults.LogFormat
	}
	if strings.TrimSpace(cfg.Logging.LogFile) == "" {
		cfg.Logging.LogFile = defaults.LogFile
	}
}

func ValidateObservabilityConfig(cfg *ObservabilityConfig) error {
	logging := &cfg.Logging
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

	if !cfg.Otel.Enabled {
		return nil
	}

	otelCfg := &cfg.Otel
	otelCfg.Endpoint = strings.TrimSpace(otelCfg.Endpoint)
	if otelCfg.Endpoint == "" {
		return errors.New("otel is enabled but endpoint is empty")
	}
	otelCfg.SecretFile = strings.TrimSpace(otelCfg.SecretFile)
	if otelCfg.SecretFile == "" {
		return errors.New("otel is enabled but secretFile is empty")
	}
	return nil
}
