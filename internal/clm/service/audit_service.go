package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// AuditService handles audit trail business logic
type AuditService struct {
	repo *repository.AuditRepository
}

// NewAuditService creates a new AuditService
func NewAuditService(repo *repository.AuditRepository) *AuditService {
	if repo == nil {
		panic("audit repository is required")
	}
	return &AuditService{repo: repo}
}

// CreateAuditRequest represents a request to create an audit entry
type CreateAuditRequest struct {
	EntityType string
	EntityID   string
	Action     domain.AuditAction
	Category   domain.AuditCategory
	UserID     domain.UserID
	UserName   string
	UserRole   string
	IPAddress  string
	UserAgent  string
	OldValues  map[string]interface{}
	NewValues  map[string]interface{}
	Metadata   map[string]interface{}
}

// Create creates a new audit entry
func (s *AuditService) Create(ctx context.Context, tenantID string, req CreateAuditRequest) fp.Result[domain.AuditEntry] {
	entry := domain.AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Action:     req.Action,
		Category:   req.Category,
		UserID:     req.UserID,
		UserName:   req.UserName,
		UserRole:   req.UserRole,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		OldValues:  req.OldValues,
		NewValues:  req.NewValues,
		Metadata:   req.Metadata,
		Timestamp:  time.Now(),
	}

	return s.repo.Create(ctx, entry)
}

// FindByEntity retrieves audit entries for an entity
func (s *AuditService) FindByEntity(ctx context.Context, tenantID, entityType, entityID string, offset, limit int) fp.Result[[]domain.AuditEntry] {
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 500 {
		limit = 500 // Max limit
	}
	return s.repo.FindByEntity(ctx, tenantID, entityType, entityID, offset, limit)
}

// FindByUser retrieves audit entries for a user
func (s *AuditService) FindByUser(ctx context.Context, tenantID string, userID domain.UserID, offset, limit int) fp.Result[[]domain.AuditEntry] {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	return s.repo.FindByUser(ctx, tenantID, userID, offset, limit)
}

// LogCreate logs a create action
func (s *AuditService) LogCreate(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName string, newEntity map[string]interface{}) fp.Result[domain.AuditEntry] {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionCreated,
		Category:   domain.AuditCategoryContract,
		UserID:     userID,
		UserName:   userName,
		NewValues:  newEntity,
	}
	return s.Create(ctx, tenantID, req)
}

// LogUpdate logs an update action
func (s *AuditService) LogUpdate(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName string, oldEntity, newEntity map[string]interface{}) fp.Result[domain.AuditEntry] {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionUpdated,
		Category:   domain.AuditCategoryContract,
		UserID:     userID,
		UserName:   userName,
		OldValues:  oldEntity,
		NewValues:  newEntity,
	}
	return s.Create(ctx, tenantID, req)
}

// LogDelete logs a delete action
func (s *AuditService) LogDelete(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName string, oldEntity map[string]interface{}) fp.Result[domain.AuditEntry] {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionDeleted,
		Category:   domain.AuditCategoryContract,
		UserID:     userID,
		UserName:   userName,
		OldValues:  oldEntity,
	}
	return s.Create(ctx, tenantID, req)
}

// LogStatusChange logs a status change action
func (s *AuditService) LogStatusChange(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName string, oldStatus, newStatus string) fp.Result[domain.AuditEntry] {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionUpdated,
		Category:   domain.AuditCategoryContract,
		UserID:     userID,
		UserName:   userName,
		OldValues:  map[string]interface{}{"status": oldStatus},
		NewValues:  map[string]interface{}{"status": newStatus},
	}
	return s.Create(ctx, tenantID, req)
}

// LogApproval logs an approval action
func (s *AuditService) LogApproval(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName, comment string) {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionApproved,
		Category:   domain.AuditCategoryWorkflow,
		UserID:     userID,
		UserName:   userName,
		Metadata:   map[string]interface{}{"comment": comment},
	}
	s.Create(ctx, tenantID, req)
}

// LogRejection logs a rejection action
func (s *AuditService) LogRejection(ctx context.Context, tenantID, entityType, entityID string, userID domain.UserID, userName, reason string) {
	req := CreateAuditRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     domain.AuditActionRejected,
		Category:   domain.AuditCategoryWorkflow,
		UserID:     userID,
		UserName:   userName,
		Metadata:   map[string]interface{}{"reason": reason},
	}
	s.Create(ctx, tenantID, req)
}
