package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ContractService handles contract business logic
type ContractService struct {
	repo      *repository.ContractRepository
	partyRepo *repository.PartyRepository
	auditRepo *repository.AuditRepository
}

// NewContractService creates a new ContractService
func NewContractService(repo *repository.ContractRepository, partyRepo *repository.PartyRepository, auditRepo *repository.AuditRepository) *ContractService {
	if repo == nil {
		panic("contract repository is required")
	}
	return &ContractService{repo: repo, partyRepo: partyRepo, auditRepo: auditRepo}
}

// CreateContractRequest represents a request to create a contract
type CreateContractRequest struct {
	ContractNumber   string
	ExternalRef      string
	Title            string
	Description      string
	ContractType     domain.ContractType
	ValueAmount      decimal.Decimal
	ValueCurrency    string
	EffectiveDate    *time.Time
	ExpirationDate   *time.Time
	AutoRenew        bool
	RenewalTermDays  int
	NoticePeriodDays int
	Terms            string
	Notes            string
	Parties          []CreateContractPartyRequest
	CreatedBy        domain.UserID
	CreatedByName    string
}

// CreateContractPartyRequest represents a party in a contract creation
type CreateContractPartyRequest struct {
	PartyID   domain.PartyID
	Role      domain.PartyRole
	SignedAt  *time.Time
	SignedBy  *domain.UserID
	IsPrimary bool
}

// Create creates a new contract
func (s *ContractService) Create(ctx context.Context, tenantID string, req CreateContractRequest) fp.Result[domain.Contract] {
	// Validate required fields
	if err := validateCreateContractRequest(req); err != nil {
		return fp.Failure[domain.Contract](err)
	}

	// Create contract with generated ID
	id := domain.ContractID(uuid.New())
	now := time.Now()

	contract := domain.Contract{
		ID:             id,
		TenantID:       tenantID,
		ContractNumber: req.ContractNumber,
		ExternalRef:    req.ExternalRef,
		Title:          req.Title,
		Description:    req.Description,
		ContractType:   req.ContractType,
		Status:         domain.ContractStatusDraft,
		Version:        1,
		Value: domain.Money{
			Amount:   req.ValueAmount,
			Currency: req.ValueCurrency,
		},
		EffectiveDate:    req.EffectiveDate,
		ExpirationDate:   req.ExpirationDate,
		AutoRenew:        req.AutoRenew,
		RenewalTermDays:  req.RenewalTermDays,
		NoticePeriodDays: req.NoticePeriodDays,
		Terms:            req.Terms,
		Notes:            req.Notes,
		IsDeleted:        false,
		CreatedAt:        now,
		CreatedBy:        req.CreatedBy,
	}

	// Add parties
	for _, p := range req.Parties {
		party := domain.ContractParty{
			PartyID:   p.PartyID,
			Role:      p.Role,
			SignedAt:  p.SignedAt,
			SignedBy:  p.SignedBy,
			IsPrimary: p.IsPrimary,
		}
		contract.Parties = append(contract.Parties, party)
	}

	// Persist
	result := s.repo.Create(ctx, contract)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionCreate, nil, &contract, req.CreatedBy, req.CreatedByName)
	}

	return result
}

// FindByID retrieves a contract by ID
func (s *ContractService) FindByID(ctx context.Context, tenantID string, id domain.ContractID) fp.Result[domain.Contract] {
	return s.repo.FindByID(ctx, tenantID, id)
}

// FindAll retrieves contracts with optional filtering
func (s *ContractService) FindAll(ctx context.Context, tenantID string, filter repository.ContractFilter, offset, limit int) fp.Result[[]domain.Contract] {
	return s.repo.FindAll(ctx, tenantID, filter, offset, limit)
}

// Count returns the total count of contracts
func (s *ContractService) Count(ctx context.Context, tenantID string, filter repository.ContractFilter) fp.Result[int] {
	return s.repo.Count(ctx, tenantID, filter)
}

// UpdateContractRequest represents a request to update a contract
type UpdateContractRequest struct {
	Title            *string
	Description      *string
	ValueAmount      *decimal.Decimal
	ValueCurrency    *string
	EffectiveDate    *time.Time
	ExpirationDate   *time.Time
	AutoRenew        *bool
	RenewalTermDays  *int
	NoticePeriodDays *int
	Terms            *string
	Notes            *string
	UpdatedBy        domain.UserID
}

