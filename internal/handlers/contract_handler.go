package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/service"
)

// maxRequestBodySize limits the size of request bodies (1MB)
const maxRequestBodySize = 1 << 20 // 1MB

// ValidContractStatuses contains all valid contract status values
var ValidContractStatuses = map[models.ContractStatus]bool{
	models.ContractStatusDraft:     true,
	models.ContractStatusPending:   true,
	models.ContractStatusActive:    true,
	models.ContractStatusSuspended: true,
	models.ContractStatusCancelled: true,
	models.ContractStatusCompleted: true,
}

// ContractHandler handles contract HTTP requests
type ContractHandler struct {
	svc *service.ContractService
}

// NewContractHandler creates a new ContractHandler
func NewContractHandler(svc *service.ContractService) *ContractHandler {
	return &ContractHandler{svc: svc}
}

// List handles GET /api/v1/contracts
func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	params := parsePagination(r)
	search := parseSearchParams(r)

	contracts, total, err := h.svc.List(r.Context(), tenantID, params, search)
	if err != nil {
		log.Printf("failed to list contracts: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.ContractResponse, len(contracts))
	for i, c := range contracts {
		responses[i] = c.ToResponse()
	}

	result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// Get handles GET /api/v1/contracts/{id}
func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	contract, err := h.svc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		log.Printf("failed to get contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}
	if contract == nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(contract.ToResponse()))
}

// Create handles POST /api/v1/contracts
func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())

	// Limit request body size to prevent excessive payloads
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req models.CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	if req.ContractNumber == "" || req.CustomerID == 0 {
		writeError(w, http.StatusBadRequest, ErrCodeValidationErr, "contract_number and customer_id are required")
		return
	}

	contract, err := h.svc.Create(r.Context(), tenantID, &req, user)
	if err != nil {
		log.Printf("failed to create contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.SuccessResponse(contract.ToResponse()))
}

// Update handles PUT /api/v1/contracts/{id}
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	// Limit request body size to prevent excessive payloads
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req models.UpdateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	contract, err := h.svc.Update(r.Context(), tenantID, id, &req, user)
	if err != nil {
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		if errors.Is(err, service.ErrContractCannotUpdate) {
			writeError(w, http.StatusConflict, "CONFLICT", "contract cannot be updated in current status")
			return
		}
		log.Printf("failed to update contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(contract.ToResponse()))
}

// UpdateStatus handles PATCH /api/v1/contracts/{id}/status
func (h *ContractHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	// Limit request body size to prevent excessive payloads
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req models.UpdateContractStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	// Validate that status is non-empty and matches allowed contract statuses
	if req.Status == "" || !ValidContractStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "invalid or missing status")
		return
	}

	ipAddress := getClientIP(r)
	if err := h.svc.UpdateStatus(r.Context(), tenantID, id, req.Status, user, ipAddress); err != nil {
		if errors.Is(err, service.ErrInvalidStatusTransition) {
			writeError(w, http.StatusConflict, "INVALID_TRANSITION", "invalid status transition")
			return
		}
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		log.Printf("failed to update contract status: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(nil))
}

// Sign handles POST /api/v1/contracts/{id}/sign
func (h *ContractHandler) Sign(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	// Limit request body size to prevent excessive payloads
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req models.SignContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	if req.SignedBy == "" {
		req.SignedBy = middleware.GetUser(r.Context())
	}

	ipAddress := getClientIP(r)
	if err := h.svc.Sign(r.Context(), tenantID, id, req.SignedBy, ipAddress); err != nil {
		if errors.Is(err, service.ErrCannotSign) {
			writeError(w, http.StatusConflict, "INVALID_STATUS", "contract cannot be signed in current status")
			return
		}
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		log.Printf("failed to sign contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(nil))
}

// GetHistory handles GET /api/v1/contracts/{id}/history
func (h *ContractHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	params := parsePagination(r)
	history, total, err := h.svc.GetHistory(r.Context(), tenantID, id, params)
	if err != nil {
		log.Printf("failed to get contract history: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.HistoryResponse, len(history))
	for i, histItem := range history {
		responses[i] = histItem.ToResponse()
	}

	result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// AddItem handles POST /api/v1/contracts/{id}/items
func (h *ContractHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	// Limit request body size to prevent excessive payloads
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req models.CreateContractItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	if req.ServiceID == 0 {
		writeError(w, http.StatusBadRequest, ErrCodeValidationErr, "service_id is required")
		return
	}

	item, err := h.svc.AddItem(r.Context(), tenantID, contractID, &req, user)
	if err != nil {
		if errors.Is(err, service.ErrCannotAddItem) {
			writeError(w, http.StatusConflict, "INVALID_STATUS", "cannot add items to contract in current status")
			return
		}
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		log.Printf("failed to add item to contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.SuccessResponse(item.ToResponse()))
}

// DeleteItem handles DELETE /api/v1/contracts/{id}/items/{itemId}
func (h *ContractHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	contractID, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}
	itemID, err := parseIDFromPath(r, "itemId")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidContractID)
		return
	}

	if err := h.svc.DeleteItem(r.Context(), tenantID, contractID, itemID, user); err != nil {
		if errors.Is(err, service.ErrCannotDeleteItem) {
			writeError(w, http.StatusConflict, "INVALID_STATUS", "cannot delete items from contract in current status")
			return
		}
		if errors.Is(err, service.ErrContractNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgContractNotFound)
			return
		}
		log.Printf("failed to delete item from contract: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	// Return 204 No Content for successful deletion
	w.WriteHeader(http.StatusNoContent)
}

// TrustProxy controls whether X-Forwarded-For and X-Real-IP headers are trusted.
// Set to true only when the service is behind a trusted reverse proxy.
var TrustProxy = false

// TrustedProxies is a list of trusted proxy IP addresses/CIDR ranges.
// If non-empty, proxy headers are only trusted when the request comes from these addresses.
var TrustedProxies []string

func getClientIP(r *http.Request) string {
	// Extract the remote address without port
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// Only trust proxy headers if TrustProxy is enabled or request is from a trusted proxy
	trustHeaders := TrustProxy
	if !trustHeaders && len(TrustedProxies) > 0 {
		for _, trusted := range TrustedProxies {
			if remoteIP == trusted {
				trustHeaders = true
				break
			}
		}
	}

	if trustHeaders {
		// Check X-Forwarded-For header
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	// Return remote address (without port)
	return remoteIP
}
