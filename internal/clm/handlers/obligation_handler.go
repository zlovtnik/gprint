package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/service"
	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/fp"
)

const (
	dateFormat             = "2006-01-02"
	dateTimeFormat         = "2006-01-02T15:04:05Z"
	msgInvalidUserID       = "invalid user ID"
	msgInvalidObligationID = "invalid obligation ID"
)

// ObligationHandler handles obligation HTTP requests
type ObligationHandler struct {
	svc    *service.ObligationService
	logger *slog.Logger
}

// NewObligationHandler creates a new ObligationHandler
func NewObligationHandler(svc *service.ObligationService, logger *slog.Logger) *ObligationHandler {
	if svc == nil {
		panic("obligation service is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	return &ObligationHandler{svc: svc, logger: logger}
}

// ObligationResponse represents an obligation in API responses
type ObligationResponse struct {
	ID               string  `json:"id"`
	ContractID       string  `json:"contract_id"`
	Type             string  `json:"type"`
	Title            string  `json:"title"`
	Description      string  `json:"description,omitempty"`
	ResponsibleParty string  `json:"responsible_party"`
	DueDate          string  `json:"due_date"`
	Status           string  `json:"status"`
	Amount           string  `json:"amount,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Frequency        string  `json:"frequency"`
	ReminderDays     int     `json:"reminder_days,omitempty"`
	Notes            string  `json:"notes,omitempty"`
	CompletedDate    *string `json:"completed_date,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        *string `json:"updated_at,omitempty"`
}

func toObligationResponse(o domain.Obligation) ObligationResponse {
	resp := ObligationResponse{
		ID:               uuid.UUID(o.ID).String(),
		ContractID:       uuid.UUID(o.ContractID).String(),
		Type:             string(o.Type),
		Title:            o.Title,
		Description:      o.Description,
		ResponsibleParty: uuid.UUID(o.ResponsibleParty).String(),
		DueDate:          o.DueDate.Format(dateFormat),
		Status:           string(o.Status),
		Frequency:        string(o.Frequency),
		ReminderDays:     o.ReminderDays,
		Notes:            o.Notes,
		CreatedAt:        o.CreatedAt.Format(dateTimeFormat),
	}
	if o.Amount != nil {
		resp.Amount = o.Amount.Amount.String()
		resp.Currency = o.Amount.Currency
	}
	if o.CompletedDate != nil {
		t := o.CompletedDate.Format(dateTimeFormat)
		resp.CompletedDate = &t
	}
	if o.UpdatedAt != nil {
		t := o.UpdatedAt.Format(dateTimeFormat)
		resp.UpdatedAt = &t
	}
	return resp
}

// CreateObligationRequest represents a request to create an obligation
type CreateObligationRequest struct {
	ContractID       string  `json:"contract_id"`
	Type             string  `json:"type"`
	Title            string  `json:"title"`
	Description      string  `json:"description,omitempty"`
	ResponsibleParty string  `json:"responsible_party"`
	DueDate          string  `json:"due_date"`
	Amount           *string `json:"amount,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Frequency        string  `json:"frequency,omitempty"`
	ReminderDays     int     `json:"reminder_days,omitempty"`
	Notes            string  `json:"notes,omitempty"`
}

// Create handles POST /api/v1/clm/obligations
func (h *ObligationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req CreateObligationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, msgInvalidUserID)
		return
	}

	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	responsibleParty, err := uuid.Parse(req.ResponsibleParty)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid responsible party ID")
		return
	}

	dueDate, err := time.Parse(dateFormat, req.DueDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid due date format (use YYYY-MM-DD)")
		return
	}

	var amount *decimal.Decimal
	var currency string
	if req.Amount != nil && *req.Amount != "" {
		amountVal, err := decimal.NewFromString(*req.Amount)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid amount")
			return
		}
		amount = &amountVal
		currency = req.Currency
		if currency == "" {
			currency = "USD"
		}
	}

	svcReq := service.CreateObligationRequest{
		ContractID:       domain.ContractID(contractID),
		Type:             domain.ObligationType(req.Type),
		Title:            req.Title,
		Description:      req.Description,
		ResponsibleParty: domain.PartyID(responsibleParty),
		DueDate:          dueDate,
		Amount:           amount,
		Currency:         currency,
		Frequency:        domain.ObligationFrequency(req.Frequency),
		ReminderDays:     req.ReminderDays,
		Notes:            req.Notes,
		CreatedBy:        domain.UserID(parsedUserID),
	}

	result := h.svc.Create(r.Context(), tenantID, svcReq)
	if fp.IsFailure(result) {
		err := fp.GetError(result)
		// Check for validation errors by type
		var validationErr fp.ValidationErrors
		if errors.As(err, &validationErr) {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, err.Error())
		} else {
			h.logger.Error("failed to create obligation", "error", err, "tenant_id", tenantID)
			writeError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
		}
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusCreated, models.SuccessResponse(toObligationResponse(obligation)))
}

// Get handles GET /api/v1/clm/obligations/{id}
func (h *ObligationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, msgInvalidObligationID)
		return
	}

	result := h.svc.FindByID(r.Context(), tenantID, domain.ObligationID(id))
	if fp.IsFailure(result) {
		err := fp.GetError(result)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, "obligation not found")
		} else {
			h.logger.Error("failed to find obligation", "error", err, "tenant_id", tenantID, "id", idStr)
			writeError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
		}
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toObligationResponse(obligation)))
}

// ListByContract handles GET /api/v1/clm/contracts/{contractId}/obligations
func (h *ObligationHandler) ListByContract(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	contractIDStr := r.PathValue("contractId")

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	result := h.svc.FindByContract(r.Context(), tenantID, domain.ContractID(contractID), offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch obligations")
		return
	}

	obligations := fp.GetValue(result)
	var responses []ObligationResponse
	for _, o := range obligations {
		responses = append(responses, toObligationResponse(o))
	}

	countResult := h.svc.CountByContract(r.Context(), tenantID, domain.ContractID(contractID))
	total := 0
	if fp.IsSuccess(countResult) {
		total = fp.GetValue(countResult)
	} else {
		h.logger.Error("failed to count obligations", "error", fp.GetError(countResult))
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to count obligations")
		return
	}

	writeJSON(w, http.StatusOK, models.NewPaginatedResponse(responses, page, pageSize, total))
}

// UpdateObligationRequest represents a request to update an obligation
type UpdateObligationRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Amount      *string `json:"amount,omitempty"`
	Currency    *string `json:"currency,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

// Update handles PUT /api/v1/clm/obligations/{id}
func (h *ObligationHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, msgInvalidObligationID)
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, msgInvalidUserID)
		return
	}

	var req UpdateObligationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		t, err := time.Parse(dateFormat, *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid due date format")
			return
		}
		dueDate = &t
	}

	var amount *decimal.Decimal
	if req.Amount != nil && *req.Amount != "" {
		amountVal, err := decimal.NewFromString(*req.Amount)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "invalid amount")
			return
		}
		amount = &amountVal
	}

	svcReq := service.UpdateObligationRequest{
		Title:       req.Title,
		Description: req.Description,
		DueDate:     dueDate,
		Amount:      amount,
		Currency:    req.Currency,
		Notes:       req.Notes,
		UpdatedBy:   domain.UserID(parsedUserID),
	}

	result := h.svc.Update(r.Context(), tenantID, domain.ObligationID(id), svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toObligationResponse(obligation)))
}

