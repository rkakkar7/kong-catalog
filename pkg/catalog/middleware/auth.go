package middleware

import (
	"net/http"
)

// APIKeyMiddleware creates a middleware that validates API keys
func APIKeyMiddleware(validAPIKeys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check endpoints
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			// Change your auth middleware to expect:
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, "Missing API key", http.StatusUnauthorized)
				return
			}

			// Extract the API key
			if apiKey == "" {
				http.Error(w, "Missing API key", http.StatusUnauthorized)
				return
			}

			// Validate the API key
			valid := false
			for _, validKey := range validAPIKeys {
				if apiKey == validKey {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// API key is valid, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}
