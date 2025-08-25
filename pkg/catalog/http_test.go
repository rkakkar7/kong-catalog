package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kong/pkg/config"
	"kong/pkg/models"
)

// testHTTPApp creates a test HTTP application using Docker Compose PostgreSQL
func testHTTPApp(t *testing.T) (*App, func()) {
	ctx := context.Background()

	// Use the same database as Docker Compose
	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		// Fallback to local database for testing
		connString = "postgres://catalog_user:catalog_password@localhost:5432/kong_catalog?sslmode=disable"
	}

	// Create test configuration
	cfg := &config.AppConfig{
		DatabaseURL:         connString,
		DBMaxConnections:    10,
		DBMinConnections:    2,
		DBMaxConnLifetime:   1 * time.Hour,
		DBMaxConnIdleTime:   15 * time.Minute,
		DBConnectTimeout:    10 * time.Second,
		DBHealthCheckPeriod: 1 * time.Minute,
		MaxPageSize:         100,
		ValidAPIKeys:        []string{"test-api-key-1", "test-api-key-2"},
	}

	// Create app
	app, err := New(ctx, cfg)
	require.NoError(t, err)

	// Drop existing schema to ensure clean state
	err = models.DropSchema(ctx, app.Pool())
	require.NoError(t, err)

	// Create fresh schema for tests
	err = models.EnsureSchema(ctx, app.Pool())
	require.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		pool := app.Pool()
		_, err := pool.Exec(ctx, "DELETE FROM service_versions")
		assert.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM services")
		assert.NoError(t, err)
		app.Close()
	}

	return app, cleanup
}

// CreateServiceRequest represents the data needed to create a service
type CreateServiceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateServiceVersionRequest represents the data needed to create a service version
type CreateServiceVersionRequest struct {
	Version string `json:"version"`
}

func TestHTTP_CreateService(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test creating a service
	t.Run("Create service successfully", func(t *testing.T) {
		reqBody := CreateServiceRequest{
			Name:        "test-service",
			Description: "A test service",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Equal(t, "test-service", response["name"])
		assert.Equal(t, "A test service", response["description"])
	})

	// Test creating duplicate service
	t.Run("Create duplicate service should fail", func(t *testing.T) {
		reqBody := CreateServiceRequest{
			Name:        "test-service", // Same name as above
			Description: "A duplicate service",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
		assert.Contains(t, response["message"], "duplicate key")
	})

	// Test missing API key
	t.Run("Missing API key should fail", func(t *testing.T) {
		reqBody := CreateServiceRequest{
			Name:        "unauthorized-service",
			Description: "This should fail",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No x-api-key header

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test invalid API key
	t.Run("Invalid API key should fail", func(t *testing.T) {
		reqBody := CreateServiceRequest{
			Name:        "invalid-key-service",
			Description: "This should fail",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "invalid-key")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestHTTP_ListServices(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Create some test services first
	services := []CreateServiceRequest{
		{Name: "api-service", Description: "API service"},
		{Name: "database-service", Description: "Database service"},
		{Name: "user-service", Description: "User service"},
	}

	for _, service := range services {
		jsonBody, _ := json.Marshal(service)
		req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Test listing all services
	t.Run("List all services", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "items")
		items := response["items"].([]interface{})
		assert.Len(t, items, 3)
	})

	// Test pagination
	t.Run("List services with pagination", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services?limit=2&offset=0", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	// Test search
	t.Run("Search services", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services?q=api", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		items := response["items"].([]interface{})
		assert.Len(t, items, 1)
		assert.Contains(t, items[0].(map[string]interface{})["name"], "api")
	})

	// Test sorting
	t.Run("Sort services by name", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services?sort=name&order=asc", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		items := response["items"].([]interface{})
		assert.Len(t, items, 3)
		// First item should be "api-service" when sorted by name ASC
		assert.Equal(t, "api-service", items[0].(map[string]interface{})["name"])
	})
}

func TestHTTP_GetService(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Create a test service first
	reqBody := CreateServiceRequest{
		Name:        "test-service",
		Description: "A test service",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "test-api-key-1")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(t, err)

	serviceID := createResponse["id"].(string)

	// Test getting the service
	t.Run("Get service by ID", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services/"+serviceID, nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, serviceID, response["id"])
		assert.Equal(t, "test-service", response["name"])
		assert.Equal(t, "A test service", response["description"])
	})

	// Test getting non-existent service
	t.Run("Get non-existent service", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req, err := http.NewRequest("GET", server.URL+"/v1/services/"+nonExistentID, nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test invalid UUID
	t.Run("Get service with invalid UUID", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services/invalid-uuid", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHTTP_CreateServiceVersion(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Create a test service first
	reqBody := CreateServiceRequest{
		Name:        "test-service",
		Description: "A test service",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "test-api-key-1")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(t, err)

	serviceID := createResponse["id"].(string)

	// Test creating a service version
	t.Run("Create service version successfully", func(t *testing.T) {
		versionReqBody := CreateServiceVersionRequest{
			Version: "1.0.0",
		}
		jsonBody, _ := json.Marshal(versionReqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services/"+serviceID+"/versions", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Equal(t, serviceID, response["service_id"])
		assert.Equal(t, "1.0.0", response["version"])
	})

	// Test creating duplicate version
	t.Run("Create duplicate version should fail", func(t *testing.T) {
		versionReqBody := CreateServiceVersionRequest{
			Version: "1.0.0", // Same version as above
		}
		jsonBody, _ := json.Marshal(versionReqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services/"+serviceID+"/versions", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
		assert.Contains(t, response["message"], "duplicate key")
	})
}

func TestHTTP_ListVersions(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Create a test service first
	reqBody := CreateServiceRequest{
		Name:        "test-service",
		Description: "A test service",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", server.URL+"/v1/services", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "test-api-key-1")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(t, err)

	serviceID := createResponse["id"].(string)

	// Create some test versions
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, version := range versions {
		versionReqBody := CreateServiceVersionRequest{Version: version}
		jsonBody, _ := json.Marshal(versionReqBody)

		req, err := http.NewRequest("POST", server.URL+"/v1/services/"+serviceID+"/versions", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Test listing versions
	t.Run("List service versions", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/v1/services/"+serviceID+"/versions", nil)
		require.NoError(t, err)
		req.Header.Set("x-api-key", "test-api-key-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "versions")
		versions := response["versions"].([]interface{})
		assert.Len(t, versions, 3)
	})
}

func TestHTTP_HealthChecks(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test health check (no auth required)
	t.Run("Health check", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/healthz", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test readiness check (no auth required)
	t.Run("Readiness check", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/readyz", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestHTTP_RequestID(t *testing.T) {
	app, cleanup := testHTTPApp(t)
	defer cleanup()

	server := httptest.NewServer(app.Router())
	defer server.Close()

	// Test that request ID is returned in headers
	t.Run("Request ID in headers", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/healthz", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header, "X-Request-ID")
		assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	})

	// Test custom request ID
	t.Run("Custom request ID", func(t *testing.T) {
		customID := "custom-request-123"
		req, err := http.NewRequest("GET", server.URL+"/healthz", nil)
		require.NoError(t, err)
		req.Header.Set("X-Request-ID", customID)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, customID, resp.Header.Get("X-Request-ID"))
	})
}
