-include .env
export

.PHONY: help deps openapi migrate up down restart api ps sp build docker-build docker-up docker-down

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Development:"
	@echo "    deps         - Install Go dependencies"
	@echo "    openapi      - Generate OpenAPI server code"
	@echo "    migrate      - Run database migrations (up)"
	@echo "    migrate-down - Rollback database migrations"
	@echo "    api          - Run API server with auto-reload"
	@echo "    ps           - Run projects sync with auto-reload"
	@echo "    sp           - Run search processing worker with auto-reload"
	@echo ""
	@echo "  Infrastructure:"
	@echo "    up           - Start MySQL and Redis services"
	@echo "    down         - Stop MySQL and Redis services"
	@echo "    restart      - Restart MySQL and Redis services"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build - Build the application Docker image"
	@echo "    docker-up    - Start full stack (app + MySQL + Redis)"
	@echo "    docker-down  - Stop full stack"
	@echo ""
	@echo "  Other:"
	@echo "    build        - Build the Go binary"

# Install Go dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod tidy
	@go mod vendor
	@echo "Dependencies installed!"

# Generate OpenAPI server code
openapi:
	@echo "Generating OpenAPI server code..."
	@oapi-codegen -config ./docs/api/http/oapi-codegen-server.yaml ./docs/api/http/openapi.yaml
	@oapi-codegen -config ./docs/api/http/oapi-codegen-models.yaml ./docs/api/http/openapi.yaml
	@echo "OpenAPI server code generated!"

# Run database migrations
migrate:
	@echo "Running migrations..."
	@DB_MIGRATIONS_PATH=./migrations go run ./cmd migrate up
	@echo "Migrations completed!"

migrate-down:
	@echo "Rolling back migrations..."
	@DB_MIGRATIONS_PATH=./migrations go run ./cmd migrate down
	@echo "Rollback completed!"

# Build the Go binary
build:
	@echo "Building..."
	@CGO_ENABLED=0 go build -o bin/app ./cmd
	@echo "Build complete: bin/app"

# Start MySQL and Redis services
up:
	@echo "Starting MySQL and Redis services..."
	@docker compose up -d gpf-mysql gpf-redis
	@echo "Services started!"

# Stop services
down:
	@echo "Stopping services..."
	@docker compose down
	@echo "Services stopped!"

# Restart services
restart: down up

# Run API server with wgo (auto-reload on file changes)
api:
	@wgo run ./cmd api

# Run projects sync with wgo
ps:
	@wgo run ./cmd projects-sync

# Run search processing worker with wgo
sp:
	@wgo run ./cmd search-processing

# Docker targets
docker-build:
	@echo "Building Docker image..."
	@docker compose build gpf-app
	@echo "Docker image built!"

docker-up:
	@echo "Starting full stack..."
	@docker compose up -d --build
	@echo "Full stack started! API available at http://localhost:58080"

docker-down:
	@echo "Stopping full stack..."
	@docker compose down
	@echo "Full stack stopped!"
