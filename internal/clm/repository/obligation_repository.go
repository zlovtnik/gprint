package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ObligationRepository handles obligation persistence
type ObligationRepository struct {
	db *sql.DB
}

// NewObligationRepository creates a new ObligationRepository
func NewObligationRepository(db *sql.DB) *ObligationRepository {
	return &ObligationRepository{db: db}
}

// Create inserts a new obligation
func (r *ObligationRepository) Create(ctx context.Context, obligation domain.Obligation) fp.Result[domain.Obligation] {
	query := `
		INSERT INTO clm_obligations (
			id, tenant_id, contract_id, obligation_type, title, description,
			responsible_party_id, status, due_date, amount, currency,
			frequency, reminder_days, notes, created_at, created_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16
		)`

	var amount, currency sql.NullString
	if obligation.Amount != nil {
		amount = sql.NullString{String: obligation.Amount.Amount.String(), Valid: true}
		currency = sql.NullString{String: obligation.Amount.Currency, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(obligation.ID).String(),
		obligation.TenantID,
		uuid.UUID(obligation.ContractID).String(),
		string(obligation.Type),
		obligation.Title,
		nullableString(obligation.Description),
		uuid.UUID(obligation.ResponsibleParty).String(),
		string(obligation.Status),
		obligation.DueDate,
		amount,
		currency,
		string(obligation.Frequency),
		obligation.ReminderDays,
		nullableString(obligation.Notes),
		obligation.CreatedAt,
		uuid.UUID(obligation.CreatedBy).String(),
	)
	if err != nil {
		return fp.Failure[domain.Obligation](err)
	}

	return fp.Success(obligation)
}

// FindByID retrieves an obligation by ID
func (r *ObligationRepository) FindByID(ctx context.Context, tenantID string, id domain.ObligationID) fp.Result[domain.Obligation] {
	query := `
		SELECT id, tenant_id, contract_id, obligation_type, title, description,
			responsible_party_id, status, due_date, completed_date,
			amount, currency, frequency, reminder_days, notes,
			created_at, updated_at, created_by, updated_by
		FROM clm_obligations
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	return scanObligation(row)
}

// FindByContract retrieves all obligations for a contract
func (r *ObligationRepository) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID, offset, limit int) fp.Result[[]domain.Obligation] {
	query := `
		SELECT id, tenant_id, contract_id, obligation_type, title, description,
			responsible_party_id, status, due_date, completed_date,
			amount, currency, frequency, reminder_days, notes,
			created_at, updated_at, created_by, updated_by
		FROM clm_obligations
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY due_date
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.UUID(contractID).String(), offset, limit)
	if err != nil {
		return fp.Failure[[]domain.Obligation](err)
	}
	defer rows.Close()

	var obligations []domain.Obligation
	for rows.Next() {
		result := scanObligationFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Obligation](fp.GetError(result))
		}
		obligations = append(obligations, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Obligation](err)
	}

	return fp.Success(obligations)
}

