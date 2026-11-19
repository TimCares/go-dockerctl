package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/TimCares/go-dockerctl/internal/docker"
	"github.com/TimCares/go-dockerctl/internal/observability"
	"github.com/TimCares/go-dockerctl/internal/project"
	"github.com/TimCares/go-dockerctl/internal/utils"
)

var validEntityNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// NamespaceConfig is a single docker compose project managed by dockerctl.
type NamespaceConfig struct {
	Name              string                 `yaml:"name"`
	Path              string                 `yaml:"path"`
	DockerComposeFile string                 `yaml:"dockerComposeFile"`
	Envs              utils.UniqueStringList `yaml:"envs,omitempty"` // nil (key absent) means all global envs.
	DisableDeploy     bool                   `yaml:"disable_deploy"` // Helpful if you have a namespace that should not always be deployed, but want to list it for consistency.
}

// For each namespace mentioned in the config, checks:
// 1. Namespace name validity and uniqueness.
// 2. Namespace path validity (if given, else set default). A relative path is relative to the project, not the working directory.
// 3. Docker compose file name validity (if given, else set default).
// 4. Check if the envs of a namespace are a subset of the global envs.
//
// Returns an error if any of the checked points are invalid, writes defaults into the config if fields are empty.
func (cfg *NamespaceConfig) Validate(projectPath string) error {
	if !validEntityNamePattern.MatchString(cfg.Name) {
		return fmt.Errorf("invalid namespace name %q: must be kebab-case and start with a lowercase letter", cfg.Name)
	}

	if cfg.Path == "" {
		cfg.Path = project.NamespaceDir(projectPath, cfg.Name)
	} else if !filepath.IsAbs(cfg.Path) {
		cfg.Path = filepath.Join(projectPath, cfg.Path) // Resolve against the path of the project dir, not the CWD.
	}
	absFilePath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return err
	}
	cfg.Path = absFilePath

	if cfg.DockerComposeFile == "" {
		cfg.DockerComposeFile = docker.DockerComposeFileName
	}

	return nil
}

// NodeInfoConfig contains all necessary info about the node dockerctl runs on.
type NodeInfoConfig struct {
	Name string `yaml:"name,omitempty"`
}

// DefaultNodeInfoConfig names the node from NODE_NAME, falling back to the hostname.
func DefaultNodeInfoConfig() NodeInfoConfig {
	name := os.Getenv("NODE_NAME")
	if name == "" {
		name, _ = os.Hostname()
	}
	return NodeInfoConfig{
		Name: name,
	}
}

// Config is the parsed dockerctl.yaml file.
type Config struct {
	APIVersion    uint                              `yaml:"apiVersion"`
	Name          string                            `yaml:"name"`
	Envs          utils.UniqueStringList            `yaml:"envs"`
	Namespaces    []NamespaceConfig                 `yaml:"namespaces"`
	Observability observability.ObservabilityConfig `yaml:"observability"`
	Node          NodeInfoConfig                    `yaml:"node,omitempty"`
}

// The major version of the dockerctl.yaml config file for this code.
//
// Only the major version, as patch and minor releases should not have
// breaking changes that require migrations.
// We do not couple it to dockerctl.Version, as we can make a major/breaking
// release to the code without the file format having a breaking change, but not the
// other way around.
const configAPIVersion uint = 1

// DefaultConfig is the base that the config file is unmarshalled onto.
func DefaultConfig() Config {
	return Config{
		APIVersion:    configAPIVersion,
		Observability: observability.DefaultObservabilityConfig(),
		Node:          DefaultNodeInfoConfig(),
	}
}

func (cfg *Config) Validate(projectPath string) error {
	if !validEntityNamePattern.MatchString(cfg.Name) {
		return fmt.Errorf("invalid project name %q: must be kebab-case and start with a lowercase letter", cfg.Name)
	}

	if len(cfg.Envs) == 0 {
		return errors.New("list of envs must not be empty")
	}
	if err := checkValidEnvNames(cfg.Envs); err != nil {
		return err
	}

	if len(cfg.Namespaces) == 0 {
		return errors.New("at least one namespace is required")
	}

	seenNames := make(map[string]struct{}, len(cfg.Namespaces))

	for i := range cfg.Namespaces {
		namespaceConfig := &cfg.Namespaces[i]

		if err := namespaceConfig.Validate(projectPath); err != nil {
			return fmt.Errorf("validating namespace config %s: %w", namespaceConfig.Name, err)
		}

		if namespaceConfig.Envs == nil {
			namespaceConfig.Envs = slices.Clone(cfg.Envs)
		} else if !utils.IsSubset(namespaceConfig.Envs, cfg.Envs) { // Also guarantees namespaceConfig.Envs have the right regex pattern.
			return fmt.Errorf("envs %v of namespace %s must be a subset of global envs %v", namespaceConfig.Envs, namespaceConfig.Name, cfg.Envs)
		}

		if _, exists := seenNames[namespaceConfig.Name]; exists {
			return fmt.Errorf("duplicate namespace name %s", namespaceConfig.Name)
		}
		seenNames[namespaceConfig.Name] = struct{}{}
	}

	if err := observability.ValidateObservabilityConfig(&cfg.Observability); err != nil {
		return err
	}

	return nil
}

func checkValidEnvNames(envs []string) error {
	for _, env := range envs {
		if !validEntityNamePattern.MatchString(env) {
			return fmt.Errorf("invalid env name %q: must be kebab-case and start with a lowercase letter", env)
		}
	}
	return nil
}
