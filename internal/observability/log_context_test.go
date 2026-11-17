package observability

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/TimCares/go-see"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogFieldsUseLeafCommandAndSetFlags(t *testing.T) {
	otelLogData := OtelLogData{
		DockerctlVersion:          "v1.0.0",
		DockerctlConfigAPIVersion: 1,
		ProjectName:               "demo",
		CurrentHostname:           "host-1",
		ExtraFields:               map[string]string{"team": "infra"},
	}

	var got string
	root := &cli.Command{
		Name: "dockerctl",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "env"},
			&cli.StringFlag{Name: "log-level", Value: "info"},
		},
		Commands: []*cli.Command{{
			Name: "identity",
			Commands: []*cli.Command{{
				Name: "init",
				Action: func(_ context.Context, cmd *cli.Command) error {
					buf, err := zapcore.NewJSONEncoder(zapcore.EncoderConfig{}).
						EncodeEntry(zapcore.Entry{}, OtelLogFields(otelLogData, cmd))
					if err != nil {
						return err
					}
					got = buf.String()
					return nil
				},
			}},
		}},
	}

	if err := root.Run(context.Background(), []string{"dockerctl", "--env", "dev", "identity", "init"}); err != nil {
		t.Fatal(err)
	}

	want := `{"service.name":"dockerctl","service.version":"v1.0.0",` +
		`"dockerctl.project.name":"demo",` +
		`"dockerctl.config.api_version":1,"dockerctl.command.name":"identity init",` +
		`"host.name":"host-1","dockerctl.command.flags":{"env":"dev"},"labels":{"team":"infra"}}` + "\n"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestForEachEnvAttachesEnvPerIteration(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	emitter, err := see.New(see.Config{Logger: zap.New(core)})
	if err != nil {
		t.Fatal(err)
	}

	envs := []string{"dev", "prod"}
	var gotEnvs []string
	err = ForEachEnv(context.Background(), envs, func(ctx context.Context, env string) error {
		gotEnvs = append(gotEnvs, env)
		emitter.Emit(ctx, plainEvent{})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(gotEnvs, envs) {
		t.Fatalf("fn called with %v, want %v", gotEnvs, envs)
	}
	entries := logs.All()
	if len(entries) != len(envs) {
		t.Fatalf("got %d log entries, want %d", len(entries), len(envs))
	}
	for i, entry := range entries {
		var envFields int
		for _, f := range entry.Context {
			if f.Key == envKey {
				envFields++
			}
		}
		if envFields != 1 {
			t.Errorf("entry %d has %d %q fields, want 1, so envs of earlier iterations must not leak", i, envFields, envKey)
		}
		if got := entry.ContextMap()[envKey]; got != envs[i] {
			t.Errorf("entry %d: %s = %v, want %s", i, envKey, got, envs[i])
		}
	}
}

func TestForEachEnvStopsAtFirstError(t *testing.T) {
	wantErr := errors.New("boom")
	var calls int
	err := ForEachEnv(context.Background(), []string{"dev", "prod"}, func(context.Context, string) error {
		calls++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
}
