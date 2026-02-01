package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// TableServices is the table name for services.
const TableServices = "SERVICES"

// ServiceRepository handles service data access.
// Uses direct SQL reads (GetByID, List, GetCategories) via db for performance/control,
// and delegates writes (Create, Update, Delete) to generic via the GenericRepository.
type ServiceRepository struct {
	db      *sql.DB
	generic *GenericRepository
}

// NewServiceRepository creates a new ServiceRepository
func NewServiceRepository(db *sql.DB) *ServiceRepository {
	if db == nil {
		panic("ServiceRepository: db is nil")
	}
	return &ServiceRepository{
		db:      db,
		generic: NewGenericRepository(db),
	}
}

// buildCreateServiceColumns builds the column values for creating a service
func buildCreateServiceColumns(req *models.CreateServiceRequest) []ColumnValue {
	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}
	priceUnit := req.PriceUnit
	if priceUnit == "" {
		priceUnit = models.PriceUnitHour
	}

	columns := []ColumnValue{
		{Name: "SERVICE_CODE", Value: req.ServiceCode},
		{Name: "NAME", Value: req.Name},
		{Name: "UNIT_PRICE", Value: req.UnitPrice, Type: "NUMBER"},
		{Name: "CURRENCY", Value: currency},
		{Name: "PRICE_UNIT", Value: string(priceUnit)},
		{Name: "ACTIVE", Value: 1, Type: "NUMBER"},
	}

	// Optional string fields
	if req.Description != "" {
		columns = append(columns, ColumnValue{Name: "DESCRIPTION", Value: req.Description})
	}
	if req.Category != "" {
		columns = append(columns, ColumnValue{Name: "CATEGORY", Value: req.Category})
	}
	if req.Subcategory != "" {
		columns = append(columns, ColumnValue{Name: "SUBCATEGORY", Value: req.Subcategory})
	}
	if req.ServiceCodeFiscal != "" {
		columns = append(columns, ColumnValue{Name: "SERVICE_CODE_FISCAL", Value: req.ServiceCodeFiscal})
	}
	columns = appendRateColumn(columns, "ISS_RATE", req.ISSRate)
	columns = appendRateColumn(columns, "IRRF_RATE", req.IRRFRate)
	columns = appendRateColumn(columns, "PIS_RATE", req.PISRate)
	columns = appendRateColumn(columns, "COFINS_RATE", req.COFINSRate)
	columns = appendRateColumn(columns, "CSLL_RATE", req.CSLLRate)
	if req.Notes != "" {
		columns = append(columns, ColumnValue{Name: "NOTES", Value: req.Notes})
	}

	// Rate fields: nil=not provided, 0=explicit 0% rate (e.g., tax-exempt)
	columns = appendRateColumn(columns, "ISS_RATE", req.ISSRate)
	columns = appendRateColumn(columns, "IRRF_RATE", req.IRRFRate)
	columns = appendRateColumn(columns, "PIS_RATE", req.PISRate)
	columns = appendRateColumn(columns, "COFINS_RATE", req.COFINSRate)
	columns = appendRateColumn(columns, "CSLL_RATE", req.CSLLRate)

	return columns
}

// appendRateColumn appends a rate column if the value is not nil
func appendRateColumn(columns []ColumnValue, name string, rate *float64) []ColumnValue {
	if rate != nil {
		columns = append(columns, ColumnValue{Name: name, Value: *rate, Type: "NUMBER"})
	}
	return columns
}

