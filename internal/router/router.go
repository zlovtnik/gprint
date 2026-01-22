package router

import (
	"log/slog"
	"net/http"

	"github.com/zlovtnik/gprint/internal/handlers"
	"github.com/zlovtnik/gprint/internal/middleware"
)

// Router holds all route handlers
type Router struct {
	mux             *http.ServeMux
	jwtSecret       string
	logger          *slog.Logger
	customerHandler *handlers.CustomerHandler
	serviceHandler  *handlers.ServiceHandler
	contractHandler *handlers.ContractHandler
	printHandler    *handlers.PrintHandler
	healthHandler   *handlers.HealthHandler
}

// NewRouter creates a new Router
func NewRouter(
	jwtSecret string,
	logger *slog.Logger,
	customerHandler *handlers.CustomerHandler,
	serviceHandler *handlers.ServiceHandler,
	contractHandler *handlers.ContractHandler,
	printHandler *handlers.PrintHandler,
	healthHandler *handlers.HealthHandler,
) *Router {
	return &Router{
		mux:             http.NewServeMux(),
		jwtSecret:       jwtSecret,
		logger:          logger,
		customerHandler: customerHandler,
		serviceHandler:  serviceHandler,
		contractHandler: contractHandler,
		printHandler:    printHandler,
		healthHandler:   healthHandler,
	}
}

// Setup configures all routes
func (r *Router) Setup() http.Handler {
	// Health endpoints (no auth required)
	r.mux.HandleFunc("GET /health", r.healthHandler.Health)
	r.mux.HandleFunc("GET /ready", r.healthHandler.Ready)

	// Customer endpoints
	r.mux.HandleFunc("GET /api/v1/customers", r.customerHandler.List)
	r.mux.HandleFunc("GET /api/v1/customers/{id}", r.customerHandler.Get)
	r.mux.HandleFunc("POST /api/v1/customers", r.customerHandler.Create)
	r.mux.HandleFunc("PUT /api/v1/customers/{id}", r.customerHandler.Update)
	r.mux.HandleFunc("DELETE /api/v1/customers/{id}", r.customerHandler.Delete)

	// Service endpoints
	r.mux.HandleFunc("GET /api/v1/services", r.serviceHandler.List)
	r.mux.HandleFunc("GET /api/v1/services/categories", r.serviceHandler.GetCategories)
	r.mux.HandleFunc("GET /api/v1/services/{id}", r.serviceHandler.Get)
	r.mux.HandleFunc("POST /api/v1/services", r.serviceHandler.Create)
	r.mux.HandleFunc("PUT /api/v1/services/{id}", r.serviceHandler.Update)
	r.mux.HandleFunc("DELETE /api/v1/services/{id}", r.serviceHandler.Delete)

	// Contract endpoints
	r.mux.HandleFunc("GET /api/v1/contracts", r.contractHandler.List)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}", r.contractHandler.Get)
	r.mux.HandleFunc("POST /api/v1/contracts", r.contractHandler.Create)
	r.mux.HandleFunc("PUT /api/v1/contracts/{id}", r.contractHandler.Update)
	r.mux.HandleFunc("PATCH /api/v1/contracts/{id}/status", r.contractHandler.UpdateStatus)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/sign", r.contractHandler.Sign)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/history", r.contractHandler.GetHistory)
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/items", r.contractHandler.AddItem)
	r.mux.HandleFunc("DELETE /api/v1/contracts/{id}/items/{itemId}", r.contractHandler.DeleteItem)

	// Print job endpoints
	r.mux.HandleFunc("POST /api/v1/contracts/{id}/print", r.printHandler.CreateJob)
	r.mux.HandleFunc("GET /api/v1/contracts/{id}/print-jobs", r.printHandler.GetJobsByContract)
	r.mux.HandleFunc("GET /api/v1/print-jobs/{id}", r.printHandler.GetJob)
	r.mux.HandleFunc("GET /api/v1/print-jobs/{id}/download", r.printHandler.Download)

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

// authMiddleware wraps the auth middleware but skips health endpoints and OPTIONS requests
func (r *Router) authMiddleware(next http.Handler) http.Handler {
	authHandler := middleware.AuthMiddleware(r.jwtSecret)(next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Skip auth for health endpoints
		if req.URL.Path == "/health" || req.URL.Path == "/ready" {
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
