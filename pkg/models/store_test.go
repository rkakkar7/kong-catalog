package models

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStore creates a test store using Docker Compose PostgreSQL
func testStore(t *testing.T) (*Store, func()) {
	ctx := context.Background()

	// Use the same database as Docker Compose
	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		// Fallback to local database for testing
		connString = "postgres://catalog_user:catalog_password@localhost:5432/kong_catalog?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	// Drop existing schema to ensure clean state
	err = DropSchema(ctx, pool)
	require.NoError(t, err)

	// Create fresh schema for tests
	err = EnsureSchema(ctx, pool)
	require.NoError(t, err)

	store := NewStore(pool, 100)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		_, err := pool.Exec(ctx, "DELETE FROM service_versions")
		assert.NoError(t, err)
		_, err = pool.Exec(ctx, "DELETE FROM services")
		assert.NoError(t, err)
		pool.Close()
	}

	return store, cleanup
}

func TestStore_CreateService(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Test creating a service
	service := &Service{
		ID:          uuid.New(),
		Name:        "test-service",
		Description: "A test service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := store.CreateService(ctx, service)
	assert.NoError(t, err)

	// Test creating duplicate service (should fail)
	err = store.CreateService(ctx, service)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")
}

func TestStore_GetService(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a service
	service := &Service{
		ID:          uuid.New(),
		Name:        "test-service",
		Description: "A test service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := store.CreateService(ctx, service)
	require.NoError(t, err)

	// Test getting the service
	retrieved, err := store.GetService(ctx, service.ID, false)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, service.Name, retrieved.Name)
	assert.Equal(t, service.Description, retrieved.Description)
	assert.Empty(t, retrieved.Versions) // includeVersions=false

	// Test getting service with versions
	retrieved, err = store.GetService(ctx, service.ID, true)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Empty(t, retrieved.Versions) // No versions yet

	// Test getting non-existent service
	nonExistentID := uuid.New()
	retrieved, err = store.GetService(ctx, nonExistentID, false)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestStore_ListServices(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple services with different names and timestamps
	baseTime := time.Now().UTC()
	services := []*Service{
		{
			ID:          uuid.New(),
			Name:        "api-service",
			Description: "API service",
			CreatedAt:   baseTime.Add(-5 * time.Hour),
			UpdatedAt:   baseTime.Add(-5 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "database-service",
			Description: "Database service",
			CreatedAt:   baseTime.Add(-4 * time.Hour),
			UpdatedAt:   baseTime.Add(-4 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "user-service",
			Description: "User service",
			CreatedAt:   baseTime.Add(-3 * time.Hour),
			UpdatedAt:   baseTime.Add(-3 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "auth-service",
			Description: "Authentication service",
			CreatedAt:   baseTime.Add(-2 * time.Hour),
			UpdatedAt:   baseTime.Add(-2 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "payment-service",
			Description: "Payment processing service",
			CreatedAt:   baseTime.Add(-1 * time.Hour),
			UpdatedAt:   baseTime.Add(-1 * time.Hour),
		},
	}

	for _, service := range services {
		err := store.CreateService(ctx, service)
		require.NoError(t, err)
	}

	// Test basic listing without pagination
	t.Run("List all services", func(t *testing.T) {
		items, err := store.ListServices(ctx, "", "", "", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
	})

	// Test pagination
	t.Run("Pagination", func(t *testing.T) {
		// First page with limit 2
		items, err := store.ListServices(ctx, "", "", "", 2, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 2)

		// Second page with limit 2
		items, err = store.ListServices(ctx, "", "", "", 2, 2, false)
		assert.NoError(t, err)
		assert.Len(t, items, 2)

		// Third page with limit 2
		items, err = store.ListServices(ctx, "", "", "", 2, 4, false)
		assert.NoError(t, err)
		assert.Len(t, items, 1)

		// Fourth page (should be empty)
		items, err = store.ListServices(ctx, "", "", "", 2, 6, false)
		assert.NoError(t, err)
		assert.Len(t, items, 0)
	})

	// Test search/filtering
	t.Run("Search filtering", func(t *testing.T) {
		// Search for "api" (should match api-service)
		items, err := store.ListServices(ctx, "api", "", "", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "api-service", items[0].Name)

		// Search for "auth" (should match auth-service)
		items, err = store.ListServices(ctx, "auth", "", "", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "auth-service", items[0].Name)

		// Search for "payment" (should match payment-service)
		items, err = store.ListServices(ctx, "payment", "", "", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "payment-service", items[0].Name)

		// Search for non-existent service
		items, err = store.ListServices(ctx, "non-existent", "", "", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 0)
	})

	// Test sorting by name
	t.Run("Sort by name", func(t *testing.T) {
		// Ascending order
		items, err := store.ListServices(ctx, "", "name", "asc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		assert.Equal(t, "api-service", items[0].Name)
		assert.Equal(t, "auth-service", items[1].Name)
		assert.Equal(t, "database-service", items[2].Name)
		assert.Equal(t, "payment-service", items[3].Name)
		assert.Equal(t, "user-service", items[4].Name)

		// Descending order
		items, err = store.ListServices(ctx, "", "name", "desc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		assert.Equal(t, "user-service", items[0].Name)
		assert.Equal(t, "payment-service", items[1].Name)
		assert.Equal(t, "database-service", items[2].Name)
		assert.Equal(t, "auth-service", items[3].Name)
		assert.Equal(t, "api-service", items[4].Name)
	})

	// Test sorting by created_at
	t.Run("Sort by created_at", func(t *testing.T) {
		// Ascending order (oldest first)
		items, err := store.ListServices(ctx, "", "created_at", "asc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		// api-service was created first (-5 hours)
		assert.Equal(t, "api-service", items[0].Name)
		// payment-service was created last (-1 hour)
		assert.Equal(t, "payment-service", items[4].Name)

		// Descending order (newest first)
		items, err = store.ListServices(ctx, "", "created_at", "desc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		// payment-service was created last (-1 hour)
		assert.Equal(t, "payment-service", items[0].Name)
		// api-service was created first (-5 hours)
		assert.Equal(t, "api-service", items[4].Name)
	})

	// Test sorting by updated_at
	t.Run("Sort by updated_at", func(t *testing.T) {
		// Ascending order
		items, err := store.ListServices(ctx, "", "updated_at", "asc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		assert.Equal(t, "api-service", items[0].Name)

		// Descending order
		items, err = store.ListServices(ctx, "", "updated_at", "desc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		assert.Equal(t, "payment-service", items[0].Name)
	})

	// Test invalid sort key (should default to name)
	t.Run("Invalid sort key", func(t *testing.T) {
		items, err := store.ListServices(ctx, "", "invalid_key", "asc", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		// Should default to sorting by name
		assert.Equal(t, "api-service", items[0].Name)
	})

	// Test invalid order (should default to ASC)
	t.Run("Invalid order", func(t *testing.T) {
		items, err := store.ListServices(ctx, "", "name", "invalid_order", 10, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		// Should default to ASC order
		assert.Equal(t, "api-service", items[0].Name)
	})

	// Test with versions included
	t.Run("Include versions", func(t *testing.T) {
		items, err := store.ListServices(ctx, "", "", "", 10, 0, true)
		assert.NoError(t, err)
		assert.Len(t, items, 5)
		for _, item := range items {
			assert.Empty(t, item.Versions) // No versions created yet
		}
	})

	// Test combination of search, sort, and pagination
	t.Run("Combined search, sort, and pagination", func(t *testing.T) {
		// Search for "api", sort by name desc, limit 2, offset 0
		items, err := store.ListServices(ctx, "api", "name", "desc", 2, 0, false)
		assert.NoError(t, err)
		if assert.Len(t, items, 1) {
			assert.Equal(t, "api-service", items[0].Name)
		}
	})

	// Test pagination edge cases
	t.Run("Pagination edge cases", func(t *testing.T) {
		// Test with limit larger than total items
		items, err := store.ListServices(ctx, "", "", "", 100, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)

		// Test with negative limit (should use default)
		items, err = store.ListServices(ctx, "", "", "", -1, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)

		// Test with zero limit (should use default)
		items, err = store.ListServices(ctx, "", "", "", 0, 0, false)
		assert.NoError(t, err)
		assert.Len(t, items, 5)

		// Test with large offset (should return empty)
		items, err = store.ListServices(ctx, "", "", "", 10, 100, false)
		assert.NoError(t, err)
		assert.Len(t, items, 0)
	})
}

func TestStore_CreateServiceVersion(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a service first
	service := &Service{
		ID:          uuid.New(),
		Name:        "test-service",
		Description: "A test service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := store.CreateService(ctx, service)
	require.NoError(t, err)

	// Test creating a service version
	version := &ServiceVersion{
		ID:        uuid.New(),
		ServiceID: service.ID,
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
	}

	err = store.CreateServiceVersion(ctx, version)
	assert.NoError(t, err)

	// Test creating duplicate version (should fail)
	err = store.CreateServiceVersion(ctx, version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")
}

func TestStore_ListVersions(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create a service
	service := &Service{
		ID:          uuid.New(),
		Name:        "test-service",
		Description: "A test service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := store.CreateService(ctx, service)
	require.NoError(t, err)

	// Create multiple versions
	versions := []*ServiceVersion{
		{
			ID:        uuid.New(),
			ServiceID: service.ID,
			Version:   "1.0.0",
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        uuid.New(),
			ServiceID: service.ID,
			Version:   "1.1.0",
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        uuid.New(),
			ServiceID: service.ID,
			Version:   "2.0.0",
			CreatedAt: time.Now().UTC(),
		},
	}

	for _, version := range versions {
		err := store.CreateServiceVersion(ctx, version)
		require.NoError(t, err)
	}

	// Test listing versions
	retrievedVersions, err := store.ListVersions(ctx, service.ID)
	assert.NoError(t, err)
	assert.Len(t, retrievedVersions, 3)

	// Test versions are ordered by created_at DESC
	assert.Equal(t, "2.0.0", retrievedVersions[0].Version)
	assert.Equal(t, "1.1.0", retrievedVersions[1].Version)
	assert.Equal(t, "1.0.0", retrievedVersions[2].Version)

	// Test getting service with versions included
	retrieved, err := store.GetService(ctx, service.ID, true)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Len(t, retrieved.Versions, 3)
}

func TestStore_Validation(t *testing.T) {
	store, cleanup := testStore(t)
	defer cleanup()

	ctx := context.Background()

	// Test creating service with empty name (should fail due to CHECK constraint)
	service := &Service{
		ID:          uuid.New(),
		Name:        "",
		Description: "A test service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := store.CreateService(ctx, service)
	assert.Error(t, err) // Should fail due to CHECK (name != '') constraint

	// Create a valid service first
	validService := &Service{
		ID:          uuid.New(),
		Name:        "valid-service",
		Description: "A valid service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err = store.CreateService(ctx, validService)
	require.NoError(t, err)

	// Now test creating a version with empty version string (should fail due to CHECK constraint)
	version := &ServiceVersion{
		ID:        uuid.New(),
		ServiceID: validService.ID,
		Version:   "",
		CreatedAt: time.Now().UTC(),
	}

	err = store.CreateServiceVersion(ctx, version)
	assert.Error(t, err) // Should fail due to CHECK (version != '') constraint

	// Test creating duplicate service (should fail)
	duplicateService := &Service{
		ID:          uuid.New(),
		Name:        "valid-service", // Same name as validService
		Description: "A duplicate service",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err = store.CreateService(ctx, duplicateService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")

	// Test creating duplicate version (should fail)
	validVersion := &ServiceVersion{
		ID:        uuid.New(),
		ServiceID: validService.ID,
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
	}
	err = store.CreateServiceVersion(ctx, validVersion)
	require.NoError(t, err)

	// Try to create the same version again
	duplicateVersion := &ServiceVersion{
		ID:        uuid.New(),
		ServiceID: validService.ID,
		Version:   "1.0.0", // Same version
		CreatedAt: time.Now().UTC(),
	}
	err = store.CreateServiceVersion(ctx, duplicateVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate key")
}
