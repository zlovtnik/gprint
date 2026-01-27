package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// parseIDFromPath extracts an int64 ID from the request path.
// The name parameter should match the path variable name (e.g., "id", "itemId").
func parseIDFromPath(r *http.Request, name string) (int64, error) {
	idStr := r.PathValue(name)
	return strconv.ParseInt(idStr, 10, 64)
}

// parsePagination extracts pagination parameters from query string
func parsePagination(r *http.Request) models.PaginationParams {
	page := 1
	pageSize := 20

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	return models.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// parseSearchParams extracts search/filter parameters from query string
func parseSearchParams(r *http.Request) models.SearchParams {
	params := models.SearchParams{
		Query:  r.URL.Query().Get("q"),
		Field:  r.URL.Query().Get("field"),
		SortBy: r.URL.Query().Get("sort_by"),
	}

	// Validate and normalize sort_dir to only accept "asc" or "desc"
	sortDir := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_dir")))
	if sortDir == "asc" || sortDir == "desc" {
		params.SortDir = sortDir
	} else {
		params.SortDir = "" // Default to empty for invalid values
	}

	if active := r.URL.Query().Get("active"); active != "" {
		b := strings.ToLower(active) == "true" || active == "1"
		params.Active = &b
	}

	return params
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Headers already sent, log the error
		log.Printf("failed to encode JSON response: %v", err)
	}
}

// writeError writes an error response in the standard format
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse(code, message, nil))
}
