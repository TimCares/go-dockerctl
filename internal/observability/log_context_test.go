package observability

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap/zapcore"

	"github.com/TimCares/go-dockerctl"
	"github.com/TimCares/go-dockerctl/internal/config"
)

func TestLogFieldsUseLeafCommandAndSetFlags(t *testing.T) {
	cfg := &config.Config{
		APIVersion: 1,
		Name:       "demo",
		Node:       config.NodeInfoConfig{Name: "host-1"},
		Runtime:    config.RuntimeConfig{ActiveEnv: "dev"},
		Observability: config.ObservabilityConfig{
			Logging: config.LoggingConfig{ExtraFields: map[string]string{"team": "infra"}},
		},
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
						EncodeEntry(zapcore.Entry{}, OtelLogFields(cfg, cmd))
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

	want := `{"service.name":"dockerctl","service.version":"` + dockerctl.Version + `",` +
		`"deployment.environment.name":"dev","dockerctl.project.name":"demo",` +
		`"dockerctl.config.api_version":1,"dockerctl.command.name":"identity init",` +
		`"host.name":"host-1","dockerctl.command.flags":{"env":"dev"},"labels":{"team":"infra"}}` + "\n"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
