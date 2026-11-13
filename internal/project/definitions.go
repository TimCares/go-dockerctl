// Definitions of dockerctl conventions.
package project

import (
	"fmt"
)

const DockerctlProjectConfigFileName string = "dockerctl.yaml"
const SecretsDirName string = ".secrets"
const ConfigDirName string = "config"
const TemplatesDirName string = "templates"
const NamespacesDirName string = "namespaces"

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
