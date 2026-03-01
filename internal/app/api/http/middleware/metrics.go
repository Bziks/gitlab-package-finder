package middleware

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"

	"github.com/bziks/gitlab-package-finder/internal/adapters/metrics"
)

// Metrics creates a middleware that works with oapi-codegen
func Metrics(m *metrics.Metrics) nethttp.StrictHTTPMiddlewareFunc {
	return nethttp.StrictHTTPMiddlewareFunc(func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return nethttp.StrictHTTPHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
			start := time.Now()
			res, err := f(ctx, w, r, request)
			duration := time.Since(start).Seconds()
			statusCode := resolveStatusCode(res, err)

			m.HTTPRequestDuration.Record(ctx, duration, metrics.WithHTTPAttr(r.URL.Path, r.Method, statusCode))
			m.HTTPRequests.Add(ctx, 1, metrics.WithHTTPAttr(r.URL.Path, r.Method, statusCode))

			return res, err
		})
	})
}

// resolveStatusCode determines the status code category from a strict handler response.
// oapi-codegen generates response types with status code in the name, e.g.:
// InternalGetProjects400JSONResponse, InternalGetProjects500JSONResponse.
func resolveStatusCode(res any, err error) string {
	if err != nil {
		return metrics.Status5xx
	}

	if res == nil {
		return metrics.Status2xx
	}

	typeName := reflect.TypeOf(res).Name()
	if strings.Contains(typeName, "500") {
		return metrics.Status5xx
	}
	if strings.Contains(typeName, "400") || strings.Contains(typeName, "404") || strings.Contains(typeName, "409") {
		return metrics.Status4xx
	}

	return metrics.Status2xx
}
