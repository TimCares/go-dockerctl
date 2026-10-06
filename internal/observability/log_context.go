package observability

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/config"
)

// OtelLogFields describe one command run and are attached to every event emitted during it.
//
// Keys follow the OpenTelemetry semantic conventions where one exists, and are
// namespaced under "dockerctl." otherwise. Extra fields from the config go under
// "labels", so they can never collide with these keys.
//
// cmd must be the command whose action runs. The command passed to a Before hook
// is the one that owns the hook, not the leaf.
func OtelLogFields(cfg *config.Config, cmd *cli.Command) []zap.Field {
	fields := []zap.Field{
		zap.String("service.name", "dockerctl"),
		zap.String("service.version", dockerctl.Version),

		zap.String("deployment.environment.name", cfg.Runtime.ActiveEnv),

		zap.String("dockerctl.project.name", cfg.Name),
		zap.Uint("dockerctl.config.api_version", cfg.APIVersion),
		zap.String("dockerctl.command.name", strings.Join(cmd.Path()[1:], " ")),
	}

	if cfg.Node.Name != "" {
		fields = append(fields, zap.String("host.name", cfg.Node.Name))
	}
	if flags := setFlags(cmd); len(flags) > 0 {
		fields = append(fields, zap.Object("dockerctl.command.flags", stringMap(flags)))
	}
	if extra := cfg.Observability.Logging.ExtraFields; len(extra) > 0 {
		fields = append(fields, zap.Object("labels", stringMap(extra)))
	}
	return fields
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
