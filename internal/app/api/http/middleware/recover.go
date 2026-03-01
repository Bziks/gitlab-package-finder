package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"

	"github.com/bziks/gitlab-package-finder/internal/adapters/metrics"
	"github.com/bziks/gitlab-package-finder/pkg/oapi"
)

func Recover(m *metrics.Metrics) nethttp.StrictHTTPMiddlewareFunc {
	return nethttp.StrictHTTPMiddlewareFunc(func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return nethttp.StrictHTTPHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			defer func() {
				if rec := recover(); rec != nil {
					sendInternalServerError(ctx, m, rec, w, r)
				}
			}()

			return f(ctx, w, r, request)
		})
	})
}

func sendInternalServerError(ctx context.Context, m *metrics.Metrics, rec interface{}, w http.ResponseWriter, r *http.Request) {
	m.HTTPRequests.Add(ctx, 1, metrics.WithHTTPAttr(r.URL.Path, r.Method, metrics.Status5xx))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	traceID := GetTraceID(ctx)

	slog.ErrorContext(ctx, "internal server error",
		"error", rec,
		"stack", string(debug.Stack()),
	)

	response := oapi.N500Response{
		Error: oapi.ErrorWithDetailsResponse{
			Message: "Internal Server Error",
		},
		Meta: &oapi.MetaResponse{
			LogId:     &traceID,
			Timestamp: time.Now(),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "failed to encode error response", "error", err)
	}
}
