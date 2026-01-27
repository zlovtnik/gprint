package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// DocumentRepository handles document persistence
type DocumentRepository struct {
	db *sql.DB
}

// NewDocumentRepository creates a new DocumentRepository
func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create inserts a new document
func (r *DocumentRepository) Create(ctx context.Context, doc domain.Document) fp.Result[domain.Document] {
	query := `
		INSERT INTO clm_documents (
			id, tenant_id, contract_id, document_type, title, file_name,
			file_path, mime_type, file_size, checksum, is_primary,
			is_signed, uploaded_at, uploaded_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14
		)`

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(doc.ID).String(),
		doc.TenantID,
		uuid.UUID(doc.ContractID).String(),
		string(doc.Type),
		doc.Title,
		doc.FileName,
		doc.FilePath,
		doc.MimeType,
		doc.FileSize,
		nullableString(doc.Checksum),
		boolToInt(doc.IsPrimary),
		boolToInt(doc.IsSigned),
		doc.UploadedAt,
		uuid.UUID(doc.UploadedBy).String(),
	)
	if err != nil {
		return fp.Failure[domain.Document](err)
	}

	return fp.Success(doc)
}

// FindByID retrieves a document by ID
func (r *DocumentRepository) FindByID(ctx context.Context, tenantID string, id domain.DocumentID) fp.Result[domain.Document] {
	query := `
		SELECT id, tenant_id, contract_id, document_type, title, file_name,
			file_path, mime_type, file_size, checksum, is_primary,
			is_signed, signed_at, signed_by, uploaded_at, uploaded_by
		FROM clm_documents
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	return scanDocument(row)
}

// FindByContract retrieves all documents for a contract
func (r *DocumentRepository) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID, offset, limit int) fp.Result[[]domain.Document] {
	query := `
		SELECT id, tenant_id, contract_id, document_type, title, file_name,
			file_path, mime_type, file_size, checksum, is_primary,
			is_signed, signed_at, signed_by, uploaded_at, uploaded_by
		FROM clm_documents
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY uploaded_at DESC
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.UUID(contractID).String(), offset, limit)
	if err != nil {
		return fp.Failure[[]domain.Document](err)
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		result := scanDocumentFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Document](fp.GetError(result))
		}
		docs = append(docs, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Document](err)
	}

	return fp.Success(docs)
}

// CountByContract returns the count of documents for a contract
func (r *DocumentRepository) CountByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[int] {
	query := `SELECT COUNT(*) FROM clm_documents WHERE tenant_id = :1 AND contract_id = :2`

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(contractID).String()).Scan(&count)
	if err != nil {
		return fp.Failure[int](err)
	}
	return fp.Success(count)
}