// Update updates an existing contract
func (s *ContractService) Update(ctx context.Context, tenantID string, id domain.ContractID, req UpdateContractRequest) fp.Result[domain.Contract] {
	// Fetch existing
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Cannot update non-draft contracts
	if existing.Status != domain.ContractStatusDraft {
		return fp.Failure[domain.Contract](errors.New("can only update draft contracts"))
	}

	// Apply updates (immutable pattern)
	updated := existing
	if req.Title != nil {
		updated = updated.WithTitle(*req.Title)
	}
	if req.Description != nil {
		updated = updated.WithDescription(*req.Description)
	}
	if req.ValueAmount != nil || req.ValueCurrency != nil {
		amount := updated.Value.Amount
		currency := updated.Value.Currency
		if req.ValueAmount != nil {
			amount = *req.ValueAmount
		}
		if req.ValueCurrency != nil {
			currency = *req.ValueCurrency
		}
		updated = updated.WithValue(domain.Money{Amount: amount, Currency: currency})
	}
	if req.EffectiveDate != nil || req.ExpirationDate != nil {
		eff := updated.EffectiveDate
		exp := updated.ExpirationDate
		if req.EffectiveDate != nil {
			eff = req.EffectiveDate
		}
		if req.ExpirationDate != nil {
			exp = req.ExpirationDate
		}
		updated = updated.WithDates(eff, exp)
	}
	if req.AutoRenew != nil || req.RenewalTermDays != nil || req.NoticePeriodDays != nil {
		autoRenew := updated.AutoRenew
		termDays := updated.RenewalTermDays
		noticeDays := updated.NoticePeriodDays
		if req.AutoRenew != nil {
			autoRenew = *req.AutoRenew
		}
		if req.RenewalTermDays != nil {
			termDays = *req.RenewalTermDays
		}
		if req.NoticePeriodDays != nil {
			noticeDays = *req.NoticePeriodDays
		}
		updated = updated.WithRenewalTerms(autoRenew, termDays, noticeDays)
	}
	if req.Terms != nil {
		updated = updated.WithTerms(*req.Terms)
	}
	if req.Notes != nil {
		updated = updated.WithNotes(*req.Notes)
	}

	now := time.Now()
	updated.UpdatedAt = &now
	updated.UpdatedBy = &req.UpdatedBy

	// Persist
	result := s.repo.Update(ctx, updated)

	// Audit if successful
	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionUpdate, &existing, &updated, req.UpdatedBy, "")
	}

	return result
}

// TransitionStatus transitions a contract to a new status
func (s *ContractService) TransitionStatus(ctx context.Context, tenantID string, id domain.ContractID, newStatus domain.ContractStatus, updatedBy domain.UserID) fp.Result[domain.Contract] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	// Attempt transition
	now := time.Now()
	updated, err := existing.TransitionTo(newStatus, updatedBy, now)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	// Persist the full updated contract
	result := s.repo.Update(ctx, updated)
	if fp.IsFailure(result) {
		return result
	}

	// Audit
	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionStatusChange, &existing, &updated, updatedBy, "")
	}

	return result
}

// Submit submits a contract for review
func (s *ContractService) Submit(ctx context.Context, tenantID string, id domain.ContractID, updatedBy domain.UserID) fp.Result[domain.Contract] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	updated, err := existing.Submit(updatedBy)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	result := s.repo.Update(ctx, updated)
	if fp.IsFailure(result) {
		return result
	}

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionStatusChange, &existing, &updated, updatedBy, "")
	}

	return result
}

// Approve approves a contract
func (s *ContractService) Approve(ctx context.Context, tenantID string, id domain.ContractID, updatedBy domain.UserID) fp.Result[domain.Contract] {
	existingResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(existingResult) {
		return existingResult
	}
	existing := fp.GetValue(existingResult)

	updated, err := existing.Approve(updatedBy)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	result := s.repo.Update(ctx, updated)
	if fp.IsFailure(result) {
		return result
	}

	if s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionApprove, &existing, &updated, updatedBy, "")
	}

	return result
}

// FindExpiring retrieves contracts expiring within the given number of days
func (s *ContractService) FindExpiring(ctx context.Context, tenantID string, days int) fp.Result[[]domain.Contract] {
	return s.repo.FindExpiring(ctx, tenantID, days)
}

// SoftDelete soft-deletes a contract
func (s *ContractService) SoftDelete(ctx context.Context, tenantID string, id domain.ContractID, deletedBy domain.UserID) fp.Result[bool] {
	result := s.repo.SoftDelete(ctx, tenantID, id, deletedBy)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "contract", uuid.UUID(id).String(), domain.AuditActionDelete, nil, nil, deletedBy, "")
	}

	return result
}

// validateCreateContractRequest validates contract creation request
func validateCreateContractRequest(req CreateContractRequest) error {
	if req.Title == "" {
		return errors.New("title is required")
	}
	// Validate ContractType has required fields
	if req.ContractType.Name == "" && req.ContractType.Code == "" {
		return errors.New("contract type is required")
	}
	// Validate currency is provided if amount is set
	if !req.ValueAmount.IsZero() && req.ValueCurrency == "" {
		return errors.New("currency is required when value amount is set")
	}
	// Validate ISO-4217 currency code (3 uppercase letters)
	if req.ValueCurrency != "" {
		if matched, _ := regexp.MatchString(`^[A-Z]{3}$`, req.ValueCurrency); !matched {
			return fmt.Errorf("invalid currency code: %s (must be 3-letter ISO-4217 code)", req.ValueCurrency)
		}
	}
	// Validate effective date is before expiration date
	if req.EffectiveDate != nil && req.ExpirationDate != nil {
		if req.ExpirationDate.Before(*req.EffectiveDate) {
			return errors.New("expiration date must be after effective date")
		}
	}
	// Validate parties
	if len(req.Parties) == 0 {
		return errors.New("at least one party is required")
	}
	for i, p := range req.Parties {
		if p.PartyID.IsZero() {
			return fmt.Errorf("party %d: party ID is required", i+1)
		}
		if p.Role == "" {
			return fmt.Errorf("party %d: role is required", i+1)
		}
	}
	return nil
}

func (s *ContractService) createAudit(ctx context.Context, tenantID, entityType, entityID string, action domain.AuditAction, oldVal, newVal interface{}, userID domain.UserID, userName string) {
	if userName == "" {
		userName = "system"
	}
	entry := domain.AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Category:   domain.AuditCategoryContract,
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
