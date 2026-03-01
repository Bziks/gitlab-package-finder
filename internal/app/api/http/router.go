package http

import (
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bziks/gitlab-package-finder/internal/app/api/http/middleware"
	oapi "github.com/bziks/gitlab-package-finder/pkg/oapi"
)

func (api *API) NewRouter() http.Handler {
	// NOTICE: Order of middlewares is important
	// Middlewares are applied on the request from last to first
	strictHandler := oapi.NewStrictHandler(api, []nethttp.StrictHTTPMiddlewareFunc{
		middleware.Recover(api.metrics),
		middleware.Metrics(api.metrics),
		middleware.Logging(),
		middleware.CORS(middleware.NewCORSConfig(api.corsAllowOrigin)),
	})

	r := http.NewServeMux()

	// Metrics
	r.Handle("GET /metrics", promhttp.HandlerFor(
		api.metrics.GetRegistry(),
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	oapi.HandlerFromMux(strictHandler, r)

	// Static files (UI) with security headers
	r.Handle("GET /", middleware.SecurityHeaders(http.FileServer(http.Dir("public"))))

	return r
}
