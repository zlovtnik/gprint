package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/service"
	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// DocumentHandler handles document HTTP requests
type DocumentHandler struct {
	svc    *service.DocumentService
	logger *slog.Logger
}

// NewDocumentHandler creates a new DocumentHandler
func NewDocumentHandler(svc *service.DocumentService, logger *slog.Logger) *DocumentHandler {
	if svc == nil {
		panic("document service is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	return &DocumentHandler{svc: svc, logger: logger}
}

// UploadDocumentAPIRequest represents the request body for uploading a document
type UploadDocumentAPIRequest struct {
	ContractID string `json:"contract_id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"file_path"`
	MimeType   string `json:"mime_type"`
	FileSize   int64  `json:"file_size"`
	Checksum   string `json:"checksum,omitempty"`
	IsPrimary  bool   `json:"is_primary"`
}

// DocumentResponse represents a document in API responses
type DocumentResponse struct {
	ID         string  `json:"id"`
	ContractID string  `json:"contract_id"`
	Type       string  `json:"type"`
	Title      string  `json:"title"`
	FileName   string  `json:"file_name"`
	FilePath   string  `json:"file_path"`
	MimeType   string  `json:"mime_type"`
	FileSize   int64   `json:"file_size"`
	Checksum   string  `json:"checksum,omitempty"`
	IsPrimary  bool    `json:"is_primary"`
	IsSigned   bool    `json:"is_signed"`
	SignedAt   *string `json:"signed_at,omitempty"`
	SignedBy   *string `json:"signed_by,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

func toDocumentResponse(d domain.Document) DocumentResponse {
	resp := DocumentResponse{
		ID:         uuid.UUID(d.ID).String(),
		ContractID: uuid.UUID(d.ContractID).String(),
		Type:       string(d.Type),
		Title:      d.Title,
		FileName:   d.FileName,
		FilePath:   d.FilePath,
		MimeType:   d.MimeType,
		FileSize:   d.FileSize,
		Checksum:   d.Checksum,
		IsPrimary:  d.IsPrimary,
		IsSigned:   d.IsSigned,
		UploadedAt: d.UploadedAt.Format("2006-01-02T15:04:05Z"),
	}

	if d.SignedAt != nil {
		t := d.SignedAt.Format("2006-01-02T15:04:05Z")
		resp.SignedAt = &t
	}
	if d.SignedBy != nil {
		id := uuid.UUID(*d.SignedBy).String()
		resp.SignedBy = &id
	}

	return resp
}

// Upload handles POST /api/v1/clm/documents
func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var req UploadDocumentAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidJSON, "invalid request body")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	contractID, err := uuid.Parse(req.ContractID)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid contract ID")
		return
	}

	svcReq := service.UploadDocumentRequest{
		ContractID: domain.ContractID(contractID),
		Type:       domain.DocumentType(req.Type),
		Title:      req.Title,
		FileName:   req.FileName,
		FilePath:   req.FilePath,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		Checksum:   req.Checksum,
		IsPrimary:  req.IsPrimary,
	}

	result := h.svc.Upload(r.Context(), tenantID, domain.UserID(parsedUserID), svcReq)
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	doc := fp.GetValue(result)
	writeJSON(w, http.StatusCreated, models.SuccessResponse(toDocumentResponse(doc)))
}

// Get handles GET /api/v1/clm/documents/{id}
func (h *DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid document ID")
		return
	}

	result := h.svc.FindByID(r.Context(), tenantID, domain.DocumentID(id))
	if fp.IsFailure(result) {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "document not found")
		return
	}

	doc := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toDocumentResponse(doc)))
}

// ListByContract handles GET /api/v1/clm/contracts/{contractId}/documents
func (h *DocumentHandler) ListByContract(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch documents")
		return
	}

	docs := fp.GetValue(result)
	responses := make([]DocumentResponse, 0, len(docs))
	for _, d := range docs {
		responses = append(responses, toDocumentResponse(d))
	}

	countResult := h.svc.CountByContract(r.Context(), tenantID, domain.ContractID(contractID))
	total := 0
	if fp.IsSuccess(countResult) {
		total = fp.GetValue(countResult)
	} else {
		h.logger.Error("failed to count documents", "error", fp.GetError(countResult))
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to count documents")
		return
	}

	writeJSON(w, http.StatusOK, models.NewPaginatedResponse(responses, page, pageSize, total))
}

// Sign handles POST /api/v1/clm/documents/{id}/sign
func (h *DocumentHandler) Sign(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid document ID")
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid user ID")
		return
	}

	result := h.svc.Sign(r.Context(), tenantID, domain.DocumentID(id), domain.UserID(parsedUserID))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	doc := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toDocumentResponse(doc)))
}

// Delete handles DELETE /api/v1/clm/documents/{id}
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid document ID")
		return
	}

	result := h.svc.Delete(r.Context(), tenantID, domain.DocumentID(id))
	if fp.IsFailure(result) {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, fp.GetError(result).Error())
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]bool{"deleted": true}))
}

// TemplateResponse represents a template in API responses
type TemplateResponse struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	ContractTypeID string                     `json:"contract_type_id"`
	FilePath       string                     `json:"file_path"`
	Version        int                        `json:"version"`
	MergeFields    []MergeFieldDefinitionResp `json:"merge_fields,omitempty"`
	IsActive       bool                       `json:"is_active"`
	CreatedAt      string                     `json:"created_at"`
	UpdatedAt      *string                    `json:"updated_at,omitempty"`
}

// MergeFieldDefinitionResp represents a merge field definition in responses
type MergeFieldDefinitionResp struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DataType     string `json:"data_type"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
}

func toTemplateResponse(t domain.DocumentTemplate) TemplateResponse {
	resp := TemplateResponse{
		ID:             uuid.UUID(t.ID).String(),
		Name:           t.Name,
		ContractTypeID: uuid.UUID(t.ContractTypeID).String(),
		FilePath:       t.FilePath,
		Version:        t.Version,
		IsActive:       t.IsActive,
		CreatedAt:      t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	for _, mf := range t.MergeFields {
		resp.MergeFields = append(resp.MergeFields, MergeFieldDefinitionResp{
			Name:         mf.Name,
			Description:  mf.Description,
			DataType:     mf.DataType,
			Required:     mf.Required,
			DefaultValue: mf.DefaultValue,
		})
	}

	if t.UpdatedAt != nil {
		u := t.UpdatedAt.Format("2006-01-02T15:04:05Z")
		resp.UpdatedAt = &u
	}

	return resp
}

// GetTemplate handles GET /api/v1/clm/templates/{id}
func (h *DocumentHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid template ID")
		return
	}

	result := h.svc.FindTemplateByID(r.Context(), tenantID, domain.TemplateID(id))
	if fp.IsFailure(result) {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "template not found")
		return
	}

	template := fp.GetValue(result)
	writeJSON(w, http.StatusOK, models.SuccessResponse(toTemplateResponse(template)))
}

// ListTemplates handles GET /api/v1/clm/templates
func (h *DocumentHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	activeOnlyStr := r.URL.Query().Get("active_only")
	activeOnly := activeOnlyStr == "true" || activeOnlyStr == "1"

	result := h.svc.FindAllTemplates(r.Context(), tenantID, activeOnly)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch templates")
		return
	}

	templates := fp.GetValue(result)
	var responses []TemplateResponse
	for _, t := range templates {
		responses = append(responses, toTemplateResponse(t))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}

// AuditHandler handles audit HTTP requests
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	if svc == nil {
		panic("audit service is required")
	}
	return &AuditHandler{svc: svc}
}

