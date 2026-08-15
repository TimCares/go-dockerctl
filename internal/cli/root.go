package cli

import (
	"context"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/logger"
)

const defaultConfigFileName = "dockerctl.yaml"

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
				Name:    "project",
				Usage:   "Path to the dockerctl project",
				Value:   ".",
				Sources: cli.EnvVars("DOCKERCTL_PROJECT_PATH"),
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Path to the dockerctl config file",
				DefaultText: "<project>/dockerctl.yaml",
				Sources:     cli.EnvVars("DOCKERCTL_CONFIG_FILE"),
			},
			&cli.StringFlag{
				Name:  "env",
				Usage: "On which env to perform an operation",
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
				Value:   logger.DefaultLogFile(),
				Sources: cli.EnvVars("DOCKERCTL_LOG_FILE"),
			},
		},
		Commands: []*cli.Command{
			identityCommand,
		},
		Before: beforeRoot,
	}
}

func beforeRoot(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if err := resolveConfigPath(cmd); err != nil {
		return ctx, err
	}
	return initLogger(ctx, cmd)
}

func resolveConfigPath(cmd *cli.Command) error {
	if cmd.IsSet("config") {
		return nil
	}
	return cmd.Set("config", filepath.Join(cmd.String("project"), defaultConfigFileName))
}

func initLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	return ctx, logger.Init(
		cmd.String("log-level"),
		cmd.String("log-format"),
		cmd.String("log-file"),
	)
}
