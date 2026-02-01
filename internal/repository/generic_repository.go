// Package repository provides dynamic database access using Oracle stored procedures.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// identifierPattern validates SQL identifiers to prevent SQL injection.
// Only allows alphanumeric characters and underscores, must start with letter or underscore.
var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateIdentifier checks if a string is a valid SQL identifier.
func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(name) > 128 {
		return fmt.Errorf("identifier too long: %q (max 128 chars)", name)
	}
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("invalid identifier: %q (must match ^[A-Za-z_][A-Za-z0-9_]*$)", name)
	}
	return nil
}

// allowedTables contains the whitelist of tables that can be accessed via GenericRepository.
// This prevents SQL injection by ensuring only known, safe table names are accepted.
var allowedTables = map[string]bool{
	"CONTRACTS":           true,
	"CONTRACT_ITEMS":      true,
	"CUSTOMERS":           true,
	"SERVICES":            true,
	"CONTRACT_HISTORY":    true,
	"CONTRACT_PRINT_JOBS": true,
	"CONTRACT_TEMPLATES":  true,
	"GENERATED_CONTRACTS": true,
}

const (
	storedProcFailedMsg = "stored procedure failed"
	queryErrFmt         = "query %s: %w"
)

// validateTableName checks if a table name is in the allowed list.
func validateTableName(name string) error {
	if !allowedTables[strings.ToUpper(name)] {
		return fmt.Errorf("table %q is not in the allowed list for generic operations", name)
	}
	return nil
}

// ColumnValue represents a column name-value pair for dynamic CRUD operations.
type ColumnValue struct {
	Name  string
	Value any
	Type  string // STRING, NUMBER, DATE, TIMESTAMP, NULL
}

// FilterCondition represents a WHERE clause filter used by `Query` and `Count`.
type FilterCondition struct {
	Column   string
	Operator string // =, <>, <, >, <=, >=, LIKE, IN, IS NULL, IS NOT NULL
	Value    any
	Type     string
}

// SortSpec represents an ORDER BY specification used by `Query`.
type SortSpec struct {
	Column    string
	Direction string // ASC or DESC
}

// CRUDResult represents the result of a CRUD operation.
// Note: ErrorCode is populated from the stored procedure's error code output when available.
type CRUDResult struct {
	Success      bool
	GeneratedID  *int64
	RowsAffected int64
	ErrorCode    string // Populated from sp_generic_* error output (e.g., "TABLE_NOT_ALLOWED")
	ErrorMessage string
}

// GenericRepository provides dynamic CRUD operations using pkg_crud.
type GenericRepository struct {
	db *sql.DB
}

// NewGenericRepository creates a new GenericRepository.
func NewGenericRepository(db *sql.DB) *GenericRepository {
	if db == nil {
		panic("GenericRepository: db is nil")
	}
	return &GenericRepository{db: db}
}

// buildColumnValuesSQL creates the t_column_values constructor SQL.
// Returns an error if any column name is not a valid SQL identifier.
func buildColumnValuesSQL(cols []ColumnValue) (string, error) {
	if len(cols) == 0 {
		return "NULL", nil
	}

	var parts []string
	for _, col := range cols {
		// Validate column name to prevent SQL injection
		if err := validateIdentifier(col.Name); err != nil {
			return "", fmt.Errorf("invalid column name: %w", err)
		}

		valueStr := formatValue(col.Value)
		colType := col.Type
		if colType == "" {
			colType = inferType(col.Value)
		}
		parts = append(parts, fmt.Sprintf(
			"t_column_value('%s', %s, '%s')",
			escapeSQLString(col.Name),
			valueStr,
			colType,
		))
	}

	return "t_column_values(" + strings.Join(parts, ", ") + ")", nil
}

// buildColumnsCSV validates and joins column names for pkg_crud.do_query.
func buildColumnsCSV(cols []string) (string, error) {
	if len(cols) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		if err := validateIdentifier(col); err != nil {
			return "", fmt.Errorf("invalid column name: %w", err)
		}
		parts = append(parts, col)
	}

	return strings.Join(parts, ","), nil
}

