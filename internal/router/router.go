package router

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/zlovtnik/gprint/internal/handlers"
	"github.com/zlovtnik/gprint/internal/middleware"
)

// Handlers groups all HTTP handlers for cleaner dependency injection
type Handlers struct {
	Customer           *handlers.CustomerHandler
	Service            *handlers.ServiceHandler
	Contract           *handlers.ContractHandler
	ContractGeneration *handlers.ContractGenerationHandler
	Print              *handlers.PrintHandler
	Health             *handlers.HealthHandler
	Auth               *handlers.AuthHandler
}

// Router holds all route handlers
type Router struct {
	mux       *http.ServeMux
	jwtSecret string
	logger    *slog.Logger
	handlers  Handlers
}

// NewRouter creates a new Router with validated handlers.
// Returns an error if any required handler is nil.
func NewRouter(
	jwtSecret string,
	logger *slog.Logger,
	h Handlers,
) (*Router, error) {
	// Validate all required handlers are set
	if h.Customer == nil {
		return nil, errors.New("customer handler is required")
	}
	if h.Service == nil {
		return nil, errors.New("service handler is required")
	}
	if h.Contract == nil {
		return nil, errors.New("contract handler is required")
	}
	if h.ContractGeneration == nil {
		return nil, errors.New("contract generation handler is required")
	}
	if h.Print == nil {
		return nil, errors.New("print handler is required")
	}
	if h.Health == nil {
		return nil, errors.New("health handler is required")
	}
	if h.Auth == nil {
		return nil, errors.New("auth handler is required")
	}

	return &Router{
		mux:       http.NewServeMux(),
		jwtSecret: jwtSecret,
		logger:    logger,
		handlers:  h,
	}, nil
}

// Setup configures all routes
func (r *Router) Setup() http.Handler {
	// Health endpoints (no auth required)
	r.mux.HandleFunc("GET /health", r.handlers.Health.Health)
	r.mux.HandleFunc("GET /ready", r.handlers.Health.Ready)

	// Auth endpoints:
	// - POST /api/v1/auth/login: public (no auth required)
	// - POST /api/v1/auth/refresh: public (no auth required)
	// - POST /api/v1/auth/logout: public (no auth required)
	// - GET /api/v1/auth/me: protected (requires valid JWT)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.handlers.Auth.Login)
	r.mux.HandleFunc("POST /api/v1/auth/refresh", r.handlers.Auth.Refresh)
	r.mux.HandleFunc("POST /api/v1/auth/logout", r.handlers.Auth.Logout)
	r.mux.HandleFunc("GET /api/v1/auth/me", r.handlers.Auth.Me)

	// Customer endpoints
	r.mux.HandleFunc("GET /api/v1/customers", r.handlers.Customer.List)
	r.mux.HandleFunc("GET /api/v1/customers/{id}", r.handlers.Customer.Get)
	r.mux.HandleFunc("POST /api/v1/customers", r.handlers.Customer.Create)
	r.mux.HandleFunc("PUT /api/v1/customers/{id}", r.handlers.Customer.Update)
	r.mux.HandleFunc("DELETE /api/v1/customers/{id}", r.handlers.Customer.Delete)

	// Service endpoints
	r.mux.HandleFunc("GET /api/v1/services", r.handlers.Service.List)
	r.mux.HandleFunc("GET /api/v1/services/categories", r.handlers.Service.GetCategories)
	r.mux.HandleFunc("GET /api/v1/services/{id}", r.handlers.Service.Get)
	r.mux.HandleFunc("POST /api/v1/services", r.handlers.Service.Create)
	r.mux.HandleFunc("PUT /api/v1/services/{id}", r.handlers.Service.Update)
	r.mux.HandleFunc("DELETE /api/v1/services/{id}", r.handlers.Service.Delete)

	// Contract endpoints
	r.mux.HandleFunc("GET /api/v1/contracts", r.handlers.Contract.List)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}", r.handlers.Contract.Get)
	r.mux.HandleFunc("POST /api/v1/contracts", r.handlers.Contract.Create)
	r.mux.HandleFunc("PUT /api/v1/contracts/{id}", r.handlers.Contract.Update)
	r.mux.HandleFunc("PATCH /api/v1/contracts/{id}/status", r.handlers.Contract.UpdateStatus)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/sign", r.handlers.Contract.Sign)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/history", r.handlers.Contract.GetHistory)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/items", r.handlers.Contract.AddItem)
	r.mux.HandleFunc("DELETE /api/v1/contracts/{id}/items/{itemId}", r.handlers.Contract.DeleteItem)

	// Print job endpoints
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/print", r.handlers.Print.CreateJob)
	r.mux.HandleFunc("GET /api/v1/print-jobs", r.handlers.Print.List)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/print-jobs", r.handlers.Print.GetJobsByContract)
	r.mux.HandleFunc("GET /api/v1/print-jobs/{id}", r.handlers.Print.GetJob)
	r.mux.HandleFunc("GET /api/v1/print-jobs/{id}/download", r.handlers.Print.Download)

	// Contract generation endpoints (all processing happens in PL/SQL for security)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/generate", r.handlers.ContractGeneration.Generate)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/generated", r.handlers.ContractGeneration.ListGenerated)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/generated/latest", r.handlers.ContractGeneration.GetLatest)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/generated/{gen_id}", r.handlers.ContractGeneration.GetContent)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/generated/{gen_id}/log/download", r.handlers.ContractGeneration.LogDownload)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/generated/{gen_id}/log/print", r.handlers.ContractGeneration.LogPrint)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/generated/{gen_id}/verify", r.handlers.ContractGeneration.VerifyIntegrity)
	r.mux.HandleFunc("GET /api/v1/contracts/generation/stats", r.handlers.ContractGeneration.GetStats)
	r.mux.HandleFunc("GET /api/v1/contracts/templates", r.handlers.ContractGeneration.ListTemplates)

	// Apply middleware stack
	var handler http.Handler = r.mux

	// Auth middleware (skip for health endpoints and OPTIONS)
	handler = r.authMiddleware(handler)

	// CORS - applied after auth so it can set headers for preflight before auth rejects
	handler = middleware.CORSMiddleware(middleware.DefaultCORSConfig())(handler)

	// Logging
	handler = middleware.LoggingMiddleware(r.logger)(handler)

	// Recovery
	handler = middleware.RecoveryMiddleware(r.logger)(handler)

	return handler
}

// unauthenticatedPaths is an explicit allowlist of paths that bypass auth middleware
var unauthenticatedPaths = map[string]bool{
	"/health":              true,
	"/ready":               true,
	"/api/v1/auth/login":   true,
	"/api/v1/auth/refresh": true,
	"/api/v1/auth/logout":  true,
	// Note: /api/v1/auth/me is NOT in this list - it requires authentication
}

// authMiddleware wraps the auth middleware but skips unauthenticated paths and OPTIONS requests
func (r *Router) authMiddleware(next http.Handler) http.Handler {
	authHandler := middleware.AuthMiddleware(r.jwtSecret)(next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Skip auth for explicitly allowed unauthenticated paths
		if unauthenticatedPaths[req.URL.Path] {
			next.ServeHTTP(w, req)
			return
		}

		// Skip auth for CORS preflight requests
		if req.Method == http.MethodOptions {
			next.ServeHTTP(w, req)
			return
		}

		// Apply auth middleware for all other paths
		authHandler.ServeHTTP(w, req)
	})
}
