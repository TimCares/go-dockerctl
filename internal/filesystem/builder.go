package filesystem

import (
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/config"
)

func makeTemplateValuesDir(envs []string, secret bool) Dir {
	sopsExt := ""
	if secret {
		sopsExt = ".sops"
	}

	file := Optional{Node: File{Secret: secret}}
	children := map[string]Node{
		fmt.Sprintf("defaults%s.yaml", sopsExt): file,
	}
	for _, env := range envs {
		// Repeated assignment of "file" is ok, as structs are copied by value.
		children[fmt.Sprintf("values.%s%s.yaml", env, sopsExt)] = file
	}

	return Dir{Secret: secret, Children: children}
}

func makeServiceGroupDir(serviceGroup *config.ServiceGroupConfig) Dir {
	return Dir{Children: map[string]Node{
		".secrets":                     Optional{Node: makeTemplateValuesDir(serviceGroup.Envs, true)},
		"config":                       Optional{Node: makeTemplateValuesDir(serviceGroup.Envs, false)},
		"templates":                    Dir{},
		serviceGroup.DockerComposeFile: File{},
	}}
}

func makeProjectDir(cfg *config.Config) Dir {
	rootSecretsConfig := makeTemplateValuesDir(cfg.Envs, true)
	rootSecretsConfig.Children["dockerctl.sops.yaml"] = Optional{File{Secret: true}}

	return Dir{Children: map[string]Node{
		"dockerctl.yaml": File{},
		".sops.yaml":     File{},
		".secrets":       Optional{Node: rootSecretsConfig},
		"config":         Optional{Node: makeTemplateValuesDir(cfg.Envs, false)},
		"templates":      Dir{},
	}}
}

// ValidateProject checks the project root and every service group directory.
// Service groups are validated at their resolved Path, meaning we allow
// groups living outside the default service-groups directory.
// However, it is encouraged to keep them in the default "service-groups" dir.
func ValidateProject(cfg *config.Config) error {
	if err := makeProjectDir(cfg).Validate(cfg.Runtime.ProjectDir); err != nil {
		return err
	}

	for i := range cfg.ServiceGroups {
		serviceGroup := &cfg.ServiceGroups[i]
		if err := makeServiceGroupDir(serviceGroup).Validate(serviceGroup.Path); err != nil {
			return fmt.Errorf("service group %q: %w", serviceGroup.Name, err)
		}
	}

	return nil
}
