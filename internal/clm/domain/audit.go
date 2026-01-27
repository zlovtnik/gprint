package domain

import (
	"fmt"
	"time"
)

// AuditAction represents the type of action in the audit trail
type AuditAction string

const (
	AuditActionCreated    AuditAction = "CREATED"
	AuditActionUpdated    AuditAction = "UPDATED"
	AuditActionDeleted    AuditAction = "DELETED"
	AuditActionApproved   AuditAction = "APPROVED"
	AuditActionRejected   AuditAction = "REJECTED"
	AuditActionSigned     AuditAction = "SIGNED"
	AuditActionExecuted   AuditAction = "EXECUTED"
	AuditActionActivated  AuditAction = "ACTIVATED"
	AuditActionTerminated AuditAction = "TERMINATED"
	AuditActionSubmitted  AuditAction = "SUBMITTED"
	AuditActionViewed     AuditAction = "VIEWED"
	AuditActionDownloaded AuditAction = "DOWNLOADED"
	AuditActionPrinted    AuditAction = "PRINTED"

	// Aliases for convenience
	AuditActionCreate       = AuditActionCreated
	AuditActionUpdate       = AuditActionUpdated
	AuditActionDelete       = AuditActionDeleted
	AuditActionApprove      = AuditActionApproved
	AuditActionReject       = AuditActionRejected
	AuditActionSign         = AuditActionSigned
	AuditActionStatusChange = AuditActionUpdated
	AuditActionDeactivate   = AuditActionUpdated
)

// AuditCategory represents the category of audit entry
type AuditCategory string

const (
	AuditCategoryContract   AuditCategory = "CONTRACT"
	AuditCategoryParty      AuditCategory = "PARTY"
	AuditCategoryObligation AuditCategory = "OBLIGATION"
	AuditCategoryWorkflow   AuditCategory = "WORKFLOW"
	AuditCategoryDocument   AuditCategory = "DOCUMENT"
	AuditCategorySecurity   AuditCategory = "SECURITY"

	// Aliases for convenience
	AuditCategoryData = AuditCategoryContract
)

// AuditEntry represents an audit trail entry (immutable)
type AuditEntry struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     AuditAction            `json:"action"`
	Category   AuditCategory          `json:"category"`
	UserID     UserID                 `json:"user_id"`
	UserName   string                 `json:"user_name"`
	UserRole   string                 `json:"user_role,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	OldValues  map[string]interface{} `json:"old_values,omitempty"`
	NewValues  map[string]interface{} `json:"new_values,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NewAuditEntry creates a new AuditEntry
func NewAuditEntry(
	tenantID string,
	entityType string,
	entityID string,
	action AuditAction,
	category AuditCategory,
	userID UserID,
	userName string,
) AuditEntry {
	return AuditEntry{
		ID:         NewDocumentID().String(), // Using DocumentID generator for unique ID
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Category:   category,
		UserID:     userID,
		UserName:   userName,
		Timestamp:  time.Now(),
	}
}

// WithUserRole returns a copy with user role
func (e AuditEntry) WithUserRole(role string) AuditEntry {
	e.UserRole = role
	return e
}

// WithIPAddress returns a copy with IP address
func (e AuditEntry) WithIPAddress(ip string) AuditEntry {
	e.IPAddress = ip
	return e
}

// WithUserAgent returns a copy with user agent
func (e AuditEntry) WithUserAgent(ua string) AuditEntry {
	e.UserAgent = ua
	return e
}

// WithChanges returns a copy with old and new values
func (e AuditEntry) WithChanges(oldValues, newValues map[string]interface{}) AuditEntry {
	if oldValues != nil {
		e.OldValues = make(map[string]interface{})
		for k, v := range oldValues {
			e.OldValues[k] = v
		}
	} else {
		e.OldValues = nil
	}
	if newValues != nil {
		e.NewValues = make(map[string]interface{})
		for k, v := range newValues {
			e.NewValues[k] = v
		}
	} else {
		e.NewValues = nil
	}
	return e
}

// WithMetadata returns a copy with metadata
func (e AuditEntry) WithMetadata(metadata map[string]interface{}) AuditEntry {
	if metadata != nil {
		e.Metadata = make(map[string]interface{})
		for k, v := range metadata {
			e.Metadata[k] = v
		}
	} else {
		e.Metadata = nil
	}
	return e
}

// DomainError represents a domain-level error
type DomainError struct {
	Message string
}

func (e DomainError) Error() string {
	return e.Message
}

// NewDomainError creates a new DomainError
func NewDomainError(format string, args ...interface{}) error {
	return DomainError{
		Message: fmt.Sprintf(format, args...),
	}
}
