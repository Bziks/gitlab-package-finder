package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// TracingHandler wraps an slog.Handler and adds trace_id and span_id from the OpenTelemetry span context.
type TracingHandler struct {
	inner slog.Handler
}

func NewTracingHandler(inner slog.Handler) *TracingHandler {
	return &TracingHandler{inner: inner}
}

func (h *TracingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *TracingHandler) Handle(ctx context.Context, record slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String("span_id", spanCtx.SpanID().String()))
	}

	return h.inner.Handle(ctx, record)
}

func (h *TracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewTracingHandler(h.inner.WithAttrs(attrs))
}

func (h *TracingHandler) WithGroup(name string) slog.Handler {
	return NewTracingHandler(h.inner.WithGroup(name))
}

// NewLogger creates a new slog.Logger with JSON output and tracing context.
func NewLogger(level slog.Level) *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	tracingHandler := NewTracingHandler(jsonHandler)

	return slog.New(tracingHandler)
}
