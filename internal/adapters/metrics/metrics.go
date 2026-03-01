package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	ometric "go.opentelemetry.io/otel/metric"
)

const (
	Status2xx = "2xx"
	Status4xx = "4xx"
	Status5xx = "5xx"
)

type Metrics struct {
	meter    ometric.Meter
	registry *prometheus.Registry

	HTTPRequests ometric.Int64Counter

	HTTPRequestDuration ometric.Float64Histogram
}

func WithHTTPAttr(route, method, statusCode string) ometric.MeasurementOption {
	return ometric.WithAttributes(
		attribute.Key("handler").String(route),
		attribute.Key("method").String(method),
		attribute.Key("status").String(statusCode),
	)
}

func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) registerMetrics() error {
	const op = "register metrics"
	var err error

	m.HTTPRequests, err = m.meter.Int64Counter(
		"http_requests_total",
		ometric.WithDescription("Total number of requests"),
	)
	if err != nil {
		return fmt.Errorf("%s: failed to create http_requests_total counter: %w", op, err)
	}

	m.HTTPRequestDuration, err = m.meter.Float64Histogram(
		"http_request_duration_seconds",
		ometric.WithDescription("Total request execution time in seconds"),
		ometric.WithUnit("s"),
		ometric.WithExplicitBucketBoundaries(0, 1, 2, 5, 10, 20, 40, 60, 120, 240),
	)
	if err != nil {
		return fmt.Errorf("%s: failed to create http_request_duration_seconds histogram: %w", op, err)
	}

	return nil
}