// buildFilterConditionsSQL creates the t_filter_conditions constructor SQL.
// Returns "NULL" when no filters are provided.
func buildFilterConditionsSQL(filters []FilterCondition) (string, error) {
	if len(filters) == 0 {
		return "NULL", nil
	}

	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		if err := validateIdentifier(f.Column); err != nil {
			return "", fmt.Errorf("invalid filter column: %w", err)
		}

		op := strings.ToUpper(strings.TrimSpace(f.Operator))
		switch op {
		case "=", "<>", "<", ">", "<=", ">=", "LIKE", "IN", "IS NULL", "IS NOT NULL":
			// allowed
		default:
			return "", fmt.Errorf("invalid filter operator: %q", f.Operator)
		}

		valueType := f.Type
		if valueType == "" {
			valueType = inferType(f.Value)
		}

		valueExpr := "NULL"
		if op != "IS NULL" && op != "IS NOT NULL" {
			valueExpr = formatValue(f.Value)
		}

		parts = append(parts, fmt.Sprintf(
			"t_filter_condition('%s', '%s', %s, '%s')",
			escapeSQLString(f.Column),
			op,
			valueExpr,
			escapeSQLString(valueType),
		))
	}

	return "t_filter_conditions(" + strings.Join(parts, ", ") + ")", nil
}

// buildSortSpecsSQL creates the t_sort_specs constructor SQL.
// Returns "NULL" when no sorts are provided.
func buildSortSpecsSQL(sorts []SortSpec) (string, error) {
	if len(sorts) == 0 {
		return "NULL", nil
	}

	parts := make([]string, 0, len(sorts))
	for _, s := range sorts {
		if err := validateIdentifier(s.Column); err != nil {
			return "", fmt.Errorf("invalid sort column: %w", err)
		}
		dir := strings.ToUpper(strings.TrimSpace(s.Direction))
		if dir == "" {
			dir = "ASC"
		}
		if dir != "ASC" && dir != "DESC" {
			return "", fmt.Errorf("invalid sort direction: %q", s.Direction)
		}

		parts = append(parts, fmt.Sprintf(
			"t_sort_spec('%s', '%s')",
			escapeSQLString(s.Column),
			dir,
		))
	}

	return "t_sort_specs(" + strings.Join(parts, ", ") + ")", nil
}

// formatValue converts a Go value to SQL literal.
func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + escapeSQLString(val) + "'"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "'1'"
		}
		return "'0'"
	default:
		return "'" + escapeSQLString(fmt.Sprintf("%v", val)) + "'"
	}
}

// escapeSQLString escapes problematic characters in SQL strings.
// Handles single quotes, null bytes, and backslashes for Oracle compatibility.
func escapeSQLString(s string) string {
	// Remove null bytes which can cause issues
	s = strings.ReplaceAll(s, "\x00", "")
	// Escape backslashes (Oracle treats them specially in some contexts)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape single quotes by doubling them
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

// inferType determines the value type for Oracle.
func inferType(v any) string {
	if v == nil {
		return "NULL"
	}
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return "NUMBER"
	case bool:
		return "NUMBER"
	default:
		return "STRING"
	}
}

