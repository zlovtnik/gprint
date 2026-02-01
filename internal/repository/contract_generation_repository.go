package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zlovtnik/gprint/internal/models"
)

// ErrUnauthorized is returned when a tenant tries to access another tenant's data
var ErrUnauthorized = errors.New("unauthorized: tenant does not own this resource")

// ContractGenerationRepository handles contract generation data access
// All sensitive operations are delegated to PL/SQL package for security
type ContractGenerationRepository struct {
	db *sql.DB
}

// NewContractGenerationRepository creates a new ContractGenerationRepository
func NewContractGenerationRepository(db *sql.DB) *ContractGenerationRepository {
	return &ContractGenerationRepository{db: db}
}

// GenerateContractParams holds parameters for generating a contract
type GenerateContractParams struct {
	TenantID     string
	ContractID   int64
	UserID       string
	TemplateCode string
	Reason       string
	IPAddress    string
	SessionID    string
}

// GenerateContract calls the PL/SQL package to generate a contract document
// All sensitive data processing happens in the database layer
// Returns both the result metadata and the generated JSON for immediate use
func (r *ContractGenerationRepository) GenerateContract(
	ctx context.Context,
	params GenerateContractParams,
) (*models.GenerateContractResponse, error) {
	// Use PL/SQL anonymous block to call the package procedure
	// The procedure returns both result object fields and the JSON content
	query := `
		DECLARE
			v_result       t_gen_result;
			v_contract_json CLOB;
		BEGIN
			pkg_contract_generation.generate_contract(
				p_tenant_id     => :1,
				p_contract_id   => :2,
				p_user_id       => :3,
				p_template_code => :4,
				p_reason        => :5,
				p_ip_address    => :6,
				p_session_id    => :7,
				p_result        => v_result,
				p_contract_json => v_contract_json
			);
			:8 := v_result.success;
			:9 := v_result.generated_id;
			:10 := v_result.content_hash;
			:11 := v_result.error_code;
			:12 := v_result.error_message;
			:13 := v_contract_json;
		END;`

	var (
		success      int
		generatedID  sql.NullInt64
		contentHash  sql.NullString
		errorCode    sql.NullString
		errorMessage sql.NullString
		contractJSON sql.NullString
	)

	_, err := r.db.ExecContext(ctx, query,
		params.TenantID,
		params.ContractID,
		params.UserID,
		sql.NullString{String: params.TemplateCode, Valid: params.TemplateCode != ""},
		sql.NullString{String: params.Reason, Valid: params.Reason != ""},
		sql.NullString{String: params.IPAddress, Valid: params.IPAddress != ""},
		sql.NullString{String: params.SessionID, Valid: params.SessionID != ""},
		sql.Out{Dest: &success},
		sql.Out{Dest: &generatedID},
		sql.Out{Dest: &contentHash},
		sql.Out{Dest: &errorCode},
		sql.Out{Dest: &errorMessage},
		sql.Out{Dest: &contractJSON},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call generate_contract: %w", err)
	}

	return &models.GenerateContractResponse{
		Success:      success == 1,
		GeneratedID:  generatedID.Int64,
		ContentHash:  contentHash.String,
		ContractJSON: []byte(contractJSON.String),
		ErrorCode:    errorCode.String,
		ErrorMessage: errorMessage.String,
	}, nil
}

