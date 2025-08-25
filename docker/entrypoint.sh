#!/bin/bash

# Docker entrypoint for Kong Catalog Service
# This script handles database setup and starts the application

set -e

echo "Starting Kong Catalog Service..."

# Set default environment variables if not provided
export DB_HOST=${DB_HOST:-db}
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-catalog_user}
export DB_NAME=${DB_NAME:-kong_catalog}
export PGPASSWORD=${POSTGRES_PASSWORD:-postgres}

echo "Database configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  User: $DB_USER"
echo "  Database: $DB_NAME"

# Database setup section
echo "Setting up database schema for Kong Catalog Service..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U postgres -d postgres; do
    echo "PostgreSQL is not ready yet, waiting..."
    sleep 2
done

echo "PostgreSQL is ready!"

# Create database if it doesn't exist
echo "Ensuring database exists..."
psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "Database already exists"

# Connect to the target database and create schema
echo "Creating schema..."
PGPASSWORD=catalog_password psql -h "$DB_HOST" -p "$DB_PORT" -U catalog_user -d "$DB_NAME" -f /app/pkg/models/schema.sql

echo "Database schema setup complete!"
echo "Tables created: services, service_versions"
echo "Indexes created: services_name_lower_idx, service_versions_by_service_and_created_at"

# Start the application
echo "Starting application..."
exec /app/catalog
