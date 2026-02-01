package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// PartyRepository handles party persistence
type PartyRepository struct {
	db *sql.DB
}

// NewPartyRepository creates a new PartyRepository
func NewPartyRepository(db *sql.DB) *PartyRepository {
	return &PartyRepository{db: db}
}

// Create inserts a new party
func (r *PartyRepository) Create(ctx context.Context, party domain.Party) fp.Result[domain.Party] {
	query := `
		INSERT INTO clm_parties (
			id, tenant_id, party_type, name, legal_name, tax_id, email, phone,
			street1, street2, city, state, postal_code, country,
			billing_street1, billing_street2, billing_city, billing_state,
			billing_postal_code, billing_country, risk_level, risk_score,
			is_active, created_at, created_by
		) VALUES (
			:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14,
			:15, :16, :17, :18, :19, :20, :21, :22, :23, :24, :25
		)`

	var billingStreet1, billingStreet2, billingCity, billingState, billingPostalCode, billingCountry sql.NullString
	if party.BillingAddress != nil {
		billingStreet1 = sql.NullString{String: party.BillingAddress.Street1, Valid: true}
		billingStreet2 = sql.NullString{String: party.BillingAddress.Street2, Valid: party.BillingAddress.Street2 != ""}
		billingCity = sql.NullString{String: party.BillingAddress.City, Valid: true}
		billingState = sql.NullString{String: party.BillingAddress.State, Valid: true}
		billingPostalCode = sql.NullString{String: party.BillingAddress.PostalCode, Valid: true}
		billingCountry = sql.NullString{String: party.BillingAddress.Country, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		uuid.UUID(party.ID).String(),
		party.TenantID,
		string(party.Type),
		party.Name,
		party.LegalName,
		nullableString(party.TaxID),
		party.Email,
		nullableString(party.Phone),
		party.Address.Street1,
		nullableString(party.Address.Street2),
		party.Address.City,
		party.Address.State,
		party.Address.PostalCode,
		party.Address.Country,
		billingStreet1,
		billingStreet2,
		billingCity,
		billingState,
		billingPostalCode,
		billingCountry,
		string(party.RiskLevel),
		party.RiskScore,
		boolToInt(party.IsActive),
		party.CreatedAt,
		uuid.UUID(party.CreatedBy).String(),
	)
	if err != nil {
		return fp.Failure[domain.Party](err)
	}

	return fp.Success(party)
}

// FindByID retrieves a party by ID
func (r *PartyRepository) FindByID(ctx context.Context, tenantID string, id domain.PartyID) fp.Result[domain.Party] {
	query := `
		SELECT id, tenant_id, party_type, name, legal_name, tax_id, email, phone,
			street1, street2, city, state, postal_code, country,
			billing_street1, billing_street2, billing_city, billing_state,
			billing_postal_code, billing_country, risk_level, risk_score,
			is_active, created_at, updated_at, created_by, updated_by
		FROM clm_parties
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	return scanParty(row)
}

// FindAll retrieves all parties with pagination
func (r *PartyRepository) FindAll(ctx context.Context, tenantID string, offset, limit int) fp.Result[[]domain.Party] {
	query := `
		SELECT id, tenant_id, party_type, name, legal_name, tax_id, email, phone,
			street1, street2, city, state, postal_code, country,
			billing_street1, billing_street2, billing_city, billing_state,
			billing_postal_code, billing_country, risk_level, risk_score,
			is_active, created_at, updated_at, created_by, updated_by
		FROM clm_parties
		WHERE tenant_id = :1 AND is_active = 1
		ORDER BY name
		OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, offset, limit)
	if err != nil {
		return fp.Failure[[]domain.Party](err)
	}
	defer rows.Close()

	var parties []domain.Party
	for rows.Next() {
		result := scanPartyFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.Party](fp.GetError(result))
		}
		parties = append(parties, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.Party](err)
	}

	return fp.Success(parties)
}

// Count returns the total number of parties
func (r *PartyRepository) Count(ctx context.Context, tenantID string) fp.Result[int] {
	query := `SELECT COUNT(*) FROM clm_parties WHERE tenant_id = :1 AND is_active = 1`

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return fp.Failure[int](err)
	}
	return fp.Success(count)
}

// Update updates an existing party
func (r *PartyRepository) Update(ctx context.Context, party domain.Party) fp.Result[domain.Party] {
	query := `
		UPDATE clm_parties SET
			name = :1, legal_name = :2, tax_id = :3, email = :4, phone = :5,
			street1 = :6, street2 = :7, city = :8, state = :9,
			postal_code = :10, country = :11, risk_level = :12, risk_score = :13,
			is_active = :14, updated_at = :15, updated_by = :16
		WHERE tenant_id = :17 AND id = :18`

	var updatedBy sql.NullString
	if party.UpdatedBy != nil {
		updatedBy = sql.NullString{String: uuid.UUID(*party.UpdatedBy).String(), Valid: true}
	}

	// Ensure updated_at is never NULL
	updatedAt := party.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
		party.UpdatedAt = updatedAt
	}

	_, err := r.db.ExecContext(ctx, query,
		party.Name,
		party.LegalName,
		nullableString(party.TaxID),
		party.Email,
		nullableString(party.Phone),
		party.Address.Street1,
		nullableString(party.Address.Street2),
		party.Address.City,
		party.Address.State,
		party.Address.PostalCode,
		party.Address.Country,
		string(party.RiskLevel),
		party.RiskScore,
		boolToInt(party.IsActive),
		updatedAt,
		updatedBy,
		party.TenantID,
		uuid.UUID(party.ID).String(),
	)
	if err != nil {
		return fp.Failure[domain.Party](err)
	}

	return fp.Success(party)
}

