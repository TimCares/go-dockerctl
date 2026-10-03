package config

// RuntimeConfig holds values resolved from CLI flags rather than the config file.
type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

// Defaults applied to a [ServiceGroup] when the config file leaves the field empty.
const (
	ServiceGroupsDefaultDirName  = "service-groups"
	DockerComposeDefaultFileName = "docker-compose.yaml"
)

// ServiceGroup is a single docker compose project managed by dockerctl.
type ServiceGroup struct {
	Name              string           `yaml:"name"`
	Path              string           `yaml:"path"`
	DockerComposeFile string           `yaml:"dockerComposeFile"`
	Envs              UniqueStringList `yaml:"envs,omitempty"` // nil (key absent) means all global envs.
	DisableDeploy     bool             `yaml:"disable_deploy"` // Helpful if you have a service group that should not always be deployed, but want to list it for consistency.
}

// configVersion is decoded before [Config], so an incompatible file is rejected before its schema is parsed.
type configVersion struct {
	APIVersion uint `yaml:"apiVersion"`
}

// Config is the parsed dockerctl.yaml file.
type Config struct {
	APIVersion           string           `yaml:"apiVersion"`
	Name                 string           `yaml:"name"`
	Envs                 UniqueStringList `yaml:"envs"`
	ManageSOPSIdentities bool             `yaml:"manageSopsIdentities"`
	ServiceGroups        []ServiceGroup   `yaml:"serviceGroups"`

	Runtime RuntimeConfig `yaml:"-"`
}
