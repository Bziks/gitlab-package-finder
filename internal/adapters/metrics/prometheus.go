package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	ometric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/bziks/gitlab-package-finder/internal/config"
)

func NewPrometheusMetrics(cfg config.MetricsConfig) (*Metrics, error) {
	const op = "init prometheus metrics"

	registry := prometheus.NewRegistry()

	exporter, err := newExporter(registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	meter := provider.Meter(cfg.Name, ometric.WithInstrumentationVersion(cfg.Version))

	m := &Metrics{
		meter:    meter,
		registry: registry,
	}

	err = m.registerMetrics()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return m, nil
}

func newExporter(registry *prometheus.Registry, cfg config.MetricsConfig) (*otelprom.Exporter, error) {
	const op = "create prometheus metrics exporter"

	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutTargetInfo(),
		otelprom.WithoutScopeInfo(),
		otelprom.WithNamespace(cfg.Namespace),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return exporter, nil
}
