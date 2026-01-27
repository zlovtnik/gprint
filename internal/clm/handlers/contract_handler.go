package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/internal/clm/service"
	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ContractHandler handles contract HTTP requests
type ContractHandler struct {
	svc *service.ContractService
}

// NewContractHandler creates a new ContractHandler
func NewContractHandler(svc *service.ContractService) *ContractHandler {
	if svc == nil {
		panic("contract service is required")
	}
	return &ContractHandler{svc: svc}
}

// ContractResponse represents a contract in API responses
type ContractResponse struct {
	ID               string                  `json:"id"`
	ContractNumber   string                  `json:"contract_number"`
	ExternalRef      string                  `json:"external_ref,omitempty"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description,omitempty"`
	ContractType     string                  `json:"contract_type"`
	Status           string                  `json:"status"`
	Parties          []ContractPartyResponse `json:"parties"`
	ValueAmount      string                  `json:"value_amount"`
	ValueCurrency    string                  `json:"value_currency"`
	EffectiveDate    *string                 `json:"effective_date,omitempty"`
	ExpirationDate   *string                 `json:"expiration_date,omitempty"`
	AutoRenew        bool                    `json:"auto_renew"`
	RenewalTermDays  int                     `json:"renewal_term_days,omitempty"`
	NoticePeriodDays int                     `json:"notice_period_days,omitempty"`
	Terms            string                  `json:"terms,omitempty"`
	Notes            string                  `json:"notes,omitempty"`
	Version          int                     `json:"version"`
	IsDeleted        bool                    `json:"is_deleted"`
	CreatedAt        string                  `json:"created_at"`
	UpdatedAt        *string                 `json:"updated_at,omitempty"`
}

// ContractPartyResponse represents a party in a contract response
type ContractPartyResponse struct {
	PartyID   string  `json:"party_id"`
	Role      string  `json:"role"`
	IsPrimary bool    `json:"is_primary"`
	SignedAt  *string `json:"signed_at,omitempty"`
}

