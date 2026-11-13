package observability

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
		zap.String("service.name", "dockerctl"),
		zap.String("service.version", data.DockerctlVersion),

		zap.String("deployment.environment.name", data.ActiveEnv),

		zap.String("dockerctl.project.name", data.ProjectName),
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
