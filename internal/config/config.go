package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"
)

type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

const serviceGroupsDefaultDirName = "service-groups"

type ServiceGroup struct {
	Name string  `yaml:"name"`
	Path *string `yaml:"path"`
}

type Config struct {
	Runtime              RuntimeConfig
	Name                 string         `yaml:"name"`
	Envs                 []string       `yaml:"envs"`
	ManageSOPSIdentities bool           `yaml:"manage_sops_identities"`
	serviceGroups        []ServiceGroup `yaml:"service_groups"`
}

func getRuntimeConfig(config Config, projectDir string, activeEnv string) (*RuntimeConfig, error) {
	if !slices.Contains(config.Envs, activeEnv) {
		err := errors.New("active env not found in list of valid environments")
		zap.L().Error(err.Error(), zap.Strings("environments", config.Envs), zap.String("env", activeEnv))
		return nil, err
	}

	info, err := os.Stat(projectDir)
	if err != nil || !info.IsDir() {
		err := errors.New("project directory does not exist")
		zap.L().Error(err.Error(), zap.String("projectDir", projectDir))
	}

	return &RuntimeConfig{
		ProjectDir: projectDir,
		ActiveEnv:  activeEnv,
	}, nil
}

func GetConfig(configFilePath string, projectDir string, activeEnv string) (*Config, error) {
	if configFilePath == "" {
		err := errors.New("config file path must not be empty")
		zap.L().Error(err.Error(), zap.String("configFilePath", configFilePath))
		return nil, err
	}

	configBody, err := os.ReadFile(configFilePath)
	if err != nil {
		zap.L().Error("error reading config file", zap.Error(err), zap.String("configFilePath", configFilePath))
		return nil, err
	}

	var cfg Config

	if err := yaml.Unmarshal(configBody, &cfg); err != nil {
		zap.L().Error("error parsing config file", zap.Error(err))
		return nil, err
	}

	for _, serviceGroup := range cfg.serviceGroups {
		if serviceGroup.Path == nil || *serviceGroup.Path == "" {
			defaultServiceGroupPath := filepath.Join(cfg.Runtime.ProjectDir, serviceGroupsDefaultDirName, serviceGroup.Name)
			serviceGroup.Path = &defaultServiceGroupPath
		}

		if info, err := os.Stat(*serviceGroup.Path); err != nil {
			if os.IsNotExist(err) {

			}
			if !info.IsDir() {

			}
			// also check general structure that we require for our prjects, but put it into separate validation func
		}
	}

	runtimeConfig, err := getRuntimeConfig(cfg, projectDir, activeEnv)
	if err != nil {
		return nil, err
	}

	cfg.Runtime = *runtimeConfig

	return &cfg, nil
}
