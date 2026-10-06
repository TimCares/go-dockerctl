// Package project ties together config parsing, filesystem validation and
// docker compose validation. It sits above config, filesystem and docker in the
// dependency graph so those packages never have to import each other.
package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/docker"
	"github.com/TimCares/go-dockerctl/internal/filesystem"
)

// Validate checks the project on disk, including every service group's docker compose file.
//
// cfg is the result of [config.Load]: paths and defaults are already filled in.
// This requires a reachable Docker daemon.
func Validate(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	if err := filesystem.ValidateProject(cfg); err != nil {
		return fmt.Errorf("validating project filesystem: %w", err)
	}

	for i := range cfg.ServiceGroups {
		if err := docker.ValidateDockerComposeFile(ctx, &cfg.ServiceGroups[i]); err != nil {
			return err
		}
	}

	return nil
}