// MarkSigned marks a document as signed
func (r *DocumentRepository) MarkSigned(ctx context.Context, tenantID string, id domain.DocumentID, signedBy domain.UserID) fp.Result[bool] {
	query := `
		UPDATE clm_documents SET
			is_signed = 1, signed_at = :1, signed_by = :2
		WHERE tenant_id = :3 AND id = :4`

	result, err := r.db.ExecContext(ctx, query,
		time.Now(),
		uuid.UUID(signedBy).String(),
		tenantID,
		uuid.UUID(id).String(),
	)
	if err != nil {
		return fp.Failure[bool](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[bool](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[bool](errors.New("document not found"))
	}

	return fp.Success(true)
}

// Delete removes a document
func (r *DocumentRepository) Delete(ctx context.Context, tenantID string, id domain.DocumentID) fp.Result[bool] {
	query := `DELETE FROM clm_documents WHERE tenant_id = :1 AND id = :2`

	result, err := r.db.ExecContext(ctx, query, tenantID, uuid.UUID(id).String())
	if err != nil {
		return fp.Failure[bool](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[bool](err)
	}
	if rowsAffected == 0 {
		return fp.Success(false)
	}

	return fp.Success(true)
}

func scanDocument(row *sql.Row) fp.Result[domain.Document] {
	var doc domain.Document
	var idStr, contractIDStr, uploadedByStr string
	var docType string
	var checksum, signedByStr sql.NullString
	var signedAt sql.NullTime
	var isPrimary, isSigned int

	err := row.Scan(
		&idStr, &doc.TenantID, &contractIDStr, &docType,
		&doc.Title, &doc.FileName, &doc.FilePath, &doc.MimeType,
		&doc.FileSize, &checksum, &isPrimary, &isSigned,
		&signedAt, &signedByStr, &doc.UploadedAt, &uploadedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Document](err)
	}

	id, _ := uuid.Parse(idStr)
	doc.ID = domain.DocumentID(id)

	contractID, _ := uuid.Parse(contractIDStr)
	doc.ContractID = domain.ContractID(contractID)

	uploadedBy, _ := uuid.Parse(uploadedByStr)
	doc.UploadedBy = domain.UserID(uploadedBy)

	doc.Type = domain.DocumentType(docType)
	doc.IsPrimary = isPrimary == 1
	doc.IsSigned = isSigned == 1

	if checksum.Valid {
		doc.Checksum = checksum.String
	}
	if signedAt.Valid {
		doc.SignedAt = &signedAt.Time
	}
	if signedByStr.Valid {
		signedBy, _ := uuid.Parse(signedByStr.String)
		doc.SignedBy = (*domain.UserID)(&signedBy)
	}

	return fp.Success(doc)
}

func scanDocumentFromRows(rows *sql.Rows) fp.Result[domain.Document] {
	var doc domain.Document
	var idStr, contractIDStr, uploadedByStr string
	var docType string
	var checksum, signedByStr sql.NullString
	var signedAt sql.NullTime
	var isPrimary, isSigned int

	err := rows.Scan(
		&idStr, &doc.TenantID, &contractIDStr, &docType,
		&doc.Title, &doc.FileName, &doc.FilePath, &doc.MimeType,
		&doc.FileSize, &checksum, &isPrimary, &isSigned,
		&signedAt, &signedByStr, &doc.UploadedAt, &uploadedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Document](err)
	}

	id, _ := uuid.Parse(idStr)
	doc.ID = domain.DocumentID(id)

	contractID, _ := uuid.Parse(contractIDStr)
	doc.ContractID = domain.ContractID(contractID)

	uploadedBy, _ := uuid.Parse(uploadedByStr)
	doc.UploadedBy = domain.UserID(uploadedBy)

	doc.Type = domain.DocumentType(docType)
	doc.IsPrimary = isPrimary == 1
	doc.IsSigned = isSigned == 1

	if checksum.Valid {
		doc.Checksum = checksum.String
	}
	if signedAt.Valid {
		doc.SignedAt = &signedAt.Time
	}
	if signedByStr.Valid {
		signedBy, _ := uuid.Parse(signedByStr.String)
		doc.SignedBy = (*domain.UserID)(&signedBy)
	}

	return fp.Success(doc)
}

// TemplateRepository handles template persistence
type TemplateRepository struct {
	db *sql.DB
}

// NewTemplateRepository creates a new TemplateRepository
func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// FindByID retrieves a template by ID
func (r *TemplateRepository) FindByID(ctx context.Context, tenantID string, id domain.TemplateID) fp.Result[domain.DocumentTemplate] {
	query := `
		SELECT id, tenant_id, name, contract_type_id, file_path, version,
			is_active, created_at, updated_at, created_by
		FROM clm_document_templates
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	return scanTemplate(row)
}

// FindAll retrieves all templates
func (r *TemplateRepository) FindAll(ctx context.Context, tenantID string, activeOnly bool) fp.Result[[]domain.DocumentTemplate] {
	query := `
		SELECT id, tenant_id, name, contract_type_id, file_path, version,
			is_active, created_at, updated_at, created_by
		FROM clm_document_templates
		WHERE tenant_id = :1`

	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return fp.Failure[[]domain.DocumentTemplate](err)
	}
	defer rows.Close()

	var templates []domain.DocumentTemplate
	for rows.Next() {
		result := scanTemplateFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.DocumentTemplate](fp.GetError(result))
		}
		templates = append(templates, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.DocumentTemplate](err)
	}

	return fp.Success(templates)
}

func scanTemplate(row *sql.Row) fp.Result[domain.DocumentTemplate] {
	var template domain.DocumentTemplate
	var idStr, typeIDStr, createdByStr string
	var updatedAt sql.NullTime
	var isActive int

	err := row.Scan(
		&idStr, &template.TenantID, &template.Name, &typeIDStr,
		&template.FilePath, &template.Version, &isActive,
		&template.CreatedAt, &updatedAt, &createdByStr,
	)
	if err != nil {
		return fp.Failure[domain.DocumentTemplate](err)
	}

	id, _ := uuid.Parse(idStr)
	template.ID = domain.TemplateID(id)

	typeID, _ := uuid.Parse(typeIDStr)
	template.ContractTypeID = domain.ContractTypeID(typeID)

	createdBy, _ := uuid.Parse(createdByStr)
	template.CreatedBy = domain.UserID(createdBy)

	template.IsActive = isActive == 1

	if updatedAt.Valid {
		template.UpdatedAt = &updatedAt.Time
	}

	return fp.Success(template)
}

func scanTemplateFromRows(rows *sql.Rows) fp.Result[domain.DocumentTemplate] {
	var template domain.DocumentTemplate
	var idStr, typeIDStr, createdByStr string
	var updatedAt sql.NullTime
	var isActive int

	err := rows.Scan(
		&idStr, &template.TenantID, &template.Name, &typeIDStr,
		&template.FilePath, &template.Version, &isActive,
		&template.CreatedAt, &updatedAt, &createdByStr,
	)
	if err != nil {
		return fp.Failure[domain.DocumentTemplate](err)
	}

	id, _ := uuid.Parse(idStr)
	template.ID = domain.TemplateID(id)

	typeID, _ := uuid.Parse(typeIDStr)
	template.ContractTypeID = domain.ContractTypeID(typeID)

	createdBy, _ := uuid.Parse(createdByStr)
	template.CreatedBy = domain.UserID(createdBy)

	template.IsActive = isActive == 1

	if updatedAt.Valid {
		template.UpdatedAt = &updatedAt.Time
	}

	return fp.Success(template)
}
