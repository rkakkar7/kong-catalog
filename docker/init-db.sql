-- Create a dedicated user for the catalog application
CREATE USER catalog_user WITH PASSWORD 'catalog_password';

-- Create the kong_catalog database
CREATE DATABASE kong_catalog;

-- Grant necessary permissions to the catalog user
GRANT ALL PRIVILEGES ON DATABASE kong_catalog TO catalog_user;

-- Connect to the kong_catalog database and set permissions
\connect kong_catalog;

-- Grant schema permissions (this will be applied to tables created by the app)
GRANT ALL ON SCHEMA public TO catalog_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO catalog_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO catalog_user;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO catalog_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO catalog_user;
