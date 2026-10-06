package observability

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/TimCares/go-dockerctl/internal/config"
)

// Shutdown needs this handle to flush.
var meterProvider *metric.MeterProvider

// InitMetrics starts the OTLP metric exporter and installs it as the global meter provider.
func InitMetrics(ctx context.Context, cfg config.OtelConfig) error {
	token, err := os.ReadFile(cfg.SecretFile) // TODO: load from dockerctl.sops.yaml
	if err != nil {
		return fmt.Errorf("reading otel secret file: %w", err)
	}
	bearerToken := strings.TrimSpace(string(token))
	if bearerToken == "" {
		return fmt.Errorf("otel secret file %s is empty", cfg.SecretFile)
	}

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"authorization": "Bearer " + bearerToken,
		}),
	)
	if err != nil {
		return err
	}

	reader := metric.NewPeriodicReader(exporter)
	meterProvider = metric.NewMeterProvider(
		metric.WithReader(reader),
	)
	otel.SetMeterProvider(meterProvider)
	return nil
}

// shutdownMetrics flushes pending metrics and stops the reader.
// A second call does nothing.
func shutdownMetrics(ctx context.Context) error {
	provider := meterProvider
	meterProvider = nil
	if provider == nil {
		return nil
	}
	return provider.Shutdown(ctx)
}
