package validation

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidateListServicesParams validates parameters for listServices endpoint
func ValidateListServicesParams(r *http.Request) error {
	var errors []ValidationError

	// Validate limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			errors = append(errors, ValidationError{
				Field:   "limit",
				Message: "must be a positive integer between 1 and 1000",
			})
		}
		// Additional validation for limit parameter
		if len(limitStr) > 10 {
			errors = append(errors, ValidationError{
				Field:   "limit",
				Message: "limit parameter string must be 10 characters or less",
			})
		}
	}

	// Validate offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			errors = append(errors, ValidationError{
				Field:   "offset",
				Message: "must be a non-negative integer",
			})
		}
	}

	// Validate sort
	if sort := r.URL.Query().Get("sort"); sort != "" {
		allowedSorts := []string{"name", "created_at", "updated_at"}
		validSort := false
		for _, allowed := range allowedSorts {
			if sort == allowed {
				validSort = true
				break
			}
		}
		if !validSort {
			errors = append(errors, ValidationError{
				Field:   "sort",
				Message: fmt.Sprintf("must be one of: %s", strings.Join(allowedSorts, ", ")),
			})
		}
	}

	// Validate order
	if order := r.URL.Query().Get("order"); order != "" {
		if order != "asc" && order != "desc" {
			errors = append(errors, ValidationError{
				Field:   "order",
				Message: "must be either 'asc' or 'desc'",
			})
		}
	}

	// Validate include_versions (boolean parameter)
	if includeVersions := r.URL.Query().Get("include_versions"); includeVersions != "" {
		if includeVersions != "true" && includeVersions != "false" {
			errors = append(errors, ValidationError{
				Field:   "include_versions",
				Message: "must be either 'true' or 'false'",
			})
		}
	}

	// Validate query length and content
	if q := r.URL.Query().Get("q"); q != "" {
		if len(q) < 1 {
			errors = append(errors, ValidationError{
				Field:   "q",
				Message: "search query must be at least 1 character long",
			})
		}
		if len(q) > 100 {
			errors = append(errors, ValidationError{
				Field:   "q",
				Message: "search query must be 100 characters or less",
			})
		}
	}

	if len(errors) > 0 {
		return ValidationErrors{Errors: errors}
	}
	return nil
}

// ValidateID validates ID path parameters - must be a valid UUIDv4
func ValidateID(id string) error {
	if id == "" {
		return ValidationError{Field: "id", Message: "ID cannot be empty"}
	}

	// Parse and validate UUID
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return ValidationError{Field: "id", Message: "ID must be a valid UUID"}
	}

	// Ensure it's a UUIDv4 specifically
	if parsedUUID.Version() != 4 {
		return ValidationError{Field: "id", Message: "ID must be a valid UUIDv4"}
	}

	return nil
}

// ValidateGetServiceParams validates parameters for getService endpoint
func ValidateGetServiceParams(r *http.Request) error {
	var errors []ValidationError

	// Validate include_versions (boolean parameter)
	if includeVersions := r.URL.Query().Get("include_versions"); includeVersions != "" {
		if includeVersions != "true" && includeVersions != "false" {
			errors = append(errors, ValidationError{
				Field:   "include_versions",
				Message: "must be either 'true' or 'false'",
			})
		}
	}

	if len(errors) > 0 {
		return ValidationErrors{Errors: errors}
	}
	return nil
}

// ValidateCreateServiceParams validates parameters for createService endpoint
func ValidateCreateServiceParams(r *http.Request) error {
	// For POST requests, we mainly validate Content-Type header
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") {
		return ValidationError{
			Field:   "Content-Type",
			Message: "must be application/json",
		}
	}
	return nil
}

// ValidateCreateServiceVersionParams validates parameters for createServiceVersion endpoint
func ValidateCreateServiceVersionParams(r *http.Request) error {
	// For POST requests, we mainly validate Content-Type header
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") {
		return ValidationError{
			Field:   "Content-Type",
			Message: "must be application/json",
		}
	}
	return nil
}
