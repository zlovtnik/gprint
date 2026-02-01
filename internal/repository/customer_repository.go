package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/zlovtnik/gprint/internal/models"
)

// TableCustomers is the table name for customers.
const TableCustomers = "CUSTOMERS"

// CustomerRepository handles customer data access
type CustomerRepository struct {
	db      *sql.DB
	generic *GenericRepository
}

func appendOptionalStringColumn(columns []ColumnValue, name string, value *string) []ColumnValue {
	if value == nil {
		return columns
	}
	return append(columns, ColumnValue{Name: name, Value: *value})
}

func appendOptionalCustomerTypeColumn(columns []ColumnValue, name string, value *models.CustomerType) []ColumnValue {
	if value == nil {
		return columns
	}
	return append(columns, ColumnValue{Name: name, Value: string(*value)})
}

func appendAddressColumns(columns []ColumnValue, address *models.AddressInput) []ColumnValue {
	if address == nil {
		return columns
	}
	columns = appendOptionalStringColumn(columns, "ADDRESS_STREET", address.Street)
	columns = appendOptionalStringColumn(columns, "ADDRESS_NUMBER", address.Number)
	columns = appendOptionalStringColumn(columns, "ADDRESS_COMP", address.Comp)
	columns = appendOptionalStringColumn(columns, "ADDRESS_DISTRICT", address.District)
	columns = appendOptionalStringColumn(columns, "ADDRESS_CITY", address.City)
	columns = appendOptionalStringColumn(columns, "ADDRESS_STATE", address.State)
	columns = appendOptionalStringColumn(columns, "ADDRESS_ZIP", address.Zip)
	columns = appendOptionalStringColumn(columns, "ADDRESS_COUNTRY", address.Country)
	return columns
}

// NewCustomerRepository creates a new CustomerRepository
func NewCustomerRepository(db *sql.DB) (*CustomerRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("NewCustomerRepository: db is nil")
	}
	return &CustomerRepository{
		db:      db,
		generic: NewGenericRepository(db),
	}, nil
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
	var createdAt, updatedAt sql.NullTime

	err := scanner.Scan(
		&c.ID, &c.TenantID, &c.CustomerCode, &c.CustomerType, &c.Name, &tradeName,
		&taxID, &stateReg, &municipalReg, &email, &phone, &mobile,
		&street, &number, &comp, &district,
		&city, &state, &zip, &country,
		&c.Active, &notes, &createdAt, &updatedAt, &createdBy, &updatedBy,
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
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		c.UpdatedAt = updatedAt.Time
	}

	return &c, nil
}

// Create creates a new customer using dynamic CRUD
func (r *CustomerRepository) Create(ctx context.Context, tenantID string, req *models.CreateCustomerRequest, createdBy string) (*models.Customer, error) {
	columns := []ColumnValue{
		{Name: "CUSTOMER_CODE", Value: req.CustomerCode},
		{Name: "CUSTOMER_TYPE", Value: string(req.CustomerType)},
		{Name: "NAME", Value: req.Name},
		{Name: "ACTIVE", Value: 1, Type: "NUMBER"},
	}

	columns = appendOptionalStringColumn(columns, "TRADE_NAME", req.TradeName)
	columns = appendOptionalStringColumn(columns, "TAX_ID", req.TaxID)
	columns = appendOptionalStringColumn(columns, "STATE_REG", req.StateReg)
	columns = appendOptionalStringColumn(columns, "MUNICIPAL_REG", req.MunicipalReg)
	columns = appendOptionalStringColumn(columns, "EMAIL", req.Email)
	columns = appendOptionalStringColumn(columns, "PHONE", req.Phone)
	columns = appendOptionalStringColumn(columns, "MOBILE", req.Mobile)
	columns = appendAddressColumns(columns, req.Address)
	columns = appendOptionalStringColumn(columns, "NOTES", req.Notes)

	result, err := r.generic.Insert(ctx, TableCustomers, tenantID, columns, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to create customer: %s", result.ErrorMessage)
	}
	if result.GeneratedID == nil {
		return nil, fmt.Errorf("failed to create customer: no ID returned")
	}

	return r.GetByID(ctx, tenantID, *result.GeneratedID)
}

// GetByID retrieves a customer by ID
// Stored procedure sp_get_customer available for ref cursor usage
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
// Stored procedure sp_list_customers available for ref cursor usage
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

// Update updates a customer using dynamic CRUD
func (r *CustomerRepository) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateCustomerRequest, updatedBy string) (*models.Customer, error) {
	var columns []ColumnValue

	columns = appendOptionalCustomerTypeColumn(columns, "CUSTOMER_TYPE", req.CustomerType)
	columns = appendOptionalStringColumn(columns, "NAME", req.Name)
	columns = appendOptionalStringColumn(columns, "TRADE_NAME", req.TradeName)
	columns = appendOptionalStringColumn(columns, "TAX_ID", req.TaxID)
	columns = appendOptionalStringColumn(columns, "STATE_REG", req.StateReg)
	columns = appendOptionalStringColumn(columns, "MUNICIPAL_REG", req.MunicipalReg)
	columns = appendOptionalStringColumn(columns, "EMAIL", req.Email)
	columns = appendOptionalStringColumn(columns, "PHONE", req.Phone)
	columns = appendOptionalStringColumn(columns, "MOBILE", req.Mobile)
	columns = appendAddressColumns(columns, req.Address)

	if len(columns) == 0 {
		return r.GetByID(ctx, tenantID, id)
	}

	result, err := r.generic.Update(ctx, TableCustomers, tenantID, id, columns, updatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to update customer: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("customer not found: tenant=%s id=%d", tenantID, id)
	}

	return r.GetByID(ctx, tenantID, id)
}

// Delete soft-deletes a customer using dynamic CRUD
func (r *CustomerRepository) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	result, err := r.generic.Delete(ctx, TableCustomers, tenantID, id, true, deletedBy)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("failed to delete customer: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("customer not found: tenant=%s id=%d", tenantID, id)
	}
	return nil
}
