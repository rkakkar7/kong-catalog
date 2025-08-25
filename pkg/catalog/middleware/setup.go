package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SetupGlobalMiddleware applies all global middleware to the router in the correct order, middlewares are applied from top to bottom (first to last)
func SetupGlobalMiddleware(r *chi.Mux, validAPIKeys []string) {
	// 1. Request ID middleware - adds unique ID to each request
	r.Use(RequestIDMiddleware)

	// 2. Logging middleware - logs request details and adds logger to context
	r.Use(LoggingMiddleware)

	// 3. Authentication middleware - validates API keys (skips health checks)
	r.Use(APIKeyMiddleware(validAPIKeys))
}

// SetupRouteSpecificMiddleware applies middleware to specific routes
// This is used for validation middleware that only applies to certain endpoints
func SetupRouteSpecificMiddleware(r chi.Router, validator func(*http.Request) error) {
	r.Use(ValidationMiddleware(validator))
}
