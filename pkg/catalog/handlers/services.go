package handlers

import (
	"encoding/json"
	"kong/pkg/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateServiceRequest represents the data needed to create a service
type CreateServiceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateServiceVersionRequest represents the data needed to create a service version
type CreateServiceVersionRequest struct {
	Version string `json:"version"`
}

// ServicesHandler handles service-related API endpoints
type ServicesHandler struct {
	store *models.Store
}

// NewServicesHandler creates a new services handler
func NewServicesHandler(store *models.Store) *ServicesHandler {
	return &ServicesHandler{store: store}
}

// ListServices lists services with validation
func (h *ServicesHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	includeVersions := r.URL.Query().Get("include_versions") == "true"

	items, err := h.store.ListServices(r.Context(), q, sort, order, limit, offset, includeVersions)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list services", err)
		return
	}

	respond(w, map[string]any{"items": items})
}

// GetService gets a service by ID with validation
func (h *ServicesHandler) GetService(w http.ResponseWriter, r *http.Request) {
	idStr := r.Context().Value("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID format", err)
		return
	}

	includeVersions := r.URL.Query().Get("include_versions") == "true"

	it, err := h.store.GetService(r.Context(), id, includeVersions)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get service", err)
		return
	}
	if it == nil {
		respondError(w, http.StatusNotFound, "Service not found", nil)
		return
	}

	respond(w, it)
}

// ListVersions lists versions for a service with validation
func (h *ServicesHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	// ID validation is handled by middleware, so we can directly extract it
	idStr := r.Context().Value("id").(string)

	// Parse the validated ID string to UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID format", err)
		return
	}

	versions, err := h.store.ListVersions(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list service versions", err)
		return
	}

	respond(w, map[string]any{"versions": versions})
}

// CreateService creates a new service
func (h *ServicesHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required", nil)
		return
	}

	if len(req.Name) > 100 {
		respondError(w, http.StatusBadRequest, "Name too long (max 100 characters)", nil)
		return
	}

	if len(req.Description) > 1000 {
		respondError(w, http.StatusBadRequest, "Description too long (max 1000 characters)", nil)
		return
	}

	// Create the service with generated values
	service := &models.Service{
		ID:          models.GenerateUUID(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Versions:    []models.ServiceVersion{}, // Empty array for new service
	}

	if err := h.store.CreateService(r.Context(), service); err != nil {
		// Check for specific database errors
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "Service with this name already exists", err)
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to create service", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(service)
}

// CreateServiceVersion creates a new service version
func (h *ServicesHandler) CreateServiceVersion(w http.ResponseWriter, r *http.Request) {
	// Get service ID from context (set by validation middleware)
	idStr := r.Context().Value("id").(string)
	serviceID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid service ID format", err)
		return
	}

	var req CreateServiceVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// Validate required fields
	if req.Version == "" {
		respondError(w, http.StatusBadRequest, "Version is required", nil)
		return
	}

	if len(req.Version) > 50 {
		respondError(w, http.StatusBadRequest, "Version too long (max 50 characters)", nil)
		return
	}

	// Create the service version with generated values
	serviceVersion := &models.ServiceVersion{
		ID:        models.GenerateUUID(),
		ServiceID: serviceID,
		Version:   req.Version,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.store.CreateServiceVersion(r.Context(), serviceVersion); err != nil {
		// Check for specific database errors
		if strings.Contains(err.Error(), "duplicate key") {
			respondError(w, http.StatusConflict, "Version already exists for this service", err)
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to create service version", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(serviceVersion)
}

// respond writes a JSON response
func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"message": message,
	}

	if err != nil {
		response["error"] = err.Error()
	}

	_ = json.NewEncoder(w).Encode(response)
}
