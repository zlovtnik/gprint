package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/repository"
)

// PrintService handles print job business logic
type PrintService struct {
	printJobRepo *repository.PrintJobRepository
	contractRepo *repository.ContractRepository
	historyRepo  *repository.HistoryRepository
	outputDir    string
	logger       *slog.Logger
}

// NewPrintService creates a new PrintService
func NewPrintService(
	printJobRepo *repository.PrintJobRepository,
	contractRepo *repository.ContractRepository,
	historyRepo *repository.HistoryRepository,
	outputDir string,
	logger *slog.Logger,
) (*PrintService, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &PrintService{
		printJobRepo: printJobRepo,
		contractRepo: contractRepo,
		historyRepo:  historyRepo,
		outputDir:    outputDir,
		logger:       logger,
	}, nil
}

// CreateJob creates a new print job
func (s *PrintService) CreateJob(ctx context.Context, tenantID string, contractID int64, format models.PrintFormat, requestedBy string) (*models.ContractPrintJob, error) {
	// Verify contract exists
	contract, err := s.contractRepo.GetByID(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return nil, ErrContractNotFound
	}

	req := &models.CreatePrintJobRequest{
		ContractID: contractID,
		Format:     format,
	}

	job, err := s.printJobRepo.Create(ctx, tenantID, req, requestedBy)
	if err != nil {
		return nil, err
	}

	// Record history
	if _, err := s.historyRepo.Create(ctx, tenantID, &models.CreateHistoryRequest{
		ContractID:  contractID,
		Action:      models.HistoryActionPrint,
		NewValue:    string(format),
		PerformedBy: requestedBy,
	}); err != nil {
		s.logger.Error("failed to create history entry",
			"tenant_id", tenantID,
			"contract_id", contractID,
			"requested_by", requestedBy,
			"error", err,
		)
	}

	return job, nil
}

// GetJob retrieves a print job by ID
func (s *PrintService) GetJob(ctx context.Context, tenantID string, id int64) (*models.ContractPrintJob, error) {
	return s.printJobRepo.GetByID(ctx, tenantID, id)
}

// GetJobsByContract retrieves print jobs for a contract
func (s *PrintService) GetJobsByContract(ctx context.Context, tenantID string, contractID int64) ([]models.ContractPrintJob, error) {
	return s.printJobRepo.GetByContractID(ctx, tenantID, contractID)
}

// List retrieves all print jobs for a tenant with pagination
func (s *PrintService) List(ctx context.Context, tenantID string, page, pageSize int) ([]models.ContractPrintJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	return s.printJobRepo.FindAll(ctx, tenantID, offset, pageSize)
}

// ProcessPendingJobs processes pending print jobs (to be called by a background worker)
func (s *PrintService) ProcessPendingJobs(ctx context.Context) error {
	jobs, err := s.printJobRepo.GetPendingJobs(ctx, 10)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := s.processJob(ctx, &job); err != nil {
			s.logger.Error("failed to process print job",
				"job_id", job.ID,
				"contract_id", job.ContractID,
				"error", err,
			)
		}
	}

	return nil
}

// processJob processes a single print job
func (s *PrintService) processJob(ctx context.Context, job *models.ContractPrintJob) error {
	// Update status to processing
	if err := s.printJobRepo.UpdateStatus(ctx, job.TenantID, job.ID, repository.UpdateStatusParams{
		Status: models.PrintJobStatusProcessing,
	}); err != nil {
		return err
	}

	// Get contract with items
	contract, err := s.contractRepo.GetByID(ctx, job.TenantID, job.ContractID)
	if err != nil {
		if err2 := s.printJobRepo.UpdateStatus(ctx, job.TenantID, job.ID, repository.UpdateStatusParams{
			Status:   models.PrintJobStatusFailed,
			ErrorMsg: err.Error(),
		}); err2 != nil {
			s.logger.Error("failed to update job status after GetByID error",
				"job_id", job.ID,
				"tenant_id", job.TenantID,
				"original_error", err.Error(),
				"update_error", err2.Error(),
			)
		}
		return err
	}
	if contract == nil {
		errMsg := "contract not found"
		if err2 := s.printJobRepo.UpdateStatus(ctx, job.TenantID, job.ID, repository.UpdateStatusParams{
			Status:   models.PrintJobStatusFailed,
			ErrorMsg: errMsg,
		}); err2 != nil {
			s.logger.Error("failed to update job status for not found contract",
				"job_id", job.ID,
				"tenant_id", job.TenantID,
				"update_error", err2.Error(),
			)
		}
		return errors.New(errMsg)
	}

	// Generate document
	outputPath, fileSize, pageCount, err := s.generateDocument(contract, job.Format)
	if err != nil {
		if err2 := s.printJobRepo.UpdateStatus(ctx, job.TenantID, job.ID, repository.UpdateStatusParams{
			Status:   models.PrintJobStatusFailed,
			ErrorMsg: err.Error(),
		}); err2 != nil {
			s.logger.Error("failed to update job status after generateDocument error",
				"job_id", job.ID,
				"tenant_id", job.TenantID,
				"original_error", err.Error(),
				"update_error", err2.Error(),
			)
		}
		return err
	}

	// Update status to completed
	return s.printJobRepo.UpdateStatus(ctx, job.TenantID, job.ID, repository.UpdateStatusParams{
		Status:     models.PrintJobStatusCompleted,
		OutputPath: outputPath,
		FileSize:   fileSize,
		PageCount:  pageCount,
	})
}

