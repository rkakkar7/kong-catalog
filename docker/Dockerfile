# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o catalog ./cmd/catalog

# Final stage
FROM postgres:17-alpine

# Install required tools for database setup
RUN apk add --no-cache postgresql-client

# Create app user
RUN adduser -D -s /bin/sh app

# Copy the binary from builder stage
COPY --from=builder /app/catalog /app/catalog

# Copy docker files
COPY --from=builder /app/docker /app/docker

# Copy configuration
COPY --from=builder /app/config /app/config

# Copy schema file
COPY --from=builder /app/pkg/models/schema.sql /app/pkg/models/schema.sql

# Set ownership
RUN chown -R app:app /app

# Switch to app user
USER app

# Set working directory
WORKDIR /app

# Make entrypoint script executable
RUN chmod +x /app/docker/entrypoint.sh

# Expose port
EXPOSE 8080

# Use the entrypoint script
ENTRYPOINT ["/app/docker/entrypoint.sh"]
