package middleware

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Logging creates a middleware that starts an otel span per request
// and sets the trace ID in the X-Trace-Id response header.
func Logging() nethttp.StrictHTTPMiddlewareFunc {
	tracer := otel.Tracer("http")

	return nethttp.StrictHTTPMiddlewareFunc(func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return nethttp.StrictHTTPHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			ctx, span := tracer.Start(ctx, operationID)
			defer span.End()

			traceID := span.SpanContext().TraceID().String()
			w.Header().Set("X-Trace-Id", traceID)

			return f(ctx, w, r, request)
		})
	})
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}
