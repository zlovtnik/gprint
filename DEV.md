# Go Microservice Development Specification

## Contract/Services/Customer Management with Oracle Database

**Version:** 1.0.0  
**Date:** January 22, 2026  
**Status:** Draft

---

## 1. Overview

This document specifies a Go (Golang) backend microservice for **contract printing management** focused on **services** (not goods). The microservice integrates with the existing Rust/Actix-Web authentication backend for login/logout and user management.

### 1.1 Core Purpose

- Manage **customers** who contract services
- Manage **services** offered (service catalog)
- Manage **contracts** binding customers to services
- Generate **printable contract documents** for services

### 1.2 Integration Points

| Component | Technology | Purpose |
|-----------|------------|---------|
| Authentication | Rust/Actix-Web (JWT) | Login/Logout/User Management |
| Database | Oracle | Contract/Services/Customers data |
| Contract Printing | Go service | PDF/Document generation |

---

## 2. Authentication Integration

The Go microservice relies on the existing Rust backend for authentication. It validates JWTs issued by the Rust backend.

### 2.1 JWT Token Structure

The Rust backend issues JWTs with the following claims:

```json
{
  "iat": 1706000000,       // Issued at (Unix timestamp)
  "exp": 1706604800,       // Expiration (Unix timestamp, default: 1 week)
  "user": "alice",         // Username
  "login_session": "uuid-v4-session-id",  // Session identifier
  "tenant_id": "tenant1"   // Multi-tenant identifier
}
```

### 2.2 JWT Signing

| Parameter | Value |
|-----------|-------|
| Algorithm | HS256 (HMAC-SHA256) |
| Secret Key | Loaded from `JWT_SECRET` env var or `secret.key` file |

### 2.3 Authentication Endpoints (Rust Backend)

The Go microservice should **delegate** authentication to these existing endpoints:

| Method | Endpoint | Description | Request Body |
|--------|----------|-------------|--------------|
| POST | `/api/auth/signup` | User registration | `SignupDTO` |
| POST | `/api/auth/login` | User login (returns JWT) | `LoginDTO` |
| POST | `/api/auth/logout` | Invalidate session | Bearer token header |
| POST | `/api/auth/refresh` | Refresh access token | `RefreshTokenRequest` |
| POST | `/api/auth/refresh-token` | Alternative refresh endpoint | `RefreshTokenRequest` |
| GET | `/api/auth/me` | Get current user info | Bearer token header |
| GET | `/api/auth/login/keycloak` | Keycloak OAuth2 redirect | - |
| POST | `/api/auth/callback/keycloak` | Keycloak OAuth2 callback | `KeycloakCallbackRequest` |

### 2.4 Data Transfer Objects (DTOs)

#### SignupDTO
```go
type SignupDTO struct {
    Username  string `json:"username" validate:"required,min=3,max=50"`
    Email     string `json:"email" validate:"required,email"`
    Password  string `json:"password" validate:"required,min=8"`
    TenantID  string `json:"tenant_id" validate:"required"`
}
```

#### LoginDTO
```go
type LoginDTO struct {
    UsernameOrEmail string `json:"username_or_email" validate:"required"`
    Password        string `json:"password" validate:"required"`
    TenantID        string `json:"tenant_id" validate:"required"`
}
```

#### LoginInfoDTO (Response)
```go
type LoginInfoDTO struct {
    Username     string `json:"username"`
    LoginSession string `json:"login_session"`
    TenantID     string `json:"tenant_id"`
}
```

#### TokenBodyResponse (Login Response)
```go
type TokenBodyResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"` // "bearer"
}
```

#### RefreshTokenRequest
```go
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
    TenantID     string `json:"tenant_id" validate:"required"`
}
```

### 2.5 Go JWT Middleware Implementation

```go
package middleware

