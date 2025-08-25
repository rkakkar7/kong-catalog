package handlers

import (
	"context"
	"kong/pkg/models"
	"net/http"
	"time"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	store *models.Store
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(store *models.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

// HealthCheck handles the /healthz endpoint
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// ReadinessCheck handles the /readyz endpoint
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()

	if err := h.store.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
