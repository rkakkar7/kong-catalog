# Kong Catalog Service

A robust, production-ready service catalog API built with Go, PostgreSQL, and Docker. This service provides a centralized registry for managing microservices, their versions, and metadata.

## 🏗️ Architecture

### Tech Stack
- **Language**: Go 1.24
- **Database**: PostgreSQL 17
- **Router**: Chi (lightweight HTTP router)
- **Logging**: Zerolog (structured logging)
- **Configuration**: Environment-based with YAML fallbacks
- **Containerization**: Docker & Docker Compose
- **Testing**: Testify with real PostgreSQL integration

### Project Structure
```
kong/
├── cmd/catalog/           # Application entry point
├── pkg/
│   ├── catalog/          # Main application logic
│   │   ├── handlers/     # HTTP request handlers
│   │   ├── middleware/   # HTTP middleware (auth, logging, validation)
│   │   ├── routes/       # Route definitions
│   │   └── validation/   # Request validation
│   ├── config/           # Configuration management
│   └── models/           # Data models and database operations
├── docker/               # Docker configuration
├── config/               # Configuration files
├── scripts/              # Utility scripts
└── .vscode/             # VS Code debugging configuration
```

## 🚀 Quick Start

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- PostgreSQL (for local development)

### Running with Docker (Recommended)
```bash
# Start all services
make docker-up

# Check logs
make docker-logs

# Stop services
make docker-down
```

### Running Locally
```bash
# Setup local database
make local

# Or run directly
go run cmd/catalog/main.go
```

## 📋 API Documentation

### Authentication
All API endpoints require authentication using API keys:
```
x-api-key: <api-key>
```

**Available API Keys:**
- `docker-api-key` (Docker environment)
- `local-api-key` (Local environment)

### Endpoints

#### Health Check
```http
GET /healthz
```

#### Services

**List Services**
```http
GET /v1/services?q=<search>&sort=<field>&order=<asc|desc>&limit=<number>&offset=<number>&include_versions=<true|false>
```

**Get Service**
```http
GET /v1/services/{id}?include_versions=<true|false>
```

**Create Service**
```http
POST /v1/services
Content-Type: application/json

{
  "name": "service-name",
  "description": "Service description"
}
```

**List Service Versions**
```http
GET /v1/services/{id}/versions
```

**Create Service Version**
```http
POST /v1/services/{id}/versions
Content-Type: application/json

{
  "version": "1.0.0"
}
```

### Query Parameters

#### List Services
- `q` - Search query (filters by service name)
- `sort` - Sort field (`name`, `created_at`, `updated_at`)
- `order` - Sort order (`asc`, `desc`)
- `limit` - Maximum items per page (default: 100, max: 1000)
- `offset` - Number of items to skip
- `include_versions` - Include service versions in response

#### Get Service
- `include_versions` - Include service versions in response

### Response Format

**Success Response**
```json
{
  "id": "uuid",
  "name": "service-name",
  "description": "Service description",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "versions": [
    {
      "id": "uuid",
      "service_id": "uuid",
      "version": "1.0.0",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Error Response**
```json
{
  "message": "Error description",
  "error": "detailed_error_message"
}
```

## 🧪 Testing

### Running Tests
```bash
# Run all tests with Docker database
make test-docker

# Run tests with local database
make test-local

# Run tests with coverage
make test-coverage
```

### Test Structure
- **Integration Tests**: Use real PostgreSQL via Docker Compose
- **Schema Validation**: Ensures database constraints work correctly
- **Pagination Tests**: Comprehensive offset/limit pagination testing
- **Search & Filter**: Tests name-based search functionality
- **Sorting**: Tests all sortable fields and orders
- **Edge Cases**: Tests boundary conditions and error scenarios

### Test Coverage
- ✅ CRUD operations for services and versions
- ✅ Pagination with offset/limit
- ✅ Search and filtering
- ✅ Sorting by multiple fields
- ✅ Database constraints (UNIQUE, CHECK)
- ✅ Error handling and validation
- ✅ Authentication middleware

## 🔧 Configuration

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://user:pass@host:port/db?sslmode=disable
DB_MAX_CONNECTIONS=10
DB_MIN_CONNECTIONS=2
DB_MAX_CONN_LIFETIME=1h
DB_MAX_CONN_IDLE_TIME=15m
DB_CONNECT_TIMEOUT=5s
DB_HEALTH_CHECK_PERIOD=1m

# Server
PORT=8080
ENV=local|production

# API Keys (comma-separated)
VALID_API_KEYS=key1,key2,key3

# Pagination
MAX_PAGE_SIZE=1000
```

### Configuration Files
- `config/default.yaml` - Default configuration
- `config/local.yaml` - Local development overrides

## 🗄️ Database Schema

### Tables

**services**
```sql
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL CHECK (name != ''),
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**service_versions**
```sql
CREATE TABLE service_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    version TEXT NOT NULL CHECK (version != ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id, version)
);
```

### Indexes
- `services_name_lower_idx` - Case-insensitive name search
- `service_versions_by_service_and_created_at` - Efficient version listing

### Constraints
- **UNIQUE**: Service names, service version combinations
- **CHECK**: Non-empty names and versions
- **FOREIGN KEY**: Service versions reference services

## 🔒 Security Features

### Authentication
- API key-based authentication
- Bearer token format
- Configurable valid keys per environment

### Validation
- Request parameter validation
- Content-Type header validation
- Database constraint enforcement
- Input sanitization

### Error Handling
- Graceful error responses
- Consistent error format
- No sensitive information leakage

## 🚀 Production Deployment

### Docker Deployment
```bash
# Build and run
docker-compose up --build -d

# Scale services
docker-compose up --scale catalog-api=3 -d
```

## 🛠️ Development

### Code Organization
- **Clean Architecture**: Separation of concerns
- **Middleware Pattern**: Reusable request processing
- **Repository Pattern**: Database abstraction
- **Validation Layer**: Input validation and sanitization

## 📝 API Examples

### Create a Service
```bash
curl -X POST http://localhost:8080/v1/services \
  -H "x-api-key: docker-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "description": "User management service"
  }'
```

### List Services with Pagination
```bash
curl "http://localhost:8080/v1/services?limit=10&offset=0&sort=name&order=asc" \
  -H "x-api-key: docker-api-key"
```

### Search Services
```bash
curl "http://localhost:8080/v1/services?q=user&include_versions=true" \
  -H "x-api-key: docker-api-key"
```

### Create Service Version
```bash
curl -X POST http://localhost:8080/v1/services/{service-id}/versions \
  -H "x-api-key: docker-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1.0.0"
  }'
```