import (
    "errors"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// UserClaims represents the JWT claims from the Rust backend
type UserClaims struct {
    jwt.RegisteredClaims
    User         string `json:"user"`
    LoginSession string `json:"login_session"`
    TenantID     string `json:"tenant_id"`
}

var jwtSecret []byte

func init() {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        // Fallback: read from file (dev only)
        data, err := os.ReadFile("secret.key")
        if err != nil {
            panic("JWT_SECRET not configured")
        }
        jwtSecret = data
    } else {
        jwtSecret = []byte(secret)
    }
}

// ValidateToken validates a JWT token from the Rust backend
func ValidateToken(tokenString string) (*UserClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
        // Check expiration
        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            return nil, errors.New("token expired")
        }
        return claims, nil
    }

    return nil, errors.New("invalid token")
}

// AuthMiddleware validates JWT tokens in incoming requests
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        // Extract bearer token
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
            http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
            return
        }

        claims, err := ValidateToken(parts[1])
        if err != nil {
            http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
            return
        }

        // Add claims to request context
        ctx := r.Context()
        ctx = context.WithValue(ctx, "user_claims", claims)
        ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2.6 Multi-Tenant Isolation

**CRITICAL**: Tenant context is server-derived, never trust client-supplied tenant IDs.

The Go microservice must:
1. Extract `tenant_id` from validated JWT claims
2. Use tenant ID to scope all database queries
3. Never allow cross-tenant data access

---

## 3. Oracle Database Structure

### 3.1 Connection Configuration

```go
package config

import (
    "database/sql"
    "fmt"
    "os"

    _ "github.com/godror/godror"
)

type OracleConfig struct {
    Host     string
    Port     string
    Service  string
    User     string
    Password string
}

func NewOracleConfig() *OracleConfig {
    return &OracleConfig{
        Host:     getEnvOrDefault("ORACLE_HOST", "localhost"),
        Port:     getEnvOrDefault("ORACLE_PORT", "1521"),
        Service:  getEnvOrDefault("ORACLE_SERVICE", "ORCL"),
        User:     os.Getenv("ORACLE_USER"),
        Password: os.Getenv("ORACLE_PASSWORD"),
    }
}

func (c *OracleConfig) DSN() string {
    return fmt.Sprintf(`user="%s" password="%s" connectString="%s:%s/%s"`,
        c.User, c.Password, c.Host, c.Port, c.Service)
}

func ConnectOracle(cfg *OracleConfig) (*sql.DB, error) {
    db, err := sql.Open("godror", cfg.DSN())
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Oracle: %w", err)
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("Oracle ping failed: %w", err)
    }
    
    return db, nil
}

func getEnvOrDefault(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

### 3.2 Database Schema

#### 3.2.1 Customers Table

```sql
-- Customers: entities that contract services
CREATE TABLE customers (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    -- Identification
    customer_code   VARCHAR2(50) NOT NULL,
    customer_type   VARCHAR2(20) DEFAULT 'INDIVIDUAL' CHECK (customer_type IN ('INDIVIDUAL', 'COMPANY')),
    
    -- Personal/Company Info
    name            VARCHAR2(255) NOT NULL,
    trade_name      VARCHAR2(255),           -- For companies
    tax_id          VARCHAR2(20),            -- CPF/CNPJ
    state_reg       VARCHAR2(30),            -- Inscrição Estadual
    municipal_reg   VARCHAR2(30),            -- Inscrição Municipal
    
    -- Contact
    email           VARCHAR2(255),
    phone           VARCHAR2(20),
    mobile          VARCHAR2(20),
    
    -- Address
    address_street  VARCHAR2(255),
    address_number  VARCHAR2(20),
    address_comp    VARCHAR2(100),           -- Complemento
    address_district VARCHAR2(100),          -- Bairro
    address_city    VARCHAR2(100),
    address_state   VARCHAR2(2),
    address_zip     VARCHAR2(10),
    address_country VARCHAR2(50) DEFAULT 'BR',
    
    -- Status & Metadata
    active          NUMBER(1) DEFAULT 1,
    notes           CLOB,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by      VARCHAR2(100),
    updated_by      VARCHAR2(100),
    
    CONSTRAINT uk_customer_tenant_code UNIQUE (tenant_id, customer_code)
);

