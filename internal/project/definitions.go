// Definitions of dockerctl conventions.
package project

import (
	"fmt"
	"path/filepath"
)

// Every path inside a project is built from these, so the layout lives in one place.
// Paths on the machine, outside of any project, are owned by the appdirs package.
const (
	configFileName     = "dockerctl.yaml"
	sopsConfigFileName = ".sops.yaml"
	secretsDirName     = ".secrets"
	configDirName      = "config"
	templatesDirName   = "templates"
	namespacesDirName  = "namespaces"
	metaDirName        = ".dockerctl"
	metaFileName       = "project.json"
)

// ConfigFile is the dockerctl config file of the project at root.
func ConfigFile(root string) string {
	return filepath.Join(root, configFileName)
}

// SOPSConfigFile is the SOPS config of the project at root, holding the creation rules of all envs.
func SOPSConfigFile(root string) string {
	return filepath.Join(root, sopsConfigFileName)
}

// NamespaceDir is the default dir of a namespace, used when its config sets no path.
func NamespaceDir(root string, namespace string) string {
	return filepath.Join(root, namespacesDirName, namespace)
}

// SecretsDir holds the SOPS files of dir, which is either the project root or a namespace dir.
func SecretsDir(dir string) string {
	return filepath.Join(dir, secretsDirName)
}

// MetaFile holds the machine-local [ProjectMetadata] of the project at root.
func MetaFile(root string) string {
	return filepath.Join(root, metaDirName, metaFileName)
}

// buildFileName creates the yaml filename of a file containing application specific
// parameters for one or more services managed by dockerctl.
//
// env declares to which environment of the service group the file belongs.
// [""] means it is the defaults file and is applied to all environments.
//
// isSecret describes whether this file contains secrets.
func buildFileName(env string, isSecret bool) string {
	var prefix string
	if env == "" {
		prefix = "defaults"
	} else {
		prefix = fmt.Sprintf("values.%s", env)
	}
	var secretComponent string
	if isSecret {
		secretComponent = ".sops"
	} else {
		secretComponent = ""
	}
	return fmt.Sprintf("%s%s.yaml", prefix, secretComponent)
}