// Deactivate soft-deletes a party
func (r *PartyRepository) Deactivate(ctx context.Context, tenantID string, id domain.PartyID, updatedBy domain.UserID) fp.Result[bool] {
	query := `
		UPDATE clm_parties 
		SET is_active = 0, updated_at = :1, updated_by = :2
		WHERE tenant_id = :3 AND id = :4`

	_, err := r.db.ExecContext(ctx, query, time.Now(), uuid.UUID(updatedBy).String(), tenantID, uuid.UUID(id).String())
	if err != nil {
		return fp.Failure[bool](err)
	}

	return fp.Success(true)
}

// Note: boolToInt and nullableString are defined in helpers.go

func scanParty(row *sql.Row) fp.Result[domain.Party] {
	var party domain.Party
	var idStr, createdByStr string
	var updatedByStr sql.NullString
	var partyType, riskLevel string
	var taxID, phone, street2 sql.NullString
	var billingStreet1, billingStreet2, billingCity, billingState, billingPostalCode, billingCountry sql.NullString
	var isActive int
	var updatedAt sql.NullTime

	err := row.Scan(
		&idStr, &party.TenantID, &partyType, &party.Name, &party.LegalName,
		&taxID, &party.Email, &phone,
		&party.Address.Street1, &street2, &party.Address.City, &party.Address.State,
		&party.Address.PostalCode, &party.Address.Country,
		&billingStreet1, &billingStreet2, &billingCity, &billingState,
		&billingPostalCode, &billingCountry, &riskLevel, &party.RiskScore,
		&isActive, &party.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Party](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Party](fmt.Errorf("parse party id %q: %w", idStr, err))
	}
	party.ID = domain.PartyID(id)
	party.Type = domain.PartyType(partyType)
	party.RiskLevel = domain.RiskLevel(riskLevel)
	party.IsActive = isActive == 1

	if taxID.Valid {
		party.TaxID = taxID.String
	}
	if phone.Valid {
		party.Phone = phone.String
	}
	if street2.Valid {
		party.Address.Street2 = street2.String
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Party](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	party.CreatedBy = domain.UserID(createdBy)

	if updatedAt.Valid {
		party.UpdatedAt = &updatedAt.Time
	}
	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Party](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		party.UpdatedBy = (*domain.UserID)(&updatedBy)
	}

	if billingStreet1.Valid {
		party.BillingAddress = &domain.Address{
			Street1:    billingStreet1.String,
			Street2:    billingStreet2.String,
			City:       billingCity.String,
			State:      billingState.String,
			PostalCode: billingPostalCode.String,
			Country:    billingCountry.String,
		}
	}

	return fp.Success(party)
}

func scanPartyFromRows(rows *sql.Rows) fp.Result[domain.Party] {
	var party domain.Party
	var idStr, createdByStr string
	var updatedByStr sql.NullString
	var partyType, riskLevel string
	var taxID, phone, street2 sql.NullString
	var billingStreet1, billingStreet2, billingCity, billingState, billingPostalCode, billingCountry sql.NullString
	var isActive int
	var updatedAt sql.NullTime

	err := rows.Scan(
		&idStr, &party.TenantID, &partyType, &party.Name, &party.LegalName,
		&taxID, &party.Email, &phone,
		&party.Address.Street1, &street2, &party.Address.City, &party.Address.State,
		&party.Address.PostalCode, &party.Address.Country,
		&billingStreet1, &billingStreet2, &billingCity, &billingState,
		&billingPostalCode, &billingCountry, &riskLevel, &party.RiskScore,
		&isActive, &party.CreatedAt, &updatedAt, &createdByStr, &updatedByStr,
	)
	if err != nil {
		return fp.Failure[domain.Party](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.Party](fmt.Errorf("parse party id %q: %w", idStr, err))
	}
	party.ID = domain.PartyID(id)
	party.Type = domain.PartyType(partyType)
	party.RiskLevel = domain.RiskLevel(riskLevel)
	party.IsActive = isActive == 1

	if taxID.Valid {
		party.TaxID = taxID.String
	}
	if phone.Valid {
		party.Phone = phone.String
	}
	if street2.Valid {
		party.Address.Street2 = street2.String
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.Party](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	party.CreatedBy = domain.UserID(createdBy)

	if updatedAt.Valid {
		party.UpdatedAt = &updatedAt.Time
	}
	if updatedByStr.Valid {
		updatedBy, err := uuid.Parse(updatedByStr.String)
		if err != nil {
			return fp.Failure[domain.Party](fmt.Errorf("parse updated_by %q: %w", updatedByStr.String, err))
		}
		party.UpdatedBy = (*domain.UserID)(&updatedBy)
	}

	if billingStreet1.Valid {
		party.BillingAddress = &domain.Address{
			Street1:    billingStreet1.String,
			Street2:    billingStreet2.String,
			City:       billingCity.String,
			State:      billingState.String,
			PostalCode: billingPostalCode.String,
			Country:    billingCountry.String,
		}
	}

	return fp.Success(party)
}
