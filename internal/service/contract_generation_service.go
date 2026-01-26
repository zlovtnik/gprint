package service

import (
	"context"
	"errors"

	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/repository"
)

// ErrUnauthorized is returned when authorization fails
var ErrUnauthorized = errors.New("unauthorized")

// ContractGenerationService handles contract generation business logic
// Delegates all sensitive data processing to the repository/database layer
type ContractGenerationService struct {
	repo *repository.ContractGenerationRepository
}

// NewContractGenerationService creates a new ContractGenerationService
func NewContractGenerationService(repo *repository.ContractGenerationRepository) *ContractGenerationService {
	return &ContractGenerationService{repo: repo}
}

// GenerateContract generates a printable contract document
// All sensitive data processing happens in the database layer
func (s *ContractGenerationService) GenerateContract(
	ctx context.Context,
	tenantID string,
	contractID int64,
	userID string,
	req *models.GenerateContractRequest,
	ipAddress string,
	sessionID string,
) (*models.GenerateContractResponse, error) {
	templateCode := ""
	reason := string(models.GenerationReasonInitial)

	if req != nil {
		if req.TemplateCode != "" {
			templateCode = req.TemplateCode
		}
		if req.Reason != "" {
			reason = string(req.Reason)
		}
	}

	return s.repo.GenerateContract(ctx, repository.GenerateContractParams{
		TenantID:     tenantID,
		ContractID:   contractID,
		UserID:       userID,
		TemplateCode: templateCode,
		Reason:       reason,
		IPAddress:    ipAddress,
		SessionID:    sessionID,
	})
}

// GetGeneratedContent retrieves the JSON content of a generated contract
func (s *ContractGenerationService) GetGeneratedContent(
	ctx context.Context,
	tenantID string,
	generatedID int64,
	userID string,
) (*models.GetGeneratedContentResponse, error) {
	return s.repo.GetGeneratedContent(ctx, tenantID, generatedID, userID)
}

// GetLatestGenerated retrieves the most recent generated version for a contract
func (s *ContractGenerationService) GetLatestGenerated(
	ctx context.Context,
	tenantID string,
	contractID int64,
	userID string,
) (*models.GetGeneratedContentResponse, error) {
	return s.repo.GetLatestGenerated(ctx, tenantID, contractID, userID)
}

// ListGeneratedContracts lists all generated versions for a contract
func (s *ContractGenerationService) ListGeneratedContracts(
	ctx context.Context,
	tenantID string,
	contractID int64,
	page int,
	pageSize int,
) ([]models.GeneratedContractListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	return s.repo.ListGeneratedContracts(ctx, tenantID, contractID, pageSize, offset)
}

// LogDownloadAction logs when a user downloads a generated contract
func (s *ContractGenerationService) LogDownloadAction(
	ctx context.Context,
	tenantID string,
	contractID int64,
	userID string,
	ipAddress string,
	sessionID string,
) error {
	return s.repo.LogContractAction(ctx, repository.LogActionParams{
		TenantID:   tenantID,
		ContractID: contractID,
		Action:     "DOWNLOAD",
		UserID:     userID,
		IPAddress:  ipAddress,
		SessionID:  sessionID,
		Status:     "SUCCESS",
	})
}

// LogPrintAction logs when a user prints a generated contract
func (s *ContractGenerationService) LogPrintAction(
	ctx context.Context,
	tenantID string,
	contractID int64,
	generatedID int64,
	userID string,
	ipAddress string,
	sessionID string,
) error {
	return s.repo.LogContractAction(ctx, repository.LogActionParams{
		TenantID:    tenantID,
		ContractID:  contractID,
		GeneratedID: generatedID,
		Action:      "PRINT",
		UserID:      userID,
		IPAddress:   ipAddress,
		SessionID:   sessionID,
		Status:      "SUCCESS",
	})
}

// GetGenerationStats retrieves generation statistics for a tenant
func (s *ContractGenerationService) GetGenerationStats(
	ctx context.Context,
	tenantID string,
) (*models.GenerationStats, error) {
	return s.repo.GetGenerationStats(ctx, tenantID)
}

// VerifyContentIntegrity verifies that a generated contract hasn't been tampered with
// Enforces tenant isolation by checking authorization first
// Returns:
//   - (true, nil) if content is valid (hash matches)
//   - (false, nil) if content is tampered (hash mismatch)
//   - (false, ErrUnauthorized) if tenant doesn't own this record or record doesn't exist for tenant
//   - (false, ErrNotFound) if record was deleted between authorization check and verification (TOCTOU race)
func (s *ContractGenerationService) VerifyContentIntegrity(
	ctx context.Context,
	tenantID string,
	generatedID int64,
) (bool, error) {
	isValid, err := s.repo.VerifyContentIntegrity(ctx, tenantID, generatedID)
	if err != nil {
		if errors.Is(err, repository.ErrUnauthorized) {
			return false, ErrUnauthorized
		}
		if errors.Is(err, repository.ErrNotFound) {
			return false, ErrNotFound
		}
		return false, err
	}
	return isValid, nil
}

// ListTemplates lists all active templates for a tenant
func (s *ContractGenerationService) ListTemplates(
	ctx context.Context,
	tenantID string,
) ([]models.ContractTemplate, error) {
	return s.repo.ListTemplates(ctx, tenantID)
}

// InitTenantTemplate ensures a tenant has a default template
func (s *ContractGenerationService) InitTenantTemplate(
	ctx context.Context,
	tenantID string,
	userID string,
) error {
	return s.repo.InitTenantTemplate(ctx, tenantID, userID)
}

// CleanupExpiredGenerations removes expired generated contracts
// For scheduled/maintenance tasks
func (s *ContractGenerationService) CleanupExpiredGenerations(
	ctx context.Context,
	tenantID string,
) (int, error) {
	return s.repo.CleanupExpiredGenerations(ctx, tenantID)
}
