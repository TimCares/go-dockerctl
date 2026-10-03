// Package config loads and validates the dockerctl.yaml config file.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/TimCares/go-see"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"
)

// The major version of the dockerctl.yaml config file for this code.
const configAPIVersion uint = 1

// Only the major version, as patch and minor releases should not have
// breaking changes that require migrations.
// We do not couple it to dockerctl.Version, as we can make a major/breaking
// release to the code without the file format having a breaking change, but not the
// other way around.

// checkMajorAPIVersionMatch finds the first occurrence of "apiVersion" in "rawConfigBody"
// (must be at the beginning of a line) and checks whether the (major) version matches
// with the major version of dockerctl currently running.
func checkMajorAPIVersionMatch(ctx context.Context, rawConfigBody []byte) error {
	var cfg configVersion
	if err := yaml.Unmarshal(rawConfigBody, &cfg); err != nil {
		return fmt.Errorf("parsing config file, missing apiVersion: %w", err)
	}

	if cfg.APIVersion == configAPIVersion {
		return nil
	}
	if cfg.APIVersion > configAPIVersion {
		return fmt.Errorf("config apiVersion too high, found %d, expected %d", cfg.APIVersion, configAPIVersion)
	}
	// cfg.APIVersion < configAPIVersion
	see.L(ctx).Warn("config file api version larger than in code", zap.Uint("file", cfg.APIVersion), zap.Uint("code", configAPIVersion))
	// Later: try to apply migrations if necessary.
	return nil
}

var validServiceGroupNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// For each service group mentioned in the config, checks:
// 1. Service group name validity and uniqueness.
// 2. Service group path validity (if given, else set default).
// 3. Docker compose file name validity (if given, else set default).
// 4. Check if the envs of a service group are a subset of the global envs.
//
// Note that it does not check the filesystem validity, only the service group configs.
// Returns an error if any of the checked points are invalid, writes defaults into the config if fields are empty.
func validateServiceGroupConfig(config *Config) error {
	seenNames := make(map[string]struct{}, len(config.ServiceGroups))

	for i := range config.ServiceGroups {
		serviceGroup := &config.ServiceGroups[i]

		if !validServiceGroupNamePattern.MatchString(serviceGroup.Name) {
			return fmt.Errorf("invalid service group name %q: must be kebab-case and start with a lowercase letter", serviceGroup.Name)
		}

		if _, exists := seenNames[serviceGroup.Name]; exists {
			return fmt.Errorf("duplicate service group name %q", serviceGroup.Name)
		}
		seenNames[serviceGroup.Name] = struct{}{}

		if serviceGroup.Path == "" {
			serviceGroup.Path = filepath.Join(config.Runtime.ProjectDir, ServiceGroupsDefaultDirName, serviceGroup.Name)
			zap.L().Debug("Service group has no explicit path, using default", zap.String("serviceGroupName", serviceGroup.Name), zap.String("defaultServiceGroupPath", serviceGroup.Path))
		} else if !filepath.IsAbs(serviceGroup.Path) {
			serviceGroup.Path = filepath.Join(config.Runtime.ProjectDir, serviceGroup.Path)
		}

		if serviceGroup.DockerComposeFile == "" {
			zap.L().Debug("Service group has no explicit docker compose file name, using default", zap.String("serviceGroupName", serviceGroup.Name), zap.String("DockerComposeDefaultFileName", DockerComposeDefaultFileName))
			serviceGroup.DockerComposeFile = DockerComposeDefaultFileName
		}

		if serviceGroup.Envs == nil {
			serviceGroup.Envs = slices.Clone(config.Envs)
		} else if !isSubset(serviceGroup.Envs, config.Envs) {
			return fmt.Errorf("envs %v of service group %q must be a subset of global envs %v", serviceGroup.Envs, serviceGroup.Name, config.Envs)
		}
	}
	return nil
}

// Load reads and validates the dockerctl config file. It does not touch the project
// filesystem or Docker; see the project package for full project validation.
func Load(ctx context.Context, configFilePath string, projectDir string, activeEnv string) (*Config, error) {
	if configFilePath == "" {
		return nil, errors.New("config file path must not be empty")
	}

	configBody, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := checkMajorAPIVersionMatch(ctx, configBody); err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(configBody, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", configFilePath, err)
	}

	if len(cfg.Envs) == 0 {
		return nil, errors.New("list of envs must not be empty")
	}

	if len(cfg.ServiceGroups) == 0 {
		return nil, errors.New("at least one service group is required")
	}

	if !slices.Contains(cfg.Envs, activeEnv) {
		return nil, fmt.Errorf("active env %q not found in list of valid environments %v", activeEnv, cfg.Envs)
	}

	cfg.Runtime = RuntimeConfig{
		ProjectDir: projectDir,
		ActiveEnv:  activeEnv,
	}

	if err := validateServiceGroupConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
