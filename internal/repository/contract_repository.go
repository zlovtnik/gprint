package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/models"
)

// ErrNotFound is returned when a requested resource does not exist
var ErrNotFound = errors.New("resource not found")

// Table names for dynamic CRUD operations
const (
	TableContracts     = "CONTRACTS"
	TableContractItems = "CONTRACT_ITEMS"
)

// ContractRepository handles contract data access
type ContractRepository struct {
	db      *sql.DB
	generic *GenericRepository
}

// NewContractRepository creates a new ContractRepository
func NewContractRepository(db *sql.DB) *ContractRepository {
	if db == nil {
		panic("ContractRepository: db is nil")
	}
	return &ContractRepository{
		db:      db,
		generic: NewGenericRepository(db),
	}
}

// decimalToFloat64 converts a decimal to float64 and logs a warning if precision is lost.
func decimalToFloat64(ctx context.Context, fieldName string, d decimal.Decimal) float64 {
	f, exact := d.Float64()
	if !exact {
		// Extract trace ID from context if available (assuming it's stored as a string value)
		traceID := "unknown"
		if traceVal := ctx.Value("trace_id"); traceVal != nil {
			if tid, ok := traceVal.(string); ok {
				traceID = tid
			}
		}

		log.Printf("WARNING: precision loss converting fieldName=%s value=%s traceID=%s", fieldName, d.String(), traceID)
	}
	return f
}