// GetGeneratedContent retrieves generated contract content with tenant validation
// Content is only returned after proper tenant verification in PL/SQL
func (r *ContractGenerationRepository) GetGeneratedContent(
	ctx context.Context,
	tenantID string,
	generatedID int64,
	userID string,
) (*models.GetGeneratedContentResponse, error) {
	query := `
		DECLARE
			v_json_data    CLOB;
			v_content_hash VARCHAR2(64);
			v_generated_at TIMESTAMP;
			v_success      NUMBER;
			v_error_code   VARCHAR2(50);
		BEGIN
			pkg_contract_generation.get_generated(
				p_tenant_id    => :1,
				p_generated_id => :2,
				p_user_id      => :3,
				p_json_data    => v_json_data,
				p_content_hash => v_content_hash,
				p_generated_at => v_generated_at,
				p_success      => v_success,
				p_error_code   => v_error_code
			);
			:4 := v_json_data;
			:5 := v_content_hash;
			:6 := v_generated_at;
			:7 := v_success;
			:8 := v_error_code;
		END;`

	var (
		jsonData    sql.NullString
		contentHash sql.NullString
		generatedAt sql.NullTime
		success     int
		errorCode   sql.NullString
	)

	_, err := r.db.ExecContext(ctx, query,
		tenantID,
		generatedID,
		userID,
		sql.Out{Dest: &jsonData},
		sql.Out{Dest: &contentHash},
		sql.Out{Dest: &generatedAt},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorCode},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get generated content: %w", err)
	}

	if success != 1 {
		return nil, fmt.Errorf("access denied: %s", errorCode.String)
	}

	// Validate that generatedAt is valid before using it
	if !generatedAt.Valid {
		return nil, fmt.Errorf("generated_at is NULL for generatedID %d", generatedID)
	}

	return &models.GetGeneratedContentResponse{
		GeneratedID:  generatedID,
		ContentHash:  contentHash.String,
		GeneratedAt:  generatedAt.Time,
		ContractJSON: []byte(jsonData.String),
	}, nil
}

// GetLatestGenerated retrieves the most recent generated version for a contract
func (r *ContractGenerationRepository) GetLatestGenerated(
	ctx context.Context,
	tenantID string,
	contractID int64,
	userID string,
) (*models.GetGeneratedContentResponse, error) {
	query := `
		DECLARE
			v_json_data    CLOB;
			v_content_hash VARCHAR2(64);
			v_gen_id       NUMBER;
			v_gen_at       TIMESTAMP;
			v_success      NUMBER;
			v_error_code   VARCHAR2(50);
		BEGIN
			pkg_contract_generation.get_latest_generated(
				p_tenant_id    => :1,
				p_contract_id  => :2,
				p_user_id      => :3,
				p_json_data    => v_json_data,
				p_content_hash => v_content_hash,
				p_gen_id       => v_gen_id,
				p_gen_at       => v_gen_at,
				p_success      => v_success,
				p_error_code   => v_error_code
			);
			:4 := v_json_data;
			:5 := v_content_hash;
			:6 := v_gen_id;
			:7 := v_gen_at;
			:8 := v_success;
			:9 := v_error_code;
		END;`

	var (
		jsonData    sql.NullString
		contentHash sql.NullString
		generatedID sql.NullInt64
		generatedAt sql.NullTime
		success     int
		errorCode   sql.NullString
	)

	_, err := r.db.ExecContext(ctx, query,
		tenantID,
		contractID,
		userID,
		sql.Out{Dest: &jsonData},
		sql.Out{Dest: &contentHash},
		sql.Out{Dest: &generatedID},
		sql.Out{Dest: &generatedAt},
		sql.Out{Dest: &success},
		sql.Out{Dest: &errorCode},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest generated: %w", err)
	}

	if success != 1 {
		return nil, ErrNotFound
	}

	// Validate that generatedID is valid before using it
	if !generatedID.Valid {
		return nil, fmt.Errorf("generated contract ID is NULL despite success: generatedID.Valid=false")
	}

	// Validate that generatedAt is valid before using it
	if !generatedAt.Valid {
		return nil, fmt.Errorf("generated_at is NULL for generatedID %d", generatedID.Int64)
	}

	return &models.GetGeneratedContentResponse{
		GeneratedID:  generatedID.Int64,
		ContentHash:  contentHash.String,
		GeneratedAt:  generatedAt.Time,
		ContractJSON: []byte(jsonData.String),
	}, nil
}

// LogActionParams holds parameters for logging a contract action
type LogActionParams struct {
	TenantID    string
	ContractID  int64
	GeneratedID int64
	Action      string
	UserID      string
	IPAddress   string
	SessionID   string
	Status      string
	ErrorCode   string
}

