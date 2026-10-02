package filesystem

import (
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/config"
)

func makeTemplateValuesDirStruct(envs []string, secrets bool) Dir {
	sopsExt := ""
	if secrets {
		sopsExt = ".sops"
	}

	defaultsFileName := fmt.Sprintf("defaults%s.yaml", sopsExt)
	secretsDirStructure := Dir{
		defaultsFileName: Optional{
			Node: File{},
		},
	}

	for _, env := range envs {
		filename := fmt.Sprintf("values.%s%s.yaml", env, sopsExt)
		secretsDirStructure[filename] = Optional{
			Node: File{},
		}
	}

	return secretsDirStructure
}

func makeServiceGroupDir(serviceGroup *config.ServiceGroup) Dir {
	return Dir{
		".secrets": Optional{
			Node: makeTemplateValuesDirStruct(serviceGroup.Envs, true),
		},
		"config": Optional{
			Node: makeTemplateValuesDirStruct(serviceGroup.Envs, false),
		},
		"templates":                    Dir{},
		serviceGroup.DockerComposeFile: File{},
	}
}

func makeProjectDir(cfg *config.Config) Dir {
	return Dir{
		"dockerctl.yaml": File{},
		".sops.yaml":     File{},
		".secrets": Optional{
			Node: makeTemplateValuesDirStruct(cfg.Envs, true),
		},
		"config": Optional{
			Node: makeTemplateValuesDirStruct(cfg.Envs, false),
		},
		"templates": Dir{},
	}
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
