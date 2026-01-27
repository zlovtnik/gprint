package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/service"
	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// WorkflowHandler handles workflow HTTP requests
type WorkflowHandler struct {
	svc    *service.WorkflowService
	logger *slog.Logger
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(svc *service.WorkflowService, logger *slog.Logger) *WorkflowHandler {
	if svc == nil {
		panic("workflow service is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	return &WorkflowHandler{svc: svc, logger: logger}
}

// WorkflowResponse represents a workflow instance in API responses
type WorkflowResponse struct {
	ID           string                 `json:"id"`
	ContractID   string                 `json:"contract_id"`
	WorkflowType string                 `json:"workflow_type"`
	Status       string                 `json:"status"`
	CurrentStep  int                    `json:"current_step"`
	Steps        []WorkflowStepResponse `json:"steps,omitempty"`
	StartedAt    string                 `json:"started_at"`
	CompletedAt  *string                `json:"completed_at,omitempty"`
}

// WorkflowStepResponse represents a workflow step in API responses
type WorkflowStepResponse struct {
	ID            string  `json:"id"`
	StepNumber    int     `json:"step_number"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	AssigneeID    *string `json:"assignee_id,omitempty"`
	AssigneeType  string  `json:"assignee_type,omitempty"`
	IsOptional    bool    `json:"is_optional"`
	ActionDate    *string `json:"action_date,omitempty"`
	ActionBy      *string `json:"action_by,omitempty"`
	ActionComment string  `json:"action_comment,omitempty"`
}

func toWorkflowResponse(wf domain.WorkflowInstance) WorkflowResponse {
	resp := WorkflowResponse{
		ID:           uuid.UUID(wf.ID).String(),
		ContractID:   uuid.UUID(wf.ContractID).String(),
		WorkflowType: wf.WorkflowType,
		Status:       string(wf.Status),
		CurrentStep:  wf.CurrentStep,
		StartedAt:    wf.StartedAt.Format("2006-01-02T15:04:05Z"),
	}

	if wf.CompletedAt != nil {
		t := wf.CompletedAt.Format("2006-01-02T15:04:05Z")
		resp.CompletedAt = &t
	}

	for _, s := range wf.Steps {
		resp.Steps = append(resp.Steps, toWorkflowStepResponse(s))
	}

	return resp
}

func toWorkflowStepResponse(s domain.WorkflowStep) WorkflowStepResponse {
	resp := WorkflowStepResponse{
		ID:            uuid.UUID(s.ID).String(),
		StepNumber:    s.StepNumber,
		Name:          s.Name,
		Type:          string(s.Type),
		Status:        string(s.Status),
		AssigneeType:  s.AssigneeType,
		IsOptional:    s.IsOptional,
		ActionComment: s.ActionComment,
	}

	if s.AssigneeID != nil {
		id := uuid.UUID(*s.AssigneeID).String()
		resp.AssigneeID = &id
	}
	if s.ActionDate != nil {
		t := s.ActionDate.Format("2006-01-02T15:04:05Z")
		resp.ActionDate = &t
	}
	if s.ActionBy != nil {
		id := uuid.UUID(*s.ActionBy).String()
		resp.ActionBy = &id
	}

	return resp
}

// Get handles GET /api/v1/clm/workflows/{id}
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid workflow ID")
		return
	}

	result := h.svc.FindByID(r.Context(), tenantID, domain.WorkflowID(id))
	if fp.IsFailure(result) {
		err := fp.GetError(result)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, "workflow not found")
		} else {
			h.logger.Error("failed to find workflow", "error", err)
			writeError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
		}
		return
	}

	wf := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toWorkflowResponse(wf)))
}

// ListByContract handles GET /api/v1/clm/contracts/{contractId}/workflows
func (h *WorkflowHandler) ListByContract(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	contractIDStr := r.PathValue("contractId")

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	result := h.svc.FindByContract(r.Context(), tenantID, domain.ContractID(contractID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch workflows")
		return
	}

	workflows := fp.GetValue(result)
	var responses []WorkflowResponse
	for _, wf := range workflows {
		responses = append(responses, toWorkflowResponse(wf))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}

// GetPendingApprovals handles GET /api/v1/clm/workflows/pending-approvals
func (h *WorkflowHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.FindPendingApprovals(r.Context(), tenantID, domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch pending approvals")
		return
	}

	steps := fp.GetValue(result)
	var responses []WorkflowStepResponse
	for _, s := range steps {
		responses = append(responses, toWorkflowStepResponse(s))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}

// ApproveStepRequest represents a request to approve a step
type ApproveStepRequest struct {
	Comment string `json:"comment,omitempty"`
}

// ApproveStep handles POST /api/v1/clm/workflow-steps/{id}/approve
func (h *WorkflowHandler) ApproveStep(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid step ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	var req ApproveStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty body is acceptable
		} else {
			h.logger.Error("failed to decode ApproveStepRequest", "error", err)
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "malformed JSON")
			return
		}
	}

	result := h.svc.ApproveStep(r.Context(), tenantID, domain.WorkflowStepID(id), domain.UserID(parsedUserID), req.Comment)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	step := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toWorkflowStepResponse(step)))
}

// RejectStepRequest represents a request to reject a step
type RejectStepRequest struct {
	Reason string `json:"reason"`
}

// RejectStep handles POST /api/v1/clm/workflow-steps/{id}/reject
func (h *WorkflowHandler) RejectStep(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid step ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	var req RejectStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "rejection reason is required")
		return
	}

	result := h.svc.RejectStep(r.Context(), tenantID, domain.WorkflowStepID(id), domain.UserID(parsedUserID), req.Reason)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	step := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toWorkflowStepResponse(step)))
}

// SkipStepRequest represents a request to skip a step
type SkipStepRequest struct {
	Reason string `json:"reason,omitempty"`
}

// SkipStep handles POST /api/v1/clm/workflow-steps/{id}/skip
func (h *WorkflowHandler) SkipStep(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid step ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	var req SkipStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty body is acceptable
		} else {
			h.logger.Error("failed to decode SkipStepRequest", "error", err)
			writeError(w, http.StatusBadRequest, ErrCodeValidation, "malformed JSON")
			return
		}
	}

	result := h.svc.SkipStep(r.Context(), tenantID, domain.WorkflowStepID(id), domain.UserID(parsedUserID), req.Reason)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	step := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toWorkflowStepResponse(step)))
}

// CancelWorkflow handles POST /api/v1/clm/workflows/{id}/cancel
func (h *WorkflowHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid workflow ID")
		return
	}

	result := h.svc.CancelWorkflow(r.Context(), tenantID, domain.WorkflowID(id))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	wf := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toWorkflowResponse(wf)))
}
