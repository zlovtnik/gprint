package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zlovtnik/gprint/internal/models"
)

// HistoryRepository handles contract history data access
type HistoryRepository struct {
	db *sql.DB
}

// NewHistoryRepository creates a new HistoryRepository
func NewHistoryRepository(db *sql.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// Create creates a new history entry
func (r *HistoryRepository) Create(ctx context.Context, tenantID string, req *models.CreateHistoryRequest) (*models.ContractHistory, error) {
	query := `
		INSERT INTO contract_history (
			tenant_id, contract_id, action, field_changed,
			old_value, new_value, performed_by, ip_address, user_agent
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9
		) RETURNING id INTO :10`

	var id int64
	_, err := r.db.ExecContext(ctx, query,
		tenantID, req.ContractID, req.Action, req.FieldChanged,
		req.OldValue, req.NewValue, req.PerformedBy, req.IPAddress, req.UserAgent,
		sql.Out{Dest: &id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create history entry: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a history entry by ID
func (r *HistoryRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.ContractHistory, error) {
	query := `
		SELECT id, tenant_id, contract_id, action, field_changed,
			old_value, new_value, performed_by, performed_at, ip_address, user_agent
		FROM contract_history
		WHERE tenant_id = :1 AND id = :2`

	var h models.ContractHistory
	var fieldChanged, oldValue, newValue, ipAddress, userAgent sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&h.ID, &h.TenantID, &h.ContractID, &h.Action, &fieldChanged,
		&oldValue, &newValue, &h.PerformedBy, &h.PerformedAt, &ipAddress, &userAgent,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get history entry: %w", err)
	}

	h.FieldChanged = fieldChanged.String
	h.OldValue = oldValue.String
	h.NewValue = newValue.String
	h.IPAddress = ipAddress.String
	h.UserAgent = userAgent.String

	return &h, nil
}

// GetByContractID retrieves history for a contract
func (r *HistoryRepository) GetByContractID(ctx context.Context, tenantID string, contractID int64, params models.PaginationParams) ([]models.ContractHistory, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM contract_history WHERE tenant_id = :1 AND contract_id = :2`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, tenantID, contractID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Main query
	query := `
		SELECT id, tenant_id, contract_id, action, field_changed,
			old_value, new_value, performed_by, performed_at, ip_address, user_agent
		FROM contract_history
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY performed_at DESC
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, contractID, params.Offset(), params.Limit())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var history []models.ContractHistory
	for rows.Next() {
		var h models.ContractHistory
		var fieldChanged, oldValue, newValue, ipAddress, userAgent sql.NullString

		err := rows.Scan(
			&h.ID, &h.TenantID, &h.ContractID, &h.Action, &fieldChanged,
			&oldValue, &newValue, &h.PerformedBy, &h.PerformedAt, &ipAddress, &userAgent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history: %w", err)
		}

		h.FieldChanged = fieldChanged.String
		h.OldValue = oldValue.String
		h.NewValue = newValue.String
		h.IPAddress = ipAddress.String
		h.UserAgent = userAgent.String

		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate history rows: %w", err)
	}

	return history, total, nil
}
