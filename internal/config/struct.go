package config

type RuntimeConfig struct {
	ProjectDir string
	ActiveEnv  string
}

const ServiceGroupsDefaultDirName = "service-groups"
const DockerComposeDefaultFileName = "docker-compose.yaml"

type ServiceGroup struct {
	Name              string           `yaml:"name"`
	Path              string           `yaml:"path"`
	DockerComposeFile string           `yaml:"docker_compose_file"`
	Envs              UniqueStringList `yaml:"envs,omitempty"` // nil (key absent) means all global envs.
	DisableDeploy     bool             `yaml:"disable_deploy"` // Helpful if you have a service group that should not always be deployed, but want to list it for consistency.
}

type Config struct {
	Runtime              RuntimeConfig    `yaml:"-"`
	Name                 string           `yaml:"name"`
	Envs                 UniqueStringList `yaml:"envs"`
	ManageSOPSIdentities bool             `yaml:"manage_sops_identities"`
	ServiceGroups        []ServiceGroup   `yaml:"service_groups"`
}
