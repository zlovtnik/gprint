package domain

import (
	"time"
)

// DocumentType represents the type of document
type DocumentType string

const (
	DocumentTypeContract   DocumentType = "CONTRACT"
	DocumentTypeAmendment  DocumentType = "AMENDMENT"
	DocumentTypeAttachment DocumentType = "ATTACHMENT"
	DocumentTypeSignature  DocumentType = "SIGNATURE"
	DocumentTypeEvidence   DocumentType = "EVIDENCE"
	DocumentTypeTemplate   DocumentType = "TEMPLATE"
)

// Document represents a document attached to a contract (immutable)
type Document struct {
	ID         DocumentID   `json:"id"`
	TenantID   string       `json:"tenant_id"`
	ContractID ContractID   `json:"contract_id"`
	Type       DocumentType `json:"type"`
	Title      string       `json:"title"`
	FileName   string       `json:"file_name"`
	FilePath   string       `json:"file_path"`
	MimeType   string       `json:"mime_type"`
	FileSize   int64        `json:"file_size"`
	Checksum   string       `json:"checksum,omitempty"`
	IsPrimary  bool         `json:"is_primary"`
	IsSigned   bool         `json:"is_signed"`
	SignedAt   *time.Time   `json:"signed_at,omitempty"`
	SignedBy   *UserID      `json:"signed_by,omitempty"`
	UploadedAt time.Time    `json:"uploaded_at"`
	UploadedBy UserID       `json:"uploaded_by"`
}

// MarkAsSigned marks the document as signed
func (d Document) MarkAsSigned(signedBy UserID) Document {
	now := time.Now()
	d.IsSigned = true
	d.SignedAt = &now
	d.SignedBy = &signedBy
	return d
}

// WithChecksum returns a copy with updated checksum
func (d Document) WithChecksum(checksum string) Document {
	d.Checksum = checksum
	return d
}

// NewDocument creates a new Document
func NewDocument(
	tenantID string,
	contractID ContractID,
	docType DocumentType,
	title string,
	fileName string,
	filePath string,
	mimeType string,
	fileSize int64,
	isPrimary bool,
	uploadedBy UserID,
) Document {
	return Document{
		ID:         NewDocumentID(),
		TenantID:   tenantID,
		ContractID: contractID,
		Type:       docType,
		Title:      title,
		FileName:   fileName,
		FilePath:   filePath,
		MimeType:   mimeType,
		FileSize:   fileSize,
		IsPrimary:  isPrimary,
		IsSigned:   false,
		UploadedAt: time.Now(),
		UploadedBy: uploadedBy,
	}
}

// MergeFieldDefinition represents a merge field in a template
type MergeFieldDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DataType     string `json:"data_type"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
}

// DocumentTemplate represents a document template (immutable)
type DocumentTemplate struct {
	ID             TemplateID             `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	Name           string                 `json:"name"`
	ContractTypeID ContractTypeID         `json:"contract_type_id"`
	FilePath       string                 `json:"file_path"`
	Version        int                    `json:"version"`
	MergeFields    []MergeFieldDefinition `json:"merge_fields,omitempty"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      *time.Time             `json:"updated_at,omitempty"`
	CreatedBy      UserID                 `json:"created_by"`
}

// WithName returns a copy with updated name
func (t DocumentTemplate) WithName(name string) DocumentTemplate {
	now := time.Now()
	t.Name = name
	t.UpdatedAt = &now
	return t
}

// WithFilePath returns a copy with updated file path
func (t DocumentTemplate) WithFilePath(filePath string) DocumentTemplate {
	now := time.Now()
	t.FilePath = filePath
	t.UpdatedAt = &now
	return t
}

// WithMergeFields returns a copy with updated merge fields
func (t DocumentTemplate) WithMergeFields(fields []MergeFieldDefinition) DocumentTemplate {
	now := time.Now()
	// Create a defensive copy of the slice
	if fields != nil {
		t.MergeFields = make([]MergeFieldDefinition, len(fields))
		copy(t.MergeFields, fields)
	} else {
		t.MergeFields = nil
	}
	t.UpdatedAt = &now
	return t
}

// Deactivate returns a copy with IsActive set to false
func (t DocumentTemplate) Deactivate() DocumentTemplate {
	now := time.Now()
	t.IsActive = false
	t.UpdatedAt = &now
	return t
}

// IncrementVersion returns a copy with incremented version
func (t DocumentTemplate) IncrementVersion() DocumentTemplate {
	now := time.Now()
	t.Version++
	t.UpdatedAt = &now
	return t
}

// NewDocumentTemplate creates a new DocumentTemplate
func NewDocumentTemplate(
	tenantID string,
	name string,
	contractTypeID ContractTypeID,
	filePath string,
	createdBy UserID,
) DocumentTemplate {
	return DocumentTemplate{
		ID:             NewTemplateID(),
		TenantID:       tenantID,
		Name:           name,
		ContractTypeID: contractTypeID,
		FilePath:       filePath,
		Version:        1,
		MergeFields:    []MergeFieldDefinition{},
		IsActive:       true,
		CreatedAt:      time.Now(),
		CreatedBy:      createdBy,
	}
}
