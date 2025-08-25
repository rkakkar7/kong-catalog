package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type Service struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Versions    []ServiceVersion `json:"versions,omitempty"`
}

type ServiceVersion struct {
	ID        uuid.UUID `json:"id"`
	ServiceID uuid.UUID `json:"service_id"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// ---- Store ----

type Store struct {
	pool    *pgxpool.Pool
	maxPage int
}

func NewStore(pool *pgxpool.Pool, maxPage int) *Store {
	return &Store{pool: pool, maxPage: maxPage}
}

// GenerateUUID generates a new UUIDv4
func GenerateUUID() uuid.UUID {
	return uuid.New()
}

// ParseUUID parses a string into a UUID, returns error if invalid
func ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// ListServices returns services with offset/limit pagination and optional search.
// sort ∈ {"name","created_at","updated_at"}; order ∈ {"asc","desc"}
func (s *Store) ListServices(ctx context.Context, q, sortKey, order string, limit int, offset int, includeVersions bool) ([]Service, error) {
	if limit <= 0 || limit > s.maxPage {
		limit = s.maxPage
	}
	col := "name"
	switch sortKey {
	case "created_at", "updated_at":
		col = sortKey
	}
	ord := "ASC"
	if strings.EqualFold(order, "desc") {
		ord = "DESC"
	}

	var where []string
	var args []any
	argn := 1
	if q != "" {
		where = append(where, fmt.Sprintf("LOWER(name) LIKE LOWER($%d) || '%%'", argn))
		args = append(args, q)
		argn++
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	sql := fmt.Sprintf(`
		SELECT id, name, coalesce(description,''), created_at, updated_at
		FROM services
		%s
		ORDER BY %s %s, id %s
		LIMIT %d OFFSET %d
	`, whereSQL, col, ord, ord, limit, offset)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Service
	for rows.Next() {
		var x Service
		if err := rows.Scan(&x.ID, &x.Name, &x.Description, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Preload versions for all services only if requested
	if includeVersions && len(items) > 0 {
		serviceIDs := make([]uuid.UUID, len(items))
		for i, service := range items {
			serviceIDs[i] = service.ID
		}

		// Build placeholders for IN clause
		placeholders := make([]string, len(serviceIDs))
		args := make([]any, len(serviceIDs))
		for i, id := range serviceIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}

		versionsSQL := fmt.Sprintf(`
			SELECT id, service_id, version, created_at
			FROM service_versions
			WHERE service_id IN (%s)
			ORDER BY service_id, created_at DESC, id DESC
		`, strings.Join(placeholders, ","))

		versionRows, err := s.pool.Query(ctx, versionsSQL, args...)
		if err != nil {
			return nil, err
		}
		defer versionRows.Close()

		// Group versions by service_id
		versionsByService := make(map[uuid.UUID][]ServiceVersion)
		for versionRows.Next() {
			var v ServiceVersion
			if err := versionRows.Scan(&v.ID, &v.ServiceID, &v.Version, &v.CreatedAt); err != nil {
				return nil, err
			}
			versionsByService[v.ServiceID] = append(versionsByService[v.ServiceID], v)
		}
		if err := versionRows.Err(); err != nil {
			return nil, err
		}

		// Assign versions to services
		for i := range items {
			if versions, exists := versionsByService[items[i].ID]; exists {
				items[i].Versions = versions
			} else {
				items[i].Versions = []ServiceVersion{}
			}
		}
	} else {
		// Set empty versions array if not requested
		for i := range items {
			items[i].Versions = []ServiceVersion{}
		}
	}

	return items, nil
}

func (s *Store) GetService(ctx context.Context, id uuid.UUID, includeVersions bool) (*Service, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, name, coalesce(description,''), created_at, updated_at FROM services WHERE id = $1`, id)
	var x Service
	if err := row.Scan(&x.ID, &x.Name, &x.Description, &x.CreatedAt, &x.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Fetch versions only if requested
	if includeVersions {
		versionRows, err := s.pool.Query(ctx, `
			SELECT id, service_id, version, created_at
			FROM service_versions
			WHERE service_id = $1
			ORDER BY created_at DESC, id DESC
		`, id)
		if err != nil {
			return nil, err
		}
		defer versionRows.Close()

		var versions []ServiceVersion
		for versionRows.Next() {
			var v ServiceVersion
			if err := versionRows.Scan(&v.ID, &v.ServiceID, &v.Version, &v.CreatedAt); err != nil {
				return nil, err
			}
			versions = append(versions, v)
		}
		if err := versionRows.Err(); err != nil {
			return nil, err
		}

		x.Versions = versions
	} else {
		x.Versions = []ServiceVersion{}
	}

	return &x, nil
}

func (s *Store) ListVersions(ctx context.Context, id uuid.UUID) ([]ServiceVersion, error) {
	sql := `
		SELECT id, service_id, version, created_at
		FROM service_versions
		WHERE service_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := s.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ServiceVersion
	for rows.Next() {
		var v ServiceVersion
		if err := rows.Scan(&v.ID, &v.ServiceID, &v.Version, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versions, nil
}

// CreateService creates a new service
func (s *Store) CreateService(ctx context.Context, service *Service) error {
	service.ID = GenerateUUID()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	return s.pool.QueryRow(ctx, `INSERT INTO services (id, name, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`, service.ID, service.Name, service.Description, service.CreatedAt, service.UpdatedAt).Scan(&service.ID)
}

// CreateServiceVersion creates a new service version
func (s *Store) CreateServiceVersion(ctx context.Context, serviceVersion *ServiceVersion) error {
	serviceVersion.ID = GenerateUUID()
	serviceVersion.CreatedAt = time.Now()

	return s.pool.QueryRow(ctx, `INSERT INTO service_versions (id, service_id, version, created_at) VALUES ($1, $2, $3, $4) RETURNING id`, serviceVersion.ID, serviceVersion.ServiceID, serviceVersion.Version, serviceVersion.CreatedAt).Scan(&serviceVersion.ID)
}