// MarkInProgress handles POST /api/v1/clm/obligations/{id}/in-progress
func (h *ObligationHandler) MarkInProgress(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, msgInvalidObligationID)
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, msgInvalidUserID)
		return
	}

	result := h.svc.MarkInProgress(r.Context(), tenantID, domain.ObligationID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toObligationResponse(obligation)))
}

// Complete handles POST /api/v1/clm/obligations/{id}/complete
func (h *ObligationHandler) Complete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, msgInvalidObligationID)
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, msgInvalidUserID)
		return
	}

	result := h.svc.Complete(r.Context(), tenantID, domain.ObligationID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toObligationResponse(obligation)))
}

// Waive handles POST /api/v1/clm/obligations/{id}/waive
func (h *ObligationHandler) Waive(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, msgInvalidObligationID)
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, msgInvalidUserID)
		return
	}

	result := h.svc.Waive(r.Context(), tenantID, domain.ObligationID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	obligation := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toObligationResponse(obligation)))
}

// FindOverdue handles GET /api/v1/clm/obligations/overdue
func (h *ObligationHandler) FindOverdue(w http.ResponseWriter, r *http.Request) {
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

	result := h.svc.FindOverdue(r.Context(), tenantID, offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch overdue obligations")
		return
	}

	obligations := fp.GetValue(result)
	responses := make([]ObligationResponse, 0, len(obligations))
	for _, o := range obligations {
		responses = append(responses, toObligationResponse(o))
	}

	countResult := h.svc.CountOverdue(r.Context(), tenantID)
	total := 0
	if fp.IsSuccess(countResult) {
		total = fp.GetValue(countResult)
	} else {
		h.logger.Error("failed to count overdue obligations", "error", fp.GetError(countResult))
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to count overdue obligations")
		return
	}

	writeJSON(w, http.StatusOK, models.NewPaginatedResponse(responses, page, pageSize, total))
}
