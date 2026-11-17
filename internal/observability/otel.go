package observability

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Shutdown needs this handle to flush.
var meterProvider *metric.MeterProvider

// InitMetrics starts the OTLP metric exporter and installs it as the global meter provider.
//
// Everything fixed for the whole run, like the host, is attached once as the resource,
// so events do not have to carry it as metric attributes.
func InitMetrics(ctx context.Context, cfg OtelConfig, data OtelLogData) error {
	token, err := os.ReadFile(cfg.SecretFile) // TODO: load from dockerctl.sops.yaml
	if err != nil {
		return fmt.Errorf("reading otel secret file: %w", err)
	}
	bearerToken := strings.TrimSpace(string(token))
	if bearerToken == "" {
		return fmt.Errorf("otel secret file %s is empty", cfg.SecretFile)
	}

	res, err := metricResource(data)
	if err != nil {
		return fmt.Errorf("building otel resource: %w", err)
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
		metric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)
	return nil
}

// metricResource uses semconv v1.43.0, the version [resource.Default] uses, so their schema URLs match when merged.
func metricResource(data OtelLogData) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(data.DockerctlVersion),
		attribute.String(projectNameKey, data.ProjectName),
	}
	if data.CurrentHostname != "" {
		attrs = append(attrs, semconv.HostName(data.CurrentHostname))
	}
	return resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, attrs...))
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
