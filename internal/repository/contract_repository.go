package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/models"
)

// ErrNotFound is returned when a requested resource does not exist
var ErrNotFound = errors.New("resource not found")

// ContractRepository handles contract data access
type ContractRepository struct {
	db *sql.DB
}

// NewContractRepository creates a new ContractRepository
func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

// Create creates a new contract with items
func (r *ContractRepository) Create(ctx context.Context, tenantID string, req *models.CreateContractRequest, createdBy string) (*models.Contract, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf(errFmtBeginTx, err)
	}
	defer func() { _ = tx.Rollback() }()

	billingCycleStr := string(req.BillingCycle)
	if billingCycleStr == "" {
		billingCycleStr = string(models.BillingCycleMonthly)
	}

	// Convert custom types to string for Oracle driver compatibility
	contractTypeStr := string(req.ContractType)

	contractQuery := `
		INSERT INTO contracts (
			tenant_id, contract_number, contract_type, customer_id,
			start_date, end_date, duration_months, auto_renew,
			payment_terms, billing_cycle, notes, terms_conditions,
			created_by, updated_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14
		) RETURNING id INTO :15`

	var contractID int64
	_, err = tx.ExecContext(ctx, contractQuery,
		tenantID, req.ContractNumber, contractTypeStr, req.CustomerID,
		req.StartDate, req.EndDate, req.DurationMonths, boolToInt(req.AutoRenew),
		req.PaymentTerms, billingCycleStr, req.Notes, req.TermsConditions,
		createdBy, createdBy,
		sql.Out{Dest: &contractID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}

	// Insert contract items using decimal for precise money calculations
	totalValue := decimal.NewFromInt(0)
	for _, item := range req.Items {
		hundred := decimal.NewFromInt(100)
		// lineTotal = quantity * unitPrice * (1 - discountPct/100)
		lineTotal := item.Quantity.Mul(item.UnitPrice).Mul(hundred.Sub(item.DiscountPct)).Div(hundred)
		totalValue = totalValue.Add(lineTotal)

		// Convert decimal to float64 for Oracle driver
		quantityFloat, _ := item.Quantity.Float64()
		unitPriceFloat, _ := item.UnitPrice.Float64()
		discountPctFloat, _ := item.DiscountPct.Float64()

		itemQuery := `
			INSERT INTO contract_items (
				tenant_id, contract_id, service_id, quantity, unit_price,
				discount_pct, start_date, end_date, delivery_date,
				description, notes
			) VALUES (
				:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11
			)`

		_, err = tx.ExecContext(ctx, itemQuery,
			tenantID, contractID, item.ServiceID, quantityFloat, unitPriceFloat,
			discountPctFloat, item.StartDate, item.EndDate, item.DeliveryDate,
			item.Description, item.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create contract item: %w", err)
		}
	}

	// Update total value (convert decimal to float64 at database boundary)
	totalValueFloat, _ := totalValue.Float64()
	_, err = tx.ExecContext(ctx, `UPDATE contracts SET total_value = :1 WHERE id = :2`, totalValueFloat, contractID)
	if err != nil {
		return nil, fmt.Errorf(errFmtUpdateTotalVal, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(errFmtCommitTx, err)
	}

	return r.GetByID(ctx, tenantID, contractID)
}

// GetByID retrieves a contract by ID with items
func (r *ContractRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.Contract, error) {
	query := `
		SELECT c.id, c.tenant_id, c.contract_number, c.contract_type, c.customer_id,
			c.start_date, c.end_date, c.duration_months, c.auto_renew,
			c.total_value, c.payment_terms, c.billing_cycle, c.status,
			c.signed_at, c.signed_by, c.document_path, c.document_hash,
			c.notes, c.terms_conditions, c.created_at, c.updated_at, c.created_by, c.updated_by
		FROM contracts c
		WHERE c.tenant_id = :1 AND c.id = :2`

	var contract models.Contract
	var endDate, signedAt sql.NullTime
	var durationMonths sql.NullInt64
	var signedBy, documentPath, documentHash, paymentTerms sql.NullString
	var notes, termsConditions, createdBy, updatedBy sql.NullString
	var totalValueFloat float64

	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&contract.ID, &contract.TenantID, &contract.ContractNumber, &contract.ContractType, &contract.CustomerID,
		&contract.StartDate, &endDate, &durationMonths, &contract.AutoRenew,
		&totalValueFloat, &paymentTerms, &contract.BillingCycle, &contract.Status,
		&signedAt, &signedBy, &documentPath, &documentHash,
		&notes, &termsConditions, &contract.CreatedAt, &contract.UpdatedAt, &createdBy, &updatedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	contract.TotalValue = decimal.NewFromFloat(totalValueFloat)
	if endDate.Valid {
		contract.EndDate = &endDate.Time
	}
	if signedAt.Valid {
		contract.SignedAt = &signedAt.Time
	}
	contract.DurationMonths = int(durationMonths.Int64)
	contract.SignedBy = signedBy.String
	contract.DocumentPath = documentPath.String
	contract.DocumentHash = documentHash.String
	contract.PaymentTerms = paymentTerms.String
	contract.Notes = notes.String
	contract.TermsConditions = termsConditions.String
	contract.CreatedBy = createdBy.String
	contract.UpdatedBy = updatedBy.String

	// Get items
	items, err := r.GetItems(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	contract.Items = items

	return &contract, nil
}

// GetItems retrieves items for a contract
func (r *ContractRepository) GetItems(ctx context.Context, tenantID string, contractID int64) ([]models.ContractItem, error) {
	query := `
		SELECT ci.id, ci.tenant_id, ci.contract_id, ci.service_id,
			ci.quantity, ci.unit_price, ci.discount_pct, ci.line_total,
			ci.start_date, ci.end_date, ci.delivery_date,
			ci.description, ci.status, ci.completed_at, ci.notes,
			ci.created_at, ci.updated_at
		FROM contract_items ci
		WHERE ci.tenant_id = :1 AND ci.contract_id = :2
		ORDER BY ci.id`

	rows, err := r.db.QueryContext(ctx, query, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract items: %w", err)
	}
	defer rows.Close()

	var items []models.ContractItem
	for rows.Next() {
		var item models.ContractItem
		var startDate, endDate, deliveryDate, completedAt sql.NullTime
		var description, notes sql.NullString

		err := rows.Scan(
			&item.ID, &item.TenantID, &item.ContractID, &item.ServiceID,
			&item.Quantity, &item.UnitPrice, &item.DiscountPct, &item.LineTotal,
			&startDate, &endDate, &deliveryDate,
			&description, &item.Status, &completedAt, &notes,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contract item: %w", err)
		}

		if startDate.Valid {
			item.StartDate = &startDate.Time
		}
		if endDate.Valid {
			item.EndDate = &endDate.Time
		}
		if deliveryDate.Valid {
			item.DeliveryDate = &deliveryDate.Time
		}
		if completedAt.Valid {
			item.CompletedAt = &completedAt.Time
		}
		item.Description = description.String
		item.Notes = notes.String

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contract items: %w", err)
	}

	return items, nil
}

// contractListAllowedSorts defines valid sort columns for contract listing
var contractListAllowedSorts = map[string]bool{
	"contract_number": true,
	"start_date":      true,
	"status":          true,
	"total_value":     true,
	"created_at":      true,
}

// getSortClause returns validated sort column and direction.
// It defensively validates defaultSort against the allowed map; if defaultSort
// is not empty and not present in allowed, it falls back to the first allowed
// key or "id" as a safe hard-coded column. When sortBy is valid and present
// in allowed, it overrides the default.
func getSortClause(sortBy, sortDir string, allowed map[string]bool, defaultSort string) (string, string) {
	// Validate defaultSort against allowed map
	col := defaultSort
	if defaultSort != "" && !allowed[defaultSort] {
		// defaultSort not in allowed map - fall back to deterministic column
		col = getDeterministicFallbackSortColumn(allowed)
	}

	// Override col when sortBy is valid and present in allowed
	if sortBy != "" && allowed[sortBy] {
		col = sortBy
	}

	// Determine sort direction
	dir := "DESC"
	if strings.ToUpper(sortDir) == "ASC" {
		dir = "ASC"
	}
	return col, dir
}

func getDeterministicFallbackSortColumn(allowed map[string]bool) string {
	if allowed["id"] {
		return "id"
	}
	if allowed["created_at"] {
		return "created_at"
	}
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return keys[0]
	}
	return "id"
}

// List retrieves contracts with pagination
func (r *ContractRepository) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Contract, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM contracts WHERE tenant_id = :1`
	args := []any{tenantID}
	argIndex := 2

	if search.Query != "" {
		countQuery += fmt.Sprintf(" AND UPPER(contract_number) LIKE UPPER(:%d)", argIndex)
		args = append(args, "%"+search.Query+"%")
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count contracts: %w", err)
	}

	// Main query
	query := `
		SELECT id, tenant_id, contract_number, contract_type, customer_id,
			start_date, end_date, duration_months, auto_renew,
			total_value, payment_terms, billing_cycle, status,
			signed_at, signed_by, document_path, document_hash,
			notes, terms_conditions, created_at, updated_at, created_by, updated_by
		FROM contracts
		WHERE tenant_id = :1`

	queryArgs := []any{tenantID}
	queryArgIndex := 2

	if search.Query != "" {
		query += fmt.Sprintf(" AND UPPER(contract_number) LIKE UPPER(:%d)", queryArgIndex)
		queryArgs = append(queryArgs, "%"+search.Query+"%")
		queryArgIndex++
	}

	// Sorting
	sortBy, sortDir := getSortClause(search.SortBy, search.SortDir, contractListAllowedSorts, "created_at")
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	// Pagination
	query += fmt.Sprintf(" OFFSET :%d ROWS FETCH NEXT :%d ROWS ONLY", queryArgIndex, queryArgIndex+1)
	queryArgs = append(queryArgs, params.Offset(), params.Limit())

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []models.Contract
	for rows.Next() {
		var c models.Contract
		var endDate, signedAt sql.NullTime
		var durationMonths sql.NullInt64
		var signedBy, documentPath, documentHash, paymentTerms sql.NullString
		var notes, termsConditions, createdBy, updatedBy sql.NullString
		var totalValueFloat float64

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.ContractNumber, &c.ContractType, &c.CustomerID,
			&c.StartDate, &endDate, &durationMonths, &c.AutoRenew,
			&totalValueFloat, &paymentTerms, &c.BillingCycle, &c.Status,
			&signedAt, &signedBy, &documentPath, &documentHash,
			&notes, &termsConditions, &c.CreatedAt, &c.UpdatedAt, &createdBy, &updatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan contract: %w", err)
		}

		c.TotalValue = decimal.NewFromFloat(totalValueFloat)
		if endDate.Valid {
			c.EndDate = &endDate.Time
		}
		if signedAt.Valid {
			c.SignedAt = &signedAt.Time
		}
		c.DurationMonths = int(durationMonths.Int64)
		c.SignedBy = signedBy.String
		c.DocumentPath = documentPath.String
		c.DocumentHash = documentHash.String
		c.PaymentTerms = paymentTerms.String
		c.Notes = notes.String
		c.TermsConditions = termsConditions.String
		c.CreatedBy = createdBy.String
		c.UpdatedBy = updatedBy.String

		contracts = append(contracts, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate contracts: %w", err)
	}

	return contracts, total, nil
}

// Update updates a contract
func (r *ContractRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateContractRequest, updatedBy string) (*models.Contract, error) {
	// Handle pointer types - convert to string values or empty string for COALESCE
	var contractType, billingCycle string
	if req.ContractType != nil {
		contractType = string(*req.ContractType)
	}
	if req.BillingCycle != nil {
		billingCycle = string(*req.BillingCycle)
	}

	query := `
		UPDATE contracts SET
			contract_type = COALESCE(NULLIF(:1, ''), contract_type),
			start_date = COALESCE(:2, start_date),
			end_date = :3,
			duration_months = COALESCE(:4, duration_months),
			payment_terms = :5,
			billing_cycle = COALESCE(NULLIF(:6, ''), billing_cycle),
			notes = :7,
			terms_conditions = :8,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = :9
		WHERE tenant_id = :10 AND id = :11`

	result, err := r.db.ExecContext(ctx, query,
		contractType, req.StartDate, req.EndDate, req.DurationMonths,
		req.PaymentTerms, billingCycle, req.Notes, req.TermsConditions,
		updatedBy, tenantID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update contract: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(errFmtRowsAffected, err)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, tenantID, id)
}

// UpdateStatus updates the contract status
func (r *ContractRepository) UpdateStatus(ctx context.Context, tenantID string, id int64, status models.ContractStatus, updatedBy string) error {
	query := `UPDATE contracts SET status = :1, updated_at = CURRENT_TIMESTAMP, updated_by = :2 WHERE tenant_id = :3 AND id = :4`
	result, err := r.db.ExecContext(ctx, query, string(status), updatedBy, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to update contract status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(errFmtRowsAffected, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Sign signs the contract
func (r *ContractRepository) Sign(ctx context.Context, tenantID string, id int64, signedBy string) error {
	now := time.Now()
	query := `UPDATE contracts SET status = :1, signed_at = :2, signed_by = :3, updated_at = CURRENT_TIMESTAMP, updated_by = :4 WHERE tenant_id = :5 AND id = :6`
	result, err := r.db.ExecContext(ctx, query, string(models.ContractStatusActive), now, signedBy, signedBy, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to sign contract: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(errFmtRowsAffected, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: tenant %s id %d", ErrNotFound, tenantID, id)
	}
	return nil
}

// AddItem adds an item to a contract
func (r *ContractRepository) AddItem(ctx context.Context, tenantID string, contractID int64, req *models.CreateContractItemRequest) (*models.ContractItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf(errFmtBeginTx, err)
	}
	defer func() { _ = tx.Rollback() }()

	itemQuery := `
		INSERT INTO contract_items (
			tenant_id, contract_id, service_id, quantity, unit_price,
			discount_pct, start_date, end_date, delivery_date,
			description, notes
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11
		) RETURNING id INTO :12`

	var itemID int64
	_, err = tx.ExecContext(ctx, itemQuery,
		tenantID, contractID, req.ServiceID, req.Quantity, req.UnitPrice,
		req.DiscountPct, req.StartDate, req.EndDate, req.DeliveryDate,
		req.Description, req.Notes,
		sql.Out{Dest: &itemID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert item: %w", err)
	}

	// Update total value
	updateQuery := `
		UPDATE contracts SET 
			total_value = (SELECT SUM(quantity * unit_price * (1 - discount_pct/100)) FROM contract_items WHERE tenant_id = :1 AND contract_id = :2),
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = :3 AND id = :4`
	_, err = tx.ExecContext(ctx, updateQuery, tenantID, contractID, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf(errFmtUpdateTotalVal, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(errFmtCommitTx, err)
	}

	// Get the created item directly by ID
	return r.GetItemByID(ctx, tenantID, contractID, itemID)
}

// GetItemByID retrieves a single contract item by ID
func (r *ContractRepository) GetItemByID(ctx context.Context, tenantID string, contractID, itemID int64) (*models.ContractItem, error) {
	query := `
		SELECT ci.id, ci.tenant_id, ci.contract_id, ci.service_id,
			ci.quantity, ci.unit_price, ci.discount_pct, ci.line_total,
			ci.start_date, ci.end_date, ci.delivery_date,
			ci.description, ci.status, ci.completed_at, ci.notes,
			ci.created_at, ci.updated_at
		FROM contract_items ci
		WHERE ci.tenant_id = :1 AND ci.contract_id = :2 AND ci.id = :3`

	var item models.ContractItem
	var startDate, endDate, deliveryDate, completedAt sql.NullTime
	var description, notes sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, contractID, itemID).Scan(
		&item.ID, &item.TenantID, &item.ContractID, &item.ServiceID,
		&item.Quantity, &item.UnitPrice, &item.DiscountPct, &item.LineTotal,
		&startDate, &endDate, &deliveryDate,
		&description, &item.Status, &completedAt, &notes,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get contract item: %w", err)
	}

	if startDate.Valid {
		item.StartDate = &startDate.Time
	}
	if endDate.Valid {
		item.EndDate = &endDate.Time
	}
	if deliveryDate.Valid {
		item.DeliveryDate = &deliveryDate.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	item.Description = description.String
	item.Notes = notes.String

	return &item, nil
}

// DeleteItem removes an item from a contract
func (r *ContractRepository) DeleteItem(ctx context.Context, tenantID string, contractID, itemID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(errFmtBeginTx, err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `DELETE FROM contract_items WHERE tenant_id = :1 AND contract_id = :2 AND id = :3`, tenantID, contractID, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(errFmtRowsAffected, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	// Update total value
	updateQuery := `
		UPDATE contracts SET 
			total_value = COALESCE((SELECT SUM(quantity * unit_price * (1 - discount_pct/100)) FROM contract_items WHERE tenant_id = :1 AND contract_id = :2), 0),
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = :3 AND id = :4`
	_, err = tx.ExecContext(ctx, updateQuery, tenantID, contractID, tenantID, contractID)
	if err != nil {
		return fmt.Errorf(errFmtUpdateTotalVal, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(errFmtCommitTx, err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
