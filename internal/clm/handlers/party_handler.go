package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/service"
	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// PartyHandler handles party HTTP requests
type PartyHandler struct {
	svc    *service.PartyService
	logger *slog.Logger
}

// NewPartyHandler creates a new PartyHandler
func NewPartyHandler(svc *service.PartyService, logger *slog.Logger) *PartyHandler {
	if svc == nil {
		panic("party service is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	return &PartyHandler{svc: svc, logger: logger}
}

// PartyResponse represents a party in API responses
type PartyResponse struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	LegalName      string  `json:"legal_name,omitempty"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone,omitempty"`
	TaxID          string  `json:"tax_id,omitempty"`
	Address        any     `json:"address,omitempty"`
	BillingAddress any     `json:"billing_address,omitempty"`
	RiskLevel      string  `json:"risk_level,omitempty"`
	RiskScore      int     `json:"risk_score,omitempty"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      *string `json:"updated_at,omitempty"`
}

func toPartyResponse(p domain.Party) PartyResponse {
	resp := PartyResponse{
		ID:        uuid.UUID(p.ID).String(),
		Type:      string(p.Type),
		Name:      p.Name,
		LegalName: p.LegalName,
		Email:     p.Email,
		Phone:     p.Phone,
		TaxID:     p.TaxID,
		Address:   p.Address,
		RiskLevel: string(p.RiskLevel),
		RiskScore: p.RiskScore,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.BillingAddress != nil {
		resp.BillingAddress = p.BillingAddress
	}
	if p.UpdatedAt != nil {
		t := p.UpdatedAt.Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &t
	}
	return resp
}

// CreatePartyRequest represents a request to create a party
type CreatePartyRequest struct {
	Type           string          `json:"type"`
	Name           string          `json:"name"`
	LegalName      string          `json:"legal_name,omitempty"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone,omitempty"`
	TaxID          string          `json:"tax_id,omitempty"`
	Address        *domain.Address `json:"address,omitempty"`
	BillingAddress *domain.Address `json:"billing_address,omitempty"`
	RiskLevel      string          `json:"risk_level,omitempty"`
	RiskScore      int             `json:"risk_score,omitempty"`
}

// Create handles POST /api/v1/clm/parties
func (h *PartyHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req CreatePartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	svcReq := service.CreatePartyRequest{
		Type:           domain.PartyType(req.Type),
		Name:           req.Name,
		LegalName:      req.LegalName,
		Email:          req.Email,
		Phone:          req.Phone,
		TaxID:          req.TaxID,
		Address:        req.Address,
		BillingAddress: req.BillingAddress,
		RiskLevel:      domain.RiskLevel(req.RiskLevel),
		RiskScore:      req.RiskScore,
		CreatedBy:      domain.UserID(parsedUserID),
	}

	result := h.svc.Create(r.Context(), tenantID, svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	party := fp.GetValue(result)
	writeJSON(w, http.StatusCreated, models.SuccessResponse(toPartyResponse(party)))
}

// Get handles GET /api/v1/clm/parties/{id}
func (h *PartyHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid party ID")
		return
	}

	result := h.svc.FindByID(r.Context(), tenantID, domain.PartyID(id))
	if fp.IsFailure(result) {
		err := fp.GetError(result)
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, "party not found")
		} else {
			h.logger.Error("failed to find party", "error", err)
			writeError(w, http.StatusInternalServerError, ErrCodeInternal, err.Error())
		}
		return
	}

	party := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toPartyResponse(party)))
}

// List handles GET /api/v1/clm/parties
func (h *PartyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	result := h.svc.FindAll(r.Context(), tenantID, offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch parties")
		return
	}

	parties := fp.GetValue(result)
	responses := make([]PartyResponse, 0, len(parties))
	for _, p := range parties {
		responses = append(responses, toPartyResponse(p))
	}

	countResult := h.svc.Count(r.Context(), tenantID)
	total := 0
	if fp.IsSuccess(countResult) {
		total = fp.GetValue(countResult)
	}

	writeJSON(w, http.StatusOK, models.NewPaginatedResponse(responses, page, pageSize, total))
}

// UpdatePartyRequest represents a request to update a party
type UpdatePartyRequest struct {
	Name           *string         `json:"name,omitempty"`
	LegalName      *string         `json:"legal_name,omitempty"`
	Email          *string         `json:"email,omitempty"`
	Phone          *string         `json:"phone,omitempty"`
	TaxID          *string         `json:"tax_id,omitempty"`
	Address        *domain.Address `json:"address,omitempty"`
	BillingAddress *domain.Address `json:"billing_address,omitempty"`
	RiskLevel      *string         `json:"risk_level,omitempty"`
	RiskScore      *int            `json:"risk_score,omitempty"`
}

// Update handles PUT /api/v1/clm/parties/{id}
func (h *PartyHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid party ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	var req UpdatePartyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	var riskLevel *domain.RiskLevel
	if req.RiskLevel != nil {
		rl := domain.RiskLevel(*req.RiskLevel)
		riskLevel = &rl
	}

	svcReq := service.UpdatePartyRequest{
		Name:           req.Name,
		LegalName:      req.LegalName,
		Email:          req.Email,
		Phone:          req.Phone,
		TaxID:          req.TaxID,
		Address:        req.Address,
		BillingAddress: req.BillingAddress,
		RiskLevel:      riskLevel,
		RiskScore:      req.RiskScore,
		UpdatedBy:      domain.UserID(parsedUserID),
	}

	result := h.svc.Update(r.Context(), tenantID, domain.PartyID(id), svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	party := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toPartyResponse(party)))
}

// Delete handles DELETE /api/v1/clm/parties/{id}
func (h *PartyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid party ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.Deactivate(r.Context(), tenantID, domain.PartyID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]bool{"deactivated": true}))
}
