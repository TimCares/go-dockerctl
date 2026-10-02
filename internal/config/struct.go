package config

type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

const (
	ServiceGroupsDefaultDirName  = "service-groups"
	DockerComposeDefaultFileName = "docker-compose.yaml"
)

type ServiceGroup struct {
	Name              string           `yaml:"name"`
	Path              string           `yaml:"path"`
	DockerComposeFile string           `yaml:"dockerComposeFile"`
	Envs              UniqueStringList `yaml:"envs,omitempty"` // nil (key absent) means all global envs.
	DisableDeploy     bool             `yaml:"disable_deploy"` // Helpful if you have a service group that should not always be deployed, but want to list it for consistency.
}

type ConfigVersion struct {
	ApiVersion uint `yaml:"apiVersion"`
}

type Config struct {
	ApiVersion           string           `yaml:"apiVersion"`
	Name                 string           `yaml:"name"`
	Envs                 UniqueStringList `yaml:"envs"`
	ManageSOPSIdentities bool             `yaml:"manageSopsIdentities"`
	ServiceGroups        []ServiceGroup   `yaml:"serviceGroups"`

	Runtime RuntimeConfig `yaml:"-"`
}
