// We need a functionality to get the path to the sops files, that is currently the only reason,
// as well as project separation and robustness of project moves.
package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TimCares/go-dockerctl/internal/definitions"
	"github.com/TimCares/go-dockerctl/internal/project"
	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/TimCares/go-see"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// The major version of the projects.json file,
// containing the definition of all dockerctl projects on the current machine.
//
// Only the major version, as patch and minor releases should not have
// breaking changes that require migrations.
// We do not couple it to dockerctl.Version, as we can make a major/breaking
// release to the code without the file format having a breaking change, but not the
// other way around.
const dockerctlProjectsFileAPIVersion uint = 1

type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type DockerctlProjects struct {
	APIVersion uint                  `json:"apiVersion"`
	Projects   map[uuid.UUID]Project `json:"projects"`
}

// defaultDockerctlProjects returns an empty collection of projects
// and the current ProjectsFile api version the code supports.
func defaultDockerctlProjects() DockerctlProjects {
	return DockerctlProjects{
		APIVersion: dockerctlProjectsFileAPIVersion,
	}
}

func projectsFilePath() string {
	parent := definitions.DockerctlAppDataDir()
	return filepath.Join(parent, definitions.DockerctlAppProjectsFileName)
}

func readProjectsFile() (*DockerctlProjects, error) {
	projectsFilePath := projectsFilePath()

	projectsFileBody, err := os.ReadFile(projectsFilePath)

	var projects DockerctlProjects

	if os.IsNotExist(err) {
		projects = defaultDockerctlProjects()
		return &projects, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading projects file %s: %w", projectsFilePath, err)
	}

	extractedVersion, err := utils.CheckMajorAPIVersionMatch(projectsFileBody, dockerctlProjectsFileAPIVersion)
	if err != nil {
		return nil, fmt.Errorf("checking projects file api version %s: %w", projectsFilePath, err)
	}
	if *extractedVersion != dockerctlProjectsFileAPIVersion {
		// TODO: apply migrations later if necessary.
	}
	// Here we kow that the versions must match, because utils.CheckMajorAPIVersionMatch
	// throws if the extracted version is higher than the one we currently have.

	if err := json.Unmarshal(projectsFileBody, &projects); err != nil {
		return nil, fmt.Errorf("parsing projects file %s: %w", projectsFilePath, err)
	}
	return &projects, nil
}

func writeProjectsFile(projects *DockerctlProjects) error {
	// Update api version if it is different: Latest change comes
	// from code with the version given by dockerctlProjectsFileAPIVersion:
	projects.APIVersion = dockerctlProjectsFileAPIVersion

	data, err := json.MarshalIndent(projects, "", "  ") // Indent by 2.
	if err != nil {
		return err
	}

	projectsPath := projectsFilePath()

	if err := os.WriteFile(projectsPath, data, utils.SecretFilePerm); err != nil {
		return err
	}
	return nil
}

func RegisterProject(name string, path string) (*project.ProjectMetadata, error) {
	projects, err := readProjectsFile()
	if err != nil {
		return nil, err
	}

	projectMetadata, projects, err := registerProject(name, path, projects)
	if err != nil {
		return nil, err
	}

	err = writeProjectsFile(projects)
	if err != nil {
		return nil, err
	}

	// Do not create the dir for the project (includes e.g. age keys) yet, as we do it on demand.

	return projectMetadata, nil
}

func registerProject(name string, path string, projects *DockerctlProjects) (*project.ProjectMetadata, *DockerctlProjects, error) {
	projectUUID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("creating id for project %s: %w", name, err)
	}

	projects.Projects[projectUUID] = Project{
		Name: name,
		Path: path,
	}

	metadata := project.ProjectMetadata{
		Id: projectUUID,
	}

	return &metadata, projects, nil

}

func checkDuplicateProject(
	ctx context.Context,
	currentProjectMetadata *project.ProjectMetadata,
	otherProjectPath string,
) (bool, error) {

	info, err := os.Stat(otherProjectPath)
	if os.IsNotExist(err) {
		see.L(ctx).Debug("Project was moved, no duplicate project.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat: checking other project path %s: %w", otherProjectPath, err)
	}
	if !info.IsDir() {
		see.L(ctx).Debug("Other project path is a file, so it cannot be a duplicate project.")
		return false, nil
	}

	// TODO: Check if old project path is still a valid dockerctl project?

	oldProjectMetadata, err := project.ReadProjectMetaFile(otherProjectPath)
	if err != nil {
		return false, fmt.Errorf("reading other project meta file %s: %w", oldProjectMetadata, err)
	}

	if oldProjectMetadata.Id != currentProjectMetadata.Id {
		see.L(ctx).Debug("Other project path poins to a dockerctl project with a different id, no duplicate project.",
			zap.String("otherProjectId", oldProjectMetadata.Id.String()),
		)
		return false, nil
	}

	see.L(ctx).Debug("Found two projects with the same id")
	return true, nil
}

type ProjectRegisteredCheckOpts struct {
	ProjectName     string
	ProjectPath     string
	ProjectMetadata *project.ProjectMetadata
}

// CheckProjectRegistered
//
// Note: Performs write operations in its own application directory to keep track of all projects.
func CheckProjectRegistered(ctx context.Context, opts ProjectRegisteredCheckOpts) (bool, error) {
	projects, err := readProjectsFile()
	if err != nil {
		return false, err
	}

	registered, err := checkProjectRegistered(ctx, opts, projects)
	if err != nil {
		return registered, err
	}

	if registered {
		// Keep project data fresh:
		// (Might be a noop of neither name or path have changed.)
		projects.Projects[opts.ProjectMetadata.Id] = Project{
			Name: opts.ProjectName,
			Path: opts.ProjectPath,
		}

		// Note: A write might be unexpected to clients, as a function starting with "Check"
		// looks like just a read op. However, the point of the write is to hide the "complexity"
		// of global project handling to the client, so we just inform any caller via the docs of the func.
		err = writeProjectsFile(projects)
		if err != nil {
			return registered, err
		}
	}

	return registered, nil
}

func checkProjectRegistered(ctx context.Context, opts ProjectRegisteredCheckOpts, projects *DockerctlProjects) (bool, error) {
	ctx = context.WithValue(ctx, "projectName", opts.ProjectName)
	ctx = context.WithValue(ctx, "projectPath", opts.ProjectPath)
	ctx = context.WithValue(ctx, "projectId", opts.ProjectMetadata.Id.String())

	project, ok := projects.Projects[opts.ProjectMetadata.Id]
	if !ok {
		return false, nil
	}

	if project.Path != opts.ProjectPath {
		otherProjectPath := project.Path
		conflictCtx := context.WithValue(ctx, "otherProjectPath", otherProjectPath)

		see.L(conflictCtx).Debug("Project path does not match the path recorded, checking for duplicate project.")

		projectIsDuplicate, err := checkDuplicateProject(conflictCtx, opts.ProjectMetadata, otherProjectPath)
		if err != nil {
			return true, fmt.Errorf("checking for duplicate project at %s: %w", otherProjectPath, err)
		}
		if projectIsDuplicate {
			return true, fmt.Errorf("found duplicate project: %s and %s", opts.ProjectPath, otherProjectPath)
		}

		see.L(conflictCtx).Debug("No duplicate found, all good.")
	}

	return true, nil

}
