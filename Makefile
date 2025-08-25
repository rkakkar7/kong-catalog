.PHONY: help local docker-up docker-down docker-logs build clean test test-docker test-local test-coverage

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

local: ## Run the application locally (requires local PostgreSQL)
	@echo "🚀 Starting catalog API locally..."
	@ENV=local go run cmd/catalog/main.go

docker-up: ## Start the application with Docker Compose
	@echo "🐳 Starting services with Docker Compose..."
	docker-compose up --build -d

docker-down: ## Stop Docker Compose services
	@echo "🛑 Stopping Docker Compose services..."
	docker-compose down

docker-logs: ## Show Docker Compose logs
	@echo "📋 Showing Docker Compose logs..."
	docker-compose logs -f

build: ## Build the application
	@echo "🔨 Building application..."
	go build -o bin/catalog ./cmd/catalog

clean: ## Clean up Docker resources
	@echo "🧹 Cleaning up Docker resources..."
	docker-compose down -v --remove-orphans
	docker system prune -f

# Run tests using Docker Compose database
test-docker: docker-up
	TEST_DATABASE_URL="postgres://catalog_user:catalog_password@localhost:5432/kong_catalog?sslmode=disable" go test ./pkg/models -v

# Run all tests
test: test-local

# Run tests with coverage
test-coverage:
	TEST_DATABASE_URL="postgres://catalog_user:catalog_password@localhost:5432/kong_catalog?sslmode=disable" go test ./pkg/models -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