// Create creates a new service using dynamic CRUD
func (r *ServiceRepository) Create(ctx context.Context, tenantID string, req *models.CreateServiceRequest, createdBy string) (*models.Service, error) {
	columns := buildCreateServiceColumns(req)

	result, err := r.generic.Insert(ctx, TableServices, tenantID, columns, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	if !result.Success {
		msg := result.ErrorMessage
		if msg == "" {
			msg = "unknown error creating service"
		}
		return nil, fmt.Errorf("failed to create service: %s", msg)
	}
	if result.GeneratedID == nil {
		return nil, fmt.Errorf("failed to create service: no ID returned")
	}

	return r.GetByID(ctx, tenantID, *result.GeneratedID)
}

// GetByID retrieves a service by ID.
// NOTE: Uses direct SQL for Go driver compatibility. Create and Update now use
// dynamic CRUD helpers (generic.Insert and generic.Update).
// Stored procedure sp_get_service is available but not used here.
// FUTURE: Migrate to sp_get_service if/when ref cursor handling is needed.
func (r *ServiceRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.Service, error) {
	query := `
		SELECT id, tenant_id, service_code, name, description, category, subcategory,
			unit_price, currency, price_unit, service_code_fiscal,
			iss_rate, irrf_rate, pis_rate, cofins_rate, csll_rate,
			active, notes, created_at, updated_at, created_by, updated_by
		FROM services
		WHERE tenant_id = :1 AND id = :2`

	var s models.Service
	var description, category, subcategory, serviceCodeFiscal sql.NullString
	var notes, createdBy, updatedBy sql.NullString
	var createdAt, updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&s.ID, &s.TenantID, &s.ServiceCode, &s.Name, &description, &category, &subcategory,
		&s.UnitPrice, &s.Currency, &s.PriceUnit, &serviceCodeFiscal,
		&s.ISSRate, &s.IRRFRate, &s.PISRate, &s.COFINSRate, &s.CSLLRate,
		&s.Active, &notes, &createdAt, &updatedAt, &createdBy, &updatedBy,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	s.Description = description.String
	s.Category = category.String
	s.Subcategory = subcategory.String
	s.ServiceCodeFiscal = serviceCodeFiscal.String
	s.Notes = notes.String
	s.CreatedBy = createdBy.String
	s.UpdatedBy = updatedBy.String
	if createdAt.Valid {
		s.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		s.UpdatedAt = updatedAt.Time
	}

	return &s, nil
}

// serviceListAllowedSorts defines valid sort columns for service listing
var serviceListAllowedSorts = map[string]bool{
	"name":         true,
	"service_code": true,
	"category":     true,
	"unit_price":   true,
	"created_at":   true,
}

// getServiceSortClause returns validated sort column and direction
func getServiceSortClause(sortBy, sortDir string) (string, string) {
	col := "created_at"
	if sortBy != "" && serviceListAllowedSorts[sortBy] {
		col = sortBy
	}
	dir := "DESC"
	if strings.ToUpper(sortDir) == "ASC" {
		dir = "ASC"
	}
	return col, dir
}

// List retrieves services with pagination
// Stored procedure sp_list_services available for ref cursor usage
func (r *ServiceRepository) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Service, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM services WHERE tenant_id = :1`
	args := []any{tenantID}
	argIndex := 2

	if search.Query != "" {
		countQuery += fmt.Sprintf(" AND UPPER(name) LIKE UPPER(:%d)", argIndex)
		args = append(args, "%"+search.Query+"%")
		argIndex++
	}

	if search.Active != nil {
		countQuery += fmt.Sprintf(" AND active = :%d", argIndex)
		args = append(args, boolToInt(*search.Active))
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}

	// Main query
	query := `
		SELECT id, tenant_id, service_code, name, description, category, subcategory,
			unit_price, currency, price_unit, service_code_fiscal,
			iss_rate, irrf_rate, pis_rate, cofins_rate, csll_rate,
			active, notes, created_at, updated_at, created_by, updated_by
		FROM services
		WHERE tenant_id = :1`

	queryArgs := []any{tenantID}
	queryArgIndex := 2

	if search.Query != "" {
		query += fmt.Sprintf(" AND UPPER(name) LIKE UPPER(:%d)", queryArgIndex)
		queryArgs = append(queryArgs, "%"+search.Query+"%")
		queryArgIndex++
	}

	if search.Active != nil {
		query += fmt.Sprintf(" AND active = :%d", queryArgIndex)
		queryArgs = append(queryArgs, boolToInt(*search.Active))
		queryArgIndex++
	}

	// Sorting
	sortBy, sortDir := getServiceSortClause(search.SortBy, search.SortDir)
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	// Pagination
	query += fmt.Sprintf(" OFFSET :%d ROWS FETCH NEXT :%d ROWS ONLY", queryArgIndex, queryArgIndex+1)
	queryArgs = append(queryArgs, params.Offset(), params.Limit())

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list services: %w", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		var description, category, subcategory, serviceCodeFiscal sql.NullString
		var notes, createdBy, updatedBy sql.NullString
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ServiceCode, &s.Name, &description, &category, &subcategory,
			&s.UnitPrice, &s.Currency, &s.PriceUnit, &serviceCodeFiscal,
			&s.ISSRate, &s.IRRFRate, &s.PISRate, &s.COFINSRate, &s.CSLLRate,
			&s.Active, &notes, &createdAt, &updatedAt, &createdBy, &updatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan service: %w", err)
		}

		s.Description = description.String
		s.Category = category.String
		s.Subcategory = subcategory.String
		s.ServiceCodeFiscal = serviceCodeFiscal.String
		s.Notes = notes.String
		s.CreatedBy = createdBy.String
		s.UpdatedBy = updatedBy.String
		if createdAt.Valid {
			s.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			s.UpdatedAt = updatedAt.Time
		}

		services = append(services, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate services: %w", err)
	}

	return services, total, nil
}

// Update updates a service using dynamic CRUD
func (r *ServiceRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateServiceRequest, updatedBy string) (*models.Service, error) {
	var columns []ColumnValue

	if req.Name != "" {
		columns = append(columns, ColumnValue{Name: "NAME", Value: req.Name})
	}
	if req.Description != "" {
		columns = append(columns, ColumnValue{Name: "DESCRIPTION", Value: req.Description})
	}
	if req.Category != "" {
		columns = append(columns, ColumnValue{Name: "CATEGORY", Value: req.Category})
	}
	if req.Subcategory != "" {
		columns = append(columns, ColumnValue{Name: "SUBCATEGORY", Value: req.Subcategory})
	}
	if req.UnitPrice != nil {
		columns = append(columns, ColumnValue{Name: "UNIT_PRICE", Value: *req.UnitPrice, Type: "NUMBER"})
	}
	if req.Currency != "" {
		columns = append(columns, ColumnValue{Name: "CURRENCY", Value: req.Currency})
	}
	if req.PriceUnit != "" {
		columns = append(columns, ColumnValue{Name: "PRICE_UNIT", Value: string(req.PriceUnit)})
	}
	if req.ServiceCodeFiscal != "" {
		columns = append(columns, ColumnValue{Name: "SERVICE_CODE_FISCAL", Value: req.ServiceCodeFiscal})
	}

	if len(columns) == 0 {
		return r.GetByID(ctx, tenantID, id)
	}

	result, err := r.generic.Update(ctx, TableServices, tenantID, id, columns, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}
	if !result.Success {
		msg := result.ErrorMessage
		if msg == "" {
			msg = "unknown error updating service"
		}
		return nil, fmt.Errorf("failed to update service: %s", msg)
	}
	if result.RowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetByID(ctx, tenantID, id)
}

// Delete soft-deletes a service using dynamic CRUD.
func (r *ServiceRepository) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	result, err := r.generic.Delete(ctx, TableServices, tenantID, id, true, deletedBy)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("failed to delete service: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetCategories retrieves distinct categories
// Stored procedure sp_get_service_categories available for ref cursor usage
func (r *ServiceRepository) GetCategories(ctx context.Context, tenantID string) ([]string, error) {
	query := `SELECT DISTINCT category FROM services WHERE tenant_id = :1 AND category IS NOT NULL AND active = 1 ORDER BY category`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate categories: %w", err)
	}

	return categories, nil
}
