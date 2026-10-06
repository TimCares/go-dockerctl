package config

import "os"

// RuntimeConfig holds values resolved from CLI flags rather than the config file.
type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

// Defaults applied to a [ServiceGroupConfig] when the config file leaves the field empty.
const (
	ServiceGroupsDefaultDirName  = "service-groups"
	DockerComposeDefaultFileName = "docker-compose.yaml"
)

// ServiceGroupConfig is a single docker compose project managed by dockerctl.
type ServiceGroupConfig struct {
	Name              string           `yaml:"name"`
	Path              string           `yaml:"path"`
	DockerComposeFile string           `yaml:"dockerComposeFile"`
	Envs              UniqueStringList `yaml:"envs,omitempty"` // nil (key absent) means all global envs.
	DisableDeploy     bool             `yaml:"disable_deploy"` // Helpful if you have a service group that should not always be deployed, but want to list it for consistency.
}

// configVersion is decoded before [Config],
// so an incompatible file is rejected before its schema is parsed.
type configVersion struct {
	APIVersion uint `yaml:"apiVersion"`
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
	APIVersion           uint                 `yaml:"apiVersion"`
	Name                 string               `yaml:"name"`
	Envs                 UniqueStringList     `yaml:"envs"`
	ManageSOPSIdentities bool                 `yaml:"manageSopsIdentities"`
	ServiceGroups        []ServiceGroupConfig `yaml:"serviceGroups"`
	Observability        ObservabilityConfig  `yaml:"observability"`
	Node                 NodeInfoConfig       `yaml:"node,omitempty"`

	Runtime RuntimeConfig `yaml:"-"`
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
		APIVersion:           configAPIVersion,
		ManageSOPSIdentities: true,
		Observability:        DefaultObservabilityConfig(),
		Node:                 DefaultNodeInfoConfig(),
	}
}
