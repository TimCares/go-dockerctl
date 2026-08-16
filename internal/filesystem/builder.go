package filesystem

import (
	"fmt"

	configModule "github.com/TimCares/go-dockerctl/internal/config"
)

func makeTemplateValuesDirStruct(envs *[]string, secrets bool) Dir {
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

	for _, env := range *envs {
		filename := fmt.Sprintf("values.%s%s.yaml", env, sopsExt)
		secretsDirStructure[filename] = Optional{
			Node: File{},
		}
	}

	return secretsDirStructure
}

func makeServiceGroupDir(envs *[]string, serviceGroupConfig *configModule.ServiceGroup) Dir {
	return Dir{
		".secrets": Optional{
			Node: makeTemplateValuesDirStruct(envs, true),
		},
		"config": Optional{
			Node: makeTemplateValuesDirStruct(envs, false),
		},
		"templates":                          Dir{},
		serviceGroupConfig.DockerComposeFile: File{},
	}
}

func MakeDockerctlFilesystem(config *configModule.Config) Dir {
	projectStructure := Dir{
		"dockerctl.yaml": File{},
		".sops.yaml":     File{},
		".secrets": Optional{
			Node: makeTemplateValuesDirStruct(&config.Envs, true),
		},
		"config": Optional{
			Node: makeTemplateValuesDirStruct(&config.Envs, false),
		},
		"templates":                              Dir{},
		configModule.ServiceGroupsDefaultDirName: Dir{},
	}

	for _, serviceGroup := range config.ServiceGroups {
		projectStructure[configModule.ServiceGroupsDefaultDirName][serviceGroup.Name] = makeServiceGroupDir(&config.Envs, &serviceGroup)
	}

	return projectStructure
}
