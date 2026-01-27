package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// Error codes for CLM handlers
const (
	ErrCodeInvalidJSON  = "INVALID_JSON"
	ErrCodeInvalidID    = "INVALID_ID"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeInternal     = "INTERNAL_ERROR"
)

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	if data == nil {
		w.WriteHeader(status)
		return
	}

	// Encode first to check for errors
	jsonData, err := json.Marshal(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(jsonData)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse(code, message, nil))
}

// parseIDFromPath extracts an ID from the URL path
func parseIDFromPath(r *http.Request, param string) (string, error) {
	// First try PathValue for Go 1.22+ routing
	if id := r.PathValue(param); id != "" {
		return id, nil
	}

	// Fallback: scan path parts
	path := r.URL.Path
	parts := strings.Split(path, "/")

	for i, part := range parts {
		if i > 0 && (parts[i-1] == param || parts[i-1] == param+"s") {
			if part != "" {
				return part, nil
			}
		}
	}

	return "", fmt.Errorf("ID not found for param %s", param)
}
