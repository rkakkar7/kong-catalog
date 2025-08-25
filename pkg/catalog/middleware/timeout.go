package middleware

import (
	"net/http"
	"time"
)

// TimeoutMiddleware creates a middleware that times out requests after the specified duration
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "request timeout")
	}
}
