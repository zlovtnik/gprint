package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// AuditRepository handles audit trail persistence
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new AuditRepository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit entry
func (r *AuditRepository) Create(ctx context.Context, entry domain.AuditEntry) fp.Result[domain.AuditEntry] {
	query := `
		INSERT INTO clm_audit_trail (
			id, tenant_id, entity_type, entity_id, action, category,
			user_id, user_name, user_role, ip_address, user_agent,
			old_values, new_values, metadata, created_at
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15
		)`

	var oldValues, newValues, metadata sql.NullString

	if entry.OldValues != nil {
		b, err := json.Marshal(entry.OldValues)
		if err != nil {
			return fp.Failure[domain.AuditEntry](fmt.Errorf("marshal audit field OldValues: %w", err))
		}
		oldValues = sql.NullString{String: string(b), Valid: true}
	}
	if entry.NewValues != nil {
		b, err := json.Marshal(entry.NewValues)
		if err != nil {
			return fp.Failure[domain.AuditEntry](fmt.Errorf("marshal audit field NewValues: %w", err))
		}
		newValues = sql.NullString{String: string(b), Valid: true}
	}
	if entry.Metadata != nil {
		b, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fp.Failure[domain.AuditEntry](fmt.Errorf("marshal audit field Metadata: %w", err))
		}
		metadata = sql.NullString{String: string(b), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.TenantID,
		entry.EntityType,
		entry.EntityID,
		string(entry.Action),
		string(entry.Category),
		uuid.UUID(entry.UserID).String(),
		entry.UserName,
		nullableString(entry.UserRole),
		nullableString(entry.IPAddress),
		nullableString(entry.UserAgent),
		oldValues,
		newValues,
		metadata,
		entry.Timestamp,
	)
	if err != nil {
		return fp.Failure[domain.AuditEntry](err)
	}

	return fp.Success(entry)
}

// FindByEntity retrieves audit entries for an entity
func (r *AuditRepository) FindByEntity(ctx context.Context, tenantID, entityType, entityID string, offset, limit int) fp.Result[[]domain.AuditEntry] {
	query := `
		SELECT id, tenant_id, entity_type, entity_id, action, category,
			user_id, user_name, user_role, ip_address, user_agent,
			old_values, new_values, metadata, created_at
		FROM clm_audit_trail
		WHERE tenant_id = :1 AND entity_type = :2 AND entity_id = :3
		ORDER BY created_at DESC
		OFFSET :4 ROWS FETCH NEXT :5 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, entityType, entityID, offset, limit)
	if err != nil {
		return fp.Failure[[]domain.AuditEntry](err)
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		result := scanAuditEntry(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.AuditEntry](fp.GetError(result))
		}
		entries = append(entries, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.AuditEntry](err)
	}

	return fp.Success(entries)
}

// FindByUser retrieves audit entries for a user
func (r *AuditRepository) FindByUser(ctx context.Context, tenantID string, userID domain.UserID, offset, limit int) fp.Result[[]domain.AuditEntry] {
	query := `
		SELECT id, tenant_id, entity_type, entity_id, action, category,
			user_id, user_name, user_role, ip_address, user_agent,
			old_values, new_values, metadata, created_at
		FROM clm_audit_trail
		WHERE tenant_id = :1 AND user_id = :2
		ORDER BY created_at DESC
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.UUID(userID).String(), offset, limit)
	if err != nil {
		return fp.Failure[[]domain.AuditEntry](err)
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		result := scanAuditEntry(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.AuditEntry](fp.GetError(result))
		}
		entries = append(entries, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.AuditEntry](err)
	}

	return fp.Success(entries)
}

func scanAuditEntry(rows *sql.Rows) fp.Result[domain.AuditEntry] {
	var entry domain.AuditEntry
	var userIDStr string
	var action, category string
	var userRole, ipAddress, userAgent sql.NullString
	var oldValues, newValues, metadata sql.NullString

	err := rows.Scan(
		&entry.ID, &entry.TenantID, &entry.EntityType, &entry.EntityID,
		&action, &category, &userIDStr, &entry.UserName,
		&userRole, &ipAddress, &userAgent,
		&oldValues, &newValues, &metadata, &entry.Timestamp,
	)
	if err != nil {
		return fp.Failure[domain.AuditEntry](err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fp.Failure[domain.AuditEntry](fmt.Errorf("parse user_id %q: %w", userIDStr, err))
	}
	entry.UserID = domain.UserID(userID)
	entry.Action = domain.AuditAction(action)
	entry.Category = domain.AuditCategory(category)

	if userRole.Valid {
		entry.UserRole = userRole.String
	}
	if ipAddress.Valid {
		entry.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		entry.UserAgent = userAgent.String
	}

	if oldValues.Valid {
		json.Unmarshal([]byte(oldValues.String), &entry.OldValues)
	}
	if newValues.Valid {
		json.Unmarshal([]byte(newValues.String), &entry.NewValues)
	}
	if metadata.Valid {
		json.Unmarshal([]byte(metadata.String), &entry.Metadata)
	}

	return fp.Success(entry)
}
