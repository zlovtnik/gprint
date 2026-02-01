package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ═══════════════════════════════════════════════════════════════════════════
// SQL NULL HELPERS - Conversion between Go types and sql.Null* types
// ═══════════════════════════════════════════════════════════════════════════

// nullableString returns a sql.NullString for a string value.
// Empty strings result in a NULL database value.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullableUUID returns a sql.NullString for a UUID-based ID.
func nullableUUID[T ~[16]byte](id *T) sql.NullString {
	if id == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: uuid.UUID(*id).String(), Valid: true}
}

// ═══════════════════════════════════════════════════════════════════════════
// BOOLEAN HELPERS - Oracle doesn't have native BOOLEAN, uses NUMBER(1)
// ═══════════════════════════════════════════════════════════════════════════

// boolToInt converts a boolean to an int for Oracle NUMBER(1) storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// intToBool converts an Oracle NUMBER(1) to boolean.
func intToBool(i int) bool {
	return i == 1
}

// ═══════════════════════════════════════════════════════════════════════════
// UUID PARSING HELPERS - Parse UUIDs with typed domain IDs
// ═══════════════════════════════════════════════════════════════════════════

// parseUUID parses a string to uuid.UUID with a descriptive error.
func parseUUID(s, fieldName string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse %s %q: %w", fieldName, s, err)
	}
	return id, nil
}

// parseContractID parses a string to ContractID.
func parseContractID(s string) fp.Result[domain.ContractID] {
	id, err := parseUUID(s, "contract_id")
	if err != nil {
		return fp.Failure[domain.ContractID](err)
	}
	return fp.Success(domain.ContractID(id))
}

// parsePartyID parses a string to PartyID.
func parsePartyID(s string) fp.Result[domain.PartyID] {
	id, err := parseUUID(s, "party_id")
	if err != nil {
		return fp.Failure[domain.PartyID](err)
	}
	return fp.Success(domain.PartyID(id))
}

// parseUserID parses a string to UserID.
func parseUserID(s string) fp.Result[domain.UserID] {
	id, err := parseUUID(s, "user_id")
	if err != nil {
		return fp.Failure[domain.UserID](err)
	}
	return fp.Success(domain.UserID(id))
}

// parseWorkflowID parses a string to WorkflowID.
func parseWorkflowID(s string) fp.Result[domain.WorkflowID] {
	id, err := parseUUID(s, "workflow_id")
	if err != nil {
		return fp.Failure[domain.WorkflowID](err)
	}
	return fp.Success(domain.WorkflowID(id))
}

// parseWorkflowStepID parses a string to WorkflowStepID.
func parseWorkflowStepID(s string) fp.Result[domain.WorkflowStepID] {
	id, err := parseUUID(s, "workflow_step_id")
	if err != nil {
		return fp.Failure[domain.WorkflowStepID](err)
	}
	return fp.Success(domain.WorkflowStepID(id))
}

// parseNullableUserID parses a sql.NullString to *UserID.
func parseNullableUserID(ns sql.NullString) fp.Result[*domain.UserID] {
	if !ns.Valid {
		return fp.Success[*domain.UserID](nil)
	}
	id, err := parseUUID(ns.String, "user_id")
	if err != nil {
		return fp.Failure[*domain.UserID](err)
	}
	uid := domain.UserID(id)
	return fp.Success(&uid)
}

// ═══════════════════════════════════════════════════════════════════════════
// NULLABLE VALUE EXTRACTORS - Extract values from sql.Null* types
// ═══════════════════════════════════════════════════════════════════════════

// stringFromNull extracts string from sql.NullString.
func stringFromNull(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// timeFromNull extracts *time.Time from sql.NullTime.
func timeFromNull(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// SCANNER HELPERS - Reduce scanning boilerplate
// ═══════════════════════════════════════════════════════════════════════════

// ScannerFunc is a function that scans a single row of results.
type ScannerFunc func(dest ...interface{}) error

// RowScanner wraps sql.Row to provide a Scanner interface.
type RowScanner struct {
	row *sql.Row
}

// Scan delegates to the underlying sql.Row.
func (r RowScanner) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

// RowsScanner wraps sql.Rows to provide a Scanner interface.
type RowsScanner struct {
	rows *sql.Rows
}

// Scan delegates to the underlying sql.Rows.
func (r RowsScanner) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

// Scanner is an interface implemented by both sql.Row and sql.Rows.
type Scanner interface {
	Scan(dest ...interface{}) error
}

// ═══════════════════════════════════════════════════════════════════════════
// RESULT HELPERS - Common fp.Result operations
// ═══════════════════════════════════════════════════════════════════════════

// CollectResults collects successful results from a slice of Result values.
// Returns failure on first error encountered.
func CollectResults[T any](results []fp.Result[T]) fp.Result[[]T] {
	items := make([]T, 0, len(results))
	for _, r := range results {
		if fp.IsFailure(r) {
			return fp.Failure[[]T](fp.GetError(r))
		}
		items = append(items, fp.GetValue(r))
	}
	return fp.Success(items)
}

// MapRows applies a scan function to each row and collects results.
func MapRows[T any](rows *sql.Rows, scanFn func(*sql.Rows) fp.Result[T]) fp.Result[[]T] {
	var items []T
	for rows.Next() {
		result := scanFn(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]T](fp.GetError(result))
		}
		items = append(items, fp.GetValue(result))
	}
	if err := rows.Err(); err != nil {
		return fp.Failure[[]T](err)
	}
	return fp.Success(items)
}