// LogContractAction logs a contract action (view, download, print)
// Includes GeneratedID to attribute actions to specific generated versions
func (r *ContractGenerationRepository) LogContractAction(
	ctx context.Context,
	params LogActionParams,
) error {
	query := `
		BEGIN
			pkg_contract_generation.log_action(
				p_tenant_id    => :1,
				p_contract_id  => :2,
				p_generated_id => :3,
				p_action       => :4,
				p_user_id      => :5,
				p_ip_address   => :6,
				p_session_id   => :7,
				p_status       => :8,
				p_error_code   => :9
			);
		END;`

	var generatedIDParam sql.NullInt64
	if params.GeneratedID > 0 {
		generatedIDParam = sql.NullInt64{Int64: params.GeneratedID, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		params.TenantID,
		params.ContractID,
		generatedIDParam,
		params.Action,
		params.UserID,
		sql.NullString{String: params.IPAddress, Valid: params.IPAddress != ""},
		sql.NullString{String: params.SessionID, Valid: params.SessionID != ""},
		params.Status,
		sql.NullString{String: params.ErrorCode, Valid: params.ErrorCode != ""},
	)
	if err != nil {
		return fmt.Errorf("failed to log contract action: %w", err)
	}

	return nil
}

// ListGeneratedContracts lists all generated versions for a contract
// Does NOT return content - only metadata
func (r *ContractGenerationRepository) ListGeneratedContracts(
	ctx context.Context,
	tenantID string,
	contractID int64,
	limit int,
	offset int,
) ([]models.GeneratedContractListItem, int, error) {
	countQuery := `SELECT COUNT(*) FROM generated_contracts WHERE tenant_id = :1 AND contract_id = :2`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, tenantID, contractID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count generated contracts: %w", err)
	}

	if total == 0 {
		return []models.GeneratedContractListItem{}, 0, nil
	}

	// Note: We explicitly exclude content_html for security
	query := `
		SELECT id, contract_id, generation_number, content_hash,
		       customer_name_snapshot, total_value_snapshot, services_count_snapshot,
		       generated_at, generated_by, generation_reason
		FROM generated_contracts
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY generated_at DESC
		OFFSET :3 ROWS FETCH NEXT :4 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, tenantID, contractID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list generated contracts: %w", err)
	}
	defer rows.Close()

	var items []models.GeneratedContractListItem
	for rows.Next() {
		var item models.GeneratedContractListItem
		var genReason sql.NullString
		err := rows.Scan(
			&item.ID,
			&item.ContractID,
			&item.GenerationNumber,
			&item.ContentHash,
			&item.CustomerNameSnapshot,
			&item.TotalValueSnapshot,
			&item.ServicesCountSnapshot,
			&item.GeneratedAt,
			&item.GeneratedBy,
			&genReason,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan generated contract: %w", err)
		}
		if genReason.Valid {
			item.GenerationReason = models.ContractGenerationReason(genReason.String)
		} // else: leave as zero value (empty string)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, total, nil
}

// GetGenerationStats retrieves generation statistics for a tenant
func (r *ContractGenerationRepository) GetGenerationStats(
	ctx context.Context,
	tenantID string,
) (*models.GenerationStats, error) {
	query := `
		DECLARE
			v_total   NUMBER;
			v_today   NUMBER;
			v_month   NUMBER;
			v_unique  NUMBER;
		BEGIN
			pkg_contract_generation.get_stats(
				p_tenant_id => :1,
				p_total     => v_total,
				p_today     => v_today,
				p_month     => v_month,
				p_unique    => v_unique
			);
			:2 := v_total;
			:3 := v_today;
			:4 := v_month;
			:5 := v_unique;
		END;`

	var stats models.GenerationStats
	_, err := r.db.ExecContext(ctx, query,
		tenantID,
		sql.Out{Dest: &stats.TotalGenerated},
		sql.Out{Dest: &stats.GeneratedToday},
		sql.Out{Dest: &stats.GeneratedMonth},
		sql.Out{Dest: &stats.UniqueContracts},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation stats: %w", err)
	}

	return &stats, nil
}

// VerifyContentIntegrity verifies the integrity of a generated contract.
// Enforces tenant isolation by checking COUNT(*) WHERE id=:1 AND tenant_id=:2 first.
//
// Result code mapping from pkg_contract_generation.verify_integrity:
//   - 1: hash matches (content valid)
//   - 0: hash mismatch (content tampered)
//   - -1: COUNT(*) = 0, meaning tenant doesn't own the record OR record doesn't exist
//   - -2: record deleted between authorization check and verify call (TOCTOU race)
//
// Returns:
//   - (true, nil) if hash matches (content valid)
//   - (false, nil) if hash mismatch (content tampered)
//   - (false, ErrUnauthorized) if tenant doesn't own the record or record doesn't exist for this tenant
//   - (false, ErrNotFound) if record was deleted between the COUNT check and verify_integrity call (rare TOCTOU race)
func (r *ContractGenerationRepository) VerifyContentIntegrity(
	ctx context.Context,
	tenantID string,
	generatedID int64,
) (bool, error) {
	query := `
		DECLARE
			v_result     NUMBER;
			v_authorized NUMBER;
		BEGIN
			-- First check tenant authorization
			SELECT COUNT(*) INTO v_authorized
			FROM generated_contracts
			WHERE id = :1 AND tenant_id = :2;
			
			IF v_authorized = 0 THEN
				:3 := -1; -- Unauthorized (tenant mismatch or not found for tenant)
			ELSE
				-- verify_integrity returns: 1=valid, 0=tampered, -2=not found
				:3 := pkg_contract_generation.verify_integrity(:1);
			END IF;
		END;`

	var result int
	_, err := r.db.ExecContext(ctx, query, generatedID, tenantID, sql.Out{Dest: &result})
	if err != nil {
		return false, fmt.Errorf("failed to verify content integrity: %w", err)
	}

	switch result {
	case 1:
		return true, nil // Valid: hash matches
	case 0:
		return false, nil // Invalid: hash mismatch (tampered)
	case -1:
		return false, ErrUnauthorized // Tenant doesn't own this record
	case -2:
		return false, ErrNotFound // Record not found
	default:
		return false, fmt.Errorf("unexpected verify_integrity result: %d", result)
	}
}

// ListTemplates lists all active templates for a tenant
func (r *ContractGenerationRepository) ListTemplates(
	ctx context.Context,
	tenantID string,
) ([]models.ContractTemplate, error) {
	query := `
		SELECT id, tenant_id, template_code, template_name, language,
		       is_default, active, version, created_at, updated_at
		FROM contract_templates
		WHERE tenant_id = :1 AND active = 1
		ORDER BY is_default DESC, template_name`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []models.ContractTemplate
	for rows.Next() {
		var t models.ContractTemplate
		var isDefault, active int
		var createdAt, updatedAt sql.NullTime
		err := rows.Scan(
			&t.ID, &t.TenantID, &t.TemplateCode, &t.TemplateName, &t.Language,
			&isDefault, &active, &t.Version, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		t.IsDefault = isDefault == 1
		t.Active = active == 1
		if createdAt.Valid {
			t.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			t.UpdatedAt = updatedAt.Time
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return templates, nil
}

// InitTenantTemplate initializes the default template for a tenant
func (r *ContractGenerationRepository) InitTenantTemplate(
	ctx context.Context,
	tenantID string,
	userID string,
) error {
	query := `
		BEGIN
			pkg_contract_generation.init_default_template(:1, :2);
		END;`

	_, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to initialize tenant template: %w", err)
	}

	return nil
}

// CleanupExpiredGenerations removes expired generated contracts
func (r *ContractGenerationRepository) CleanupExpiredGenerations(
	ctx context.Context,
	tenantID string,
) (int, error) {
	query := `
		DECLARE
			v_deleted NUMBER;
		BEGIN
			pkg_contract_generation.cleanup_expired_generations(:1, v_deleted);
			:2 := v_deleted;
		END;`

	var deleted int
	var nullableTenantID sql.NullString
	if tenantID != "" {
		nullableTenantID = sql.NullString{String: tenantID, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query, nullableTenantID, sql.Out{Dest: &deleted})
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired generations: %w", err)
	}

	return deleted, nil
}
