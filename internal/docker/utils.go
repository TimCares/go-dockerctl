package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"

	"github.com/TimCares/go-dockerctl/internal/config"
)

// GetDockerComposeService lazily creates a single compose service for the whole process.
var GetDockerComposeService = sync.OnceValues(func() (api.Compose, error) {
	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("creating docker cli: %w", err)
	}

	if err := dockerCLI.Initialize(&flags.ClientOptions{}); err != nil {
		return nil, fmt.Errorf("initializing docker cli: %w", err)
	}

	composeService, err := compose.NewComposeService(dockerCLI)
	if err != nil {
		return nil, fmt.Errorf("creating compose service: %w", err)
	}

	return composeService, nil
})

var envVariablePattern = regexp.MustCompile(
	`\$\{[A-Za-z_][A-Za-z0-9_]*(?::?-[^}]*)?\}`,
)

var errEnvVariablePlaceholder = errors.New("docker compose file contains env variable placeholders, which are not allowed in dockerctl; use Go templating '{{ value }}' instead")

func ValidateDockerComposeFile(ctx context.Context, serviceGroup *config.ServiceGroup) error {
	dockerComposeFilePath := filepath.Join(serviceGroup.Path, serviceGroup.DockerComposeFile)
	dockerComposeBody, err := os.ReadFile(dockerComposeFilePath) // Do not use os.Stat here, as we are also interested in the contents.
	if err != nil {
		return fmt.Errorf("service group %q: %w", serviceGroup.Name, err)
	}

	if envVariablePattern.Match(dockerComposeBody) {
		return fmt.Errorf("service group %q (%s): %w", serviceGroup.Name, dockerComposeFilePath, errEnvVariablePlaceholder)
	}

	composeService, err := GetDockerComposeService()
	if err != nil {
		return err
	}

	// This validates the compose format as a side effect, which is the only thing we are currently interested it.
	// Later, this operation will be performed again when actually starting the service group.
	_, err = composeService.LoadProject(ctx, api.ProjectLoadOptions{
		ProjectName: serviceGroup.Name,
		ConfigPaths: []string{dockerComposeFilePath},
		WorkingDir:  serviceGroup.Path,
	})
	if err != nil {
		return fmt.Errorf("service group %q: invalid compose file %s: %w", serviceGroup.Name, dockerComposeFilePath, err)
	}

	return nil
}
