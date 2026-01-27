package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// ContractRepository handles contract persistence
type ContractRepository struct {
	db        *sql.DB
	partyRepo *PartyRepository
}

// NewContractRepository creates a new ContractRepository
func NewContractRepository(db *sql.DB, partyRepo *PartyRepository) *ContractRepository {
	return &ContractRepository{db: db, partyRepo: partyRepo}
}

// ContractFilter represents filter criteria for contracts
type ContractFilter struct {
	Status     *domain.ContractStatus
	PartyID    *domain.PartyID
	TypeID     *domain.ContractTypeID
	SearchTerm string
}

// Create inserts a new contract
func (r *ContractRepository) Create(ctx context.Context, contract domain.Contract) fp.Result[domain.Contract] {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO clm_contracts (
			id, tenant_id, contract_number, external_ref, title, description,
			contract_type_id, status, value_amount, value_currency,
			effective_date, expiration_date, auto_renew, renewal_term_days,
			notice_period_days, terms, notes, version, is_deleted,
			created_at, created_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14,
			:15, :16, :17, :18, :19, :20, :21
		)`

	_, err = tx.ExecContext(ctx, query,
		uuid.UUID(contract.ID).String(),
		contract.TenantID,
		contract.ContractNumber,
		nullableString(contract.ExternalRef),
		contract.Title,
		nullableString(contract.Description),
		uuid.UUID(contract.ContractType.ID).String(),
		string(contract.Status),
		contract.Value.Amount.String(),
		contract.Value.Currency,
		contract.EffectiveDate,
		contract.ExpirationDate,
		boolToInt(contract.AutoRenew),
		contract.RenewalTermDays,
		contract.NoticePeriodDays,
		nullableString(contract.Terms),
		nullableString(contract.Notes),
		contract.Version,
		boolToInt(contract.IsDeleted),
		contract.CreatedAt,
		uuid.UUID(contract.CreatedBy).String(),
	)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	// Insert contract parties using transaction
	for _, party := range contract.Parties {
		if err := r.insertContractPartyTx(ctx, tx, contract.ID, party); err != nil {
			return fp.Failure[domain.Contract](err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fp.Failure[domain.Contract](err)
	}

	return fp.Success(contract)
}

func (r *ContractRepository) insertContractPartyTx(ctx context.Context, tx *sql.Tx, contractID domain.ContractID, party domain.ContractParty) error {
	query := `
		INSERT INTO clm_contract_parties (contract_id, party_id, role, is_primary, signed_at, signed_by)
		VALUES (:1, :2, :3, :4, :5, :6)`

	var signedBy sql.NullString
	if party.SignedBy != nil {
		signedBy = sql.NullString{String: uuid.UUID(*party.SignedBy).String(), Valid: true}
	}

	_, err := tx.ExecContext(ctx, query,
		uuid.UUID(contractID).String(),
		uuid.UUID(party.PartyID).String(),
		string(party.Role),
		boolToInt(party.IsPrimary),
		party.SignedAt,
		signedBy,
	)
	return err
}

// FindByID retrieves a contract by ID
func (r *ContractRepository) FindByID(ctx context.Context, tenantID string, id domain.ContractID) fp.Result[domain.Contract] {
	query := `
		SELECT c.id, c.tenant_id, c.contract_number, c.external_ref, c.title, c.description,
			c.contract_type_id, ct.name as type_name, ct.code as type_code,
			ct.requires_approval, ct.approval_levels,
			c.status, c.value_amount, c.value_currency,
			c.effective_date, c.expiration_date, c.termination_date,
			c.auto_renew, c.renewal_term_days, c.notice_period_days,
			c.terms, c.notes, c.version, c.previous_version_id, c.is_deleted,
			c.created_at, c.updated_at, c.created_by, c.updated_by
		FROM clm_contracts c
		JOIN clm_contract_types ct ON c.contract_type_id = ct.id
		WHERE c.tenant_id = :1 AND c.id = :2 AND c.is_deleted = 0`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	contractResult := scanContract(row)
	if fp.IsFailure(contractResult) {
		return contractResult
	}

	contract := fp.GetValue(contractResult)

	// Load contract parties
	partiesResult := r.findContractParties(ctx, id)
	if fp.IsFailure(partiesResult) {
		return fp.Failure[domain.Contract](fp.GetError(partiesResult))
	}
	contract.Parties = fp.GetValue(partiesResult)

	return fp.Success(contract)
}

func (r *ContractRepository) findContractParties(ctx context.Context, contractID domain.ContractID) fp.Result[[]domain.ContractParty] {
	query := `
		SELECT party_id, role, is_primary, signed_at, signed_by
		FROM clm_contract_parties
		WHERE contract_id = :1`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(contractID).String())
	if err != nil {
		return fp.Failure[[]domain.ContractParty](err)
	}
	defer rows.Close()

	var parties []domain.ContractParty
	for rows.Next() {
		var partyIDStr string
		var role string
		var isPrimary int
		var signedAt sql.NullTime
		var signedByStr sql.NullString

		if err := rows.Scan(&partyIDStr, &role, &isPrimary, &signedAt, &signedByStr); err != nil {
			return fp.Failure[[]domain.ContractParty](err)
		}

		partyID, err := uuid.Parse(partyIDStr)
		if err != nil {
			return fp.Failure[[]domain.ContractParty](fmt.Errorf("parse party_id %q: %w", partyIDStr, err))
		}
		party := domain.ContractParty{
			PartyID:   domain.PartyID(partyID),
			Role:      domain.PartyRole(role),
			IsPrimary: isPrimary == 1,
		}
		if signedAt.Valid {
			party.SignedAt = &signedAt.Time
		}
		if signedByStr.Valid {
			signedBy, err := uuid.Parse(signedByStr.String)
			if err != nil {
				return fp.Failure[[]domain.ContractParty](fmt.Errorf("parse signed_by %q: %w", signedByStr.String, err))
			}
			party.SignedBy = (*domain.UserID)(&signedBy)
		}
		parties = append(parties, party)
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.ContractParty](err)
	}

	return fp.Success(parties)
}

// FindAll retrieves all contracts with pagination and filters
func (r *ContractRepository) FindAll(ctx context.Context, tenantID string, filter ContractFilter, offset, limit int) fp.Result[[]domain.Contract] {
	query := `
		SELECT c.id, c.tenant_id, c.contract_number, c.external_ref, c.title, c.description,
			c.contract_type_id, ct.name as type_name, ct.code as type_code,
			ct.requires_approval, ct.approval_levels,
			c.status, c.value_amount, c.value_currency,
			c.effective_date, c.expiration_date, c.termination_date,
			c.auto_renew, c.renewal_term_days, c.notice_period_days,
			c.terms, c.notes, c.version, c.previous_version_id, c.is_deleted,
			c.created_at, c.updated_at, c.created_by, c.updated_by
		FROM clm_contracts c
		JOIN clm_contract_types ct ON c.contract_type_id = ct.id
		WHERE c.tenant_id = :1 AND c.is_deleted = 0`

	args := []interface{}{tenantID}
	paramIdx := 2

	// Apply filters
	if filter.Status != nil {
		query += fmt.Sprintf(" AND c.status = :%d", paramIdx)
		args = append(args, string(*filter.Status))
		paramIdx++
	}
	if filter.TypeID != nil {
		query += fmt.Sprintf(" AND c.contract_type_id = :%d", paramIdx)
		args = append(args, uuid.UUID(*filter.TypeID).String())
		paramIdx++
	}
	if filter.PartyID != nil {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM clm_contract_parties cp WHERE cp.contract_id = c.id AND cp.party_id = :%d)", paramIdx)
		args = append(args, uuid.UUID(*filter.PartyID).String())
		paramIdx++
	}
	if filter.SearchTerm != "" {
		query += fmt.Sprintf(" AND (UPPER(c.title) LIKE UPPER(:%d) OR UPPER(c.contract_number) LIKE UPPER(:%d))", paramIdx, paramIdx)
		args = append(args, "%"+filter.SearchTerm+"%")
		paramIdx++
	}

	query += fmt.Sprintf(" ORDER BY c.created_at DESC OFFSET :%d ROWS FETCH NEXT :%d ROWS ONLY", paramIdx, paramIdx+1)
	args = append(args, offset, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fp.Failure[[]domain.Contract](err)
	}
	defer rows.Close()

	var contracts []domain.Contract
	for rows.Next() {
		result := scanContractFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Contract](fp.GetError(result))
		}
		contracts = append(contracts, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Contract](err)
	}

	return fp.Success(contracts)
}

// Count returns the total number of contracts
func (r *ContractRepository) Count(ctx context.Context, tenantID string, filter ContractFilter) fp.Result[int] {
	query := `SELECT COUNT(*) FROM clm_contracts c WHERE c.tenant_id = :1 AND c.is_deleted = 0`

	args := []interface{}{tenantID}
	paramIdx := 2

	// Apply same filters as FindAll
	if filter.Status != nil {
		query += fmt.Sprintf(" AND c.status = :%d", paramIdx)
		args = append(args, string(*filter.Status))
		paramIdx++
	}
	if filter.TypeID != nil {
		query += fmt.Sprintf(" AND c.contract_type_id = :%d", paramIdx)
		args = append(args, uuid.UUID(*filter.TypeID).String())
		paramIdx++
	}
	if filter.PartyID != nil {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM clm_contract_parties cp WHERE cp.contract_id = c.id AND cp.party_id = :%d)", paramIdx)
		args = append(args, uuid.UUID(*filter.PartyID).String())
		paramIdx++
	}
	if filter.SearchTerm != "" {
		query += fmt.Sprintf(" AND (UPPER(c.title) LIKE UPPER(:%d) OR UPPER(c.contract_number) LIKE UPPER(:%d))", paramIdx, paramIdx)
		args = append(args, "%"+filter.SearchTerm+"%")
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return fp.Failure[int](err)
	}
	return fp.Success(count)
}

// Update updates an existing contract
func (r *ContractRepository) Update(ctx context.Context, contract domain.Contract) fp.Result[domain.Contract] {
	query := `
		UPDATE clm_contracts SET
			title = :1, description = :2, value_amount = :3, value_currency = :4,
			effective_date = :5, expiration_date = :6, auto_renew = :7,
			renewal_term_days = :8, notice_period_days = :9, terms = :10,
			notes = :11, updated_at = :12, updated_by = :13
		WHERE tenant_id = :14 AND id = :15`

	var updatedBy sql.NullString
	if contract.UpdatedBy != nil {
		updatedBy = sql.NullString{String: uuid.UUID(*contract.UpdatedBy).String(), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		contract.Title,
		nullableString(contract.Description),
		contract.Value.Amount.String(),
		contract.Value.Currency,
		contract.EffectiveDate,
		contract.ExpirationDate,
		boolToInt(contract.AutoRenew),
		contract.RenewalTermDays,
		contract.NoticePeriodDays,
		nullableString(contract.Terms),
		nullableString(contract.Notes),
		contract.UpdatedAt,
		updatedBy,
		contract.TenantID,
		uuid.UUID(contract.ID).String(),
	)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	return fp.Success(contract)
}

// UpdateStatus updates the contract status
func (r *ContractRepository) UpdateStatus(ctx context.Context, tenantID string, id domain.ContractID, status domain.ContractStatus, updatedBy domain.UserID) fp.Result[bool] {
	query := `
		UPDATE clm_contracts SET
			status = :1, updated_at = :2, updated_by = :3
		WHERE tenant_id = :4 AND id = :5`

	_, err := r.db.ExecContext(ctx, query,
		string(status),
		time.Now(),
		uuid.UUID(updatedBy).String(),
		tenantID,
		uuid.UUID(id).String(),
	)
	if err != nil {
		return fp.Failure[bool](err)
	}

	return fp.Success(true)
}

// SoftDelete marks a contract as deleted
func (r *ContractRepository) SoftDelete(ctx context.Context, tenantID string, id domain.ContractID, deletedBy domain.UserID) fp.Result[bool] {
	query := `
		UPDATE clm_contracts SET
			is_deleted = 1, updated_at = :1, updated_by = :2
		WHERE tenant_id = :3 AND id = :4`

	_, err := r.db.ExecContext(ctx, query,
		time.Now(),
		uuid.UUID(deletedBy).String(),
		tenantID,
		uuid.UUID(id).String(),
	)
	if err != nil {
		return fp.Failure[bool](err)
	}

	return fp.Success(true)
}

// FindExpiring returns contracts expiring within the given days
func (r *ContractRepository) FindExpiring(ctx context.Context, tenantID string, days int) fp.Result[[]domain.Contract] {
	query := `
		SELECT c.id, c.tenant_id, c.contract_number, c.external_ref, c.title, c.description,
			c.contract_type_id, ct.name as type_name, ct.code as type_code,
			ct.requires_approval, ct.approval_levels,
			c.status, c.value_amount, c.value_currency,
			c.effective_date, c.expiration_date, c.termination_date,
			c.auto_renew, c.renewal_term_days, c.notice_period_days,
			c.terms, c.notes, c.version, c.previous_version_id, c.is_deleted,
			c.created_at, c.updated_at, c.created_by, c.updated_by
		FROM clm_contracts c
		JOIN clm_contract_types ct ON c.contract_type_id = ct.id
		WHERE c.tenant_id = :1 AND c.is_deleted = 0 AND c.status = 'ACTIVE'
			AND c.expiration_date <= :2 AND c.expiration_date > SYSDATE
		ORDER BY c.expiration_date`

	threshold := time.Now().AddDate(0, 0, days)
	rows, err := r.db.QueryContext(ctx, query, tenantID, threshold)
	if err != nil {
		return fp.Failure[[]domain.Contract](err)
	}
	defer rows.Close()

	var contracts []domain.Contract
	for rows.Next() {
		result := scanContractFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Contract](fp.GetError(result))
		}
		contracts = append(contracts, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Contract](err)
	}

	return fp.Success(contracts)
}

func scanContract(row *sql.Row) fp.Result[domain.Contract] {
	var contract domain.Contract
	var idStr, typeIDStr, createdByStr string
	var updatedByStr, externalRef, description, terms, notes sql.NullString
	var previousVersionStr sql.NullString
	var status, typeName, typeCode string
	var valueAmount string
	var effectiveDate, expirationDate, terminationDate sql.NullTime
	var updatedAt sql.NullTime
	var autoRenew, isDeleted, requiresApproval int
	var approvalLevels int

	err := row.Scan(
		&idStr, &contract.TenantID, &contract.ContractNumber, &externalRef,
		&contract.Title, &description, &typeIDStr, &typeName, &typeCode,
		&requiresApproval, &approvalLevels,
		&status, &valueAmount, &contract.Value.Currency,
		&effectiveDate, &expirationDate, &terminationDate,
		&autoRenew, &contract.RenewalTermDays, &contract.NoticePeriodDays,
		&terms, &notes, &contract.Version, &previousVersionStr, &isDeleted,
		&contract.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse contract id %q: %w", idStr, err))
	}
	contract.ID = domain.ContractID(id)
	contract.Status = domain.ContractStatus(status)
	contract.AutoRenew = autoRenew == 1
	contract.IsDeleted = isDeleted == 1

	typeID, err := uuid.Parse(typeIDStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse contract_type_id %q: %w", typeIDStr, err))
	}
	contract.ContractType = domain.ContractType{
		ID:               domain.ContractTypeID(typeID),
		Name:             typeName,
		Code:             typeCode,
		RequiresApproval: requiresApproval == 1,
		ApprovalLevels:   approvalLevels,
	}

	amount, err := decimal.NewFromString(valueAmount)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse value_amount %q: %w", valueAmount, err))
	}
	contract.Value.Amount = amount

	if externalRef.Valid {
		contract.ExternalRef = externalRef.String
	}
	if description.Valid {
		contract.Description = description.String
	}
	if terms.Valid {
		contract.Terms = terms.String
	}
	if notes.Valid {
		contract.Notes = notes.String
	}
	if effectiveDate.Valid {
		contract.EffectiveDate = &effectiveDate.Time
	}
	if expirationDate.Valid {
		contract.ExpirationDate = &expirationDate.Time
	}
	if terminationDate.Valid {
		contract.TerminationDate = &terminationDate.Time
	}
	if updatedAt.Valid {
		contract.UpdatedAt = &updatedAt.Time
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	contract.CreatedBy = domain.UserID(createdBy)

	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Contract](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		contract.UpdatedBy = (*domain.UserID)(&updatedBy)
	}
	if previousVersionStr.Valid {
		prevID, err := uuid.Parse(previousVersionStr.String)
		if err != nil {
			return fp.Failure[domain.Contract](fmt.Errorf("parse previous_version_id %q: %w", previousVersionStr.String, err))
		}
		contract.PreviousVersion = (*domain.ContractID)(&prevID)
	}

	return fp.Success(contract)
}

func scanContractFromRows(rows *sql.Rows) fp.Result[domain.Contract] {
	var contract domain.Contract
	var idStr, typeIDStr, createdByStr string
	var updatedByStr, externalRef, description, terms, notes sql.NullString
	var previousVersionStr sql.NullString
	var status, typeName, typeCode string
	var valueAmount string
	var effectiveDate, expirationDate, terminationDate sql.NullTime
	var updatedAt sql.NullTime
	var autoRenew, isDeleted, requiresApproval int
	var approvalLevels int

	err := rows.Scan(
		&idStr, &contract.TenantID, &contract.ContractNumber, &externalRef,
		&contract.Title, &description, &typeIDStr, &typeName, &typeCode,
		&requiresApproval, &approvalLevels,
		&status, &valueAmount, &contract.Value.Currency,
		&effectiveDate, &expirationDate, &terminationDate,
		&autoRenew, &contract.RenewalTermDays, &contract.NoticePeriodDays,
		&terms, &notes, &contract.Version, &previousVersionStr, &isDeleted,
		&contract.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Contract](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse contract id %q: %w", idStr, err))
	}
	contract.ID = domain.ContractID(id)
	contract.Status = domain.ContractStatus(status)
	contract.AutoRenew = autoRenew == 1
	contract.IsDeleted = isDeleted == 1

	typeID, err := uuid.Parse(typeIDStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse contract_type_id %q: %w", typeIDStr, err))
	}
	contract.ContractType = domain.ContractType{
		ID:               domain.ContractTypeID(typeID),
		Name:             typeName,
		Code:             typeCode,
		RequiresApproval: requiresApproval == 1,
		ApprovalLevels:   approvalLevels,
	}

	amount, err := decimal.NewFromString(valueAmount)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse value_amount %q: %w", valueAmount, err))
	}
	contract.Value.Amount = amount

	if externalRef.Valid {
		contract.ExternalRef = externalRef.String
	}
	if description.Valid {
		contract.Description = description.String
	}
	if terms.Valid {
		contract.Terms = terms.String
	}
	if notes.Valid {
		contract.Notes = notes.String
	}
	if effectiveDate.Valid {
		contract.EffectiveDate = &effectiveDate.Time
	}
	if expirationDate.Valid {
		contract.ExpirationDate = &expirationDate.Time
	}
	if terminationDate.Valid {
		contract.TerminationDate = &terminationDate.Time
	}
	if updatedAt.Valid {
		contract.UpdatedAt = &updatedAt.Time
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Contract](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	contract.CreatedBy = domain.UserID(createdBy)

	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Contract](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		contract.UpdatedBy = (*domain.UserID)(&updatedBy)
	}
	if previousVersionStr.Valid {
		prevID, err := uuid.Parse(previousVersionStr.String)
		if err != nil {
			return fp.Failure[domain.Contract](fmt.Errorf("parse previous_version_id %q: %w", previousVersionStr.String, err))
		}
		contract.PreviousVersion = (*domain.ContractID)(&prevID)
	}

	return fp.Success(contract)
}