// AuditEntryResponse represents an audit entry in API responses
type AuditEntryResponse struct {
	ID         string                 `json:"id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	Category   string                 `json:"category"`
	UserID     string                 `json:"user_id"`
	UserName   string                 `json:"user_name"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	OldValues  map[string]interface{} `json:"old_values,omitempty"`
	NewValues  map[string]interface{} `json:"new_values,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

func toAuditEntryResponse(e domain.AuditEntry) AuditEntryResponse {
	return AuditEntryResponse{
		ID:         e.ID,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Action:     string(e.Action),
		Category:   string(e.Category),
		UserID:     uuid.UUID(e.UserID).String(),
		UserName:   e.UserName,
		IPAddress:  e.IPAddress,
		UserAgent:  e.UserAgent,
		OldValues:  e.OldValues,
		NewValues:  e.NewValues,
		Timestamp:  e.Timestamp.Format("2006-01-02T15:04:05Z"),
	}
}

// ListByEntity handles GET /api/v1/clm/audit/entity/{entityType}/{entityId}
func (h *AuditHandler) ListByEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	entityType := r.PathValue("entityType")
	entityID := r.PathValue("entityId")

	if entityType == "" || entityID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "entity type and ID are required")
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

	result := h.svc.FindByEntity(r.Context(), tenantID, entityType, entityID, offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch audit entries")
		return
	}

	entries := fp.GetValue(result)
	responses := make([]AuditEntryResponse, 0, len(entries))
	for _, e := range entries {
		responses = append(responses, toAuditEntryResponse(e))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}

// ListByUser handles GET /api/v1/clm/audit/user/{userId}
func (h *AuditHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	userIDStr := r.PathValue("userId")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, "invalid user ID")
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

	result := h.svc.FindByUser(r.Context(), tenantID, domain.UserID(userID), offset, pageSize)
	if fp.IsFailure(result) {
		writeError(w, http.StatusInternalServerError, ErrCodeInternal, "failed to fetch audit entries")
		return
	}

	entries := fp.GetValue(result)
	responses := make([]AuditEntryResponse, 0, len(entries))
	for _, e := range entries {
		responses = append(responses, toAuditEntryResponse(e))
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(responses))
}
