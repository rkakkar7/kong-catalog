package middleware

import (
	"fmt"
	"net/http"

	"kong/pkg/catalog/validation"
)

// ValidationMiddleware validates request parameters based on validation function
func ValidationMiddleware(validator func(*http.Request) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate request parameters
			if err := validator(r); err != nil {
				handleValidationError(w, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// handleValidationError handles validation errors and returns appropriate HTTP response
func handleValidationError(w http.ResponseWriter, err error) {
	if validationErr, ok := err.(validation.ValidationErrors); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		// Convert validation errors to our format
		errors := make([]map[string]string, len(validationErr.Errors))
		for i, ve := range validationErr.Errors {
			errors[i] = map[string]string{
				"field":   ve.Field,
				"message": ve.Message,
			}
		}
		// Simple JSON encoding for now
		fmt.Fprintf(w, `{"error":"Validation failed","errors":[`)
		for i, err := range errors {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"field":"%s","message":"%s"}`, err["field"], err["message"])
		}
		fmt.Fprint(w, "]}")
		return
	}
	if validationErr, ok := err.(validation.ValidationError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"Validation failed","errors":[{"field":"%s","message":"%s"}]}`,
			validationErr.Field, validationErr.Message)
		return
	}

	// Fallback for other errors
	http.Error(w, err.Error(), http.StatusBadRequest)
}