// Insert performs a generic INSERT operation.
func (r *GenericRepository) Insert(
	ctx context.Context,
	tableName string,
	tenantID string,
	columns []ColumnValue,
	createdBy string,
) (*CRUDResult, error) {
	// Validate table name against allowlist to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}

	colsSQL, err := buildColumnValuesSQL(columns)
	if err != nil {
		return nil, fmt.Errorf("insert %s: %w", tableName, err)
	}

	query := fmt.Sprintf(`
		DECLARE
			v_cols t_column_values := %s;
			v_id NUMBER;
			v_success NUMBER;
			v_error VARCHAR2(4000);
		BEGIN
			sp_generic_insert(:1, :2, v_cols, :3, v_id, v_success, v_error);
			:4 := v_id;
			:5 := v_success;
			:6 := v_error;
		END;
	`, colsSQL)

	var id sql.NullInt64
	var success int
	var errorMsg sql.NullString

	_, err = r.db.ExecContext(ctx, query,
		tableName,
		tenantID,
		sql.NullString{String: createdBy, Valid: createdBy != ""},
		sql.Out{Dest: &id},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("insert %s: %w", tableName, err)
	}

	// Check stored procedure success status
	if success != 1 {
		errorStr := storedProcFailedMsg
		if errorMsg.Valid && errorMsg.String != "" {
			errorStr = errorMsg.String
		}
		return nil, fmt.Errorf("insert %s: %s", tableName, errorStr)
	}

	result := &CRUDResult{
		Success:      true,
		RowsAffected: 1, // INSERT affects 1 row on success
	}
	if id.Valid {
		result.GeneratedID = &id.Int64
	}
	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
		// Extract error code from message if it follows pattern "CODE: message"
		if idx := strings.Index(errorMsg.String, ":"); idx > 0 && idx < 30 {
			result.ErrorCode = strings.TrimSpace(errorMsg.String[:idx])
		}
	}

	return result, nil
}

// Update performs a generic UPDATE operation.
func (r *GenericRepository) Update(
	ctx context.Context,
	tableName string,
	tenantID string,
	id int64,
	columns []ColumnValue,
	updatedBy string,
) (*CRUDResult, error) {
	// Validate table name against allowlist to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	colsSQL, err := buildColumnValuesSQL(columns)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", tableName, err)
	}

	query := fmt.Sprintf(`
		DECLARE
			v_cols t_column_values := %s;
			v_rows NUMBER;
			v_success NUMBER;
			v_error VARCHAR2(4000);
		BEGIN
			sp_generic_update(:1, :2, :3, v_cols, :4, v_rows, v_success, v_error);
			:5 := v_rows;
			:6 := v_success;
			:7 := v_error;
		END;
	`, colsSQL)

	var rows int64
	var success int
	var errorMsg sql.NullString

	_, err = r.db.ExecContext(ctx, query,
		tableName,
		tenantID,
		id,
		sql.NullString{String: updatedBy, Valid: updatedBy != ""},
		sql.Out{Dest: &rows},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", tableName, err)
	}

	// Check stored procedure success status
	if success != 1 {
		errorStr := storedProcFailedMsg
		if errorMsg.Valid && errorMsg.String != "" {
			errorStr = errorMsg.String
		}
		return nil, fmt.Errorf("update %s: %s", tableName, errorStr)
	}

	result := &CRUDResult{
		Success:      true,
		RowsAffected: rows,
	}
	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
		// Extract error code from message if it follows pattern "CODE: message"
		if idx := strings.Index(errorMsg.String, ":"); idx > 0 && idx < 30 {
			result.ErrorCode = strings.TrimSpace(errorMsg.String[:idx])
		}
	}

	return result, nil
}

// Delete performs a generic DELETE operation (soft delete by default).
// If the table supports DELETED_BY, the value is passed to sp_generic_delete.
func (r *GenericRepository) Delete(
	ctx context.Context,
	tableName string,
	tenantID string,
	id int64,
	softDelete bool,
	deletedBy string,
) (*CRUDResult, error) {
	// Validate table name against allowlist to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}

	soft := 1
	if !softDelete {
		soft = 0
	}

	query := `
		DECLARE
			v_rows NUMBER;
			v_success NUMBER;
			v_error VARCHAR2(4000);
		BEGIN
			sp_generic_delete(:1, :2, :3, :4, :5, v_rows, v_success, v_error);
			:6 := v_rows;
			:7 := v_success;
			:8 := v_error;
		END;
	`

	var rows int64
	var success int
	var errorMsg sql.NullString

	_, err := r.db.ExecContext(ctx, query,
		tableName,
		tenantID,
		id,
		soft,
		sql.NullString{String: deletedBy, Valid: deletedBy != ""},
		sql.Out{Dest: &rows},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("delete %s: %w", tableName, err)
	}

	// Check stored procedure success status
	if success != 1 {
		errorStr := storedProcFailedMsg
		if errorMsg.Valid && errorMsg.String != "" {
			errorStr = errorMsg.String
		}
		return nil, fmt.Errorf("delete %s: %s", tableName, errorStr)
	}

	result := &CRUDResult{
		Success:      true,
		RowsAffected: rows,
	}
	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
		// Extract error code from message if it follows pattern "CODE: message"
		if idx := strings.Index(errorMsg.String, ":"); idx > 0 && idx < 30 {
			result.ErrorCode = strings.TrimSpace(errorMsg.String[:idx])
		}
	}

	return result, nil
}

