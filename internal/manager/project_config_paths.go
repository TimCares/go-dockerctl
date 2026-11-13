package manager

import (
	"path/filepath"

	"github.com/TimCares/go-dockerctl/internal/definitions"
	"github.com/TimCares/go-dockerctl/internal/project"
)

func DockerctlProjectConfigDir(projectMetadata *project.ProjectMetadata) (string, error) {
	parent, err := definitions.DockerctlAppConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(parent, definitions.DockerctlAppProjectsDirName, projectMetadata.Id.String())
	return path, nil
}
