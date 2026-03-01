# Contributing to GitLab Package Finder

Thank you for your interest in contributing! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://gitlab.com/your-username/gitlab-package-finder.git`
3. Create a branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Submit a merge request

## Development Setup

### Prerequisites

- Go 1.26+
- Docker and Docker Compose
- A GitLab personal access token with `read_api` scope

### Setup

```bash
# Install dependencies
make deps

# Copy env and configure your GitLab token
cp .env.example .env
# Edit .env with your GITLAB_TOKEN and GITLAB_BASE_URL

# Start MySQL and Redis
make up

# Run database migrations
go run ./cmd migrate up

# Start the API server (with auto-reload)
make api
```

### Running Commands

```bash
# Sync projects from GitLab
make ps

# Run search processing worker
make sp
```

## Architecture

The project follows Hexagonal Architecture with DDD principles:

```
internal/
├── domain/          # Pure business logic (zero external imports)
├── ports/           # Interface definitions (one per file)
├── services/        # Business orchestration
├── adapters/        # Port implementations (MySQL, Redis, GitLab)
├── app/             # HTTP handlers, factories, init helpers
├── jobs/            # Background job implementations
└── worker/          # Generic worker framework
```

**Dependency rule**: `domain <- ports <- services <- app/jobs <- cmd`

### Adding a New Package Manager

1. Create `internal/domain/packagemanager/yourpm/packagemanager.go`
2. Implement the `packagemanager.PackageManager` interface
3. Register it in `cmd/api.go` and `cmd/searchprocessing/command.go`
4. Add the package type to the database migration

## Commit Messages

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Common types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `ci`.

Examples:
- `feat: add pip package manager support`
- `fix(search): prevent duplicate scans for the same package`
- `docs: update configuration table in README`

## API Development (OpenAPI-First)

This project follows an **OpenAPI-first** approach for HTTP API development. The OpenAPI specification is the single source of truth for all API endpoints.

### How It Works

1. **Define the endpoint** in `docs/api/http/openapi.yaml` (paths, parameters, request/response schemas)
2. **Generate server code** by running:
   ```bash
   make openapi
   ```
   This runs [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) and produces:
   - `pkg/oapi/api.gen.go` — server interface, strict handler, request/response types, routing
   - `pkg/oapi/models.gen.go` — model structs from component schemas
3. **Implement the handler** by adding a method on `API` that satisfies the generated `StrictServerInterface`

### Adding a New Endpoint

1. Add the path and operation to `docs/api/http/openapi.yaml`, following existing conventions (reuse shared schemas like `400Response`, `500Response`, `MetaResponse`, etc.)
2. Run `make openapi` to regenerate
3. The build will fail until you implement the new method on `API` (the generated strict interface enforces this)
4. Add your handler method in the appropriate file under `internal/app/api/http/`

### Important Rules

- **Never edit generated files** (`pkg/oapi/api.gen.go`, `pkg/oapi/models.gen.go`) — your changes will be overwritten
- **Never register routes manually** — all routing is handled by the generated `HandlerFromMux`
- **Always run `make openapi`** after modifying `openapi.yaml` to keep generated code in sync
- Non-API endpoints (monitoring: `/metrics`, `/kubernetes/*`, `/ping`) are excluded from strict generation via `exclude-operation-ids` in the codegen config

## Code Guidelines

- Follow existing code patterns
- Keep domain layer free of infrastructure imports
- Write tests for new services
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Run `go build ./...` and `go test ./...` before submitting

## Reporting Issues

- Use the issue tracker to report bugs
- Include steps to reproduce, expected behavior, and actual behavior
- For security vulnerabilities, please report privately
