package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zlovtnik/gprint/internal/models"
)

// TablePrintJobs is the table name for print job operations
const TablePrintJobs = "CONTRACT_PRINT_JOBS"

// PrintJobRepository handles print job data access
type PrintJobRepository struct {
	db      *sql.DB
	generic *GenericRepository
}

// UpdateStatusParams contains parameters for updating a print job's status.
// This groups related parameters to reduce function argument count.
type UpdateStatusParams struct {
	Status     models.PrintJobStatus
	OutputPath string
	FileSize   int64
	PageCount  int
	ErrorMsg   string
	UpdatedBy  string // User who triggered the update (for audit trail)
}

// NewPrintJobRepository creates a new PrintJobRepository
func NewPrintJobRepository(db *sql.DB) *PrintJobRepository {
	return &PrintJobRepository{
		db:      db,
		generic: NewGenericRepository(db),
	}
}

// Create creates a new print job using dynamic CRUD
func (r *PrintJobRepository) Create(ctx context.Context, tenantID string, req *models.CreatePrintJobRequest, requestedBy string) (*models.ContractPrintJob, error) {
	if req == nil {
		return nil, fmt.Errorf("create print job request cannot be nil")
	}
	format := req.Format
	if format == "" {
		format = models.PrintFormatPDF
	}

	columns := []ColumnValue{
		{Name: "CONTRACT_ID", Value: req.ContractID, Type: "NUMBER"},
		{Name: "FORMAT", Value: string(format), Type: "STRING"},
		{Name: "REQUESTED_BY", Value: requestedBy, Type: "STRING"},
		{Name: "STATUS", Value: string(models.PrintJobStatusQueued), Type: "STRING"},
	}

	result, err := r.generic.Insert(ctx, TablePrintJobs, tenantID, columns, requestedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create print job: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("failed to create print job: %s", result.ErrorMessage)
	}
	if result.GeneratedID == nil {
		return nil, fmt.Errorf("failed to create print job: no ID returned")
	}

	return r.GetByID(ctx, tenantID, *result.GeneratedID)
}

// GetByID retrieves a print job by ID.
// NOTE: Stored procedure sp_get_print_job is available but this method uses inline SQL
// for Go driver compatibility. Create uses generic.Insert and UpdateStatus uses generic.Update.
// FUTURE: Migrate to sp_get_print_job if/when ref cursor handling is needed.
func (r *PrintJobRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM ` + TablePrintJobs + `
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, id)
	job, err := scanPrintJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get print job: %w", err)
	}

	return &job, nil
}

// GetByContractID retrieves print jobs for a contract
// Stored procedure sp_get_print_jobs_by_contract available for ref cursor usage
func (r *PrintJobRepository) GetByContractID(ctx context.Context, tenantID string, contractID int64) ([]models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM ` + TablePrintJobs + `
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY queued_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to list print jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.ContractPrintJob
	for rows.Next() {
		job, err := scanPrintJob(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan print job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contract jobs: %w", err)
	}

	return jobs, nil
}

// FindAll retrieves all print jobs for a tenant with pagination
func (r *PrintJobRepository) FindAll(ctx context.Context, tenantID string, offset, limit int) ([]models.ContractPrintJob, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM ` + TablePrintJobs + ` WHERE tenant_id = :1`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting print jobs: %w", err)
	}

	// Get paginated results
	// Stored procedure sp_list_print_jobs available for ref cursor usage
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM ` + TablePrintJobs + `
		WHERE tenant_id = :1
		ORDER BY queued_at DESC
		OFFSET :2 ROWS FETCH NEXT :3 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying print jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.ContractPrintJob
	for rows.Next() {
		job, err := scanPrintJob(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning print job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating print jobs: %w", err)
	}

	return jobs, total, nil
}

// UpdateStatus updates the print job status using dynamic CRUD
func (r *PrintJobRepository) UpdateStatus(ctx context.Context, tenantID string, id int64, params UpdateStatusParams) error {
	columns := []ColumnValue{
		{Name: "STATUS", Value: string(params.Status), Type: "STRING"},
	}
	// Add optional fields if present
	if params.OutputPath != "" {
		columns = append(columns, ColumnValue{Name: "OUTPUT_PATH", Value: params.OutputPath, Type: "STRING"})
	}
	if params.FileSize > 0 {
		columns = append(columns, ColumnValue{Name: "FILE_SIZE", Value: params.FileSize, Type: "NUMBER"})
	}
	if params.PageCount > 0 {
		columns = append(columns, ColumnValue{Name: "PAGE_COUNT", Value: params.PageCount, Type: "NUMBER"})
	}
	if params.ErrorMsg != "" {
		columns = append(columns, ColumnValue{Name: "ERROR_MESSAGE", Value: params.ErrorMsg, Type: "STRING"})
	}
	// Set timestamps based on status
	if params.Status == models.PrintJobStatusProcessing {
		columns = append(columns, ColumnValue{Name: "STARTED_AT", Value: "SYSDATE", Type: "DATE"})
	}
	if params.Status == models.PrintJobStatusCompleted || params.Status == models.PrintJobStatusFailed {
		columns = append(columns, ColumnValue{Name: "COMPLETED_AT", Value: "SYSDATE", Type: "DATE"})
	}

	result, err := r.generic.Update(ctx, TablePrintJobs, tenantID, id, columns, params.UpdatedBy)
	if err != nil {
		return fmt.Errorf("failed to update print job status: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("failed to update print job status: %s", result.ErrorMessage)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("print job not found or tenant mismatch")
	}
	return nil
}

// GetPendingJobs retrieves pending print jobs
// Stored procedure sp_get_pending_print_jobs available for ref cursor usage
func (r *PrintJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM ` + TablePrintJobs + `
		WHERE status = :1
		ORDER BY queued_at ASC
		FETCH FIRST :2 ROWS ONLY`

	rows, err := r.db.QueryContext(ctx, query, string(models.PrintJobStatusQueued), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.ContractPrintJob
	for rows.Next() {
		job, err := scanPrintJob(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan print job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending jobs: %w", err)
	}

	return jobs, nil
}

type printJobScanner interface {
	Scan(dest ...any) error
}

func scanPrintJob(scanner printJobScanner) (models.ContractPrintJob, error) {
	var job models.ContractPrintJob
	var outputPath, errorMessage sql.NullString
	var fileSize, pageCount sql.NullInt64
	var startedAt, completedAt sql.NullTime

	if err := scanner.Scan(
		&job.ID, &job.TenantID, &job.ContractID, &job.Status, &job.Format,
		&outputPath, &fileSize, &pageCount,
		&job.QueuedAt, &startedAt, &completedAt,
		&job.RetryCount, &errorMessage, &job.RequestedBy,
	); err != nil {
		return models.ContractPrintJob{}, err
	}

	job.OutputPath = outputPath.String
	job.FileSize = fileSize.Int64
	job.PageCount = int(pageCount.Int64)
	job.ErrorMessage = errorMessage.String
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return job, nil
}
