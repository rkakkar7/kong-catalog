package models

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaFS embed.FS

// GetSchemaSQL reads the schema from the embedded SQL file
func GetSchemaSQL() (string, error) {
	content, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return "", fmt.Errorf("failed to read schema.sql: %w", err)
	}
	return string(content), nil
}

// EnsureSchema creates all tables and indexes if they don't exist
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Read schema from SQL file
	schemaSQL, err := GetSchemaSQL()
	if err != nil {
		return err
	}

	// Execute the schema SQL
	_, err = pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// DropSchema drops all tables (useful for testing)
func DropSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// Drop in reverse order due to foreign key constraints
	dropSQL := []string{
		"DROP TABLE IF EXISTS service_versions CASCADE;",
		"DROP TABLE IF EXISTS services CASCADE;",
	}

	for _, sql := range dropSQL {
		_, err := pool.Exec(ctx, sql)
		if err != nil {
			return err
		}
	}

	return nil
}
