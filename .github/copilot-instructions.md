# gprint - AI Coding Instructions

## Architecture Overview

Go microservice for contract printing management. Uses a **layered architecture**:

```
handlers → service → repository → Oracle DB
```

- **Handlers** (`internal/handlers/`): HTTP request handling, JSON serialization, error responses
- **Service** (`internal/service/`): Business logic, coordinates repositories
- **Repository** (`internal/repository/`): Raw SQL with `database/sql`, no ORM
- **Models** (`internal/models/`): Domain entities + Request/Response DTOs

Multi-tenant isolation: All data operations require `tenant_id` from JWT claims.

## Key Conventions

### Handler Pattern

```go
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
    tenantID := middleware.GetTenantID(r.Context())  // Always extract first
    id, err := parseIDFromPath(r, "id")              // Defined in customer_handler.go
    // ... call service, use writeJSON() or writeError()
}
```

**Helper function** `parseIDFromPath` is defined in `internal/handlers/customer_handler.go`:

```go
func parseIDFromPath(r *http.Request, name string) (int64, error) {
    idStr := r.PathValue(name)
    return strconv.ParseInt(idStr, 10, 64)
}
```

- Use `middleware.GetTenantID(ctx)` and `middleware.GetUser(ctx)` for auth context
- Use `writeJSON(w, status, data)` and `writeError(w, status, code, msg)` helpers
- Error codes defined in `internal/handlers/errors.go`

### Model Pattern
Each entity has 3 types:
- `Customer` - full domain model with all fields
- `CreateCustomerRequest` / `UpdateCustomerRequest` - input DTOs
- `CustomerResponse` - output DTO (excludes `tenant_id`, internal fields)
- Use `entity.ToResponse()` to convert domain → response

### Repository Pattern
- Raw SQL with positional params (`:1`, `:2` for Oracle)
- `sql.NullString` for nullable fields, convert to struct fields after scan
- All queries filter by `tenant_id` for isolation

### API Response Format
```go
// Success: models.SuccessResponse(data)
{"success": true, "data": {...}}

// Error: models.ErrorResponse(code, message, details)
{"success": false, "error": {"code": "...", "message": "..."}}

// Paginated: models.NewPaginatedResponse(items, page, pageSize, total)
{"data": [...], "page": 1, "page_size": 20, "total_count": 100, "total_pages": 5}
```

## Development Commands

```bash
# Run locally (sets up Oracle Instant Client paths)
make run

# Run with hot reload
make dev  # requires: go install github.com/cosmtrek/air@latest

# Tests with race detection
make test

# Lint (requires golangci-lint)
make lint
```

**Oracle Instant Client**: Libraries in `./lib/`, wallet in `./wallet/`. The Makefile automatically sets the correct library path environment variable per platform:

- **macOS**: `DYLD_LIBRARY_PATH`
- **Linux**: `LD_LIBRARY_PATH`
- **Windows**: `PATH`

`TNS_ADMIN` is also set automatically to point to the wallet directory.

## Environment Variables

Required:

- `JWT_PUBLIC_KEY` - PEM-encoded RS256 public key for token verification (auth service holds the private key)
- `ORACLE_USER`, `ORACLE_PASSWORD` - database credentials

For Oracle Cloud (ADB):

- `ORACLE_WALLET_PATH`, `ORACLE_TNS_ALIAS` - wallet-based auth

## Authentication

JWT tokens issued by external Rust/Actix-Web service. This service **validates** tokens only (no token creation):

- Algorithm: RS256 (asymmetric)
- Claims: `user`, `tenant_id`, `login_session`
- Validation in `internal/middleware/auth.go` using `JWT_PUBLIC_KEY`
- The auth service signs tokens with its private key; this service only needs the public key to verify

## Adding a New Entity

1. Define models in `internal/models/entity.go` (Entity, CreateRequest, UpdateRequest, Response, ToResponse method)
2. Create repository in `internal/repository/entity_repository.go`
3. Create service in `internal/service/entity_service.go`
4. Create handler in `internal/handlers/entity_handler.go`
5. Register routes in `internal/router/router.go`
6. Wire up in `cmd/server/main.go` (repo → service → handler → router)

## Routes

Go 1.22+ routing pattern: `"METHOD /path/{param}"`
```go
r.mux.HandleFunc("GET /api/v1/customers/{id}", h.Get)
r.mux.HandleFunc("POST /api/v1/contracts/{id}/print", h.CreateJob)
```
Health endpoints (`/health`, `/ready`) bypass auth middleware.
