package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ObligationService handles obligation business logic
type ObligationService struct {
	repo      *repository.ObligationRepository
	auditRepo *repository.AuditRepository
}

// NewObligationService creates a new ObligationService
func NewObligationService(repo *repository.ObligationRepository, auditRepo *repository.AuditRepository) *ObligationService {
	if repo == nil {
		panic("obligation repository is required")
	}
	return &ObligationService{repo: repo, auditRepo: auditRepo}
}

// CreateObligationRequest represents a request to create an obligation
type CreateObligationRequest struct {
	ContractID       domain.ContractID
	Type             domain.ObligationType
	Title            string
	Description      string
	ResponsibleParty domain.PartyID
	DueDate          time.Time
	Amount           *decimal.Decimal
	Currency         string
	Frequency        domain.ObligationFrequency
	ReminderDays     int
	Notes            string
	CreatedBy        domain.UserID
}

// Create creates a new obligation
func (s *ObligationService) Create(ctx context.Context, tenantID string, req CreateObligationRequest) fp.Result[domain.Obligation] {
	// Validate required fields
	if req.Title == "" {
		return fp.Failure[domain.Obligation](errors.New("title is required"))
	}
	if req.ContractID.IsZero() {
		return fp.Failure[domain.Obligation](errors.New("contract is required"))
	}

	// Create obligation with generated ID
	id := domain.ObligationID(uuid.New())
	now := time.Now()

	obligation := domain.Obligation{
		ID:               id,
		TenantID:         tenantID,
		ContractID:       req.ContractID,
		Type:             req.Type,
		Title:            req.Title,
		Description:      req.Description,
		ResponsibleParty: req.ResponsibleParty,
		Status:           domain.ObligationStatusPending,
		DueDate:          req.DueDate,
		Frequency:        req.Frequency,
		ReminderDays:     req.ReminderDays,
		Notes:            req.Notes,
		CreatedAt:        now,
		CreatedBy:        req.CreatedBy,
	}

	if req.Amount != nil {
		obligation.Amount = &domain.Money{
			Amount:   *req.Amount,
			Currency: req.Currency,
		}
	}

	// Persist
	result := s.repo.Create(ctx, obligation)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "obligation", uuid.UUID(id).String(), domain.AuditActionCreate, nil, &obligation, req.CreatedBy, "")
	}

	return result
}

// FindByID retrieves an obligation by ID
func (s *ObligationService) FindByID(ctx context.Context, tenantID string, id domain.ObligationID) fp.Result[domain.Obligation] {
	return s.repo.FindByID(ctx, tenantID, id)
}

// FindByContract retrieves all obligations for a contract
func (s *ObligationService) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID, offset, limit int) fp.Result[[]domain.Obligation] {
	return s.repo.FindByContract(ctx, tenantID, contractID, offset, limit)
}

// CountByContract returns the count of obligations for a contract
func (s *ObligationService) CountByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[int] {
	return s.repo.CountByContract(ctx, tenantID, contractID)
}

// FindOverdue retrieves overdue obligations
func (s *ObligationService) FindOverdue(ctx context.Context, tenantID string, offset, limit int) fp.Result[[]domain.Obligation] {
	return s.repo.FindOverdue(ctx, tenantID, offset, limit)
}

// UpdateObligationRequest represents a request to update an obligation
type UpdateObligationRequest struct {
	Title       *string
	Description *string
	DueDate     *time.Time
	Amount      *decimal.Decimal
	Currency    *string
	Notes       *string
	UpdatedBy   domain.UserID
}

// Update updates an existing obligation
func (s *ObligationService) Update(ctx context.Context, tenantID string, id domain.ObligationID, req UpdateObligationRequest) fp.Result[domain.Obligation] {
	// Fetch existing
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Apply updates (immutable pattern)
	updated := existing
	if req.Title != nil {
		updated = updated.WithTitle(*req.Title, req.UpdatedBy)
	}
	if req.Description != nil {
		updated = updated.WithDescription(*req.Description, req.UpdatedBy)
	}
	if req.DueDate != nil {
		updated = updated.WithDueDate(*req.DueDate, req.UpdatedBy)
	}
	if req.Amount != nil {
		currency := ""
		if updated.Amount != nil {
			currency = updated.Amount.Currency
		}
		if req.Currency != nil {
			currency = *req.Currency
		}
		updated = updated.WithAmount(&domain.Money{Amount: *req.Amount, Currency: currency}, req.UpdatedBy)
	} else if req.Currency != nil && updated.Amount != nil {
		// Handle currency-only update - only if existing amount is present
		updated = updated.WithAmount(&domain.Money{Amount: updated.Amount.Amount, Currency: *req.Currency}, req.UpdatedBy)
	}
	if req.Notes != nil {
		updated = updated.WithNotes(*req.Notes, req.UpdatedBy)
	}

	// Persist
	result := s.repo.Update(ctx, updated)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "obligation", uuid.UUID(id).String(), domain.AuditActionUpdate, &existing, &updated, req.UpdatedBy, "")
	}

	return result
}

