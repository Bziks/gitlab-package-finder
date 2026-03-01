package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

// CORSConfig holds the configuration for CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORSConfig returns a CORS configuration with the given allowed origin.
func NewCORSConfig(allowOrigin string) CORSConfig {
	origins := []string{"*"}
	if allowOrigin != "" && allowOrigin != "*" {
		origins = strings.Split(allowOrigin, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
	}
	return CORSConfig{
		AllowedOrigins: origins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS creates a CORS middleware that works with oapi-codegen
func CORS(config CORSConfig) nethttp.StrictHTTPMiddlewareFunc {
	return nethttp.StrictHTTPMiddlewareFunc(func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return nethttp.StrictHTTPHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (response interface{}, err error) {
			setCORSHeaders(w, r, config)

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return nil, nil
			}

			return f(ctx, w, r, request)
		})
	})
}

// setCORSHeaders sets the appropriate CORS headers based on the configuration
func setCORSHeaders(w http.ResponseWriter, r *http.Request, config CORSConfig) {
	origin := r.Header.Get("Origin")

	// Set Access-Control-Allow-Origin
	if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else if isOriginAllowed(origin, config.AllowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}

	// Set Access-Control-Allow-Credentials
	if config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Set Access-Control-Allow-Methods
	if len(config.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	}

	// Set Access-Control-Allow-Headers
	if len(config.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
	}

	// Set Access-Control-Expose-Headers
	if len(config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
	}

	// Set Access-Control-Max-Age
	if config.MaxAge > 0 && r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
	}
}

// isOriginAllowed checks if the given origin is in the list of allowed origins
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := allowedOrigin[2:]
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	return false
}
