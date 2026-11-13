package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/goccy/go-yaml"
)

// Load reads the dockerctl config file.
//
// It does not touch the project filesystem or Docker.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("config file path must not be empty")
	}

	configBody, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if _, err := utils.CheckMajorAPIVersionMatch(configBody, configAPIVersion); err != nil {
		return nil, err // TODO: custom error for api version mismatch.
	}

	// Defaults first, then the file. Unmarshalling keeps fields the file omits.
	// applyDefaults fills observability strings the file left blank.
	// Settings then win: path and env are not in the file.
	// A non-nil log field is an explicit flag or environment variable.
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(configBody, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	cfg.Observability.ApplyDefaults()

	return &cfg, nil
}