func toContractResponse(c domain.Contract) ContractResponse {
	resp := ContractResponse{
		ID:               uuid.UUID(c.ID).String(),
		ContractNumber:   c.ContractNumber,
		ExternalRef:      c.ExternalRef,
		Title:            c.Title,
		Description:      c.Description,
		ContractType:     c.ContractType.Code,
		Status:           string(c.Status),
		ValueAmount:      c.Value.Amount.String(),
		ValueCurrency:    c.Value.Currency,
		AutoRenew:        c.AutoRenew,
		RenewalTermDays:  c.RenewalTermDays,
		NoticePeriodDays: c.NoticePeriodDays,
		Terms:            c.Terms,
		Notes:            c.Notes,
		Version:          c.Version,
		IsDeleted:        c.IsDeleted,
		CreatedAt:        c.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Format parties
	for _, p := range c.Parties {
		pr := ContractPartyResponse{
			PartyID:   uuid.UUID(p.PartyID).String(),
			Role:      string(p.Role),
			IsPrimary: p.IsPrimary,
		}
		if p.SignedAt != nil {
			t := p.SignedAt.Format("2006-01-02T15:04:05Z")
			pr.SignedAt = &t
		}
		resp.Parties = append(resp.Parties, pr)
	}

	if c.EffectiveDate != nil {
		t := c.EffectiveDate.Format("2006-01-02")
		resp.EffectiveDate = &t
	}
	if c.ExpirationDate != nil {
		t := c.ExpirationDate.Format("2006-01-02")
		resp.ExpirationDate = &t
	}
	if c.UpdatedAt != nil {
		t := c.UpdatedAt.Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &t
	}
	return resp
}

// CreateContractRequest represents a request to create a contract
type CreateContractRequest struct {
	ContractNumber   string                       `json:"contract_number"`
	ExternalRef      string                       `json:"external_ref,omitempty"`
	Title            string                       `json:"title"`
	Description      string                       `json:"description,omitempty"`
	ContractType     CreateContractTypeRequest    `json:"contract_type"`
	Parties          []CreateContractPartyRequest `json:"parties"`
	ValueAmount      string                       `json:"value_amount"`
	ValueCurrency    string                       `json:"value_currency"`
	EffectiveDate    *string                      `json:"effective_date,omitempty"`
	ExpirationDate   *string                      `json:"expiration_date,omitempty"`
	AutoRenew        bool                         `json:"auto_renew"`
	RenewalTermDays  int                          `json:"renewal_term_days,omitempty"`
	NoticePeriodDays int                          `json:"notice_period_days,omitempty"`
	Terms            string                       `json:"terms,omitempty"`
	Notes            string                       `json:"notes,omitempty"`
}

// CreateContractTypeRequest represents contract type in a request
type CreateContractTypeRequest struct {
	Name                string `json:"name"`
	Code                string `json:"code"`
	Description         string `json:"description,omitempty"`
	DefaultDurationDays int    `json:"default_duration_days,omitempty"`
	RequiresApproval    bool   `json:"requires_approval"`
	ApprovalLevels      int    `json:"approval_levels,omitempty"`
}

// CreateContractPartyRequest represents a party in a contract creation request
type CreateContractPartyRequest struct {
	PartyID   string `json:"party_id"`
	Role      string `json:"role"`
	IsPrimary bool   `json:"is_primary"`
}

// Create handles POST /api/v1/clm/contracts
func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req CreateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	// Parse value amount
	valueAmount, err := decimal.NewFromString(req.ValueAmount)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid value amount")
		return
	}
	currency := req.ValueCurrency
	if currency == "" {
		currency = "USD"
	}

	// Parse dates
	var effectiveDate *time.Time
	if req.EffectiveDate != nil {
		t, err := time.Parse("2006-01-02", *req.EffectiveDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid effective date format (use YYYY-MM-DD)")
			return
		}
		effectiveDate = &t
	}

	var expirationDate *time.Time
	if req.ExpirationDate != nil {
		t, err := time.Parse("2006-01-02", *req.ExpirationDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid expiration date format (use YYYY-MM-DD)")
			return
		}
		expirationDate = &t
	}

	// Build contract type
	contractType := domain.ContractType{
		Name:                req.ContractType.Name,
		Code:                req.ContractType.Code,
		Description:         req.ContractType.Description,
		DefaultDurationDays: req.ContractType.DefaultDurationDays,
		RequiresApproval:    req.ContractType.RequiresApproval,
		ApprovalLevels:      req.ContractType.ApprovalLevels,
	}

	// Build parties
	var parties []service.CreateContractPartyRequest
	for _, p := range req.Parties {
		partyID, err := uuid.Parse(p.PartyID)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid party ID: "+p.PartyID)
			return
		}
		parties = append(parties, service.CreateContractPartyRequest{
			PartyID:   domain.PartyID(partyID),
			Role:      domain.PartyRole(p.Role),
			IsPrimary: p.IsPrimary,
		})
	}

	svcReq := service.CreateContractRequest{
		ContractNumber:   req.ContractNumber,
		ExternalRef:      req.ExternalRef,
		Title:            req.Title,
		Description:      req.Description,
		ContractType:     contractType,
		ValueAmount:      valueAmount,
		ValueCurrency:    currency,
		EffectiveDate:    effectiveDate,
		ExpirationDate:   expirationDate,
		AutoRenew:        req.AutoRenew,
		RenewalTermDays:  req.RenewalTermDays,
		NoticePeriodDays: req.NoticePeriodDays,
		Terms:            req.Terms,
		Notes:            req.Notes,
		Parties:          parties,
		CreatedBy:        domain.UserID(parsedUserID),
	}

	result := h.svc.Create(r.Context(), tenantID, svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	contract := fp.GetValue(result)
	writeJSON(w, http.StatusCreated, models.SuccessResponse(toContractResponse(contract)))
}

// Get handles GET /api/v1/clm/contracts/{id}
func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	result := h.svc.FindByID(r.Context(), tenantID, domain.ContractID(id))
	if fp.IsFailure(result) {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "contract not found")
		return
	}

	contract := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toContractResponse(contract)))
}

