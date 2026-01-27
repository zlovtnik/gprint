package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// DocumentService handles document business logic
type DocumentService struct {
	repo         *repository.DocumentRepository
	templateRepo *repository.TemplateRepository
	auditRepo    *repository.AuditRepository
}

// NewDocumentService creates a new DocumentService
func NewDocumentService(repo *repository.DocumentRepository, templateRepo *repository.TemplateRepository, auditRepo *repository.AuditRepository) *DocumentService {
	if repo == nil {
		panic("document repository is required")
	}
	return &DocumentService{repo: repo, templateRepo: templateRepo, auditRepo: auditRepo}
}

// UploadDocumentRequest represents a request to upload a document
type UploadDocumentRequest struct {
	ContractID domain.ContractID
	Type       domain.DocumentType
	Title      string
	FileName   string
	FilePath   string
	MimeType   string
	FileSize   int64
	Checksum   string
	IsPrimary  bool
}

// Upload uploads a new document
func (s *DocumentService) Upload(ctx context.Context, tenantID string, uploadedBy domain.UserID, req UploadDocumentRequest) fp.Result[domain.Document] {
	if req.Title == "" {
		return fp.Failure[domain.Document](errors.New("title is required"))
	}
	if req.ContractID.IsZero() {
		return fp.Failure[domain.Document](errors.New("contract is required"))
	}
	if req.FilePath == "" {
		return fp.Failure[domain.Document](errors.New("file path is required"))
	}

	id := domain.DocumentID(uuid.New())
	now := time.Now()

	doc := domain.Document{
		ID:         id,
		TenantID:   tenantID,
		ContractID: req.ContractID,
		Type:       req.Type,
		Title:      req.Title,
		FileName:   req.FileName,
		FilePath:   req.FilePath,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		Checksum:   req.Checksum,
		IsPrimary:  req.IsPrimary,
		IsSigned:   false,
		UploadedAt: now,
		UploadedBy: uploadedBy,
	}

	result := s.repo.Create(ctx, doc)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "document", uuid.UUID(id).String(), domain.AuditActionCreate, nil, &doc, uploadedBy, "")
	}

	return result
}

// FindByID retrieves a document by ID
func (s *DocumentService) FindByID(ctx context.Context, tenantID string, id domain.DocumentID) fp.Result[domain.Document] {
	return s.repo.FindByID(ctx, tenantID, id)
}

// FindByContract retrieves documents for a contract with pagination
func (s *DocumentService) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID, offset, limit int) fp.Result[[]domain.Document] {
	return s.repo.FindByContract(ctx, tenantID, contractID, offset, limit)
}

// CountByContract returns the count of documents for a contract
func (s *DocumentService) CountByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[int] {
	return s.repo.CountByContract(ctx, tenantID, contractID)
}

// Sign marks a document as signed
func (s *DocumentService) Sign(ctx context.Context, tenantID string, id domain.DocumentID, signedBy domain.UserID) fp.Result[domain.Document] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	if existing.IsSigned {
		return fp.Failure[domain.Document](errors.New("document is already signed"))
	}

	result := s.repo.MarkSigned(ctx, tenantID, id, signedBy)
	if fp.IsFailure(result) {
		return fp.Failure[domain.Document](fp.GetError(result))
	}

	// Re-fetch the document to get the persisted state
	updatedResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(updatedResult) {
		return updatedResult
	}
	updated := fp.GetValue(updatedResult)

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "document", uuid.UUID(id).String(), domain.AuditActionSign, &existing, &updated, signedBy, "")
	}

	return fp.Success(updated)
}

// Delete removes a document
func (s *DocumentService) Delete(ctx context.Context, tenantID string, id domain.DocumentID) fp.Result[bool] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return fp.Failure[bool](fp.GetError(existingResult))
	}
	existing := fp.GetValue(existingResult)

	if existing.IsSigned {
		return fp.Failure[bool](errors.New("cannot delete signed documents"))
	}

	result := s.repo.Delete(ctx, tenantID, id)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "document", uuid.UUID(id).String(), domain.AuditActionDelete, &existing, nil, domain.UserID(uuid.Nil), "")
	}

	return result
}

// FindTemplateByID retrieves a template by ID
func (s *DocumentService) FindTemplateByID(ctx context.Context, tenantID string, id domain.TemplateID) fp.Result[domain.DocumentTemplate] {
	if s.templateRepo == nil {
		return fp.Failure[domain.DocumentTemplate](errors.New("template repository not configured"))
	}
	return s.templateRepo.FindByID(ctx, tenantID, id)
}

// FindAllTemplates retrieves all templates
func (s *DocumentService) FindAllTemplates(ctx context.Context, tenantID string, activeOnly bool) fp.Result[[]domain.DocumentTemplate] {
	if s.templateRepo == nil {
		return fp.Failure[[]domain.DocumentTemplate](errors.New("template repository not configured"))
	}
	return s.templateRepo.FindAll(ctx, tenantID, activeOnly)
}

func (s *DocumentService) createAudit(ctx context.Context, tenantID, entityType, entityID string, action domain.AuditAction, oldVal, newVal interface{}, userID domain.UserID, userName string) {
	if userName == "" {
		userName = "system"
	}
	entry := domain.AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Category:   domain.AuditCategoryDocument,
		UserID:     userID,
		UserName:   userName,
		Timestamp:  time.Now(),
	}

	if oldVal != nil {
		entry.OldValues = map[string]interface{}{"data": oldVal}
	}
	if newVal != nil {
		entry.NewValues = map[string]interface{}{"data": newVal}
	}

	result := s.auditRepo.Create(ctx, entry)
	if fp.IsFailure(result) {
		log.Printf("audit create failed for %s/%s: %v", entityType, entityID, fp.GetError(result))
	}
}
