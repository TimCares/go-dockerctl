package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	infractl "github.com/TimCares/go-infractl"
	"github.com/TimCares/go-infractl/internal/logger"
)

func New() *cli.Command {
	return &cli.Command{
		Name:                   "infractl",
		Usage:                  "infractl CLI",
		Description:            "A Go CLI tool for managing multiple docker compose projects with SOPS encryption.",
		Version:                infractl.Version,
		EnableShellCompletion:  true,
		Suggest:                true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "log `LEVEL`: debug, info, warn, error",
				Value:   "info",
				Sources: cli.EnvVars("INFRACTL_LOG_LEVEL"),
			},
			&cli.StringFlag{
				Name:    "log-format",
				Usage:   "log `FORMAT`: console or json",
				Value:   "console",
				Sources: cli.EnvVars("INFRACTL_LOG_FORMAT"),
			},
		},
		Commands: []*cli.Command{
			identityCommand,
		},
		Before: initLogger,
	}
}

func initLogger(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	return ctx, logger.Init(cmd.String("log-level"), cmd.String("log-format"))
}
