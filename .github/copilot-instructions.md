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
    id, err := parseIDFromPath(r, "id")              // Shared helper in internal/handlers/helpers.go
    // ... call service, use writeJSON() or writeError()
}
```

- Use `middleware.GetTenantID(ctx)` and `middleware.GetUser(ctx)` for auth context
- Use `writeJSON(w, status, data)` and `writeError(w, status, code, msg)` helpers (defined in `customer_handler.go`)
- Error codes/messages defined in `internal/handlers/errors.go`
- Handler constructors panic on nil dependencies for fail-fast startup

### Model Pattern

Each entity has 3-4 types in `internal/models/`:
- `Customer` - full domain model with all fields including `tenant_id`
- `CreateCustomerRequest` / `UpdateCustomerRequest` - input DTOs
- `CustomerResponse` - output DTO (excludes `tenant_id`, internal audit fields)
- Use `entity.ToResponse()` method to convert domain → response

### Repository Pattern

- Raw SQL with Oracle positional params (`:1`, `:2`)
- Use `sql.NullString` for nullable columns, convert to struct fields after `Scan()`
- **All queries must filter by `tenant_id`** for multi-tenant isolation
- Use `scanCustomer()` pattern to handle nullable fields consistently

### API Response Format

```go
// Success: wrap data with models.SuccessResponse()
writeJSON(w, http.StatusOK, models.SuccessResponse(customer.ToResponse()))

// Error: use writeError() which calls models.ErrorResponse()
writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgCustomerNotFound)

// Paginated lists: use models.NewPaginatedResponse()
result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
```

### Routing

Go 1.22+ routing with method prefix: `"METHOD /path/{param}"`

```go
r.mux.HandleFunc("GET /api/v1/customers/{id}", h.Get)
r.mux.HandleFunc("POST /api/v1/contracts/{id}/print", h.CreateJob)
```

Health endpoints (`/health`, `/ready`) bypass auth middleware.

## Development Commands

```bash
make run     # Run locally (auto-sets Oracle Instant Client paths)
make dev     # Hot reload with air (go install github.com/cosmtrek/air@latest)
make test    # Tests with race detection
make lint    # Requires golangci-lint
```

**Oracle Instant Client**: Libraries in `./lib/` (macOS) and `./lib-linux/` (Linux). Makefile auto-sets:
- macOS: `DYLD_LIBRARY_PATH`
- Linux: `LD_LIBRARY_PATH`
- `TNS_ADMIN` → `./wallet/`

## Environment Variables

**Required:**
- `JWT_SECRET` - HMAC secret for HS256 token validation
- `ORACLE_USER`, `ORACLE_PASSWORD`

**Oracle Cloud (ADB):**
- `ORACLE_WALLET_PATH`, `ORACLE_TNS_ALIAS`

## Authentication

JWT validation only (tokens issued by external Rust/Actix-Web auth service):
- Algorithm: HS256 (symmetric) via `pkg/auth/jwt.go` (Keycloak integration in `pkg/auth/keycloak.go`)
- Claims: `user`, `tenant_id`, `login_session`
- Middleware extracts to context in `internal/middleware/auth.go`

## Adding a New Entity

1. **Models** in `internal/models/entity.go`: Entity, CreateRequest, UpdateRequest, Response + `ToResponse()` method
2. **Repository** in `internal/repository/entity_repository.go`: CRUD with tenant isolation
3. **Service** in `internal/service/entity_service.go`: business logic, error wrapping
4. **Handler** in `internal/handlers/entity_handler.go`: HTTP handlers, validation
5. **Routes** in `internal/router/router.go`: register endpoints
6. **Wire up** in `cmd/server/main.go`: add to `repositories`, `services`, `handlerSet` structs
