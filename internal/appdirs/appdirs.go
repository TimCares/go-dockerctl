// Package appdirs owns every path dockerctl uses on the current machine, outside of any project.
//
// Paths inside a project are owned by the project package.
package appdirs

import (
	"path/filepath"

	"github.com/google/uuid"
)

const (
	appDirName       = "dockerctl"
	projectsFileName = "projects.json"
	projectsDirName  = "projects"
	logFileName      = "dockerctl.log"
)

// ProjectsFile is the registry of all dockerctl projects on this machine.
func ProjectsFile() (string, error) {
	base, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName, projectsFileName), nil
}

// ProjectEnvDir holds the machine-local files of one env of a project, e.g. its age key.
//
// It is keyed by the project id, not the path, so it survives moving the project.
func ProjectEnvDir(projectID uuid.UUID, env string) (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName, projectsDirName, projectID.String(), env), nil
}

// LogFile is the default internal log file.
func LogFile() (string, error) {
	base, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName, logFileName), nil
}
