package models

import (
	"sync/atomic"
	"time"
)

// HistoryAction represents the type of action in history
type HistoryAction string

const (
	HistoryActionCreate       HistoryAction = "CREATE"
	HistoryActionUpdate       HistoryAction = "UPDATE"
	HistoryActionStatusChange HistoryAction = "STATUS_CHANGE"
	HistoryActionSign         HistoryAction = "SIGN"
	HistoryActionPrint        HistoryAction = "PRINT"
	HistoryActionDelete       HistoryAction = "DELETE"
)

// DataRetentionConfig defines retention policy for history personal data
// IPAddress and UserAgent are considered personal/tracking data and should be:
// - Subject to configurable retention periods
// - Anonymized or purged after retention period expires
// - Only collected with appropriate privacy notices and user consent
type DataRetentionConfig struct {
	// IPAddressRetentionDays is the number of days to retain IP addresses (0 = disabled)
	IPAddressRetentionDays int
	// UserAgentRetentionDays is the number of days to retain user agent strings (0 = disabled)
	UserAgentRetentionDays int
	// CollectIPAddress indicates whether IP addresses should be collected
	CollectIPAddress bool
	// CollectUserAgent indicates whether user agents should be collected
	CollectUserAgent bool
}

// historyDataRetention is an atomic pointer for thread-safe access to retention config
var historyDataRetention atomic.Pointer[DataRetentionConfig]

func init() {
	historyDataRetention.Store(&DataRetentionConfig{
		IPAddressRetentionDays: 90,
		UserAgentRetentionDays: 90,
		CollectIPAddress:       true,
		CollectUserAgent:       true,
	})
}

// GetHistoryDataRetention returns a copy of the current data retention configuration
func GetHistoryDataRetention() DataRetentionConfig {
	return *historyDataRetention.Load()
}

// UpdateHistoryDataRetention atomically updates the data retention configuration
func UpdateHistoryDataRetention(cfg *DataRetentionConfig) {
	if cfg == nil {
		return
	}
	// Store a copy to prevent external mutation
	configCopy := *cfg
	historyDataRetention.Store(&configCopy)
}

// ContractHistory represents a history entry for contract changes
type ContractHistory struct {
	ID           int64         `json:"id"`
	TenantID     string        `json:"tenant_id"`
	ContractID   int64         `json:"contract_id"`
	Action       HistoryAction `json:"action"`
	FieldChanged string        `json:"field_changed,omitempty"`
	OldValue     string        `json:"old_value,omitempty"`
	NewValue     string        `json:"new_value,omitempty"`
	PerformedBy  string        `json:"performed_by"`
	PerformedAt  time.Time     `json:"performed_at"`
	// IPAddress contains the client IP address. This is personal/tracking data
	// subject to retention policy defined in HistoryDataRetention.
	// Use Anonymize() to remove this data after retention period.
	IPAddress string `json:"ip_address,omitempty"`
	// UserAgent contains the client user agent string. This is personal/tracking data
	// subject to retention policy defined in HistoryDataRetention.
	// Use Anonymize() to remove this data after retention period.
	UserAgent string `json:"user_agent,omitempty"`
}

// Anonymize selectively removes personal/tracking data (IPAddress and/or UserAgent) from the history entry
// based on the configured retention periods. Only clears fields whose retention period has expired.
// For explicit full-erasure requests, use ForceAnonymize() instead.
func (h *ContractHistory) Anonymize() {
	cfg := GetHistoryDataRetention()
	now := time.Now()

	// Check and clear IPAddress if retention period has passed
	if h.IPAddress != "" && cfg.IPAddressRetentionDays > 0 {
		ipRetention := time.Duration(cfg.IPAddressRetentionDays) * 24 * time.Hour
		if now.Sub(h.PerformedAt) > ipRetention {
			h.IPAddress = ""
		}
	}

	// Check and clear UserAgent if retention period has passed
	if h.UserAgent != "" && cfg.UserAgentRetentionDays > 0 {
		uaRetention := time.Duration(cfg.UserAgentRetentionDays) * 24 * time.Hour
		if now.Sub(h.PerformedAt) > uaRetention {
			h.UserAgent = ""
		}
	}
}

// ForceAnonymize unconditionally removes all personal/tracking data (IPAddress and UserAgent)
// from the history entry. Use this for explicit full-erasure requests (e.g., GDPR data deletion).
func (h *ContractHistory) ForceAnonymize() {
	h.IPAddress = ""
	h.UserAgent = ""
}

// IsEligibleForAnonymization returns true if this history entry's personal data
// is older than the configured retention period and should be anonymized.
func (h *ContractHistory) IsEligibleForAnonymization() bool {
	if h.IPAddress == "" && h.UserAgent == "" {
		return false // Already anonymized
	}

	cfg := GetHistoryDataRetention()
	now := time.Now()
	ipRetention := time.Duration(cfg.IPAddressRetentionDays) * 24 * time.Hour
	uaRetention := time.Duration(cfg.UserAgentRetentionDays) * 24 * time.Hour

	ipEligible := h.IPAddress != "" && cfg.IPAddressRetentionDays > 0 && now.Sub(h.PerformedAt) > ipRetention
	uaEligible := h.UserAgent != "" && cfg.UserAgentRetentionDays > 0 && now.Sub(h.PerformedAt) > uaRetention

	return ipEligible || uaEligible
}

// CreateHistoryRequest represents the request to create a history entry
type CreateHistoryRequest struct {
	ContractID   int64         `json:"contract_id"`
	Action       HistoryAction `json:"action"`
	FieldChanged string        `json:"field_changed,omitempty"`
	OldValue     string        `json:"old_value,omitempty"`
	NewValue     string        `json:"new_value,omitempty"`
	PerformedBy  string        `json:"performed_by"`
	IPAddress    string        `json:"ip_address,omitempty"`
	UserAgent    string        `json:"user_agent,omitempty"`
}

// HistoryResponse represents the API response for a history entry
type HistoryResponse struct {
	ID           int64         `json:"id"`
	ContractID   int64         `json:"contract_id"`
	Action       HistoryAction `json:"action"`
	FieldChanged string        `json:"field_changed,omitempty"`
	OldValue     string        `json:"old_value,omitempty"`
	NewValue     string        `json:"new_value,omitempty"`
	PerformedBy  string        `json:"performed_by"`
	PerformedAt  time.Time     `json:"performed_at"`
}

// ToResponse converts a ContractHistory to HistoryResponse
func (h *ContractHistory) ToResponse() HistoryResponse {
	return HistoryResponse{
		ID:           h.ID,
		ContractID:   h.ContractID,
		Action:       h.Action,
		FieldChanged: h.FieldChanged,
		OldValue:     h.OldValue,
		NewValue:     h.NewValue,
		PerformedBy:  h.PerformedBy,
		PerformedAt:  h.PerformedAt,
	}
}
