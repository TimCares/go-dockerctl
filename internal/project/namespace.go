// Package project ties together config parsing, filesystem validation and
// docker compose validation. It sits above config, filesystem and docker in the
// dependency graph so those packages never have to import each other.
package project

import (
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/docker"
	fs "github.com/TimCares/go-dockerctl/internal/filesystem"
	"github.com/TimCares/go-dockerctl/internal/utils"
)

type ParameterGroup struct {
	IsSecret bool
	Envs     []string
}

func (p ParameterGroup) MakeFilesystemLayout() fs.Dir {
	file := fs.Optional{Node: fs.File{Secret: p.IsSecret}}

	children := map[string]fs.Node{
		buildFileName("", p.IsSecret): fs.Optional{Node: fs.File{Secret: p.IsSecret}},
	}
	for _, env := range p.Envs {
		// Repeated assignment of "file" is ok, as structs are copied by value.
		children[buildFileName(env, p.IsSecret)] = file
	}
	return fs.Dir{Secret: p.IsSecret, Children: children}
}

func (p ParameterGroup) IterFilesOrdered(envs []string) []string {
	files := []string{buildFileName("", p.IsSecret)}
	for _, env := range envs {
		files = append(files, buildFileName(env, p.IsSecret)) // Envs.
	}
	return files
}

type Namespace struct {
	Name         string
	Envs         []string
	secretsGroup ParameterGroup
	valuesGroup  ParameterGroup
}

func NewNamespace(name string, envs []string) Namespace {
	return Namespace{
		Name: name,
		Envs: envs,
		secretsGroup: ParameterGroup{
			IsSecret: true,
			Envs:     envs,
		},
		valuesGroup: ParameterGroup{
			IsSecret: false,
			Envs:     envs,
		},
	}
}
func (n Namespace) checkValidEnvs(envs []string) error {
	invalidEnvs := utils.FindExcluded(envs, n.Envs)
	if len(invalidEnvs) > 0 {
		return fmt.Errorf("invalid envs for namespace, found %q, allowed: %q", invalidEnvs, n.Envs)
	}
	return nil
}

func (n Namespace) iterParameterGroupFilesOrdered(p ParameterGroup, envs []string) ([]string, error) {
	err := n.checkValidEnvs(envs)
	if err != nil {
		return []string{}, nil
	}
	return p.IterFilesOrdered(envs), nil

}

func (n Namespace) IterValueFilesOrdered(envs []string) ([]string, error) {
	return n.iterParameterGroupFilesOrdered(n.valuesGroup, envs)
}

func (n Namespace) IterSecretFilesOrdered(envs []string) ([]string, error) {
	return n.iterParameterGroupFilesOrdered(n.secretsGroup, envs)
}

func (n Namespace) IterFilesOrdered(envs []string) ([]string, error) {
	valueFiles, err := n.IterValueFilesOrdered(envs)
	if err != nil {
		return []string{}, err
	}
	secretFiles, err := n.IterSecretFilesOrdered(envs)
	if err != nil {
		return []string{}, err
	}
	return append(valueFiles, secretFiles...), nil

}

func (n Namespace) Validate(path string) error {
	err := n.ValidateLayout(path)
	if err != nil {
		return fmt.Errorf("validating layout: %w", err)
	}

	return nil
}

func (n Namespace) ValidateLayout(path string) error {
	return n.makeFilesystemLayout().Validate(path)
}

func (n Namespace) makeFilesystemLayout() fs.Dir {
	return fs.Dir{Children: map[string]fs.Node{
		secretsDirName:               fs.Optional{Node: n.secretsGroup.MakeFilesystemLayout()},
		configDirName:                fs.Optional{Node: n.valuesGroup.MakeFilesystemLayout()},
		templatesDirName:             fs.Optional{Node: fs.Dir{}},
		docker.DockerComposeFileName: fs.File{},
	}}
}
