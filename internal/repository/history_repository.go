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

// Create creates a new history entry using stored procedure
func (r *HistoryRepository) Create(ctx context.Context, tenantID string, req *models.CreateHistoryRequest) (*models.ContractHistory, error) {
	// Use stored procedure sp_insert_history
	var id int64
	var success int
	var errorMsg string

	_, err := r.db.ExecContext(ctx, `BEGIN sp_insert_history(
		p_tenant_id => :1,
		p_contract_id => :2,
		p_action => :3,
		p_field_changed => :4,
		p_old_value => :5,
		p_new_value => :6,
		p_performed_by => :7,
		p_ip_address => :8,
		p_user_agent => :9,
		p_id => :10,
		p_success => :11,
		p_error_msg => :12
	); END;`,
		tenantID, req.ContractID, req.Action, req.FieldChanged,
		req.OldValue, req.NewValue, req.PerformedBy, req.IPAddress, req.UserAgent,
		sql.Out{Dest: &id},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create history entry: %w", err)
	}
	if success != 1 {
		return nil, fmt.Errorf("failed to create history entry: %s", errorMsg)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a history entry by ID.
// NOTE: Uses direct SQL SELECT against contract_history table.
// Stored procedure sp_get_history is available but not used here for Go driver compatibility.
// FUTURE: Migrate to sp_get_history if/when ref cursor handling is needed.
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

	// Main query - uses direct SQL for Go driver compatibility
	// Stored procedure sp_get_history_by_contract is available but not used here.
	// FUTURE: Migrate to sp_get_history_by_contract if/when ref cursor handling is needed.
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