CREATE INDEX idx_customers_tenant ON customers(tenant_id);
CREATE INDEX idx_customers_tax_id ON customers(tenant_id, tax_id);
CREATE INDEX idx_customers_name ON customers(tenant_id, UPPER(name));
```

#### 3.2.2 Services Table

```sql
-- Services: catalog of services offered (NOT goods)
CREATE TABLE services (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    -- Identification
    service_code    VARCHAR2(50) NOT NULL,
    name            VARCHAR2(255) NOT NULL,
    description     CLOB,
    
    -- Classification
    category        VARCHAR2(100),
    subcategory     VARCHAR2(100),
    
    -- Pricing
    unit_price      NUMBER(15,2) NOT NULL,
    currency        VARCHAR2(3) DEFAULT 'BRL',
    price_unit      VARCHAR2(20) DEFAULT 'HOUR',  -- HOUR, DAY, MONTH, PROJECT, UNIT
    
    -- Tax/Fiscal
    service_code_fiscal VARCHAR2(20),        -- Código de Serviço Municipal
    iss_rate        NUMBER(5,2) DEFAULT 0,   -- ISS percentage
    irrf_rate       NUMBER(5,2) DEFAULT 0,   -- IRRF percentage
    pis_rate        NUMBER(5,2) DEFAULT 0,
    cofins_rate     NUMBER(5,2) DEFAULT 0,
    csll_rate       NUMBER(5,2) DEFAULT 0,
    
    -- Status
    active          NUMBER(1) DEFAULT 1,
    
    -- Metadata
    notes           CLOB,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by      VARCHAR2(100),
    updated_by      VARCHAR2(100),
    
    CONSTRAINT uk_service_tenant_code UNIQUE (tenant_id, service_code)
);

CREATE INDEX idx_services_tenant ON services(tenant_id);
CREATE INDEX idx_services_category ON services(tenant_id, category);
```

#### 3.2.3 Contracts Table

```sql
-- Contracts: binding agreement between customer and services
CREATE TABLE contracts (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    -- Identification
    contract_number VARCHAR2(50) NOT NULL,
    contract_type   VARCHAR2(30) DEFAULT 'SERVICE' CHECK (contract_type IN ('SERVICE', 'RECURRING', 'PROJECT')),
    
    -- Parties
    customer_id     NUMBER NOT NULL REFERENCES customers(id),
    
    -- Terms
    start_date      DATE NOT NULL,
    end_date        DATE,
    duration_months NUMBER(4),
    auto_renew      NUMBER(1) DEFAULT 0,
    
    -- Financial
    total_value     NUMBER(15,2),
    payment_terms   VARCHAR2(100),           -- e.g., "NET30", "IMMEDIATE"
    billing_cycle   VARCHAR2(20) DEFAULT 'MONTHLY',  -- MONTHLY, QUARTERLY, YEARLY, ONE_TIME
    
    -- Status
    status          VARCHAR2(20) DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PENDING', 'ACTIVE', 'SUSPENDED', 'CANCELLED', 'COMPLETED')),
    signed_at       TIMESTAMP,
    signed_by       VARCHAR2(100),
    
    -- Document
    document_path   VARCHAR2(500),           -- Path to generated PDF
    document_hash   VARCHAR2(128),           -- SHA-512 for integrity
    
    -- Metadata
    notes           CLOB,
    terms_conditions CLOB,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by      VARCHAR2(100),
    updated_by      VARCHAR2(100),
    
    CONSTRAINT uk_contract_tenant_number UNIQUE (tenant_id, contract_number)
);