// Create creates a new contract with items using dynamic CRUD
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

	// Build columns dynamically
	columns := []ColumnValue{
		{Name: "CONTRACT_NUMBER", Value: req.ContractNumber},
		{Name: "CONTRACT_TYPE", Value: string(req.ContractType)},
		{Name: "CUSTOMER_ID", Value: req.CustomerID, Type: "NUMBER"},
		{Name: "START_DATE", Value: req.StartDate.Format("2006-01-02"), Type: "DATE"},
		{Name: "DURATION_MONTHS", Value: req.DurationMonths, Type: "NUMBER"},
		{Name: "AUTO_RENEW", Value: boolToInt(req.AutoRenew), Type: "NUMBER"},
		{Name: "BILLING_CYCLE", Value: billingCycleStr},
		{Name: "STATUS", Value: "DRAFT"},
		{Name: "TOTAL_VALUE", Value: 0, Type: "NUMBER"},
	}

	if req.EndDate != nil {
		columns = append(columns, ColumnValue{Name: "END_DATE", Value: req.EndDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.PaymentTerms != "" {
		columns = append(columns, ColumnValue{Name: "PAYMENT_TERMS", Value: req.PaymentTerms})
	}
	if req.Notes != "" {
		columns = append(columns, ColumnValue{Name: "NOTES", Value: req.Notes})
	}
	if req.TermsConditions != "" {
		columns = append(columns, ColumnValue{Name: "TERMS_CONDITIONS", Value: req.TermsConditions})
	}

	// Insert contract using generic CRUD
	result, err := r.generic.Insert(ctx, TableContracts, tenantID, columns, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to create contract: %s", result.ErrorMessage)
	}
	if result.GeneratedID == nil {
		return nil, fmt.Errorf("failed to create contract: no ID returned")
	}

	contractID := *result.GeneratedID

	// Insert contract items
	for _, item := range req.Items {
		if err := r.insertContractItem(ctx, tenantID, contractID, item, createdBy); err != nil {
			return nil, err
		}
	}

	// Update total using aggregate
	aggResult, err := r.generic.UpdateAggregate(ctx, TableContracts, contractID, tenantID, TableContractItems)
	if err != nil {
		return nil, fmt.Errorf(errFmtUpdateTotalVal, err)
	}
	if !aggResult.Success {
		return nil, fmt.Errorf("failed to update total: %s", aggResult.ErrorMessage)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(errFmtCommitTx, err)
	}

	return r.GetByID(ctx, tenantID, contractID)
}

// insertContractItem inserts a single contract item using dynamic CRUD.
func (r *ContractRepository) insertContractItem(ctx context.Context, tenantID string, contractID int64, item models.CreateContractItemRequest, createdBy string) error {
	columns := []ColumnValue{
		{Name: "CONTRACT_ID", Value: contractID, Type: "NUMBER"},
		{Name: "SERVICE_ID", Value: item.ServiceID, Type: "NUMBER"},
		{Name: "QUANTITY", Value: decimalToFloat64(ctx, "Quantity", item.Quantity), Type: "NUMBER"},
		{Name: "UNIT_PRICE", Value: decimalToFloat64(ctx, "UnitPrice", item.UnitPrice), Type: "NUMBER"},
		{Name: "DISCOUNT_PCT", Value: decimalToFloat64(ctx, "DiscountPct", item.DiscountPct), Type: "NUMBER"},
		{Name: "STATUS", Value: "PENDING"},
	}

	if item.StartDate != nil {
		columns = append(columns, ColumnValue{Name: "START_DATE", Value: item.StartDate.Format("2006-01-02"), Type: "DATE"})
	}
	if item.EndDate != nil {
		columns = append(columns, ColumnValue{Name: "END_DATE", Value: item.EndDate.Format("2006-01-02"), Type: "DATE"})
	}
	if item.DeliveryDate != nil {
		columns = append(columns, ColumnValue{Name: "DELIVERY_DATE", Value: item.DeliveryDate.Format("2006-01-02"), Type: "DATE"})
	}
	if item.Description != "" {
		columns = append(columns, ColumnValue{Name: "DESCRIPTION", Value: item.Description})
	}
	if item.Notes != "" {
		columns = append(columns, ColumnValue{Name: "NOTES", Value: item.Notes})
	}

	result, err := r.generic.Insert(ctx, TableContractItems, tenantID, columns, createdBy)
	if err != nil {
		return fmt.Errorf("failed to create contract item: %w", err)
	}
	if !result.Success {
		msg := result.ErrorMessage
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("failed to create contract item: %s", msg)
	}
	return nil
}

// GetByID retrieves a contract by ID with items
func (r *ContractRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.Contract, error) {
	return r.getByIDDirect(ctx, tenantID, id)
}

// getByIDDirect retrieves a contract by ID with items using direct SQL
func (r *ContractRepository) getByIDDirect(ctx context.Context, tenantID string, id int64) (*models.Contract, error) {
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
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&contract.ID, &contract.TenantID, &contract.ContractNumber, &contract.ContractType, &contract.CustomerID,
		&contract.StartDate, &endDate, &durationMonths, &contract.AutoRenew,
		&totalValueFloat, &paymentTerms, &contract.BillingCycle, &contract.Status,
		&signedAt, &signedBy, &documentPath, &documentHash,
		&notes, &termsConditions, &createdAt, &updatedAt, &createdBy, &updatedBy,
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
	if createdAt.Valid {
		contract.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		contract.UpdatedAt = updatedAt.Time
	}

	// Get items
	items, err := r.GetItems(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	contract.Items = items

	return &contract, nil
}

// contractItemScanDest holds scan destinations for contract item queries.
type contractItemScanDest struct {
	item                                          models.ContractItem
	startDate, endDate, deliveryDate, completedAt sql.NullTime
	description, notes                            sql.NullString
	createdAt, updatedAt                          sql.NullTime
}

// scanArgs returns the slice of pointers for sql.Rows.Scan.
func (d *contractItemScanDest) scanArgs() []any {
	return []any{
		&d.item.ID, &d.item.TenantID, &d.item.ContractID, &d.item.ServiceID,
		&d.item.Quantity, &d.item.UnitPrice, &d.item.DiscountPct, &d.item.LineTotal,
		&d.startDate, &d.endDate, &d.deliveryDate,
		&d.description, &d.item.Status, &d.completedAt, &d.notes,
		&d.createdAt, &d.updatedAt,
	}
}

// toContractItem converts scanned nullable fields to a ContractItem.
func (d *contractItemScanDest) toContractItem() models.ContractItem {
	d.item.StartDate = TimeFromNull(d.startDate)
	d.item.EndDate = TimeFromNull(d.endDate)
	d.item.DeliveryDate = TimeFromNull(d.deliveryDate)
	d.item.CompletedAt = TimeFromNull(d.completedAt)
	d.item.Description = StringFromNull(d.description)
	d.item.Notes = StringFromNull(d.notes)
	d.item.CreatedAt = TimeValueFromNull(d.createdAt)
	d.item.UpdatedAt = TimeValueFromNull(d.updatedAt)
	return d.item
}

// GetItems retrieves items for a contract using stored procedure
func (r *ContractRepository) GetItems(ctx context.Context, tenantID string, contractID int64) ([]models.ContractItem, error) {
	// Stored procedure sp_get_contract_items is available for ref cursor usage
	// Using direct query for Go driver compatibility
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
		var dest contractItemScanDest
		if err := rows.Scan(dest.scanArgs()...); err != nil {
			return nil, fmt.Errorf("failed to scan contract item: %w", err)
		}
		items = append(items, dest.toContractItem())
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
	// Empty-map fallback: return "id" as a safe default when no columns are allowed.
	// This differs from the earlier allowed["id"] check, which only triggers when "id"
	// is explicitly in the allowed map. This branch handles the degenerate case.
	return "id"
}

// contractScanDest holds scan destinations for contract queries.
type contractScanDest struct {
	contract                             models.Contract
	endDate, signedAt                    sql.NullTime
	durationMonths                       sql.NullInt64
	signedBy, documentPath, documentHash sql.NullString
	paymentTerms, notes, termsConditions sql.NullString
	createdBy, updatedBy                 sql.NullString
	createdAt, updatedAt                 sql.NullTime
	totalValueFloat                      float64
}

// scanArgs returns the slice of pointers for sql.Rows.Scan.
func (d *contractScanDest) scanArgs() []any {
	return []any{
		&d.contract.ID, &d.contract.TenantID, &d.contract.ContractNumber, &d.contract.ContractType, &d.contract.CustomerID,
		&d.contract.StartDate, &d.endDate, &d.durationMonths, &d.contract.AutoRenew,
		&d.totalValueFloat, &d.paymentTerms, &d.contract.BillingCycle, &d.contract.Status,
		&d.signedAt, &d.signedBy, &d.documentPath, &d.documentHash,
		&d.notes, &d.termsConditions, &d.createdAt, &d.updatedAt, &d.createdBy, &d.updatedBy,
	}
}

// toContract converts scanned nullable fields to a Contract.
func (d *contractScanDest) toContract() models.Contract {
	d.contract.TotalValue = decimal.NewFromFloat(d.totalValueFloat)
	d.contract.EndDate = TimeFromNull(d.endDate)
	d.contract.SignedAt = TimeFromNull(d.signedAt)
	d.contract.DurationMonths = IntFromNullInt64(d.durationMonths)
	d.contract.SignedBy = StringFromNull(d.signedBy)
	d.contract.DocumentPath = StringFromNull(d.documentPath)
	d.contract.DocumentHash = StringFromNull(d.documentHash)
	d.contract.PaymentTerms = StringFromNull(d.paymentTerms)
	d.contract.Notes = StringFromNull(d.notes)
	d.contract.TermsConditions = StringFromNull(d.termsConditions)
	d.contract.CreatedBy = StringFromNull(d.createdBy)
	d.contract.UpdatedBy = StringFromNull(d.updatedBy)
	d.contract.CreatedAt = TimeValueFromNull(d.createdAt)
	d.contract.UpdatedAt = TimeValueFromNull(d.updatedAt)
	return d.contract
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

	// Main query - stored procedure sp_list_contracts available for ref cursor usage
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
		var dest contractScanDest
		if err := rows.Scan(dest.scanArgs()...); err != nil {
			return nil, 0, fmt.Errorf("failed to scan contract: %w", err)
		}
		contracts = append(contracts, dest.toContract())
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate contracts: %w", err)
	}

	return contracts, total, nil
}

// Update updates a contract using dynamic CRUD
func (r *ContractRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateContractRequest, updatedBy string) (*models.Contract, error) {
	var columns []ColumnValue

	if req.ContractType != nil {
		columns = append(columns, ColumnValue{Name: "CONTRACT_TYPE", Value: string(*req.ContractType)})
	}
	if req.StartDate != nil {
		columns = append(columns, ColumnValue{Name: "START_DATE", Value: req.StartDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.EndDate != nil {
		columns = append(columns, ColumnValue{Name: "END_DATE", Value: req.EndDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.DurationMonths != nil {
		columns = append(columns, ColumnValue{Name: "DURATION_MONTHS", Value: *req.DurationMonths, Type: "NUMBER"})
	}
	if req.AutoRenew != nil {
		columns = append(columns, ColumnValue{Name: "AUTO_RENEW", Value: boolToInt(*req.AutoRenew), Type: "NUMBER"})
	}
	// PaymentTerms: nil=no change, &""=clear, &"value"=set
	if req.PaymentTerms != nil {
		columns = append(columns, ColumnValue{Name: "PAYMENT_TERMS", Value: *req.PaymentTerms})
	}
	if req.BillingCycle != nil {
		columns = append(columns, ColumnValue{Name: "BILLING_CYCLE", Value: string(*req.BillingCycle)})
	}
	// Notes: nil=no change, &""=clear, &"value"=set
	if req.Notes != nil {
		columns = append(columns, ColumnValue{Name: "NOTES", Value: *req.Notes})
	}
	// TermsConditions: nil=no change, &""=clear, &"value"=set
	if req.TermsConditions != nil {
		columns = append(columns, ColumnValue{Name: "TERMS_CONDITIONS", Value: *req.TermsConditions})
	}

	if len(columns) == 0 {
		return r.GetByID(ctx, tenantID, id)
	}

	result, err := r.generic.Update(ctx, TableContracts, tenantID, id, columns, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update contract: %w", err)
	}
	if !result.Success {
		if result.ErrorMessage == "Record not found" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update contract: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
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

// AddItem adds an item to a contract using dynamic CRUD
// Note: createdBy is extracted from the caller context; pass empty string if unknown.
func (r *ContractRepository) AddItem(ctx context.Context, tenantID string, contractID int64, req *models.CreateContractItemRequest, createdBy string) (*models.ContractItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf(errFmtBeginTx, err)
	}
	defer func() { _ = tx.Rollback() }()

	columns := []ColumnValue{
		{Name: "CONTRACT_ID", Value: contractID, Type: "NUMBER"},
		{Name: "SERVICE_ID", Value: req.ServiceID, Type: "NUMBER"},
		{Name: "QUANTITY", Value: decimalToFloat64(ctx, "Quantity", req.Quantity), Type: "NUMBER"},
		{Name: "UNIT_PRICE", Value: decimalToFloat64(ctx, "UnitPrice", req.UnitPrice), Type: "NUMBER"},
		{Name: "DISCOUNT_PCT", Value: decimalToFloat64(ctx, "DiscountPct", req.DiscountPct), Type: "NUMBER"},
		{Name: "STATUS", Value: "PENDING"},
	}

	if req.StartDate != nil {
		columns = append(columns, ColumnValue{Name: "START_DATE", Value: req.StartDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.EndDate != nil {
		columns = append(columns, ColumnValue{Name: "END_DATE", Value: req.EndDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.DeliveryDate != nil {
		columns = append(columns, ColumnValue{Name: "DELIVERY_DATE", Value: req.DeliveryDate.Format("2006-01-02"), Type: "DATE"})
	}
	if req.Description != "" {
		columns = append(columns, ColumnValue{Name: "DESCRIPTION", Value: req.Description})
	}
	if req.Notes != "" {
		columns = append(columns, ColumnValue{Name: "NOTES", Value: req.Notes})
	}

	result, err := r.generic.Insert(ctx, TableContractItems, tenantID, columns, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert item: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to insert item: %s", result.ErrorMessage)
	}
	if result.GeneratedID == nil {
		return nil, fmt.Errorf("failed to insert item: no ID returned")
	}

	itemID := *result.GeneratedID

	// Update total using aggregate
	aggResult, err := r.generic.UpdateAggregate(ctx, TableContracts, contractID, tenantID, TableContractItems)
	if err != nil {
		return nil, fmt.Errorf(errFmtUpdateTotalVal, err)
	}
	if !aggResult.Success {
		return nil, fmt.Errorf("failed to update total: %s", aggResult.ErrorMessage)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(errFmtCommitTx, err)
	}

	return r.GetItemByID(ctx, tenantID, contractID, itemID)
}

// GetItemByID retrieves a single contract item by ID
// Stored procedure sp_get_contract_item is available for ref cursor usage
func (r *ContractRepository) GetItemByID(ctx context.Context, tenantID string, contractID, itemID int64) (*models.ContractItem, error) {
	query := `
		SELECT ci.id, ci.tenant_id, ci.contract_id, ci.service_id,
			ci.quantity, ci.unit_price, ci.discount_pct, ci.line_total,
			ci.start_date, ci.end_date, ci.delivery_date,
			ci.description, ci.status, ci.completed_at, ci.notes,
			ci.created_at, ci.updated_at
		FROM contract_items ci
		WHERE ci.tenant_id = :1 AND ci.contract_id = :2 AND ci.id = :3`

	var dest contractItemScanDest
	err := r.db.QueryRowContext(ctx, query, tenantID, contractID, itemID).Scan(dest.scanArgs()...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get contract item: %w", err)
	}

	item := dest.toContractItem()
	return &item, nil
}

// DeleteItem removes an item from a contract using dynamic CRUD
func (r *ContractRepository) DeleteItem(ctx context.Context, tenantID string, contractID, itemID int64, deletedBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(errFmtBeginTx, err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete item (hard delete since contract items don't have ACTIVE column)
	result, err := r.generic.Delete(ctx, TableContractItems, tenantID, itemID, false, deletedBy)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	if !result.Success {
		if result.ErrorMessage == "Record not found" {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete item: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	// Update total using aggregate
	aggResult, err := r.generic.UpdateAggregate(ctx, TableContracts, contractID, tenantID, TableContractItems)
	if err != nil {
		return fmt.Errorf(errFmtUpdateTotalVal, err)
	}
	if !aggResult.Success {
		return fmt.Errorf("failed to update total: %s", aggResult.ErrorMessage)
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
