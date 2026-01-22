package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

// HealthHandler handles health check HTTP requests
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Ready handles GET /ready
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Create a context with a short timeout to prevent blocking indefinitely
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database connection with context
	if err := h.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