CREATE INDEX idx_contracts_tenant ON contracts(tenant_id);
CREATE INDEX idx_contracts_customer ON contracts(tenant_id, customer_id);
CREATE INDEX idx_contracts_status ON contracts(tenant_id, status);
CREATE INDEX idx_contracts_dates ON contracts(tenant_id, start_date, end_date);
```

#### 3.2.4 Contract Items Table

```sql
-- Contract Items: services included in a contract
CREATE TABLE contract_items (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    contract_id     NUMBER NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    service_id      NUMBER NOT NULL REFERENCES services(id),
    
    -- Pricing (can override service defaults)
    quantity        NUMBER(10,2) DEFAULT 1,
    unit_price      NUMBER(15,2) NOT NULL,
    discount_pct    NUMBER(5,2) DEFAULT 0,
    line_total      NUMBER(15,2) GENERATED ALWAYS AS (quantity * unit_price * (1 - discount_pct/100)) VIRTUAL,
    
    -- Scheduling
    start_date      DATE,
    end_date        DATE,
    delivery_date   DATE,
    
    -- Description override
    description     CLOB,
    
    -- Status
    status          VARCHAR2(20) DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    completed_at    TIMESTAMP,
    
    -- Metadata
    notes           CLOB,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_contract_items_contract ON contract_items(tenant_id, contract_id);
CREATE INDEX idx_contract_items_service ON contract_items(tenant_id, service_id);
```

#### 3.2.5 Contract History (Audit)

```sql
-- Contract History: audit trail for contract changes
CREATE TABLE contract_history (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    contract_id     NUMBER NOT NULL REFERENCES contracts(id),
    action          VARCHAR2(20) NOT NULL,   -- CREATE, UPDATE, STATUS_CHANGE, SIGN, PRINT
    field_changed   VARCHAR2(100),
    old_value       CLOB,
    new_value       CLOB,
    
    -- Actor
    performed_by    VARCHAR2(100) NOT NULL,
    performed_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address      VARCHAR2(50),
    user_agent      VARCHAR2(500)
);

CREATE INDEX idx_contract_history_contract ON contract_history(tenant_id, contract_id);
CREATE INDEX idx_contract_history_date ON contract_history(tenant_id, performed_at);
```

#### 3.2.6 Contract Print Jobs

```sql
-- Contract Print Jobs: track document generation
CREATE TABLE contract_print_jobs (
    id              NUMBER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    tenant_id       VARCHAR2(100) NOT NULL,
    
    contract_id     NUMBER NOT NULL REFERENCES contracts(id),
    
    -- Job Status
    status          VARCHAR2(20) DEFAULT 'QUEUED' CHECK (status IN ('QUEUED', 'PROCESSING', 'COMPLETED', 'FAILED')),
    format          VARCHAR2(10) DEFAULT 'PDF' CHECK (format IN ('PDF', 'DOCX', 'HTML')),
    
    -- Output
    output_path     VARCHAR2(500),
    file_size       NUMBER,
    page_count      NUMBER,
    
    -- Timing
    queued_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    
    -- Error handling
    retry_count     NUMBER DEFAULT 0,
    error_message   CLOB,
    
    -- Metadata
    requested_by    VARCHAR2(100) NOT NULL
);

CREATE INDEX idx_print_jobs_contract ON contract_print_jobs(tenant_id, contract_id);
CREATE INDEX idx_print_jobs_status ON contract_print_jobs(tenant_id, status);
```

---

## 4. Go Domain Models

### 4.1 Customer Model

```go
package models

import (
    "database/sql"
    "time"
)

type CustomerType string

const (
    CustomerTypeIndividual CustomerType = "INDIVIDUAL"
    CustomerTypeCompany    CustomerType = "COMPANY"
)

type Customer struct {
    ID              int64          `json:"id" db:"id"`
    TenantID        string         `json:"-" db:"tenant_id"`
    
    // Identification
    CustomerCode    string         `json:"customer_code" db:"customer_code"`
    CustomerType    CustomerType   `json:"customer_type" db:"customer_type"`
    
    // Personal/Company Info
    Name            string         `json:"name" db:"name"`
    TradeName       sql.NullString `json:"trade_name,omitempty" db:"trade_name"`
    TaxID           sql.NullString `json:"tax_id,omitempty" db:"tax_id"`
    StateReg        sql.NullString `json:"state_reg,omitempty" db:"state_reg"`
    MunicipalReg    sql.NullString `json:"municipal_reg,omitempty" db:"municipal_reg"`
    
    // Contact
    Email           sql.NullString `json:"email,omitempty" db:"email"`
    Phone           sql.NullString `json:"phone,omitempty" db:"phone"`
    Mobile          sql.NullString `json:"mobile,omitempty" db:"mobile"`
    
    // Address
    Address         Address        `json:"address"`
    
    // Status
    Active          bool           `json:"active" db:"active"`
    Notes           sql.NullString `json:"notes,omitempty" db:"notes"`
    
    // Metadata
    CreatedAt       time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
    CreatedBy       sql.NullString `json:"created_by,omitempty" db:"created_by"`
    UpdatedBy       sql.NullString `json:"updated_by,omitempty" db:"updated_by"`
}

type Address struct {
    Street   string `json:"street" db:"address_street"`
    Number   string `json:"number" db:"address_number"`
    Comp     string `json:"complement,omitempty" db:"address_comp"`
    District string `json:"district" db:"address_district"`
    City     string `json:"city" db:"address_city"`
    State    string `json:"state" db:"address_state"`
    Zip      string `json:"zip" db:"address_zip"`
    Country  string `json:"country" db:"address_country"`
}

// CreateCustomerDTO for customer creation
type CreateCustomerDTO struct {
    CustomerCode string       `json:"customer_code" validate:"required,max=50"`
    CustomerType CustomerType `json:"customer_type" validate:"required,oneof=INDIVIDUAL COMPANY"`
    Name         string       `json:"name" validate:"required,max=255"`
    TradeName    *string      `json:"trade_name,omitempty"`
    TaxID        *string      `json:"tax_id,omitempty"`
    Email        *string      `json:"email,omitempty" validate:"omitempty,email"`
    Phone        *string      `json:"phone,omitempty"`
    Address      *Address     `json:"address,omitempty"`
}

// UpdateCustomerDTO for customer updates
type UpdateCustomerDTO struct {
    Name      *string  `json:"name,omitempty" validate:"omitempty,max=255"`
    TradeName *string  `json:"trade_name,omitempty"`
    Email     *string  `json:"email,omitempty" validate:"omitempty,email"`
    Phone     *string  `json:"phone,omitempty"`
    Mobile    *string  `json:"mobile,omitempty"`
    Address   *Address `json:"address,omitempty"`
    Active    *bool    `json:"active,omitempty"`
    Notes     *string  `json:"notes,omitempty"`
}
```

### 4.2 Service Model

```go
package models

type PriceUnit string

const (
    PriceUnitHour    PriceUnit = "HOUR"
    PriceUnitDay     PriceUnit = "DAY"
    PriceUnitMonth   PriceUnit = "MONTH"
    PriceUnitProject PriceUnit = "PROJECT"
    PriceUnitUnit    PriceUnit = "UNIT"
)

type Service struct {
    ID                int64          `json:"id" db:"id"`
    TenantID          string         `json:"-" db:"tenant_id"`
    
    ServiceCode       string         `json:"service_code" db:"service_code"`
    Name              string         `json:"name" db:"name"`
    Description       sql.NullString `json:"description,omitempty" db:"description"`
    
    Category          sql.NullString `json:"category,omitempty" db:"category"`
    Subcategory       sql.NullString `json:"subcategory,omitempty" db:"subcategory"`
    
    // Pricing
    UnitPrice         float64        `json:"unit_price" db:"unit_price"`
    Currency          string         `json:"currency" db:"currency"`
    PriceUnit         PriceUnit      `json:"price_unit" db:"price_unit"`
    
    // Tax rates
    ServiceCodeFiscal sql.NullString `json:"service_code_fiscal,omitempty" db:"service_code_fiscal"`
    ISSRate           float64        `json:"iss_rate" db:"iss_rate"`
    IRRFRate          float64        `json:"irrf_rate" db:"irrf_rate"`
    PISRate           float64        `json:"pis_rate" db:"pis_rate"`
    COFINSRate        float64        `json:"cofins_rate" db:"cofins_rate"`
    CSLLRate          float64        `json:"csll_rate" db:"csll_rate"`
    
    Active            bool           `json:"active" db:"active"`
    Notes             sql.NullString `json:"notes,omitempty" db:"notes"`
    
    CreatedAt         time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

type CreateServiceDTO struct {
    ServiceCode       string    `json:"service_code" validate:"required,max=50"`
    Name              string    `json:"name" validate:"required,max=255"`
    Description       *string   `json:"description,omitempty"`
    Category          *string   `json:"category,omitempty"`
    UnitPrice         float64   `json:"unit_price" validate:"required,gt=0"`
    PriceUnit         PriceUnit `json:"price_unit" validate:"required"`
    ServiceCodeFiscal *string   `json:"service_code_fiscal,omitempty"`
    ISSRate           *float64  `json:"iss_rate,omitempty"`
}
```

### 4.3 Contract Model

```go
package models

type ContractType string
type ContractStatus string
type BillingCycle string

const (
    ContractTypeService   ContractType = "SERVICE"
    ContractTypeRecurring ContractType = "RECURRING"
    ContractTypeProject   ContractType = "PROJECT"
    
    ContractStatusDraft     ContractStatus = "DRAFT"
    ContractStatusPending   ContractStatus = "PENDING"
    ContractStatusActive    ContractStatus = "ACTIVE"
    ContractStatusSuspended ContractStatus = "SUSPENDED"
    ContractStatusCancelled ContractStatus = "CANCELLED"
    ContractStatusCompleted ContractStatus = "COMPLETED"
    
    BillingCycleMonthly   BillingCycle = "MONTHLY"
    BillingCycleQuarterly BillingCycle = "QUARTERLY"
    BillingCycleYearly    BillingCycle = "YEARLY"
    BillingCycleOneTime   BillingCycle = "ONE_TIME"
)

type Contract struct {
    ID              int64            `json:"id" db:"id"`
    TenantID        string           `json:"-" db:"tenant_id"`
    
    ContractNumber  string           `json:"contract_number" db:"contract_number"`
    ContractType    ContractType     `json:"contract_type" db:"contract_type"`
    
    CustomerID      int64            `json:"customer_id" db:"customer_id"`
    Customer        *Customer        `json:"customer,omitempty"` // Populated on detail queries
    
    StartDate       time.Time        `json:"start_date" db:"start_date"`
    EndDate         *time.Time       `json:"end_date,omitempty" db:"end_date"`
    DurationMonths  *int             `json:"duration_months,omitempty" db:"duration_months"`
    AutoRenew       bool             `json:"auto_renew" db:"auto_renew"`
    
    TotalValue      *float64         `json:"total_value,omitempty" db:"total_value"`
    PaymentTerms    sql.NullString   `json:"payment_terms,omitempty" db:"payment_terms"`
    BillingCycle    BillingCycle     `json:"billing_cycle" db:"billing_cycle"`
    
    Status          ContractStatus   `json:"status" db:"status"`
    SignedAt        *time.Time       `json:"signed_at,omitempty" db:"signed_at"`
    SignedBy        sql.NullString   `json:"signed_by,omitempty" db:"signed_by"`
    
    DocumentPath    sql.NullString   `json:"document_path,omitempty" db:"document_path"`
    DocumentHash    sql.NullString   `json:"-" db:"document_hash"`
    
    Notes           sql.NullString   `json:"notes,omitempty" db:"notes"`
    TermsConditions sql.NullString   `json:"terms_conditions,omitempty" db:"terms_conditions"`
    
    Items           []ContractItem   `json:"items,omitempty"` // Populated on detail queries
    
    CreatedAt       time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

type ContractItem struct {
    ID          int64          `json:"id" db:"id"`
    TenantID    string         `json:"-" db:"tenant_id"`
    ContractID  int64          `json:"contract_id" db:"contract_id"`
    ServiceID   int64          `json:"service_id" db:"service_id"`
    Service     *Service       `json:"service,omitempty"`
    
    Quantity    float64        `json:"quantity" db:"quantity"`
    UnitPrice   float64        `json:"unit_price" db:"unit_price"`
    DiscountPct float64        `json:"discount_pct" db:"discount_pct"`
    LineTotal   float64        `json:"line_total" db:"line_total"` // Computed column
    
    StartDate   *time.Time     `json:"start_date,omitempty" db:"start_date"`
    EndDate     *time.Time     `json:"end_date,omitempty" db:"end_date"`
    Description sql.NullString `json:"description,omitempty" db:"description"`
    Status      string         `json:"status" db:"status"`
    
    CreatedAt   time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// CreateContractDTO for contract creation
type CreateContractDTO struct {
    ContractNumber  string           `json:"contract_number" validate:"required,max=50"`
    ContractType    ContractType     `json:"contract_type" validate:"required"`
    CustomerID      int64            `json:"customer_id" validate:"required"`
    StartDate       time.Time        `json:"start_date" validate:"required"`
    EndDate         *time.Time       `json:"end_date,omitempty"`
    DurationMonths  *int             `json:"duration_months,omitempty"`
    AutoRenew       bool             `json:"auto_renew"`
    PaymentTerms    *string          `json:"payment_terms,omitempty"`
    BillingCycle    BillingCycle     `json:"billing_cycle" validate:"required"`
    Notes           *string          `json:"notes,omitempty"`
    TermsConditions *string          `json:"terms_conditions,omitempty"`
    Items           []CreateItemDTO  `json:"items" validate:"required,min=1,dive"`
}

type CreateItemDTO struct {
    ServiceID   int64    `json:"service_id" validate:"required"`
    Quantity    float64  `json:"quantity" validate:"required,gt=0"`
    UnitPrice   *float64 `json:"unit_price,omitempty"` // Override service price
    DiscountPct *float64 `json:"discount_pct,omitempty"`
    Description *string  `json:"description,omitempty"`
}
```

---

## 5. API Endpoints

### 5.1 Customers API

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/customers` | List customers (paginated) | ✓ |
| GET | `/api/v1/customers/{id}` | Get customer by ID | ✓ |
| POST | `/api/v1/customers` | Create customer | ✓ |
| PUT | `/api/v1/customers/{id}` | Update customer | ✓ |
| DELETE | `/api/v1/customers/{id}` | Soft delete customer | ✓ |
| GET | `/api/v1/customers/search` | Search customers | ✓ |

### 5.2 Services API

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/services` | List services (paginated) | ✓ |
| GET | `/api/v1/services/{id}` | Get service by ID | ✓ |
| POST | `/api/v1/services` | Create service | ✓ |
| PUT | `/api/v1/services/{id}` | Update service | ✓ |
| DELETE | `/api/v1/services/{id}` | Soft delete service | ✓ |
| GET | `/api/v1/services/categories` | List service categories | ✓ |

### 5.3 Contracts API

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/contracts` | List contracts (paginated) | ✓ |
| GET | `/api/v1/contracts/{id}` | Get contract with items | ✓ |
| POST | `/api/v1/contracts` | Create contract with items | ✓ |
| PUT | `/api/v1/contracts/{id}` | Update contract | ✓ |
| PATCH | `/api/v1/contracts/{id}/status` | Change contract status | ✓ |
| DELETE | `/api/v1/contracts/{id}` | Cancel contract | ✓ |
| POST | `/api/v1/contracts/{id}/sign` | Sign contract | ✓ |
| GET | `/api/v1/contracts/{id}/history` | Get contract audit history | ✓ |

### 5.4 Contract Items API

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/v1/contracts/{id}/items` | Add item to contract | ✓ |
| PUT | `/api/v1/contracts/{id}/items/{itemId}` | Update contract item | ✓ |
| DELETE | `/api/v1/contracts/{id}/items/{itemId}` | Remove item from contract | ✓ |

### 5.5 Contract Printing API

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/v1/contracts/{id}/print` | Queue contract for printing | ✓ |
| GET | `/api/v1/contracts/{id}/print/status` | Get print job status | ✓ |
| GET | `/api/v1/contracts/{id}/document` | Download generated document | ✓ |
| GET | `/api/v1/print-jobs` | List print jobs | ✓ |

---

## 6. Project Structure

```
contract-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go              # Configuration loading
│   │   └── oracle.go              # Oracle connection
│   ├── middleware/
│   │   ├── auth.go                # JWT validation middleware
│   │   ├── tenant.go              # Tenant context middleware
│   │   └── logging.go             # Request logging
│   ├── handlers/
│   │   ├── customer.go            # Customer handlers
│   │   ├── service.go             # Service handlers
│   │   ├── contract.go            # Contract handlers
│   │   └── print.go               # Print job handlers
│   ├── models/
│   │   ├── customer.go
│   │   ├── service.go
│   │   ├── contract.go
│   │   └── print_job.go
│   ├── repository/
│   │   ├── customer_repo.go
│   │   ├── service_repo.go
│   │   ├── contract_repo.go
│   │   └── print_job_repo.go
│   ├── services/
│   │   ├── customer_service.go
│   │   ├── contract_service.go
│   │   └── print_service.go       # PDF generation
│   └── templates/
│       └── contract_template.html # Contract PDF template
├── pkg/
│   ├── auth/
│   │   └── jwt.go                 # JWT token utilities
│   └── pdf/
│       └── generator.go           # PDF generation utilities
├── migrations/
│   └── oracle/
│       ├── 001_create_customers.sql
│       ├── 002_create_services.sql
│       ├── 003_create_contracts.sql
│       └── ...
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

## 7. Environment Variables

```env
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Oracle Database
ORACLE_HOST=localhost
ORACLE_PORT=1521
ORACLE_SERVICE=ORCL
ORACLE_USER=contract_svc
ORACLE_PASSWORD=<secure-password>

# JWT (must match Rust backend)
JWT_SECRET=<shared-secret-with-rust-backend>

# Rust Auth Backend
AUTH_SERVICE_URL=http://localhost:8000

# PDF Generation
PDF_STORAGE_PATH=/app/documents
PDF_TEMPLATE_PATH=/app/templates

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## 8. Dependencies

```go
// go.mod
module github.com/yourorg/contract-service

go 1.22

require (
    github.com/godror/godror v0.42.0          // Oracle driver
    github.com/golang-jwt/jwt/v5 v5.2.0       // JWT handling
    github.com/go-chi/chi/v5 v5.0.12          // HTTP router
    github.com/go-playground/validator/v10 v10.18.0  // Validation
    github.com/jung-kurt/gofpdf v1.16.2       // PDF generation
    github.com/rs/zerolog v1.32.0             // Structured logging
    github.com/joho/godotenv v1.5.1           // Env loading
)
```

---

## 9. Security Considerations

1. **JWT Validation**: Always validate JWT signature with shared secret
2. **Tenant Isolation**: Extract tenant from JWT, scope all queries
3. **SQL Injection**: Use parameterized queries only
4. **Rate Limiting**: Implement per-tenant rate limits
5. **Audit Logging**: Log all contract modifications
6. **Document Integrity**: Store SHA-512 hash of generated documents

---

## 10. Next Steps

1. [ ] Set up Go project structure
2. [ ] Configure Oracle database connection
3. [ ] Implement JWT middleware integration
4. [ ] Create database migrations
5. [ ] Implement CRUD repositories
6. [ ] Build API handlers
7. [ ] Implement PDF generation
8. [ ] Add integration tests
9. [ ] Docker containerization
10. [ ] CI/CD pipeline setup
