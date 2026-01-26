package models

import (
	"encoding/json"
	"time"
)

// ContractGenerationReason represents why a contract was generated
type ContractGenerationReason string

const (
	GenerationReasonInitial    ContractGenerationReason = "INITIAL"
	GenerationReasonUpdate     ContractGenerationReason = "UPDATE"
	GenerationReasonRenewal    ContractGenerationReason = "RENEWAL"
	GenerationReasonCorrection ContractGenerationReason = "CORRECTION"
)

// ContractGenerationAction represents contract generation actions
type ContractGenerationAction string

const (
	GenerationActionGenerate ContractGenerationAction = "GENERATE"
	GenerationActionView     ContractGenerationAction = "VIEW"
	GenerationActionDownload ContractGenerationAction = "DOWNLOAD"
	GenerationActionPrint    ContractGenerationAction = "PRINT"
)

// GeneratedContract represents a generated contract document
type GeneratedContract struct {
	ID                    int64                    `json:"id"`
	TenantID              string                   `json:"tenant_id"`
	ContractID            int64                    `json:"contract_id"`
	TemplateID            int64                    `json:"template_id"`
	GenerationNumber      int                      `json:"generation_number"`
	ContractJSON          json.RawMessage          `json:"contract_data,omitempty"` // Only returned when explicitly requested
	ContentHash           string                   `json:"content_hash"`
	CustomerNameSnapshot  string                   `json:"customer_name_snapshot"`
	TotalValueSnapshot    int64                    `json:"total_value_snapshot"` // Value in cents for precision
	ServicesCountSnapshot int                      `json:"services_count_snapshot"`
	GeneratedAt           time.Time                `json:"generated_at"`
	GeneratedBy           string                   `json:"generated_by"`
	GenerationReason      ContractGenerationReason `json:"generation_reason,omitempty"`
	ExpiresAt             *time.Time               `json:"expires_at,omitempty"`
}

// ContractTemplate represents a contract template
type ContractTemplate struct {
	ID           int64     `json:"id"`
	TenantID     string    `json:"tenant_id"`
	TemplateCode string    `json:"template_code"`
	TemplateName string    `json:"template_name"`
	Language     string    `json:"language"`
	IsDefault    bool      `json:"is_default"`
	Active       bool      `json:"active"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ContractGenerationLog represents an audit log entry
type ContractGenerationLog struct {
	ID            int64                    `json:"id"`
	TenantID      string                   `json:"tenant_id"`
	ContractID    int64                    `json:"contract_id"`
	Action        ContractGenerationAction `json:"action"`
	ActionStatus  string                   `json:"action_status"`
	PerformedBy   string                   `json:"performed_by"`
	PerformedAt   time.Time                `json:"performed_at"`
	ErrorCode     string                   `json:"error_code,omitempty"`
	ErrorCategory string                   `json:"error_category,omitempty"`
}

// GenerateContractRequest represents a request to generate a contract document
type GenerateContractRequest struct {
	TemplateCode string                   `json:"template_code,omitempty"`
	Reason       ContractGenerationReason `json:"reason,omitempty"`
}

// GenerateContractResponse represents the response from contract generation
// Note: Optionally returns the generated JSON for immediate use
type GenerateContractResponse struct {
	Success      bool            `json:"success"`
	GeneratedID  int64           `json:"generated_id,omitempty"`
	ContentHash  string          `json:"content_hash,omitempty"`
	ContractJSON json.RawMessage `json:"contract_data,omitempty"` // Clean JSON for PDF rendering
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// GetGeneratedContentResponse represents the response when fetching generated content
type GetGeneratedContentResponse struct {
	GeneratedID  int64           `json:"generated_id"`
	ContentHash  string          `json:"content_hash"`
	GeneratedAt  time.Time       `json:"generated_at"`
	ContractJSON json.RawMessage `json:"contract_data"` // Clean JSON structure for PDF rendering
}

// GeneratedContractListItem represents an item in the list of generated contracts
// Excludes sensitive content data
type GeneratedContractListItem struct {
	ID                    int64                    `json:"id"`
	ContractID            int64                    `json:"contract_id"`
	GenerationNumber      int                      `json:"generation_number"`
	ContentHash           string                   `json:"content_hash"`
	CustomerNameSnapshot  string                   `json:"customer_name_snapshot"`
	TotalValueSnapshot    int64                    `json:"total_value_snapshot"` // Value in cents for precision
	ServicesCountSnapshot int                      `json:"services_count_snapshot"`
	GeneratedAt           time.Time                `json:"generated_at"`
	GeneratedBy           string                   `json:"generated_by"`
	GenerationReason      ContractGenerationReason `json:"generation_reason,omitempty"`
}

// GenerationStats represents contract generation statistics
type GenerationStats struct {
	TotalGenerated  int64 `json:"total_generated"`
	GeneratedToday  int64 `json:"generated_today"`
	GeneratedMonth  int64 `json:"generated_month"`
	UniqueContracts int64 `json:"unique_contracts"`
}

// ToListItem converts a GeneratedContract to GeneratedContractListItem (excludes content)
func (g *GeneratedContract) ToListItem() GeneratedContractListItem {
	return GeneratedContractListItem{
		ID:                    g.ID,
		ContractID:            g.ContractID,
		GenerationNumber:      g.GenerationNumber,
		ContentHash:           g.ContentHash,
		CustomerNameSnapshot:  g.CustomerNameSnapshot,
		TotalValueSnapshot:    g.TotalValueSnapshot,
		ServicesCountSnapshot: g.ServicesCountSnapshot,
		GeneratedAt:           g.GeneratedAt,
		GeneratedBy:           g.GeneratedBy,
		GenerationReason:      g.GenerationReason,
	}
}

// ContractTemplateResponse represents the API response for a template
type ContractTemplateResponse struct {
	ID           int64     `json:"id"`
	TemplateCode string    `json:"template_code"`
	TemplateName string    `json:"template_name"`
	Language     string    `json:"language"`
	IsDefault    bool      `json:"is_default"`
	Active       bool      `json:"active"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToResponse converts a ContractTemplate to ContractTemplateResponse
func (t *ContractTemplate) ToResponse() ContractTemplateResponse {
	return ContractTemplateResponse{
		ID:           t.ID,
		TemplateCode: t.TemplateCode,
		TemplateName: t.TemplateName,
		Language:     t.Language,
		IsDefault:    t.IsDefault,
		Active:       t.Active,
		Version:      t.Version,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
