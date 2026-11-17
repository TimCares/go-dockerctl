package observability

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/TimCares/go-see"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Shared by the log fields and the metric resource.
const (
	serviceName    = "dockerctl"
	projectNameKey = "dockerctl.project.name"
)

const envKey = "deployment.environment.name"

// OtelLogData describes one command run. It is the same for logs and metrics.
type OtelLogData struct {
	DockerctlVersion          string
	DockerctlConfigAPIVersion uint
	ProjectName               string
	CurrentHostname           string
	ExtraFields               map[string]string
}

// OtelLogFields describe one command run and are attached to every event emitted during it.
//
// Keys follow the OpenTelemetry semantic conventions where one exists, and are
// namespaced under "dockerctl." otherwise. Extra fields from the config go under
// "labels", so they can never collide with these keys.
//
// cmd must be the command whose action runs. The command passed to a Before hook
// is the one that owns the hook, not the leaf.
func OtelLogFields(data OtelLogData, cmd *cli.Command) []zap.Field {
	fields := []zap.Field{
		zap.String("service.name", serviceName),
		zap.String("service.version", data.DockerctlVersion),

		zap.String(projectNameKey, data.ProjectName),
		zap.Uint("dockerctl.config.api_version", data.DockerctlConfigAPIVersion),
		zap.String("dockerctl.command.name", strings.Join(cmd.Path()[1:], " ")),
	}

	if data.CurrentHostname != "" {
		fields = append(fields, zap.String("host.name", data.CurrentHostname))
	}
	if flags := setFlags(cmd); len(flags) > 0 {
		fields = append(fields, zap.Object("dockerctl.command.flags", stringMap(flags)))
	}
	if extra := data.ExtraFields; len(extra) > 0 {
		fields = append(fields, zap.Object("labels", stringMap(extra)))
	}
	return fields
}

// WithEnv attaches the env an operation runs for to every event emitted with the returned ctx.
//
// The env is not part of [OtelLogData], because a single command usually operates on several envs.
func WithEnv(ctx context.Context, env string) context.Context {
	return see.With(ctx, zap.String(envKey, env))
}

// ForEachEnv calls fn once per env, with the env attached to ctx through [WithEnv].
// It stops at the first error.
func ForEachEnv(ctx context.Context, envs []string, fn func(ctx context.Context, env string) error) error {
	for _, env := range envs {
		if err := fn(WithEnv(ctx, env), env); err != nil {
			return err
		}
	}
	return nil
}

// setFlags returns the flags explicitly set on cmd or its ancestors, keyed by primary name.
// Defaults are left out, so the log shows what the user actually passed.
func setFlags(cmd *cli.Command) map[string]string {
	flags := map[string]string{}
	for _, c := range cmd.Lineage() {
		for _, f := range c.Flags {
			if !f.IsSet() {
				continue
			}
			name := f.Names()[0]
			if _, seen := flags[name]; !seen {
				flags[name] = fmt.Sprint(cmd.Value(name))
			}
		}
	}
	return flags
}

type stringMap map[string]string

func (m stringMap) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, k := range slices.Sorted(maps.Keys(m)) {
		enc.AddString(k, m[k])
	}
	return nil
}
