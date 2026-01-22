# gprint - Contract Printing Microservice

A Go microservice for contract printing management focused on services (not goods). Integrates with an existing Rust/Actix-Web authentication backend for login/logout and user management.

## Features

- **Customer Management**: CRUD operations for customers (individuals and companies)
- **Service Catalog**: Manage service offerings with pricing and tax rates
- **Contract Management**: Create, update, sign, and manage service contracts
- **Contract Printing**: Generate contract documents in PDF, DOCX, and HTML formats
- **Multi-tenant Support**: Full tenant isolation with JWT-based authentication
- **Audit Trail**: Complete history tracking for contract changes

## Architecture

```
cmd/
└── server/
    └── main.go           # Application entry point

internal/
├── config/               # Configuration management
│   ├── config.go
│   └── database.go
├── handlers/             # HTTP request handlers
│   ├── customer_handler.go
│   ├── service_handler.go
│   ├── contract_handler.go
│   ├── print_handler.go
│   └── health_handler.go
├── middleware/           # HTTP middleware
│   ├── auth.go
│   ├── cors.go
│   ├── logging.go
│   └── recovery.go
├── models/               # Domain models and DTOs
│   ├── customer.go
│   ├── service.go
│   ├── contract.go
│   ├── history.go
│   ├── print_job.go
│   └── common.go
├── repository/           # Data access layer
│   ├── customer_repository.go
│   ├── service_repository.go
│   ├── contract_repository.go
│   ├── history_repository.go
│   └── print_job_repository.go
├── router/               # HTTP routing
│   └── router.go
└── service/              # Business logic layer
    ├── customer_service.go
    ├── service_service.go
    ├── contract_service.go
    └── print_service.go

migrations/               # Database migrations
└── 001_initial_schema.sql
```

## Prerequisites

- Go 1.22+
- Oracle Database 19c+
- Oracle Instant Client (for godror driver)

## Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/zlovtnik/gprint.git
cd gprint
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your configuration
```

### 3. Install Dependencies

```bash
make deps
```

### 4. Run Database Migrations

```bash
# Using SQL*Plus or your preferred Oracle client
sqlplus user/password@//localhost:1521/ORCL @migrations/001_initial_schema.sql
```

### 5. Run the Application

```bash
make run
```

## API Endpoints

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

### Customers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/customers` | List customers (paginated) |
| GET | `/api/v1/customers/{id}` | Get customer by ID |
| POST | `/api/v1/customers` | Create customer |
| PUT | `/api/v1/customers/{id}` | Update customer |
| DELETE | `/api/v1/customers/{id}` | Soft delete customer |

### Services

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/services` | List services (paginated) |
| GET | `/api/v1/services/{id}` | Get service by ID |
| GET | `/api/v1/services/categories` | List service categories |
| POST | `/api/v1/services` | Create service |
| PUT | `/api/v1/services/{id}` | Update service |
| DELETE | `/api/v1/services/{id}` | Soft delete service |

### Contracts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/contracts` | List contracts (paginated) |
| GET | `/api/v1/contracts/{id}` | Get contract with items |
| POST | `/api/v1/contracts` | Create contract with items |
| PUT | `/api/v1/contracts/{id}` | Update contract |
| PATCH | `/api/v1/contracts/{id}/status` | Change contract status |
| DELETE | `/api/v1/contracts/{id}` | Cancel contract |
| POST | `/api/v1/contracts/{id}/sign` | Sign contract |
| GET | `/api/v1/contracts/{id}/history` | Get contract audit history |

### Contract Items

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/contracts/{id}/items` | Add item to contract |
| DELETE | `/api/v1/contracts/{id}/items/{itemId}` | Remove item from contract |

### Contract Printing

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/contracts/{id}/print` | Create print job |
| GET | `/api/v1/contracts/{id}/print-jobs` | List print jobs for contract |
| GET | `/api/v1/print-jobs/{id}` | Get print job status |
| GET | `/api/v1/print-jobs/{id}/download` | Download generated document |

## Authentication

All `/api/*` endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <jwt-token>
```

The JWT must be issued by the Rust authentication backend and contain:
- `user`: Username
- `login_session`: Session ID
- `tenant_id`: Tenant identifier

## Docker

### Build Image

```bash
make docker-build
```

### Run Container

```bash
make docker-run
```

## Development

### Run with Hot Reload

```bash
# Install air first
go install github.com/cosmtrek/air@latest

# Run with hot reload
make dev
```

### Run Tests

```bash
make test
```

### Run Tests with Coverage

```bash
make test-coverage
```

### Lint Code

```bash
# Install golangci-lint first
make lint
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server bind address | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |
| `ORACLE_HOST` | Oracle database host | `localhost` |
| `ORACLE_PORT` | Oracle database port | `1521` |
| `ORACLE_SERVICE` | Oracle service name | `ORCL` |
| `ORACLE_USER` | Oracle username | - |
| `ORACLE_PASSWORD` | Oracle password | - |
| `JWT_SECRET` | JWT signing secret (must match Rust backend) | - |
| `AUTH_SERVICE_URL` | Rust auth service URL | `http://localhost:8081` |

## License

MIT License
