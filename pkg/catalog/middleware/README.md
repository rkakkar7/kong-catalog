# Middleware Package

This package contains all the middleware used by the catalog service, organized in separate files for clarity and maintainability.

## Middleware Files

### 1. `logging.go`
- **RequestIDMiddleware**: Generates and adds unique request IDs to each request
- **LoggingMiddleware**: Logs request details and adds logger to request context
- **RequestLogger interface**: Defines logging contract
- **ZeroLogger**: Implements RequestLogger using zerolog

### 2. `validation.go`
- **ValidationMiddleware**: Validates request parameters based on validation functions
- **handleValidationError**: Handles validation errors and returns appropriate HTTP responses

### 3. `timeout.go`
- **TimeoutMiddleware**: Creates a timeout handler that cancels requests after specified duration

### 4. `auth.go`
- **APIKeyMiddleware**: Validates API keys from x-api-key headers
- Skips authentication for health check endpoints (`/healthz`, `/readyz`)
- Expects `x-api-key: <api-key>` format

### 5. `setup.go`
- **SetupGlobalMiddleware**: Applies all global middleware in the correct order
- **SetupRouteSpecificMiddleware**: Applies middleware to specific routes

## Middleware Order

The middleware is applied in the following order (from first to last):

1. **Request ID Middleware** - Adds unique ID to each request
2. **Logging Middleware** - Logs request details and adds logger to context
3. **Authentication Middleware** - Validates API keys (skips health checks)
4. **Timeout Middleware** - Cancels requests that take too long (5 seconds)

## Usage

### Global Middleware
```go
// In app.go
middleware.SetupGlobalMiddleware(r, cfg.ValidAPIKeys)
```

### Route-Specific Middleware
```go
// In routes.go
r.With(middleware.ValidationMiddleware(validation.ValidateListServicesParams)).
  Get("/services", servicesHandler.ListServices)
```

## Configuration

API keys are configured in the configuration files:
- `config/default.yaml`
- `config/local.yaml`
- `docker-compose.yaml` (via environment variable)

The `VALID_API_KEYS` environment variable can override the YAML configuration.

## Health Check Endpoints

The following endpoints are exempt from authentication:
- `/healthz` - Health check
- `/readyz` - Readiness check

## Example Request

```bash
curl -H "x-api-key: your-api-key-here" \
     -H "X-Request-ID: custom-request-id" \
     http://localhost:8080/v1/services
```
