package service

import (
	"context"
	"fmt"
	"log"

	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/repository"
)

// ContractService handles contract business logic
type ContractService struct {
	contractRepo *repository.ContractRepository
	historyRepo  *repository.HistoryRepository
}

// NewContractService creates a new ContractService
func NewContractService(contractRepo *repository.ContractRepository, historyRepo *repository.HistoryRepository) *ContractService {
	return &ContractService{
		contractRepo: contractRepo,
		historyRepo:  historyRepo,
	}
}

// Create creates a new contract
func (s *ContractService) Create(ctx context.Context, tenantID string, req *models.CreateContractRequest, createdBy string) (*models.Contract, error) {
	contract, err := s.contractRepo.Create(ctx, tenantID, req, createdBy)
	if err != nil {
		return nil, err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:  contract.ID,
		Action:      models.HistoryActionCreate,
		PerformedBy: createdBy,
	}); err != nil {
		// Log but don't fail the operation
		log.Printf("failed to record contract creation history (tenant=%s, contractID=%d, performedBy=%s): %v", tenantID, contract.ID, createdBy, err)
	}

	return contract, nil
}

// GetByID retrieves a contract by ID
func (s *ContractService) GetByID(ctx context.Context, tenantID string, id int64) (*models.Contract, error) {
	return s.contractRepo.GetByID(ctx, tenantID, id)
}

// List retrieves contracts with pagination
func (s *ContractService) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Contract, int, error) {
	return s.contractRepo.List(ctx, tenantID, params, search)
}

// Update updates a contract
func (s *ContractService) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateContractRequest, updatedBy string) (*models.Contract, error) {
	existing, err := s.contractRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrContractNotFound
	}

	// Only allow updates on DRAFT or PENDING contracts
	if existing.Status != models.ContractStatusDraft && existing.Status != models.ContractStatusPending {
		return nil, fmt.Errorf("%w: cannot update contract in %s status", ErrContractCannotUpdate, existing.Status)
	}

	contract, err := s.contractRepo.Update(ctx, tenantID, id, req, updatedBy)
	if err != nil {
		return nil, err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:  id,
		Action:      models.HistoryActionUpdate,
		PerformedBy: updatedBy,
	}); err != nil {
		log.Printf("failed to record contract update history (tenant=%s, contractID=%d, performedBy=%s): %v", tenantID, id, updatedBy, err)
	}

	return contract, nil
}

// UpdateStatus updates the contract status
func (s *ContractService) UpdateStatus(ctx context.Context, tenantID string, id int64, newStatus models.ContractStatus, updatedBy, ipAddress string) error {
	existing, err := s.contractRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrContractNotFound
	}

	// Validate status transition
	if !isValidStatusTransition(existing.Status, newStatus) {
		return fmt.Errorf("%w: from %s to %s", ErrInvalidStatusTransition, existing.Status, newStatus)
	}

	if err := s.contractRepo.UpdateStatus(ctx, tenantID, id, newStatus, updatedBy); err != nil {
		return err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:   id,
		Action:       models.HistoryActionStatusChange,
		FieldChanged: "status",
		OldValue:     string(existing.Status),
		NewValue:     string(newStatus),
		PerformedBy:  updatedBy,
		IPAddress:    ipAddress,
	}); err != nil {
		log.Printf("failed to record contract status change history (tenant=%s, contractID=%d, action=STATUS_CHANGE, performedBy=%s): %v", tenantID, id, updatedBy, err)
	}

	return nil
}

// Sign signs the contract
func (s *ContractService) Sign(ctx context.Context, tenantID string, id int64, signedBy, ipAddress string) error {
	existing, err := s.contractRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrContractNotFound
	}

	// Only allow signing of PENDING contracts
	if existing.Status != models.ContractStatusPending {
		return fmt.Errorf("%w: can only sign contracts in PENDING status, current status: %s", ErrCannotSign, existing.Status)
	}

	if err := s.contractRepo.Sign(ctx, tenantID, id, signedBy); err != nil {
		return err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:  id,
		Action:      models.HistoryActionSign,
		PerformedBy: signedBy,
		IPAddress:   ipAddress,
	}); err != nil {
		log.Printf("failed to record contract sign history (tenant=%s, contractID=%d, action=SIGN, performedBy=%s): %v", tenantID, id, signedBy, err)
	}

	return nil
}

// GetHistory retrieves contract history
func (s *ContractService) GetHistory(ctx context.Context, tenantID string, contractID int64, params models.PaginationParams) ([]models.ContractHistory, int, error) {
	return s.historyRepo.GetByContractID(ctx, tenantID, contractID, params)
}

// AddItem adds an item to a contract
func (s *ContractService) AddItem(ctx context.Context, tenantID string, contractID int64, req *models.CreateContractItemRequest, createdBy string) (*models.ContractItem, error) {
	existing, err := s.contractRepo.GetByID(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrContractNotFound
	}

	// Only allow adding items to DRAFT contracts
	if existing.Status != models.ContractStatusDraft {
		return nil, fmt.Errorf("%w: can only add items to contracts in DRAFT status", ErrCannotAddItem)
	}

	item, err := s.contractRepo.AddItem(ctx, tenantID, contractID, req)
	if err != nil {
		return nil, err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:   contractID,
		Action:       models.HistoryActionUpdate,
		FieldChanged: "items",
		NewValue:     fmt.Sprintf("Added item with service_id=%d", req.ServiceID),
		PerformedBy:  createdBy,
	}); err != nil {
		log.Printf("failed to record contract add item history (tenant=%s, contractID=%d, itemServiceID=%d, performedBy=%s): %v", tenantID, contractID, req.ServiceID, createdBy, err)
	}

	return item, nil
}

// DeleteItem removes an item from a contract
func (s *ContractService) DeleteItem(ctx context.Context, tenantID string, contractID, itemID int64, deletedBy string) error {
	existing, err := s.contractRepo.GetByID(ctx, tenantID, contractID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrContractNotFound
	}

	// Only allow deleting items from DRAFT contracts
	if existing.Status != models.ContractStatusDraft {
		return fmt.Errorf("%w: can only delete items from contracts in DRAFT status", ErrCannotDeleteItem)
	}

	if err := s.contractRepo.DeleteItem(ctx, tenantID, contractID, itemID); err != nil {
		return err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:   contractID,
		Action:       models.HistoryActionUpdate,
		FieldChanged: "items",
		NewValue:     fmt.Sprintf("Removed item_id=%d", itemID),
		PerformedBy:  deletedBy,
	}); err != nil {
		log.Printf("failed to record contract delete item history (tenant=%s, contractID=%d, itemID=%d, performedBy=%s): %v", tenantID, contractID, itemID, deletedBy, err)
	}

	return nil
}

// isValidStatusTransition checks if a status transition is valid
func isValidStatusTransition(from, to models.ContractStatus) bool {
	validTransitions := map[models.ContractStatus][]models.ContractStatus{
		models.ContractStatusDraft:     {models.ContractStatusPending, models.ContractStatusCancelled},
		models.ContractStatusPending:   {models.ContractStatusActive, models.ContractStatusCancelled, models.ContractStatusDraft},
		models.ContractStatusActive:    {models.ContractStatusSuspended, models.ContractStatusCompleted, models.ContractStatusCancelled},
		models.ContractStatusSuspended: {models.ContractStatusActive, models.ContractStatusCancelled},
		models.ContractStatusCompleted: {},
		models.ContractStatusCancelled: {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}

	return false
}
