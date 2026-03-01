package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/bziks/gitlab-package-finder/internal/config"
)

// TracerProvider wraps an OpenTelemetry TracerProvider with a Stop method.
type TracerProvider struct {
	provider *trace.TracerProvider
}

// Stop shuts down the tracer provider, flushing any remaining spans.
func (tp *TracerProvider) Stop(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}

// NewTracerProvider creates a new TracerProvider. When tracing is disabled, it sets a no-op provider.
func NewTracerProvider(ctx context.Context, cfg config.TracingConfig) (*TracerProvider, error) {
	if !cfg.Enabled {
		return &TracerProvider{}, nil
	}

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("create stdout trace exporter: %w", err)
	}

	sampler := trace.TraceIDRatioBased(cfg.SampleRate)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)

	return &TracerProvider{provider: tp}, nil
}
