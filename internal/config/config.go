package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"

	"github.com/TimCares/go-dockerctl/internal/docker"
	dockerctlFilesystem "github.com/TimCares/go-dockerctl/internal/filesystem"
)

type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

const ServiceGroupsDefaultDirName = "service-groups"
const DockerComposeDefaultFileName = "docker-compose.yaml"

type ServiceGroup struct {
	Path              string    `yaml:"path"`
	Name              string    `yaml:"name"`
	DockerComposeFile string    `yaml:"docker_compose_file"`
	Envs              *[]string `yaml:"envs,omitempty"`
	DisableDeploy     bool      `yaml:"disable_deploy"`  // Helpful if you have a service group that should not always be deployed, but want to list it for consistency.
}

//TODO: func (g ServiceGroup) filename() string

type Config struct {
	Runtime              RuntimeConfig
	Name                 string         `yaml:"name"`
	Envs                 []string       `yaml:"envs"`
	ManageSOPSIdentities bool           `yaml:"manage_sops_identities"`
	ServiceGroups        []ServiceGroup `yaml:"service_groups"`
}

func getRuntimeConfig(config Config, projectDir string, activeEnv string) (*RuntimeConfig, error) {
	if !slices.Contains(config.Envs, activeEnv) {
		err := errors.New("active env not found in list of valid environments")
		zap.L().Error(err.Error(), zap.Strings("environments", config.Envs), zap.String("env", activeEnv))
		return nil, err
	}

	return &RuntimeConfig{
		ProjectDir: projectDir,
		ActiveEnv:  activeEnv,
	}, nil
}

// kebab-case
var validServiceGroupNamePattern = regexp.MustCompile(`^[a-z]+(?:-[a-z0-9]+)*$`)

// For each service group mentioned in the config, checks:
// 1. Service group name validity.
// 2. Service group path validity (if given, else set default).
// 3. Docker compose file name validity (if given, else set default).
// 4. Check if the envs of a service group are a subset of the global envs.
//
// Note that it does not check the filesystem validity, only the service group configs.
// Returns an error if any of the checked points are invalid, writes defaults into the config if fields are empty.
func validateServiceGroupConfig(config *Config) error {
	for _, serviceGroup := range config.ServiceGroups {
		if !validServiceGroupNamePattern.MatchString(serviceGroup.Name) {
			errorMsg := fmt.Sprintf("Invalid service group name, must be kebab-case and start with a lowercase letter, found '%s'", serviceGroup.Name)
			err := errors.New(errorMsg)
			zap.L().Error(err.Error(), zap.String("serviceGroupName", serviceGroup.Name), zap.String("expectedPattern", validServiceGroupNamePattern.String()))
			return err
		}

		if serviceGroup.Path == "" {
			defaultServiceGroupPath := filepath.Join(config.Runtime.ProjectDir, ServiceGroupsDefaultDirName, serviceGroup.Name)
			zap.L().Debug("Service group has no explicit path, using default", zap.String("serviceGroupName", serviceGroup.Name), zap.String("defaultServiceGroupPath", defaultServiceGroupPath))
			serviceGroup.Path = defaultServiceGroupPath
		}

		if serviceGroup.DockerComposeFile == "" {
			zap.L().Debug("Service group has no explicit docker compose file name, using default", zap.String("serviceGroupName", serviceGroup.Name), zap.String("DockerComposeDefaultFileName", DockerComposeDefaultFileName))
			serviceGroup.DockerComposeFile = DockerComposeDefaultFileName
		}

		if serviceGroup.Envs == nil {
			serviceGroup.Envs = &slices.Clone(config.Envs)
		} else {
			// TODO: check serviceGroup.Envs is subset of config.Envs + no duplications
		}
	}
	return nil
}

func GetConfig(configFilePath string, projectDir string, activeEnv string) (*Config, error) {
	if configFilePath == "" {
		err := errors.New("config file path must not be empty")
		zap.L().Error(err.Error(), zap.String("configFilePath", configFilePath))
		return nil, err
	}

	configBody, readErr := os.ReadFile(configFilePath)
	if readErr != nil {
		zap.L().Error("error reading config file", zap.Error(readErr), zap.String("configFilePath", configFilePath))
		return nil, readErr
	}

	var cfg Config

	if yamlErr := yaml.Unmarshal(configBody, &cfg); yamlErr != nil {
		zap.L().Error("error parsing config file", zap.Error(yamlErr))
		return nil, yamlErr
	}

	runtimeConfig, runtimeConfigErr := getRuntimeConfig(cfg, projectDir, activeEnv)
	if runtimeConfigErr != nil {
		// TODO: log
		return nil, runtimeConfigErr
	}

	cfg.Runtime = *runtimeConfig

	configValidationErr := validateServiceGroupConfig(&cfg)
	if configValidationErr != nil {
		// TODO: log
		return nil, configValidationErr
	}

	if len(cfg.Envs) == 0 {
		envsLenError := errors.New("List of `Envs` must not be empty.")
		zap.L().Error(envsLenError.Error())
		return nil, envsLenError
	}

	if len(cfg.ServiceGroups) == 0 {
		serviceGroupsLenError := errors.New("At least one service group must be contained in `ServiceGroups`.")
		zap.L().Error(serviceGroupsLenError.Error())
		return nil, serviceGroupsLenError
	}

	// Validate the dockerctl filesystem structure.
	dockerctlFilesystem := dockerctlFilesystem.MakeDockerctlFilesystem(&cfg)
	filesystemErr := dockerctlFilesystem.Validate(cfg.Runtime.ProjectDir)
	if configValidationErr != nil {
		// TODO: log
		return nil, filesystemErr
	}

	for _, serviceGroup := range cfg.ServiceGroups {
		dockerComposeValidationErr := docker.ValidateDockerComposeFile(&serviceGroup)
		if dockerComposeValidationErr != nil {
			// TODO: log
			return nil, dockerComposeValidationErr
		}

	}

	return &cfg, nil
}
