package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zlovtnik/gprint/internal/models"
)

// PrintJobRepository handles print job data access
type PrintJobRepository struct {
	db *sql.DB
}

// NewPrintJobRepository creates a new PrintJobRepository
func NewPrintJobRepository(db *sql.DB) *PrintJobRepository {
	return &PrintJobRepository{db: db}
}

// Create creates a new print job
func (r *PrintJobRepository) Create(ctx context.Context, tenantID string, req *models.CreatePrintJobRequest, requestedBy string) (*models.ContractPrintJob, error) {
	if req == nil {
		return nil, fmt.Errorf("create print job request cannot be nil")
	}
	format := req.Format
	if format == "" {
		format = models.PrintFormatPDF
	}

	query := `
		INSERT INTO contract_print_jobs (
			tenant_id, contract_id, format, requested_by
		) VALUES (
			:1, :2, :3, :4
		) RETURNING id INTO :5`

	var id int64
	_, err := r.db.ExecContext(ctx, query, tenantID, req.ContractID, format, requestedBy, sql.Out{Dest: &id})
	if err != nil {
		return nil, fmt.Errorf("failed to create print job: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a print job by ID
func (r *PrintJobRepository) GetByID(ctx context.Context, tenantID string, id int64) (*models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM contract_print_jobs
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
func (r *PrintJobRepository) GetByContractID(ctx context.Context, tenantID string, contractID int64) ([]models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM contract_print_jobs
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
	countQuery := `SELECT COUNT(*) FROM contract_print_jobs WHERE tenant_id = :1`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting print jobs: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM contract_print_jobs
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

// UpdateStatus updates the print job status
func (r *PrintJobRepository) UpdateStatus(ctx context.Context, tenantID string, id int64, status models.PrintJobStatus, outputPath string, fileSize int64, pageCount int, errorMsg string) error {
	query := `
		UPDATE contract_print_jobs SET
			status = :1,
			output_path = :2,
			file_size = :3,
			page_count = :4,
			error_message = :5,
			started_at = CASE WHEN :1 = 'PROCESSING' AND started_at IS NULL THEN CURRENT_TIMESTAMP ELSE started_at END,
			completed_at = CASE WHEN :1 IN ('COMPLETED', 'FAILED') THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE tenant_id = :6 AND id = :7`

	result, err := r.db.ExecContext(ctx, query, string(status), outputPath, fileSize, pageCount, errorMsg, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to update print job status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("print job not found or tenant mismatch")
	}
	return nil
}

// GetPendingJobs retrieves pending print jobs
func (r *PrintJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.ContractPrintJob, error) {
	query := `
		SELECT id, tenant_id, contract_id, status, format,
			output_path, file_size, page_count,
			queued_at, started_at, completed_at,
			retry_count, error_message, requested_by
		FROM contract_print_jobs
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
