package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// CustomerRepository handles customer data access
type CustomerRepository struct {
	db *sql.DB
}

// NewCustomerRepository creates a new CustomerRepository
func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// searchCondition holds a SQL condition fragment and its argument
type searchCondition struct {
	clause string
	arg    any
}

// buildSearchConditions builds WHERE clause conditions from search params
func buildSearchConditions(search models.SearchParams, startIndex int) ([]searchCondition, int) {
	var conditions []searchCondition
	argIndex := startIndex

	if search.Query != "" {
		conditions = append(conditions, searchCondition{
			clause: fmt.Sprintf(" AND UPPER(name) LIKE UPPER(:%d)", argIndex),
			arg:    "%" + search.Query + "%",
		})
		argIndex++
	}

	if search.Active != nil {
		activeVal := 0
		if *search.Active {
			activeVal = 1
		}
		conditions = append(conditions, searchCondition{
			clause: fmt.Sprintf(" AND active = :%d", argIndex),
			arg:    activeVal,
		})
		argIndex++
	}

	return conditions, argIndex
}

// applySorting returns the ORDER BY clause for customer queries
func applySorting(search models.SearchParams) string {
	sortBy := "created_at"
	allowedSorts := map[string]bool{"name": true, "customer_code": true, "created_at": true, "updated_at": true}
	if search.SortBy != "" && allowedSorts[search.SortBy] {
		sortBy = search.SortBy
	}

	sortDir := "DESC"
	if strings.ToUpper(search.SortDir) == "ASC" {
		sortDir = "ASC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)
}

// scanCustomer scans a row into a Customer struct
func scanCustomer(scanner interface{ Scan(...any) error }) (*models.Customer, error) {
	var c models.Customer
	var tradeName, taxID, stateReg, municipalReg, email, phone, mobile sql.NullString
	var street, number, comp, district, city, state, zip, country sql.NullString
	var notes, createdBy, updatedBy sql.NullString

	err := scanner.Scan(
		&c.ID, &c.TenantID, &c.CustomerCode, &c.CustomerType, &c.Name, &tradeName,
		&taxID, &stateReg, &municipalReg, &email, &phone, &mobile,
		&street, &number, &comp, &district,
		&city, &state, &zip, &country,
		&c.Active, &notes, &c.CreatedAt, &c.UpdatedAt, &createdBy, &updatedBy,
	)
	if err != nil {
		return nil, err
	}

	c.TradeName = tradeName.String
	c.TaxID = taxID.String
	c.StateReg = stateReg.String
	c.MunicipalReg = municipalReg.String
	c.Email = email.String
	c.Phone = phone.String
	c.Mobile = mobile.String
	c.Address = &models.Address{
		Street:   street.String,
		Number:   number.String,
		Comp:     comp.String,
		District: district.String,
		City:     city.String,
		State:    state.String,
		Zip:      zip.String,
		Country:  country.String,
	}
	c.Notes = notes.String
	c.CreatedBy = createdBy.String
	c.UpdatedBy = updatedBy.String

	return &c, nil
}

// Create creates a new customer
func (r *CustomerRepository) Create(ctx context.Context, tenantID string, req *models.CreateCustomerRequest, createdBy string) (*models.Customer, error) {
	// Address is a value type in CreateCustomerRequest
	address := req.Address

	query := `
		INSERT INTO customers (
			tenant_id, customer_code, customer_type, name, trade_name,
			tax_id, state_reg, municipal_reg, email, phone, mobile,
			address_street, address_number, address_comp, address_district,
			address_city, address_state, address_zip, address_country,
			notes, created_by, updated_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11,
			:12, :13, :14, :15, :16, :17, :18, :19, :20, :21, :22
		) RETURNING id INTO :23`

	var id int64
	_, err := r.db.ExecContext(ctx, query,
		tenantID, req.CustomerCode, req.CustomerType, req.Name, req.TradeName,
		req.TaxID, req.StateReg, req.MunicipalReg, req.Email, req.Phone, req.Mobile,
		address.Street, address.Number, address.Comp, address.District,
		address.City, address.State, address.Zip, address.Country,
		req.Notes, createdBy, createdBy,
		sql.Out{Dest: &id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.Customer, error) {
	query := `
		SELECT id, tenant_id, customer_code, customer_type, name, trade_name,
			tax_id, state_reg, municipal_reg, email, phone, mobile,
			address_street, address_number, address_comp, address_district,
			address_city, address_state, address_zip, address_country,
			active, notes, created_at, updated_at, created_by, updated_by
		FROM customers
		WHERE tenant_id = :1 AND id = :2`

	customer, err := scanCustomer(r.db.QueryRowContext(ctx, query, tenantID, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return customer, nil
}

// List retrieves customers with pagination
func (r *CustomerRepository) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Customer, int, error) {
	conditions, lastIndex := buildSearchConditions(search, 2)

	// Count query
	total, err := r.countCustomers(ctx, tenantID, conditions)
	if err != nil {
		return nil, 0, err
	}

	// Main query
	customers, err := r.fetchCustomers(ctx, tenantID, conditions, search, params, lastIndex)
	if err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

// countCustomers executes the count query with search conditions
func (r *CustomerRepository) countCustomers(ctx context.Context, tenantID string, conditions []searchCondition) (int, error) {
	countQuery := `SELECT COUNT(*) FROM customers WHERE tenant_id = :1`
	args := []any{tenantID}

	for _, cond := range conditions {
		countQuery += cond.clause
		args = append(args, cond.arg)
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count customers: %w", err)
	}
	return total, nil
}

// fetchCustomers executes the main query with search, sorting, and pagination
func (r *CustomerRepository) fetchCustomers(ctx context.Context, tenantID string, conditions []searchCondition, search models.SearchParams, params models.PaginationParams, argIndex int) ([]models.Customer, error) {
	query := `
		SELECT id, tenant_id, customer_code, customer_type, name, trade_name,
			tax_id, state_reg, municipal_reg, email, phone, mobile,
			address_street, address_number, address_comp, address_district,
			address_city, address_state, address_zip, address_country,
			active, notes, created_at, updated_at, created_by, updated_by
		FROM customers
		WHERE tenant_id = :1`

	queryArgs := []any{tenantID}
	for _, cond := range conditions {
		query += cond.clause
		queryArgs = append(queryArgs, cond.arg)
	}

	query += applySorting(search)
	query += fmt.Sprintf(" OFFSET :%d ROWS FETCH NEXT :%d ROWS ONLY", argIndex, argIndex+1)
	queryArgs = append(queryArgs, params.Offset(), params.Limit())

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		customer, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, *customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate customers: %w", err)
	}

	return customers, nil
}

// Update updates a customer
func (r *CustomerRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateCustomerRequest, updatedBy string) (*models.Customer, error) {
	// Address is a value type in UpdateCustomerRequest
	address := req.Address

	query := `
		UPDATE customers SET
			customer_type = COALESCE(NULLIF(:1, ''), customer_type),
			name = COALESCE(NULLIF(:2, ''), name),
			trade_name = :3,
			tax_id = :4,
			state_reg = :5,
			municipal_reg = :6,
			email = :7,
			phone = :8,
			mobile = :9,
			address_street = COALESCE(NULLIF(:10, ''), address_street),
			address_number = COALESCE(NULLIF(:11, ''), address_number),
			address_comp = :12,
			address_district = COALESCE(NULLIF(:13, ''), address_district),
			address_city = COALESCE(NULLIF(:14, ''), address_city),
			address_state = COALESCE(NULLIF(:15, ''), address_state),
			address_zip = COALESCE(NULLIF(:16, ''), address_zip),
			address_country = COALESCE(NULLIF(:17, ''), address_country),
			updated_at = CURRENT_TIMESTAMP,
			updated_by = :18
		WHERE tenant_id = :19 AND id = :20`

	result, err := r.db.ExecContext(ctx, query,
		req.CustomerType, req.Name, req.TradeName, req.TaxID,
		req.StateReg, req.MunicipalReg, req.Email, req.Phone, req.Mobile,
		address.Street, address.Number, address.Comp, address.District,
		address.City, address.State, address.Zip, address.Country,
		updatedBy, tenantID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("customer not found: tenant=%s id=%d", tenantID, id)
	}

	return r.GetByID(ctx, tenantID, id)
}

// Delete soft-deletes a customer
func (r *CustomerRepository) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	query := `UPDATE customers SET active = 0, updated_at = CURRENT_TIMESTAMP, updated_by = :1 WHERE tenant_id = :2 AND id = :3`
	result, err := r.db.ExecContext(ctx, query, deletedBy, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("customer not found: tenant=%s id=%d", tenantID, id)
	}
	return nil
}