// List handles GET /api/v1/clm/contracts
func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filter := repository.ContractFilter{}

	if status := query.Get("status"); status != "" {
		s := domain.ContractStatus(status)
		filter.Status = &s
	}

	if partyID := query.Get("party_id"); partyID != "" {
		if id, err := uuid.Parse(partyID); err == nil {
			pid := domain.PartyID(id)
			filter.PartyID = &pid
		}
	}

	if searchTerm := query.Get("search"); searchTerm != "" {
		filter.SearchTerm = searchTerm
	}

	result := h.svc.FindAll(r.Context(), tenantID, filter, offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch contracts")
		return
	}

	contracts := fp.GetValue(result)
	var responses []ContractResponse
	for _, c := range contracts {
		responses = append(responses, toContractResponse(c))
	}

	countResult := h.svc.Count(r.Context(), tenantID, filter)
	total := 0
	if fp.IsSuccess(countResult) {
		total = fp.GetValue(countResult)
	}

	writeJSON(w, http.StatusOK, models.NewPaginatedResponse(responses, page, pageSize, total))
}

// UpdateContractRequest represents a request to update a contract
type UpdateContractRequest struct {
	Title            *string `json:"title,omitempty"`
	Description      *string `json:"description,omitempty"`
	ValueAmount      *string `json:"value_amount,omitempty"`
	ValueCurrency    *string `json:"value_currency,omitempty"`
	EffectiveDate    *string `json:"effective_date,omitempty"`
	ExpirationDate   *string `json:"expiration_date,omitempty"`
	AutoRenew        *bool   `json:"auto_renew,omitempty"`
	RenewalTermDays  *int    `json:"renewal_term_days,omitempty"`
	NoticePeriodDays *int    `json:"notice_period_days,omitempty"`
	Terms            *string `json:"terms,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

// Update handles PUT /api/v1/clm/contracts/{id}
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	var req UpdateContractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	var effectiveDate *time.Time
	if req.EffectiveDate != nil {
		t, err := time.Parse("2006-01-02", *req.EffectiveDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid effective date format")
			return
		}
		effectiveDate = &t
	}

	var expirationDate *time.Time
	if req.ExpirationDate != nil {
		t, err := time.Parse("2006-01-02", *req.ExpirationDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid expiration date format")
			return
		}
		expirationDate = &t
	}

	var valueAmount *decimal.Decimal
	if req.ValueAmount != nil && *req.ValueAmount != "" {
		amount, err := decimal.NewFromString(*req.ValueAmount)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid value amount")
			return
		}
		valueAmount = &amount
	}

	svcReq := service.UpdateContractRequest{
		Title:            req.Title,
		Description:      req.Description,
		ValueAmount:      valueAmount,
		ValueCurrency:    req.ValueCurrency,
		EffectiveDate:    effectiveDate,
		ExpirationDate:   expirationDate,
		AutoRenew:        req.AutoRenew,
		RenewalTermDays:  req.RenewalTermDays,
		NoticePeriodDays: req.NoticePeriodDays,
		Terms:            req.Terms,
		Notes:            req.Notes,
		UpdatedBy:        domain.UserID(parsedUserID),
	}

	result := h.svc.Update(r.Context(), tenantID, domain.ContractID(id), svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	contract := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toContractResponse(contract)))
}

// Submit handles POST /api/v1/clm/contracts/{id}/submit
func (h *ContractHandler) Submit(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.Submit(r.Context(), tenantID, domain.ContractID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	contract := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toContractResponse(contract)))
}

// Approve handles POST /api/v1/clm/contracts/{id}/approve
func (h *ContractHandler) Approve(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.Approve(r.Context(), tenantID, domain.ContractID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	contract := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toContractResponse(contract)))
}

// Delete handles DELETE /api/v1/clm/contracts/{id}
func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.SoftDelete(r.Context(), tenantID, domain.ContractID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]bool{"deleted": true}))
}

// FindExpiring handles GET /api/v1/clm/contracts/expiring
func (h *ContractHandler) FindExpiring(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	daysStr := r.URL.Query().Get("days")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 30
	}

	result := h.svc.FindExpiring(r.Context(), tenantID, days)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch expiring contracts")
		return
	}

	contracts := fp.GetValue(result)
	var responses []ContractResponse
	for _, c := range contracts {
		responses = append(responses, toContractResponse(c))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}
