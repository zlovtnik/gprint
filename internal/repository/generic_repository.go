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

// FilterCondition represents a WHERE clause filter.
// TODO: Implement filtering support in GenericRepository methods (List, Query, etc.)
// Planned for future enhancement to support dynamic WHERE clauses in CRUD operations.
type FilterCondition struct {
	Column   string
	Operator string // =, <>, <, >, <=, >=, LIKE, IN, IS NULL, IS NOT NULL
	Value    any
	Type     string
}

// SortSpec represents an ORDER BY specification.
// TODO: Implement sorting support in GenericRepository methods (List, Query, etc.)
// Planned for future enhancement to support dynamic ORDER BY clauses in CRUD operations.
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
		errorStr := "stored procedure failed"
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
		errorStr := "stored procedure failed"
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
// Note: deletedBy is currently not passed to sp_generic_delete as the stored procedure
// doesn't support it yet. For soft deletes, UPDATED_BY is set via UPDATED_AT trigger or
// you can call Update separately to set DELETED_BY if needed.
func (r *GenericRepository) Delete(
	ctx context.Context,
	tableName string,
	tenantID string,
	id int64,
	softDelete bool,
	deletedBy string, // Currently unused - reserved for future sp_generic_delete enhancement
) (*CRUDResult, error) {
	// Validate table name against allowlist to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}

	soft := 1
	if !softDelete {
		soft = 0
	}

	// TODO: Update sp_generic_delete to accept deletedBy parameter and set DELETED_BY column
	_ = deletedBy // Silence unused variable warning until sp is updated

	query := `
		DECLARE
			v_rows NUMBER;
			v_success NUMBER;
			v_error VARCHAR2(4000);
		BEGIN
			sp_generic_delete(:1, :2, :3, :4, v_rows, v_success, v_error);
			:5 := v_rows;
			:6 := v_success;
			:7 := v_error;
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
		sql.Out{Dest: &rows},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorMsg},
	)
	if err != nil {
		return nil, fmt.Errorf("delete %s: %w", tableName, err)
	}

	// Check stored procedure success status
	if success != 1 {
		errorStr := "stored procedure failed"
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
