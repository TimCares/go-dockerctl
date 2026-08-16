package config

import (
	"errors"
	"os"
	"slices"

	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"
)

type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

type Config struct {
	Runtime              RuntimeConfig
	Name                 string   `yaml:"name"`
	Envs                 []string `yaml:"envs"`
	ManageSOPSIdentities bool     `yaml:"manage_sops_identities"`
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

	runtimeConfig, err := getRuntimeConfig(cfg, projectDir, activeEnv)
	if err != nil {
		return nil, err
	}

	cfg.Runtime = *runtimeConfig

	return &cfg, nil
}
