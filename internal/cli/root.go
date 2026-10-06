// Package cli defines the dockerctl command tree, its flags, and the setup that runs before every command.
package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/TimCares/go-see"
	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/observability"
	"github.com/TimCares/go-dockerctl/internal/observability/events"
)

const (
	shellCompletionFlag   = "--generate-shell-completion"
	defaultConfigFileName = "dockerctl.yaml"
)

// New builds the root dockerctl command with its global flags and subcommands.
func New() *cli.Command {
	root := &cli.Command{
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
				Value:   config.DefaultLogFile(),
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
		Before: beforeRoot,
	}

	// Commands are built fresh on every call, so each action is wrapped exactly once.
	_ = root.Walk(func(c *cli.Command) error {
		if c.Action != nil {
			c.Action = withLogContext(c.Action)
		}
		return nil
	})
	return root
}

func beforeRoot(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	settings := config.Settings{
		ProjectPath: cmd.String("project-path"),
		ConfigPath:  cmd.String("config-path"),
		Env:         cmd.String("env"),
		LogLevel:    optionalString(cmd, "log-level"),
		LogFormat:   optionalString(cmd, "log-format"),
		LogFile:     optionalString(cmd, "log-file"),
	}

	if settings.ConfigPath == "" {
		settings.ConfigPath = filepath.Join(settings.ProjectPath, defaultConfigFileName)
	}
	cfg, err := config.Load(ctx, settings)
	if err != nil {
		return ctx, err
	}

	// Tab completion should not open a collector connection.
	obs := cfg.Observability
	if shellCompletion() {
		obs.Otel.Enabled = false
	}

	if err := observability.Init(ctx, obs); err != nil {
		return ctx, err
	}

	see.Emit(ctx, events.ConfigDump{Cfg: *cfg})

	return config.ContextWithConfig(ctx, *cfg), nil
}

// withLogContext attaches [observability.OtelLogFields] for the running command.
// It wraps the action rather than living in beforeRoot,
// because Before receives the root command instead of the leaf.
func withLogContext(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cfg, err := config.FromContext(ctx)
		if err != nil {
			return err
		}
		ctx = see.With(ctx, observability.OtelLogFields(cfg, cmd)...)
		return action(ctx, cmd)
	}
}

func optionalString(cmd *cli.Command, name string) *string {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.String(name)
	return &value
}

func shellCompletion() bool {
	return len(os.Args) > 0 && os.Args[len(os.Args)-1] == shellCompletionFlag
}
