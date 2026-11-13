// Package cli defines the dockerctl command tree, its flags, and the setup that runs before every command.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TimCares/go-see"
	"github.com/urfave/cli/v3"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/config"
	"github.com/TimCares/go-dockerctl/internal/manager"
	"github.com/TimCares/go-dockerctl/internal/observability"
	"github.com/TimCares/go-dockerctl/internal/observability/events"
	"github.com/TimCares/go-dockerctl/internal/project"
	"github.com/TimCares/go-dockerctl/internal/utils"
)

const (
	shellCompletionFlag = "--generate-shell-completion"
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

func applyCLISettingsToConfig(cfg *config.Config, settings CLISettings) (*config.Config, error) {
	if settings.LogLevel != nil {
		cfg.Observability.Logging.LogLevel = *settings.LogLevel
	}
	if settings.LogFormat != nil {
		cfg.Observability.Logging.LogFormat = *settings.LogFormat
	}
	if settings.LogFile != nil {
		cfg.Observability.Logging.LogFile = *settings.LogFile
	}

	return cfg, nil
}

func beforeRoot(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	settings := CLISettings{
		ProjectPath: cmd.String("project-path"),
		ConfigPath:  cmd.String("config-path"),

		// config.Config.Envs tells us the available/possible Envs,
		// while this tells us on which Envs to operate for the current command!
		// => We do not use it to override the Envs in applyCLISettingsToConfig.
		Envs:      cmd.StringSlice("envs"),
		LogLevel:  optionalString(cmd, "log-level"),
		LogFormat: optionalString(cmd, "log-format"),
		LogFile:   optionalString(cmd, "log-file"),
	}

	if settings.ConfigPath == "" {
		settings.ConfigPath = filepath.Join(settings.ProjectPath, project.DockerctlProjectConfigFileName)
	}
	cfg, err := config.LoadConfig(settings.ConfigPath)
	if err != nil {
		return ctx, fmt.Errorf("loading config: %w", err)
	}

	cfg, err = applyCLISettingsToConfig(cfg, settings) // Before we validate!
	if err != nil {
		return ctx, fmt.Errorf("applying cli settings to config: %w", err)
	}

	if err := cfg.Validate(settings.ProjectPath); err != nil {
		return ctx, fmt.Errorf("validating config: %w", err)
	}

	if len(settings.Envs) == 0 {
		settings.Envs = cfg.Envs
	} else if invalidEnvs := utils.FindExcluded(settings.Envs, cfg.Envs); len(invalidEnvs) > 0 { // Only do this check after validation.
		// ... Also guarantees settings.Envs have the right regex pattern.
		return nil, fmt.Errorf("envs %v not found in list of valid environments %v", invalidEnvs, cfg.Envs)
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

	err = ensureProjectRegistered(ctx, cfg.Name, settings.ProjectPath)
	if err != nil {
		return ctx, err
	}

	ctx = config.ContextWithConfig(ctx, *cfg)

	return ctx, nil
}

func ensureProjectRegistered(ctx context.Context, projectName string, projectPath string) error {
	projectMetadata, err := project.ReadProjectMetaFile(projectPath)
	if err != nil {
		return err
	}

	var registered bool = false

	if projectMetadata != nil {
		checkOpts := manager.ProjectRegisteredCheckOpts{
			ProjectName:     projectName,
			ProjectPath:     projectPath,
			ProjectMetadata: projectMetadata,
		}
		registered, err = manager.CheckProjectRegistered(ctx, checkOpts)
		if err != nil {
			return fmt.Errorf("checking project registered %s: %w", projectName, err)
		}
	}

	if !registered {
		projectMetadata, err = manager.RegisterProject(projectName, projectPath)
		if err != nil {
			return fmt.Errorf("registering project %s: %w", projectName, err)
		}
		project.WriteProjectMetaFile(projectPath, projectMetadata)
	}

	project.MaybeAddProjectMetaFileToGitignore(ctx, projectPath)

	return nil
}

// withLogContext attaches [observability.OtelLogFields] for the running command.
// It wraps the action rather than living in beforeRoot,
// because Before receives the root command instead of the leaf.
func withLogContext(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cfg, err := config.ConfigFromContext(ctx)
		if err != nil {
			return err
		}
		otelLogData := observability.OtelLogData{
			DockerctlVersion:          dockerctl.Version,
			DockerctlConfigAPIVersion: cfg.APIVersion,
			ProjectName:               cfg.Name,
			CurrentHostname:           utils.Hostname(),
			ExtraFields:               cfg.Observability.Logging.ExtraFields,
		}
		ctx = see.With(ctx, observability.OtelLogFields(otelLogData, cmd)...)
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