// generateDocument generates the contract document
func (s *PrintService) generateDocument(contract *models.Contract, format models.PrintFormat) (string, int64, int, error) {
	// Sanitize contract number for safe filename
	safeContractNumber := sanitizeFilename(contract.ContractNumber)
	if safeContractNumber == "" {
		safeContractNumber = "unknown"
	}

	// Generate filename
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("contract_%s_%s", safeContractNumber, timestamp)

	var ext string
	switch format {
	case models.PrintFormatPDF:
		ext = ".pdf"
	case models.PrintFormatDOCX:
		ext = ".docx"
	case models.PrintFormatHTML:
		ext = ".html"
	default:
		ext = ".pdf"
	}

	outputPath := filepath.Join(s.outputDir, contract.TenantID, filename+ext)

	// Ensure tenant directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", 0, 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate HTML content (base for all formats)
	htmlContent := s.generateHTML(contract)

	switch format {
	case models.PrintFormatHTML:
		if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
			return "", 0, 0, fmt.Errorf("failed to write HTML: %w", err)
		}
	case models.PrintFormatPDF:
		// NOTE: PDF conversion requires external dependency (wkhtmltopdf or chromedp)
		return "", 0, 0, fmt.Errorf("%w: PDF export not implemented", ErrFormatNotSupported)
	case models.PrintFormatDOCX:
		// NOTE: DOCX conversion requires external dependency (unioffice)
		return "", 0, 0, fmt.Errorf("%w: DOCX export not implemented", ErrFormatNotSupported)
	default:
		return "", 0, 0, fmt.Errorf("%w: unrecognized format %s", ErrFormatNotSupported, format)
	}

	// Get file info
	info, err := os.Stat(outputPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to stat output file: %w", err)
	}

	return outputPath, info.Size(), 1, nil // pageCount is estimated
}

// sanitizeFilename removes or replaces characters that are unsafe for filenames
func sanitizeFilename(name string) string {
	// Remove path separators and other unsafe characters
	re := regexp.MustCompile(`[/\\:*?"<>|.\s]+`)
	safe := re.ReplaceAllString(name, "_")
	// Trim leading/trailing underscores
	safe = regexp.MustCompile(`^_+|_+$`).ReplaceAllString(safe, "")
	// Limit to alphanumeric, underscore, and hyphen
	safe = regexp.MustCompile(`[^A-Za-z0-9_-]`).ReplaceAllString(safe, "")
	return safe
}

// generateHTML generates HTML content for the contract
func (s *PrintService) generateHTML(contract *models.Contract) string {
	// Escape user-provided content to prevent XSS
	escapedContractNumber := html.EscapeString(contract.ContractNumber)
	escapedContractType := html.EscapeString(string(contract.ContractType))
	escapedStatus := html.EscapeString(string(contract.Status))
	escapedTermsConditions := html.EscapeString(contract.TermsConditions)

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Contract %s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .section { margin: 20px 0; }
        .label { font-weight: bold; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 10px; text-align: left; }
        th { background-color: #f5f5f5; }
        .total { font-size: 1.2em; font-weight: bold; text-align: right; }
    </style>
</head>
<body>
    <h1>Service Contract</h1>
    <div class="section">
        <p><span class="label">Contract Number:</span> %s</p>
        <p><span class="label">Type:</span> %s</p>
        <p><span class="label">Status:</span> %s</p>
        <p><span class="label">Start Date:</span> %s</p>
    </div>
    
    <h2>Services</h2>
    <table>
        <tr>
            <th>Description</th>
            <th>Quantity</th>
            <th>Unit Price</th>
            <th>Discount</th>
            <th>Total</th>
        </tr>`,
		escapedContractNumber,
		escapedContractNumber,
		escapedContractType,
		escapedStatus,
		contract.StartDate.Format("2006-01-02"),
	)

	for _, item := range contract.Items {
		desc := item.Description
		if desc == "" && item.Service != nil {
			desc = item.Service.Name
		}
		escapedDesc := html.EscapeString(desc)
		htmlContent += fmt.Sprintf(`
        <tr>
            <td>%s</td>
            <td>%.2f</td>
            <td>R$ %.2f</td>
            <td>%.1f%%</td>
            <td>R$ %.2f</td>
        </tr>`,
			escapedDesc,
			item.Quantity.InexactFloat64(),
			item.UnitPrice.InexactFloat64(),
			item.DiscountPct.InexactFloat64(),
			item.LineTotal.InexactFloat64(),
		)
	}

	htmlContent += fmt.Sprintf(`
    </table>
    <p class="total">Total: R$ %.2f</p>
    
    <div class="section">
        <h2>Terms and Conditions</h2>
        <p>%s</p>
    </div>
</body>
</html>`,
		contract.TotalValue.InexactFloat64(),
		escapedTermsConditions,
	)

	return htmlContent
}

// DownloadJob returns the file path for a completed job
func (s *PrintService) DownloadJob(ctx context.Context, tenantID string, jobID int64) (string, error) {
	job, err := s.printJobRepo.GetByID(ctx, tenantID, jobID)
	if err != nil {
		return "", err
	}
	if job == nil {
		return "", ErrPrintJobNotFound
	}

	if job.Status != models.PrintJobStatusCompleted {
		return "", fmt.Errorf("%w: current status is %s", ErrJobNotCompleted, job.Status)
	}

	if job.OutputPath == "" {
		return "", ErrOutputFileNotFound
	}

	// Verify the file exists on disk
	if _, err := os.Stat(job.OutputPath); os.IsNotExist(err) {
		return "", ErrOutputFileNotFound
	} else if err != nil {
		return "", fmt.Errorf("failed to access output file: %w", err)
	}

	return job.OutputPath, nil
}
