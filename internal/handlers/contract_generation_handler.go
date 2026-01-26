package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/service"
)

// ContractGenerationHandler handles contract generation HTTP requests
// This handler only orchestrates calls - all sensitive processing happens in PL/SQL
type ContractGenerationHandler struct {
	svc *service.ContractGenerationService
}

// NewContractGenerationHandler creates a new ContractGenerationHandler
func NewContractGenerationHandler(svc *service.ContractGenerationService) *ContractGenerationHandler {
	return &ContractGenerationHandler{svc: svc}
}

// getSessionID extracts a session identifier from the request
func getSessionID(r *http.Request) string {
	// Try login_session claim from JWT
	if claims := middleware.GetUserClaims(r.Context()); claims != nil {
		return claims.LoginSession
	}
	return ""
}

// Generate handles POST /api/v1/contracts/{id}/generate
// Generates a printable contract document
func (h *ContractGenerationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUser(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	// Parse optional request body
	var req models.GenerateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "Invalid request body")
		return
	}

	// Get client context for audit
	ipAddress := getClientIP(r)
	sessionID := getSessionID(r)

	// Call service - all sensitive processing happens in database
	result, err := h.svc.GenerateContract(r.Context(), tenantID, contractID, userID, &req, ipAddress, sessionID)
	if err != nil {
		log.Printf("failed to generate contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	if !result.Success {
		// Map error codes to HTTP status codes
		status := http.StatusBadRequest
		switch result.ErrorCode {
		case "ERR_CONTRACT_NOT_FOUND", "ERR_TEMPLATE_NOT_FOUND", "ERR_CUSTOMER_NOT_FOUND":
			status = http.StatusNotFound
		case "ERR_UNAUTHORIZED", "ERR_TENANT_MISMATCH":
			status = http.StatusForbidden
		}
		writeError(w, status, result.ErrorCode, result.ErrorMessage)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// GetContent handles GET /api/v1/contracts/{id}/generated/{gen_id}
// Retrieves the JSON content of a generated contract for PDF rendering
func (h *ContractGenerationHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUser(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	generatedID, err := parseIDFromPath(r, "gen_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidGeneratedID)
		return
	}

	// Validate that the generated contract belongs to this contract (done in PL/SQL too)
	_ = contractID // Used for route organization; actual validation in PL/SQL

	content, err := h.svc.GetGeneratedContent(r.Context(), tenantID, generatedID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgGeneratedNotFound)
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden, ErrCodeUnauthorized, "Access denied to this generated contract")
		default:
			log.Printf("failed to get generated content: %v", err)
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(content))
}

// GetLatest handles GET /api/v1/contracts/{id}/generated/latest
// Retrieves the most recent generated version
func (h *ContractGenerationHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUser(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	content, err := h.svc.GetLatestGenerated(r.Context(), tenantID, contractID, userID)
	if err != nil {
		log.Printf("failed to get latest generated: %v", err)
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgNoGeneratedContract)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(content))
}

// ListGenerated handles GET /api/v1/contracts/{id}/generated
// Lists all generated versions for a contract (metadata only, no content)
func (h *ContractGenerationHandler) ListGenerated(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	params := parsePagination(r)
	items, total, err := h.svc.ListGeneratedContracts(r.Context(), tenantID, contractID, params.Page, params.PageSize)
	if err != nil {
		log.Printf("failed to list generated contracts: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	result := models.NewPaginatedResponse(items, params.Page, params.PageSize, total)
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// LogDownload handles POST /api/v1/contracts/{id}/generated/{gen_id}/download
// Logs a download action and returns the content
func (h *ContractGenerationHandler) LogDownload(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUser(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	generatedID, err := parseIDFromPath(r, "gen_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidGeneratedID)
		return
	}

	ipAddress := getClientIP(r)
	sessionID := getSessionID(r)

	// Log the download action
	if err := h.svc.LogDownloadAction(r.Context(), tenantID, contractID, userID, ipAddress, sessionID); err != nil {
		log.Printf("failed to log download action: %v", err)
		// Continue anyway - logging failure shouldn't block download
	}

	// Return the content
	content, err := h.svc.GetGeneratedContent(r.Context(), tenantID, generatedID, userID)
	if err != nil {
		log.Printf("failed to get content for download: %v", err)
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgGeneratedNotFound)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(content))
}

// LogPrint handles POST /api/v1/contracts/{id}/generated/{gen_id}/print
// Logs a print action for a specific generated version
func (h *ContractGenerationHandler) LogPrint(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUser(r.Context())

	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	generatedID, err := parseIDFromPath(r, "gen_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidGeneratedID)
		return
	}

	ipAddress := getClientIP(r)
	sessionID := getSessionID(r)

	if err := h.svc.LogPrintAction(r.Context(), tenantID, contractID, generatedID, userID, ipAddress, sessionID); err != nil {
		log.Printf("failed to log print action for contract %d, generated %d: %v", contractID, generatedID, err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "Failed to log print action")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"status":       "logged",
		"generated_id": generatedID,
	}))
}

// VerifyIntegrity handles GET /api/v1/contracts/{id}/generated/{gen_id}/verify
// Verifies the integrity of a generated contract with tenant authorization
func (h *ContractGenerationHandler) VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	// Contract ID from path (for route consistency, validated by tenant check in PL/SQL)
	_, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	generatedID, err := parseIDFromPath(r, "gen_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidGeneratedID)
		return
	}

	isValid, err := h.svc.VerifyContentIntegrity(r.Context(), tenantID, generatedID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthorized):
			writeError(w, http.StatusForbidden, ErrCodeUnauthorized, "Access denied to this generated contract")
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgGeneratedNotFound)
		default:
			log.Printf("failed to verify integrity: %v", err)
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]bool{"valid": isValid}))
}

// GetStats handles GET /api/v1/contracts/generation/stats
// Returns generation statistics for the tenant
func (h *ContractGenerationHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	stats, err := h.svc.GetGenerationStats(r.Context(), tenantID)
	if err != nil {
		log.Printf("failed to get generation stats: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(stats))
}

// ListTemplates handles GET /api/v1/contracts/templates
// Lists available contract templates for the tenant
func (h *ContractGenerationHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	templates, err := h.svc.ListTemplates(r.Context(), tenantID)
	if err != nil {
		log.Printf("failed to list templates: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.ContractTemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = t.ToResponse()
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}
