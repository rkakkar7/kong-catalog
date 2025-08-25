-- Kong Catalog Service Database Schema
-- This file contains all table and index definitions

-- Services table
CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL CHECK (name != ''),
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Service versions table
CREATE TABLE IF NOT EXISTS service_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    version TEXT NOT NULL CHECK (version != ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id, version)
);

-- Indexes
CREATE INDEX IF NOT EXISTS services_name_lower_idx ON services (LOWER(name));
CREATE INDEX IF NOT EXISTS service_versions_by_service_and_created_at ON service_versions (service_id, created_at DESC, id DESC);
