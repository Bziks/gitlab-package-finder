package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bziks/gitlab-package-finder/internal/config"
)

func NewHTTPServer(cfg config.ApiConfig, handler http.Handler) *http.Server {
	addr := fmt.Sprintf(":%d", cfg.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	slog.Info("http server will be started at port " + addr)

	return srv
}
