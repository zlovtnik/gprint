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

// PartyService handles party business logic
type PartyService struct {
	repo      *repository.PartyRepository
	auditRepo *repository.AuditRepository
}

// NewPartyService creates a new PartyService
func NewPartyService(repo *repository.PartyRepository, auditRepo *repository.AuditRepository) *PartyService {
	if repo == nil {
		panic("party repository is required")
	}
	return &PartyService{repo: repo, auditRepo: auditRepo}
}

// CreatePartyRequest represents a request to create a party
type CreatePartyRequest struct {
	Type           domain.PartyType
	Name           string
	LegalName      string
	Email          string
	Phone          string
	TaxID          string
	Address        *domain.Address
	BillingAddress *domain.Address
	RiskLevel      domain.RiskLevel
	RiskScore      int
	CreatedBy      domain.UserID
}

// Create creates a new party
func (s *PartyService) Create(ctx context.Context, tenantID string, req CreatePartyRequest) fp.Result[domain.Party] {
	// Validate required fields
	if req.Name == "" {
		return fp.Failure[domain.Party](errors.New("name is required"))
	}
	if req.Type == "" {
		return fp.Failure[domain.Party](errors.New("type is required"))
	}
	// Validate party type is a known value
	switch req.Type {
	case domain.PartyTypeIndividual, domain.PartyTypeCorporation, domain.PartyTypeGovernment, domain.PartyTypeNonProfit, domain.PartyTypePartnership:
		// Valid type
	default:
		return fp.Failure[domain.Party](errors.New("invalid party type"))
	}

	// Create party with generated ID
	id := domain.PartyID(uuid.New())
	now := time.Now()

	party := domain.Party{
		ID:        id,
		TenantID:  tenantID,
		Type:      req.Type,
		Name:      req.Name,
		LegalName: req.LegalName,
		Email:     req.Email,
		Phone:     req.Phone,
		TaxID:     req.TaxID,
		IsActive:  true,
		CreatedAt: now,
		CreatedBy: req.CreatedBy,
	}

	if req.Address != nil {
		party = party.WithAddress(*req.Address)
	}
	if req.BillingAddress != nil {
		party = party.WithBillingAddress(req.BillingAddress)
	}
	if req.RiskLevel != "" {
		party = party.WithRiskAssessment(req.RiskLevel, req.RiskScore)
	}

	// Persist
	result := s.repo.Create(ctx, party)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "party", uuid.UUID(id).String(), domain.AuditActionCreate, nil, &party, req.CreatedBy, "")
	}

	return result
}

// FindByID retrieves a party by ID
func (s *PartyService) FindByID(ctx context.Context, tenantID string, id domain.PartyID) fp.Result[domain.Party] {
	return s.repo.FindByID(ctx, tenantID, id)
}

// FindAll retrieves all parties with pagination
func (s *PartyService) FindAll(ctx context.Context, tenantID string, offset, limit int) fp.Result[[]domain.Party] {
	return s.repo.FindAll(ctx, tenantID, offset, limit)
}

// Count returns the total count of parties
func (s *PartyService) Count(ctx context.Context, tenantID string) fp.Result[int] {
	return s.repo.Count(ctx, tenantID)
}

// UpdatePartyRequest represents a request to update a party
type UpdatePartyRequest struct {
	Name           *string
	LegalName      *string
	Email          *string
	Phone          *string
	TaxID          *string
	Address        *domain.Address
	BillingAddress *domain.Address
	RiskLevel      *domain.RiskLevel
	RiskScore      *int
	UpdatedBy      domain.UserID
}

// Update updates an existing party
func (s *PartyService) Update(ctx context.Context, tenantID string, id domain.PartyID, req UpdatePartyRequest) fp.Result[domain.Party] {
	// Fetch existing
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Apply updates (immutable pattern)
	updated := existing
	if req.Name != nil {
		updated = updated.WithName(*req.Name)
	}
	if req.LegalName != nil {
		updated = updated.WithLegalName(*req.LegalName)
	}
	if req.Email != nil {
		updated = updated.WithEmail(*req.Email)
	}
	if req.Phone != nil {
		updated = updated.WithPhone(*req.Phone)
	}
	if req.Address != nil {
		updated = updated.WithAddress(*req.Address)
	}
	if req.BillingAddress != nil {
		updated = updated.WithBillingAddress(req.BillingAddress)
	}
	if req.RiskLevel != nil {
		score := updated.RiskScore
		if req.RiskScore != nil {
			score = *req.RiskScore
		}
		updated = updated.WithRiskAssessment(*req.RiskLevel, score)
	} else if req.RiskScore != nil {
		// Handle RiskScore-only update - use existing risk level
		updated = updated.WithRiskAssessment(updated.RiskLevel, *req.RiskScore)
	}

	// Update timestamp
	now := time.Now()
	updated.UpdatedAt = &now
	updated.UpdatedBy = &req.UpdatedBy

	// Persist
	result := s.repo.Update(ctx, updated)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "party", uuid.UUID(id).String(), domain.AuditActionUpdate, &existing, &updated, req.UpdatedBy, "")
	}

	return result
}

// Deactivate deactivates a party
func (s *PartyService) Deactivate(ctx context.Context, tenantID string, id domain.PartyID, deactivatedBy domain.UserID) fp.Result[bool] {
	result := s.repo.Deactivate(ctx, tenantID, id, deactivatedBy)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "party", uuid.UUID(id).String(), domain.AuditActionDeactivate, nil, nil, deactivatedBy, "")
	}

	return result
}

func (s *PartyService) createAudit(ctx context.Context, tenantID, entityType, entityID string, action domain.AuditAction, oldVal, newVal interface{}, userID domain.UserID, userName string) {
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
		log.Printf("audit create failed for partyID=%s: %v", entityID, fp.GetError(result))
	}
}