// MarkInProgress marks an obligation as in progress
func (s *ObligationService) MarkInProgress(ctx context.Context, tenantID string, id domain.ObligationID, updatedBy domain.UserID) fp.Result[domain.Obligation] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Validate transition is allowed
	if existing.Status != domain.ObligationStatusPending {
		return fp.Failure[domain.Obligation](errors.New("can only mark pending obligations as in progress"))
	}

	result := s.repo.UpdateStatus(ctx, tenantID, id, domain.ObligationStatusInProgress, updatedBy)
	if fp.IsFailure(result) {
		return fp.Failure[domain.Obligation](fp.GetError(result))
	}

	// Re-fetch to get persisted state
	updatedResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(updatedResult) {
		return updatedResult
	}
	updated := fp.GetValue(updatedResult)

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "obligation", uuid.UUID(id).String(), domain.AuditActionStatusChange, &existing, &updated, updatedBy, "")
	}

	return fp.Success(updated)
}

// Complete marks an obligation as completed
func (s *ObligationService) Complete(ctx context.Context, tenantID string, id domain.ObligationID, updatedBy domain.UserID) fp.Result[domain.Obligation] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Validate transition is allowed
	if existing.Status != domain.ObligationStatusPending && existing.Status != domain.ObligationStatusInProgress {
		return fp.Failure[domain.Obligation](errors.New("can only complete pending or in-progress obligations"))
	}

	result := s.repo.UpdateStatus(ctx, tenantID, id, domain.ObligationStatusCompleted, updatedBy)
	if fp.IsFailure(result) {
		return fp.Failure[domain.Obligation](fp.GetError(result))
	}

	// Re-fetch to get persisted state
	updatedResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(updatedResult) {
		return updatedResult
	}
	updated := fp.GetValue(updatedResult)

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "obligation", uuid.UUID(id).String(), domain.AuditActionStatusChange, &existing, &updated, updatedBy, "")
	}

	return fp.Success(updated)
}

// Waive marks an obligation as waived
func (s *ObligationService) Waive(ctx context.Context, tenantID string, id domain.ObligationID, updatedBy domain.UserID) fp.Result[domain.Obligation] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Validate transition is allowed
	if existing.Status != domain.ObligationStatusPending && existing.Status != domain.ObligationStatusInProgress {
		return fp.Failure[domain.Obligation](errors.New("can only waive pending or in-progress obligations"))
	}

	result := s.repo.UpdateStatus(ctx, tenantID, id, domain.ObligationStatusWaived, updatedBy)
	if fp.IsFailure(result) {
		return fp.Failure[domain.Obligation](fp.GetError(result))
	}

	// Re-fetch to get persisted state
	updatedResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(updatedResult) {
		return updatedResult
	}
	updated := fp.GetValue(updatedResult)

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "obligation", uuid.UUID(id).String(), domain.AuditActionStatusChange, &existing, &updated, updatedBy, "")
	}

	return fp.Success(updated)
}

// CountOverdue returns the count of overdue obligations
func (s *ObligationService) CountOverdue(ctx context.Context, tenantID string) fp.Result[int] {
	return s.repo.CountOverdue(ctx, tenantID)
}

func (s *ObligationService) createAudit(ctx context.Context, tenantID, entityType, entityID string, action domain.AuditAction, oldVal, newVal interface{}, userID domain.UserID, userName string) {
	if userName == "" {
		userName = "system"
	}
	entry := domain.AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Category:   domain.AuditCategoryData,
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
		log.Printf("audit create failed for %s/%s action=%s tenantID=%s: %v", entityType, entityID, action, tenantID, fp.GetError(result))
	}
}
