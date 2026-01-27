package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/service"
)

// PrintHandler handles print job HTTP requests
type PrintHandler struct {
	svc *service.PrintService
}

// NewPrintHandler creates a new PrintHandler
func NewPrintHandler(svc *service.PrintService) *PrintHandler {
	return &PrintHandler{svc: svc}
}

// CreateJob handles POST /api/v1/contracts/{id}/print
func (h *PrintHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid contract ID")
		return
	}

	var req struct {
		Format models.PrintFormat `json:"format"`
	}

	// Read the entire body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read request body")
		return
	}

	// Check if body is empty (after trimming whitespace)
	trimmedBody := bytes.TrimSpace(body)
	if len(trimmedBody) == 0 {
		// Empty body, default to PDF
		req.Format = models.PrintFormatPDF
	} else {
		// Try to unmarshal
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON in request body")
			return
		}
	}

	if req.Format == "" {
		req.Format = models.PrintFormatPDF
	}

	job, err := h.svc.CreateJob(r.Context(), tenantID, contractID, req.Format, user)
	if err != nil {
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		log.Printf("failed to create print job: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.SuccessResponse(job.ToResponse()))
}

// List handles GET /api/v1/print-jobs
func (h *PrintHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	// Parse pagination parameters
	params := parsePagination(r)

	jobs, total, err := h.svc.List(r.Context(), tenantID, params.Page, params.PageSize)
	if err != nil {
		log.Printf("failed to list print jobs: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.PrintJobResponse, len(jobs))
	for i, j := range jobs {
		responses[i] = j.ToResponse()
	}

	result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, int(total))
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// GetJob handles GET /api/v1/print-jobs/{id}
func (h *PrintHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidPrintJobID)
		return
	}

	job, err := h.svc.GetJob(r.Context(), tenantID, id)
	if err != nil {
		log.Printf("failed to retrieve print job (id=%d, tenant=%s): %v", id, tenantID, err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgFailedToRetrieveJob)
		return
	}
	if job == nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgPrintJobNotFound)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(job.ToResponse()))
}

// GetJobsByContract handles GET /api/v1/contracts/{id}/print-jobs
func (h *PrintHandler) GetJobsByContract(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	jobs, err := h.svc.GetJobsByContract(r.Context(), tenantID, contractID)
	if err != nil {
		log.Printf("failed to get print jobs for contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.PrintJobResponse, len(jobs))
	for i, j := range jobs {
		responses[i] = j.ToResponse()
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}

// Download handles GET /api/v1/print-jobs/{id}/download
func (h *PrintHandler) Download(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidPrintJobID)
		return
	}

	filePath, err := h.svc.DownloadJob(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrPrintJobNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgPrintJobNotFound)
			return
		}
		if errors.Is(err, service.ErrJobNotCompleted) {
			writeError(w, http.StatusConflict, ErrCodeNotReady, MsgJobNotCompleted)
			return
		}
		if errors.Is(err, service.ErrOutputFileNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeFileNotFound, MsgFileNotFound)
			return
		}
		log.Printf("failed to download print job: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, ErrCodeFileNotFound, MsgFileNotFound)
		return
	}

	// Determine content type
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".html":
		contentType = "text/html"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	// Sanitize filename for Content-Disposition header
	safeName := filepath.Base(filePath)
	// Remove control characters and replace quotes
	safeName = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r < 32 {
			return -1
		}
		if r == '"' {
			return '\''
		}
		return r
	}, safeName)
	// Build Content-Disposition with both filename and filename* (RFC5987)
	disposition := mime.FormatMediaType("attachment", map[string]string{
		"filename": safeName,
	})
	// Add filename* for UTF-8 encoding support
	disposition += "; filename*=UTF-8''" + url.PathEscape(safeName)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", disposition)
	http.ServeFile(w, r, filePath)
}
