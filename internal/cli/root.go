// Package cli defines the dockerctl command tree, its flags, and the setup each command opts into.
package cli

import (
	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/observability"
)

// New builds the root dockerctl command with its global flags and subcommands.
//
// Actions that operate on a project are wrapped with [withProject].
func New() *cli.Command {
	return &cli.Command{
		Name:                   "dockerctl",
		Usage:                  "dockerctl CLI",
		Description:            "A Go CLI tool for managing multiple docker compose projects with SOPS encryption.",
		Version:                dockerctl.Version,
		EnableShellCompletion:  true,
		Suggest:                true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project-path",
				Usage:   "Path to the dockerctl project",
				Value:   ".",
				Sources: cli.EnvVars("DOCKERCTL_PROJECT_PATH"),
			},
			&cli.StringFlag{
				Name:        "config-path",
				Usage:       "Path to the dockerctl config file",
				DefaultText: "<project>/dockerctl.yaml",
				Sources:     cli.EnvVars("DOCKERCTL_CONFIG_FILE"),
			},
			&cli.StringSliceFlag{
				Name:        "envs",
				Usage:       "On which envs to perform the operation",
				DefaultText: "Envs of <project>/dockerctl.yaml",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "log `LEVEL`: debug, info, warn, error",
				Value:   "info",
				Sources: cli.EnvVars("DOCKERCTL_LOG_LEVEL"),
			},
			&cli.StringFlag{
				Name:    "log-format",
				Usage:   "log `FORMAT`: console or json",
				Value:   "console",
				Sources: cli.EnvVars("DOCKERCTL_LOG_FORMAT"),
			},
			&cli.StringFlag{
				Name:    "log-file",
				Usage:   "internal JSON log `PATH` (\"none\" disables)",
				Value:   observability.DefaultLogFile(),
				Sources: cli.EnvVars("DOCKERCTL_LOG_FILE"),
			},
			// TODO?
			// &cli.BoolFlag{
			// 	Name:    "enable-docker-env-var-templating",
			// 	Usage:   "Fall back to env var templating (${...}) in docker compose instead of go templating ({{ ... }})",
			// 	Value:   false,
			// 	Sources: cli.EnvVars("DOCKERCTL_ENABLE_DOCKER_ENV_VAR_TEMPLATING"),
			// },
		},
		Commands: []*cli.Command{
			newIdentityCommand(),
		},
	}
}
