# Models Package

This package contains the data models, database schema, and data access layer for the catalog service.

## Package Structure

### `store.go`
- **Store**: Main data access layer with methods for CRUD operations
- **Service**: Service entity with UUID ID, name, description, and timestamps
- **ServiceVersion**: Service version entity with UUID ID, service reference, and version info
- **UUID utilities**: Helper functions for UUID generation and parsing

### `schema.go`
- **Schema**: Database table definitions as SQL strings
- **Indexes**: Database index definitions for performance optimization
- **EnsureSchema**: Creates all tables and indexes if they don't exist
- **DropSchema**: Drops all tables (useful for testing)

## Data Models

### Service
```go
type Service struct {
    ID          uuid.UUID        `json:"id"`
    Name        string           `json:"name"`
    Description string           `json:"description"`
    CreatedAt   time.Time        `json:"created_at"`
    UpdatedAt   time.Time        `json:"updated_at"`
    Versions    []ServiceVersion `json:"versions,omitempty"`
}
```

### ServiceVersion
```go
type ServiceVersion struct {
    ID        uuid.UUID `json:"id"`
    ServiceID uuid.UUID `json:"service_id"`
    Version   string    `json:"version"`
    CreatedAt time.Time `json:"created_at"`
}
```

## Database Schema

### Services Table
- `id`: UUID PRIMARY KEY (auto-generated)
- `name`: TEXT NOT NULL
- `description`: TEXT NOT NULL DEFAULT ''
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT now()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT now()

### Service Versions Table
- `id`: UUID PRIMARY KEY (auto-generated)
- `service_id`: UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE
- `version`: TEXT NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT now()

## Indexes

1. **services_name_lower_idx**: Case-insensitive name search
2. **services_fts_idx**: Full-text search on name and description
3. **service_versions_by_service**: Optimized service version queries

## UUID Usage

All entity IDs use UUIDv4 format:
- **Generation**: `models.GenerateUUID()` creates new UUIDs
- **Parsing**: `models.ParseUUID(string)` validates and parses UUIDs
- **Validation**: Middleware ensures all IDs are valid UUIDv4
- **Database**: Uses PostgreSQL's `gen_random_uuid()` for auto-generation

## Cursor-based Pagination

The service supports cursor-based pagination for efficient large dataset handling:
- **Format**: `{sort_key}|{uuid_id}`
- **Example**: `2024-01-01T00:00:00Z|550e8400-e29b-41d4-a716-446655440000`
- **Functions**: `makeUUIDCursor()` and `splitUUIDCursor()` handle cursor operations

## Usage Examples

### Creating a new service
```go
service := &models.Service{
    Name: "My Service",
    Description: "Service description",
}
// ID will be auto-generated when saved to database
```

### Parsing a UUID from string
```go
id, err := models.ParseUUID("550e8400-e29b-41d4-a716-446655440000")
if err != nil {
    // Handle invalid UUID
}
```

### Schema management
```go
// Ensure all tables and indexes exist
err := store.EnsureSchema(ctx)

// Drop all tables (for testing)
err := models.DropSchema(ctx, pool)
```
