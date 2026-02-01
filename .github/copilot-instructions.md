# gprint - AI Coding Instructions

## Architecture Overview

Go microservice for contract printing management (services-only, multi-tenant). Uses a **layered architecture**:

```
handlers → service → repository → Oracle DB
```

- **Handlers** in [internal/handlers](internal/handlers): HTTP request handling, JSON serialization, error responses.
- **Service** in [internal/service](internal/service): business logic, orchestrates repository calls.
- **Repository** in [internal/repository](internal/repository): raw SQL via `database/sql`, Oracle positional params (`:1`, `:2`).
- **Models** in [internal/models](internal/models): domain entities + request/response DTOs.

Multi-tenant isolation: every data access must filter by `tenant_id` from JWT claims.

## Critical Conventions & Patterns

### Handler Pattern

```go
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
    tenantID := middleware.GetTenantID(r.Context())  // Always extract first
    id, err := parseIDFromPath(r, "id")              // helpers.go
    // ... call service, use writeJSON() or writeError()
}
```

- Use `middleware.GetTenantID(ctx)` and `middleware.GetUser(ctx)` for auth context.
- Response helpers `writeJSON()` / `writeError()` live in [internal/handlers/customer_handler.go](internal/handlers/customer_handler.go).
- Error codes/messages in [internal/handlers/errors.go](internal/handlers/errors.go).
- Handler constructors should panic on nil dependencies (fail-fast).

### Model Pattern

Each entity has 3–4 types under [internal/models](internal/models):

- `Entity` (full domain, includes `tenant_id`), `CreateEntityRequest`, `UpdateEntityRequest`, `EntityResponse`.
- Convert domain → response via `entity.ToResponse()`.

### Repository Pattern

- Always include `tenant_id` filters in SQL queries.
- Use `sql.NullString` for nullable columns and a `scanX()` helper for consistent `Scan()` handling.

### API Response Format

```go
writeJSON(w, http.StatusOK, models.SuccessResponse(entity.ToResponse()))
writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgCustomerNotFound)
result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
```

### Routing

Go 1.22+ method-based routing in [internal/router/router.go](internal/router/router.go):

```go
r.mux.HandleFunc("GET /api/v1/customers/{id}", h.Get)
r.mux.HandleFunc("POST /api/v1/contracts/{id}/print", h.CreateJob)
```

Health endpoints `/health` and `/ready` bypass auth middleware.

## Integration Points

- Oracle DB (19c+) using `godror` driver; Instant Client libs live under [lib](lib) (macOS) and [lib-linux](lib-linux).
- External Rust/Actix-Web auth service issues JWTs; HS256 validation in [pkg/auth/jwt.go](pkg/auth/jwt.go).
- Tenant and user claims are added to context in [internal/middleware/auth.go](internal/middleware/auth.go).

## Developer Workflows

- `make run` (auto-sets Instant Client env: `DYLD_LIBRARY_PATH`/`LD_LIBRARY_PATH`, `TNS_ADMIN` → [wallet](wallet)).
- `make dev` for hot reload (requires `air`).
- `make test`, `make test-coverage`, `make lint`.
- Migrations in [migrations](migrations) are Oracle SQL scripts (see [README.md](README.md)).

## Adding a New Entity (Order Matters)

1. Add models in [internal/models](internal/models) with `ToResponse()`.
2. Add repository in [internal/repository](internal/repository) (tenant filters + `scanX()`).
3. Add service in [internal/service](internal/service).
4. Add handler in [internal/handlers](internal/handlers).
5. Register routes in [internal/router/router.go](internal/router/router.go).
6. Wire in [cmd/server/main.go](cmd/server/main.go).
