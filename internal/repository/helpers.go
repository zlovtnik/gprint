package repository

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════
// SQL NULL HELPERS - Conversion between Go types and sql.Null* types
// ═══════════════════════════════════════════════════════════════════════════

// NullableString returns a sql.NullString for a string value.
// Empty strings result in a NULL database value.
func NullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullableUUID returns a sql.NullString for a UUID value formatted as string.
func NullableUUID[T ~[16]byte](id *T) sql.NullString {
	if id == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: uuid.UUID(*id).String(), Valid: true}
}

// NullableTime returns a sql.NullTime for a *time.Time value.
func NullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// NullableInt64 returns a sql.NullInt64 for an *int64 value.
func NullableInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// ═══════════════════════════════════════════════════════════════════════════
// NULLABLE VALUE EXTRACTORS - Extract values from sql.Null* types
// ═══════════════════════════════════════════════════════════════════════════

// StringFromNull extracts the string value from a sql.NullString.
// Returns empty string if null.
func StringFromNull(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// TimeFromNull extracts the *time.Time value from a sql.NullTime.
// Returns nil if null.
func TimeFromNull(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// Int64FromNull extracts the int64 value from a sql.NullInt64.
// Returns 0 if null.
func Int64FromNull(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// IntFromNullInt64 extracts int from sql.NullInt64.
// Returns 0 if null. On 32-bit systems, values outside [math.MinInt, math.MaxInt]
// are clamped to the respective boundary to prevent overflow.
// For unclamped int64 values, use Int64FromNull instead.
func IntFromNullInt64(ni sql.NullInt64) int {
	v := Int64FromNull(ni)
	if v > int64(math.MaxInt) {
		return math.MaxInt
	}
	if v < int64(math.MinInt) {
		return math.MinInt
	}
	return int(v)
}

// ═══════════════════════════════════════════════════════════════════════════
// BOOLEAN HELPERS - Oracle doesn't have native BOOLEAN, uses NUMBER(1)
// ═══════════════════════════════════════════════════════════════════════════

// BoolToInt converts a boolean to an int for Oracle NUMBER(1) storage.
// true -> 1, false -> 0
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IntToBool converts an Oracle NUMBER(1) to boolean.
// 1 -> true, anything else -> false
func IntToBool(i int) bool {
	return i == 1
}

// ═══════════════════════════════════════════════════════════════════════════
// UUID PARSING HELPERS - Common UUID parsing with wrapped errors
// ═══════════════════════════════════════════════════════════════════════════

// ParseUUID parses a string to uuid.UUID with a descriptive error.
func ParseUUID(s, fieldName string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse %s %q: %w", fieldName, s, err)
	}
	return id, nil
}

// ParseNullableUUID parses a sql.NullString to *uuid.UUID.
// Returns nil if the string is not valid, otherwise returns pointer to UUID.
func ParseNullableUUID(ns sql.NullString, fieldName string) (*uuid.UUID, error) {
	if !ns.Valid {
		return nil, nil
	}
	id, err := ParseUUID(ns.String, fieldName)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// IN CLAUSE BUILDER - Build Oracle IN clauses with positional parameters
// ═══════════════════════════════════════════════════════════════════════════

// MaxInClauseSize is the maximum number of items in an IN clause.
// Oracle has a limit of 1000 items per IN clause.
const MaxInClauseSize = 1000

// InClauseBuilder helps build Oracle IN clauses with positional params.
type InClauseBuilder struct {
	placeholders []string
	args         []interface{}
	startIdx     int
}

// NewInClauseBuilder creates a new InClauseBuilder starting at the given parameter index.
func NewInClauseBuilder(startIdx int) *InClauseBuilder {
	return &InClauseBuilder{
		startIdx: startIdx,
	}
}

// Add adds a value to the IN clause.
func (b *InClauseBuilder) Add(value interface{}) {
	idx := b.startIdx + len(b.args)
	b.placeholders = append(b.placeholders, fmt.Sprintf(":%d", idx))
	b.args = append(b.args, value)
}

// Placeholders returns the comma-separated placeholder string.
func (b *InClauseBuilder) Placeholders() string {
	return strings.Join(b.placeholders, ", ")
}

// Args returns the argument values.
func (b *InClauseBuilder) Args() []interface{} {
	return b.args
}

// NextIndex returns the next available parameter index.
func (b *InClauseBuilder) NextIndex() int {
	return b.startIdx + len(b.args)
}

// ═══════════════════════════════════════════════════════════════════════════
// QUERY BUILDER HELPERS - Dynamic WHERE clause construction
// ═══════════════════════════════════════════════════════════════════════════

// QueryBuilder helps construct dynamic SQL queries with Oracle positional params.
type QueryBuilder struct {
	conditions []string
	args       []interface{}
	nextIdx    int
}

// NewQueryBuilder creates a new QueryBuilder starting at the given parameter index.
func NewQueryBuilder(startIdx int) *QueryBuilder {
	return &QueryBuilder{
		nextIdx: startIdx,
	}
}

// AddCondition adds a condition with a single placeholder.
// The placeholder in the condition should be %d which will be replaced with :N.
func (b *QueryBuilder) AddCondition(conditionFmt string, value interface{}) {
	condition := fmt.Sprintf(conditionFmt, b.nextIdx)
	b.conditions = append(b.conditions, condition)
	b.args = append(b.args, value)
	b.nextIdx++
}

// AddConditionMultiple adds a condition with multiple same-value placeholders.
// Useful for LIKE queries that need the same value in multiple positions.
func (b *QueryBuilder) AddConditionMultiple(conditionFmt string, value interface{}, count int) {
	indices := make([]interface{}, count)
	for i := 0; i < count; i++ {
		indices[i] = b.nextIdx
	}
	condition := fmt.Sprintf(conditionFmt, indices...)
	b.conditions = append(b.conditions, condition)
	b.args = append(b.args, value)
	b.nextIdx++
}

// WhereClause returns the WHERE conditions joined with AND.
// Returns empty string if no conditions.
func (b *QueryBuilder) WhereClause() string {
	if len(b.conditions) == 0 {
		return ""
	}
	return " AND " + strings.Join(b.conditions, " AND ")
}

// Args returns all accumulated arguments.
func (b *QueryBuilder) Args() []interface{} {
	return b.args
}

// NextIndex returns the next available parameter index.
func (b *QueryBuilder) NextIndex() int {
	return b.nextIdx
}

// ═══════════════════════════════════════════════════════════════════════════
// CHUNK PROCESSOR - Process slices in chunks to avoid DB limits
// ═══════════════════════════════════════════════════════════════════════════

// ChunkSlice splits a slice into chunks of the specified size.
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if chunkSize <= 0 {
		chunkSize = MaxInClauseSize
	}
	if len(slice) == 0 {
		return nil
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
