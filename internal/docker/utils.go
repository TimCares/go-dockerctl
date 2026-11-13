// Package docker wraps the docker compose library used to load and validate compose files.
package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
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

// ValidateDockerComposeFile checks that a compose file exists, contains no
// ${VAR} placeholders, and parses as a valid compose project.
func ValidateDockerComposeFile(ctx context.Context, projectName string, workingDirectory string, dockerComposeFilePath string) error {
	dockerComposeBody, err := os.ReadFile(dockerComposeFilePath) // Do not use os.Stat here, as we are also interested in the contents.
	if err != nil {
		return fmt.Errorf("reading docker compose file %s: %w", dockerComposeFilePath, err)
	}

	if envVariablePattern.Match(dockerComposeBody) {
		return fmt.Errorf("env variables not allowed in docker compose file %s: %w", dockerComposeFilePath, errEnvVariablePlaceholder)
	}

	composeService, err := GetDockerComposeService()
	if err != nil {
		return err
	}

	// TODO: Validate compose file only after templating
	// TODO: "name" key in compose file should be the same as namespace (check after templating)
	// BUT: docker module should have no knowledge of Namespace.

	// This validates the compose format as a side effect, which is the only thing we are currently interested it.
	_, err = composeService.LoadProject(ctx, api.ProjectLoadOptions{
		ProjectName: projectName,
		ConfigPaths: []string{dockerComposeFilePath},
		WorkingDir:  workingDirectory,
	})
	if err != nil {
		return fmt.Errorf("invalid compose file %s: %w", dockerComposeFilePath, err)
	}

	return nil
}
