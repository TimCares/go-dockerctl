package definitions

import (
	"path/filepath"

	"github.com/TimCares/go-dockerctl/internal/utils"
)

const DockerctlAppDirName string = "dockerctl"
const DockerctlAppProjectsFileName string = "projects.json"
const DockerctlAppProjectsDirName string = "projects"

func DockerctlAppConfigDir() (string, error) {
	parent, err := utils.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(parent, DockerctlAppDirName)
	return path, nil
}

func DockerctlAppDataDir() string {
	parent := utils.UserAppDataDir()
	path := filepath.Join(parent, DockerctlAppDirName)
	return path
}