// FindOverdue retrieves all overdue obligations
func (r *ObligationRepository) FindOverdue(ctx context.Context, tenantID string, offset, limit int) fp.Result[[]domain.Obligation] {
	query := `
		SELECT id, tenant_id, contract_id, obligation_type, title, description,
			responsible_party_id, status, due_date, completed_date,
			amount, currency, frequency, reminder_days, notes,
			created_at, updated_at, created_by, updated_by
		FROM clm_obligations
		WHERE tenant_id = :1 AND status IN ('PENDING', 'IN_PROGRESS')
			AND due_date < SYSDATE
		ORDER BY due_date
		OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, offset, limit)
	if err != nil {
		return fp.Failure[[]domain.Obligation](err)
	}
	defer rows.Close()

	var obligations []domain.Obligation
	for rows.Next() {
		result := scanObligationFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Obligation](fp.GetError(result))
		}
		obligations = append(obligations, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Obligation](err)
	}

	return fp.Success(obligations)
}

// UpdateStatus updates the obligation status
func (r *ObligationRepository) UpdateStatus(ctx context.Context, tenantID string, id domain.ObligationID, status domain.ObligationStatus, updatedBy domain.UserID) fp.Result[bool] {
	query := `
		UPDATE clm_obligations SET
			status = :1, updated_at = :2, updated_by = :3
		WHERE tenant_id = :4 AND id = :5`

	result, err := r.db.ExecContext(ctx, query,
		string(status),
		time.Now(),
		uuid.UUID(updatedBy).String(),
		tenantID,
		uuid.UUID(id).String(),
	)
	if err != nil {
		return fp.Failure[bool](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[bool](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[bool](errors.New("obligation not found"))
	}

	return fp.Success(true)
}

// Update updates an obligation
func (r *ObligationRepository) Update(ctx context.Context, obligation domain.Obligation) fp.Result[domain.Obligation] {
	query := `
		UPDATE clm_obligations SET
			title = :1, description = :2, due_date = :3, completed_date = :4,
			status = :5, amount = :6, currency = :7, notes = :8,
			updated_at = :9, updated_by = :10
		WHERE tenant_id = :11 AND id = :12`

	var amount, currency sql.NullString
	if obligation.Amount != nil {
		amount = sql.NullString{String: obligation.Amount.Amount.String(), Valid: true}
		currency = sql.NullString{String: obligation.Amount.Currency, Valid: true}
	}

	var updatedBy sql.NullString
	if obligation.UpdatedBy != nil {
		updatedBy = sql.NullString{String: uuid.UUID(*obligation.UpdatedBy).String(), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		obligation.Title,
		nullableString(obligation.Description),
		obligation.DueDate,
		obligation.CompletedDate,
		string(obligation.Status),
		amount,
		currency,
		nullableString(obligation.Notes),
		obligation.UpdatedAt,
		updatedBy,
		obligation.TenantID,
		uuid.UUID(obligation.ID).String(),
	)
	if err != nil {
		return fp.Failure[domain.Obligation](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[domain.Obligation](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[domain.Obligation](errors.New("obligation not found"))
	}

	return fp.Success(obligation)
}

// CountByContract returns the count of obligations for a contract
func (r *ObligationRepository) CountByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[int] {
	query := `SELECT COUNT(*) FROM clm_obligations WHERE tenant_id = :1 AND contract_id = :2`

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(contractID).String()).Scan(&count)
	if err != nil {
		return fp.Failure[int](err)
	}
	return fp.Success(count)
}

// CountOverdue returns the count of overdue obligations
func (r *ObligationRepository) CountOverdue(ctx context.Context, tenantID string) fp.Result[int] {
	query := `SELECT COUNT(*) FROM clm_obligations WHERE tenant_id = :1 AND status IN ('PENDING', 'IN_PROGRESS') AND due_date < SYSDATE`

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return fp.Failure[int](err)
	}
	return fp.Success(count)
}

func scanObligation(row *sql.Row) fp.Result[domain.Obligation] {
	var obligation domain.Obligation
	var idStr, contractIDStr, partyIDStr, createdByStr string
	var updatedByStr sql.NullString
	var obligationType, status, frequency string
	var description, notes sql.NullString
	var completedDate, updatedAt sql.NullTime
	var amount, currency sql.NullString

	err := row.Scan(
		&idStr, &obligation.TenantID, &contractIDStr, &obligationType,
		&obligation.Title, &description, &partyIDStr, &status,
		&obligation.DueDate, &completedDate, &amount, &currency,
		&frequency, &obligation.ReminderDays, &notes,
		&obligation.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Obligation](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse obligation id %q: %w", idStr, err))
	}
	obligation.ID = domain.ObligationID(id)

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse contract_id %q: %w", contractIDStr, err))
	}
	obligation.ContractID = domain.ContractID(contractID)

	partyID, err := uuid.Parse(partyIDStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse responsible_party_id %q: %w", partyIDStr, err))
	}
	obligation.ResponsibleParty = domain.PartyID(partyID)

	obligation.Type = domain.ObligationType(obligationType)
	obligation.Status = domain.ObligationStatus(status)
	obligation.Frequency = domain.ObligationFrequency(frequency)

	if description.Valid {
		obligation.Description = description.String
	}
	if notes.Valid {
		obligation.Notes = notes.String
	}
	if completedDate.Valid {
		obligation.CompletedDate = &completedDate.Time
	}
	if updatedAt.Valid {
		obligation.UpdatedAt = &updatedAt.Time
	}
	if amount.Valid && currency.Valid {
		amt, err := decimal.NewFromString(amount.String)
		if err != nil {
			return fp.Failure[domain.Obligation](fmt.Errorf("parse amount %q: %w", amount.String, err))
		}
		obligation.Amount = &domain.Money{Amount: amt, Currency: currency.String}
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	obligation.CreatedBy = domain.UserID(createdBy)

	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Obligation](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		obligation.UpdatedBy = (*domain.UserID)(&updatedBy)
	}

	return fp.Success(obligation)
}

func scanObligationFromRows(rows *sql.Rows) fp.Result[domain.Obligation] {
	var obligation domain.Obligation
	var idStr, contractIDStr, partyIDStr, createdByStr string
	var updatedByStr sql.NullString
	var obligationType, status, frequency string
	var description, notes sql.NullString
	var completedDate, updatedAt sql.NullTime
	var amount, currency sql.NullString

	err := rows.Scan(
		&idStr, &obligation.TenantID, &contractIDStr, &obligationType,
		&obligation.Title, &description, &partyIDStr, &status,
		&obligation.DueDate, &completedDate, &amount, &currency,
		&frequency, &obligation.ReminderDays, &notes,
		&obligation.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Obligation](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse obligation id %q: %w", idStr, err))
	}
	obligation.ID = domain.ObligationID(id)

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse contract_id %q: %w", contractIDStr, err))
	}
	obligation.ContractID = domain.ContractID(contractID)

	partyID, err := uuid.Parse(partyIDStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse responsible_party_id %q: %w", partyIDStr, err))
	}
	obligation.ResponsibleParty = domain.PartyID(partyID)

	obligation.Type = domain.ObligationType(obligationType)
	obligation.Status = domain.ObligationStatus(status)
	obligation.Frequency = domain.ObligationFrequency(frequency)

	if description.Valid {
		obligation.Description = description.String
	}
	if notes.Valid {
		obligation.Notes = notes.String
	}
	if completedDate.Valid {
		obligation.CompletedDate = &completedDate.Time
	}
	if updatedAt.Valid {
		obligation.UpdatedAt = &updatedAt.Time
	}
	if amount.Valid && currency.Valid {
		amt, err := decimal.NewFromString(amount.String)
		if err != nil {
			return fp.Failure[domain.Obligation](fmt.Errorf("parse amount %q: %w", amount.String, err))
		}
		obligation.Amount = &domain.Money{Amount: amt, Currency: currency.String}
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Obligation](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	obligation.CreatedBy = domain.UserID(createdBy)

	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Obligation](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		obligation.UpdatedBy = (*domain.UserID)(&updatedBy)
	}

	return fp.Success(obligation)
}
