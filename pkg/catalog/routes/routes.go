package routes

import (
	"context"
	"kong/pkg/catalog/handlers"
	"kong/pkg/catalog/middleware"
	"kong/pkg/catalog/validation"
	"kong/pkg/models"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SetupRoutes configures all the routes with middleware
func SetupRoutes(store *models.Store, r *chi.Mux) {
	// Health checks (no validation needed)
	healthHandler := handlers.NewHealthHandler(store)
	r.Get("/healthz", healthHandler.HealthCheck)
	r.Get("/readyz", healthHandler.ReadinessCheck)

	// API routes with validation middleware
	servicesHandler := handlers.NewServicesHandler(store)

	r.Route("/v1", func(r chi.Router) {
		// List services with validation
		r.With(middleware.ValidationMiddleware(validation.ValidateListServicesParams)).
			Get("/services", servicesHandler.ListServices)

		// Get service by ID with validation
		r.With(middleware.ValidationMiddleware(func(r *http.Request) error {
			// Extract ID from URL parameter and validate
			id := chi.URLParam(r, "id")
			if err := validation.ValidateID(id); err != nil {
				return err
			}

			// Store validated ID in context for handler to use
			ctx := context.WithValue(r.Context(), "id", id)
			*r = *r.WithContext(ctx)

			// Also validate query parameters
			return validation.ValidateGetServiceParams(r)
		})).Get("/services/{id}", servicesHandler.GetService)

		// List versions with ID validation
		r.With(middleware.ValidationMiddleware(func(r *http.Request) error {
			// Extract ID from URL parameter and validate
			id := chi.URLParam(r, "id")
			if err := validation.ValidateID(id); err != nil {
				return err
			}
			// Store validated ID in context for handler to use
			ctx := context.WithValue(r.Context(), "id", id)
			*r = *r.WithContext(ctx)
			return nil
		})).Get("/services/{id}/versions", servicesHandler.ListVersions)

		// Create service with validation
		r.With(middleware.ValidationMiddleware(validation.ValidateCreateServiceParams)).
			Post("/services", servicesHandler.CreateService)

		// Create service version with validation
		r.With(middleware.ValidationMiddleware(func(r *http.Request) error {
			// Extract ID from URL parameter and validate
			id := chi.URLParam(r, "id")
			if err := validation.ValidateID(id); err != nil {
				return err
			}
			// Store validated ID in context for handler to use
			ctx := context.WithValue(r.Context(), "id", id)
			*r = *r.WithContext(ctx)
			return nil
		})).With(middleware.ValidationMiddleware(validation.ValidateCreateServiceVersionParams)).
			Post("/services/{id}/versions", servicesHandler.CreateServiceVersion)
	})
}
