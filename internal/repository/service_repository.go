package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// ServiceRepository handles service data access
type ServiceRepository struct {
	db *sql.DB
}

// NewServiceRepository creates a new ServiceRepository
func NewServiceRepository(db *sql.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create creates a new service
func (r *ServiceRepository) Create(ctx context.Context, tenantID string, req *models.CreateServiceRequest, createdBy string) (*models.Service, error) {
	currency := req.Currency
	if currency == "" {
		currency = "BRL"
	}
	priceUnit := req.PriceUnit
	if priceUnit == "" {
		priceUnit = models.PriceUnitHour
	}

	query := `
		INSERT INTO services (
			tenant_id, service_code, name, description, category, subcategory,
			unit_price, currency, price_unit, service_code_fiscal,
			iss_rate, irrf_rate, pis_rate, cofins_rate, csll_rate,
			notes, created_by, updated_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, :18
		) RETURNING id INTO :19`

	var id int64
	_, err := r.db.ExecContext(ctx, query,
		tenantID, req.ServiceCode, req.Name, req.Description, req.Category, req.Subcategory,
		req.UnitPrice, currency, string(priceUnit), req.ServiceCodeFiscal,
		req.ISSRate, req.IRRFRate, req.PISRate, req.COFINSRate, req.CSLLRate,
		req.Notes, createdBy, createdBy,
		sql.Out{Dest: &id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a service by ID
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

// Update updates a service
func (r *ServiceRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateServiceRequest, updatedBy string) (*models.Service, error) {
	query := `
		UPDATE services SET
			name = COALESCE(NULLIF(:1, ''), name),
			description = :2,
			category = :3,
			subcategory = :4,
			unit_price = COALESCE(:5, unit_price),
			currency = COALESCE(NULLIF(:6, ''), currency),
			price_unit = COALESCE(NULLIF(:7, ''), price_unit),
			service_code_fiscal = :8,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = :9
		WHERE tenant_id = :10 AND id = :11`

	result, err := r.db.ExecContext(ctx, query,
		req.Name, req.Description, req.Category, req.Subcategory,
		req.UnitPrice, req.Currency, string(req.PriceUnit), req.ServiceCodeFiscal,
		updatedBy, tenantID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	return r.GetByID(ctx, tenantID, id)
}

// Delete soft-deletes a service
func (r *ServiceRepository) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	query := `UPDATE services SET active = 0, updated_at = CURRENT_TIMESTAMP, updated_by = :1 WHERE tenant_id = :2 AND id = :3`
	result, err := r.db.ExecContext(ctx, query, deletedBy, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetCategories retrieves distinct categories
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
