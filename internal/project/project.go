// Package project ties together config parsing, filesystem validation and
// docker compose validation. It sits above config, filesystem and docker in the
// dependency graph so those packages never have to import each other.
package project

import (
	"context"
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/docker"
	"github.com/TimCares/go-dockerctl/internal/filesystem"
)

// Load parses the config and validates the project on disk, including every
// service group's docker compose file. This requires a reachable Docker daemon.
func Load(ctx context.Context, configFilePath, projectDir, activeEnv string) (*config.Config, error) {
	cfg, err := config.Load(configFilePath, projectDir, activeEnv)
	if err != nil {
		return nil, err
	}

	if err := filesystem.ValidateProject(cfg); err != nil {
		return nil, fmt.Errorf("validating project filesystem: %w", err)
	}

	for i := range cfg.ServiceGroups {
		if err := docker.ValidateDockerComposeFile(ctx, &cfg.ServiceGroups[i]); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
