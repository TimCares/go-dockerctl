package docker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/docker"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"go.uber.org/zap"
)

var dockerComposeService *api.Compose

func GetDockerComposeService() (*api.Compose, error) {
	if dockerComposeService != nil {
		return dockerComposeService, nil
	}

	dockerCLI, dockerCLIErr := command.NewDockerCli()
	if dockerCLIErr != nil {
		return nil, dockerCLIErr
	}

	if initErr := dockerCLI.Initialize(&flags.ClientOptions{}); initErr != nil {
		return nil, initErr
	}

	composeService, composeErr := compose.NewComposeService(dockerCLI)
	if composeErr != nil {
		return nil, composeErr
	}

	dockerComposeService = &composeService

	return dockerComposeService, nil
}

var envVariablePattern = regexp.MustCompile(
	`\$\{[A-Za-z_][A-Za-z0-9_]*(?::?-[^}]*)?\}`,
)

func ValidateDockerComposeFile(serviceGroup *config.ServiceGroup) error {
	dockerComposeFilePath := filepath.Join(serviceGroup.Path, serviceGroup.DockerComposeFile)
	dockerComposeBody, readErr := os.ReadFile(dockerComposeFilePath) // Do not use os.Stat here, as we are also interested in the contents.

	if readErr != nil {
		zap.L().Error(readErr.Error(), zap.String("serviceGroupName", serviceGroup.Name), zap.String("dockerComposeFilePath", dockerComposeFilePath))
		return readErr
	}

	composeService, getComposeErr := docker.GetDockerComposeService()
	if getComposeErr != nil {
		return getComposeErr
	}

	ctx := context.Background()

	// This validates the compose format as a side effect, which is the only thing we are currently interested it.
	// Later, this operation will be performed again when actually starting the service group.
	_, composeLoadErr := (*composeService).LoadProject(ctx, api.ProjectLoadOptions{
		ProjectName: serviceGroup.Name,
		ConfigPaths: []string{dockerComposeFilePath},
		WorkingDir:  serviceGroup.Path,
	})
	if composeLoadErr != nil {
		return composeLoadErr
	}

	if envVariablePattern.Match(dockerComposeBody) { // Checks for any match.
		err := errors.New("Docker compose file contains env variable placeholders, which are not allowed in dockerctl. Use Go templating '{{ value }}' instead.")
		zap.L().Error(err.Error(), zap.String("serviceGroupName", serviceGroup.Name), zap.String("dockerComposeFilePath", dockerComposeFilePath))
		return err
	}

	return nil
}
