package cli

import (
	"context"
	"fmt"
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

// Project is a loaded, validated and registered dockerctl project.
type Project struct {
	Settings CLISettings
	Config   config.Config
	Metadata *project.ProjectMetadata

	// Envs are the envs to operate on for the current command:
	// --envs if given, else all envs of the config. Always a subset of Config.Envs.
	Envs []string
}

// ProjectActionFunc is an action that operates on a dockerctl project.
type ProjectActionFunc func(ctx context.Context, cmd *cli.Command, p Project) error

// Setup lives in action wrappers rather than Before hooks, because a Before hook receives
// the command owning it instead of the leaf, and is also run for shell completion.

// withProject loads and validates the config, resolves the envs to operate on,
// sets up observability and ensures the project is registered.
func withProject(action ProjectActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		settings := readSettings(cmd)

		cfg, err := loadConfig(settings)
		if err != nil {
			return err
		}

		envs, err := resolveEnvs(settings.Envs, cfg.Envs)
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

		if err := observability.Init(ctx, cfg.Observability, otelLogData); err != nil {
			return err
		}

		ctx = see.With(ctx, observability.OtelLogFields(otelLogData, cmd)...)

		see.Emit(ctx, events.ConfigDump{Cfg: *cfg})

		metadata, err := ensureProjectRegistered(ctx, cfg.Name, settings.ProjectPath)
		if err != nil {
			return err
		}

		return action(ctx, cmd, Project{
			Settings: settings,
			Config:   *cfg,
			Metadata: metadata,
			Envs:     envs,
		})
	}
}

// readSettings reads the global flags. Flags of ancestors are visible from the leaf command.
func readSettings(cmd *cli.Command) CLISettings {
	settings := CLISettings{
		ProjectPath: cmd.String("project-path"),
		ConfigPath:  cmd.String("config-path"),
		Envs:        cmd.StringSlice("envs"),
		LogLevel:    optionalString(cmd, "log-level"),
		LogFormat:   optionalString(cmd, "log-format"),
		LogFile:     optionalString(cmd, "log-file"),
	}
	if settings.ConfigPath == "" {
		settings.ConfigPath = filepath.Join(settings.ProjectPath, project.DockerctlProjectConfigFileName)
	}
	return settings
}

func loadConfig(settings CLISettings) (*config.Config, error) {
	cfg, err := config.LoadConfig(settings.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	applyLogSettings(&cfg.Observability, settings) // Before we validate!

	if err := cfg.Validate(settings.ProjectPath); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

// resolveEnvs picks the envs to operate on.
//
// configEnvs are the available envs, while requested are the ones the current command operates on,
// so requested never overrides the envs of the config.
// configEnvs must be validated, which also guarantees the naming pattern of the returned envs.
func resolveEnvs(requested, configEnvs []string) ([]string, error) {
	if len(requested) == 0 {
		return configEnvs, nil
	}
	if invalidEnvs := utils.FindExcluded(requested, configEnvs); len(invalidEnvs) > 0 {
		return nil, fmt.Errorf("envs %v not found in list of valid environments %v", invalidEnvs, configEnvs)
	}
	return requested, nil
}

func applyLogSettings(obs *observability.ObservabilityConfig, settings CLISettings) {
	if settings.LogLevel != nil {
		obs.Logging.LogLevel = *settings.LogLevel
	}
	if settings.LogFormat != nil {
		obs.Logging.LogFormat = *settings.LogFormat
	}
	if settings.LogFile != nil {
		obs.Logging.LogFile = *settings.LogFile
	}
}

// ensureProjectRegistered returns the metadata of the project, registering it first if needed.
func ensureProjectRegistered(ctx context.Context, projectName string, projectPath string) (*project.ProjectMetadata, error) {
	projectMetadata, err := project.ReadProjectMetaFile(projectPath)
	if err != nil {
		return nil, err
	}

	registered := false

	if projectMetadata != nil {
		checkOpts := manager.ProjectRegisteredCheckOpts{
			ProjectName:     projectName,
			ProjectPath:     projectPath,
			ProjectMetadata: projectMetadata,
		}
		registered, err = manager.CheckProjectRegistered(ctx, checkOpts)
		if err != nil {
			return nil, fmt.Errorf("checking project registered %s: %w", projectName, err)
		}
	}

	if !registered {
		projectMetadata, err = manager.RegisterProject(projectName, projectPath)
		if err != nil {
			return nil, fmt.Errorf("registering project %s: %w", projectName, err)
		}
		if err := project.WriteProjectMetaFile(projectPath, projectMetadata); err != nil {
			return nil, fmt.Errorf("writing project meta file of %s: %w", projectName, err)
		}
	}

	project.MaybeAddProjectMetaFileToGitignore(ctx, projectPath)

	return projectMetadata, nil
}

func optionalString(cmd *cli.Command, name string) *string {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.String(name)
	return &value
}