// UpdateAggregate recalculates an aggregate column in a parent table.
func (r *GenericRepository) UpdateAggregate(
	ctx context.Context,
	parentTable string,
	parentID int64,
	tenantID string,
	childTable string,
) (*CRUDResult, error) {
	query := `
		DECLARE
			v_success NUMBER;
			v_error VARCHAR2(4000);
		BEGIN
			sp_generic_aggregate(:1, :2, :3, :4, v_success, v_error);
			:5 := v_success;
			:6 := v_error;
		END;
	`

	var success int
	var errorMsg sql.NullString

	_, err := r.db.ExecContext(ctx, query,
		parentTable,
		parentID,
		tenantID,
		childTable,
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate %s<-%s: %w", parentTable, childTable, err)
	}

	result := &CRUDResult{
		Success: success == 1,
	}
	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
	}

	return result, nil
}

// QueryOptions defines optional parameters for Query.
type QueryOptions struct {
	Columns []string
	Filters []FilterCondition
	Sort    []SortSpec
	Offset  int
	Limit   int
}

// Query retrieves rows from a table using pkg_crud.do_query with optional filters and sorting.
// Note: The caller is responsible for closing the returned rows.
func (r *GenericRepository) Query(
	ctx context.Context,
	tableName string,
	tenantID string,
	opts QueryOptions,
) (*sql.Rows, error) {
	if err := validateTableName(tableName); err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	if opts.Offset < 0 || opts.Limit < 0 {
		return nil, fmt.Errorf("invalid query options: negative offset/limit")
	}

	colsCSV, err := buildColumnsCSV(opts.Columns)
	if err != nil {
		return nil, fmt.Errorf(queryErrFmt, tableName, err)
	}

	filtersSQL, err := buildFilterConditionsSQL(opts.Filters)
	if err != nil {
		return nil, fmt.Errorf(queryErrFmt, tableName, err)
	}

	sortSQL, err := buildSortSpecsSQL(opts.Sort)
	if err != nil {
		return nil, fmt.Errorf(queryErrFmt, tableName, err)
	}

	query := fmt.Sprintf(`
		BEGIN
			:1 := pkg_crud.do_query(:2, :3, :4, %s, %s, :5, :6);
		END;
	`, filtersSQL, sortSQL)

	var cursor *sql.Rows
	_, err = r.db.ExecContext(ctx, query,
		sql.Out{Dest: &cursor},
		tableName,
		tenantID,
		sql.NullString{String: colsCSV, Valid: colsCSV != ""},
		opts.Offset,
		opts.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf(queryErrFmt, tableName, err)
	}

	return cursor, nil
}

// Count returns the number of rows matching the provided filters using pkg_crud.do_count.
func (r *GenericRepository) Count(
	ctx context.Context,
	tableName string,
	tenantID string,
	filters []FilterCondition,
) (int64, error) {
	if err := validateTableName(tableName); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}

	filtersSQL, err := buildFilterConditionsSQL(filters)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", tableName, err)
	}

	query := fmt.Sprintf(`
		BEGIN
			:1 := pkg_crud.do_count(:2, :3, %s);
		END;
	`, filtersSQL)

	var count int64
	_, err = r.db.ExecContext(ctx, query,
		sql.Out{Dest: &count},
		tableName,
		tenantID,
	)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", tableName, err)
	}

	return count, nil
}
