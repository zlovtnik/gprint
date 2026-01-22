package models

import "time"

// PrintJobStatus represents the status of a print job
type PrintJobStatus string

const (
	PrintJobStatusQueued     PrintJobStatus = "QUEUED"
	PrintJobStatusProcessing PrintJobStatus = "PROCESSING"
	PrintJobStatusCompleted  PrintJobStatus = "COMPLETED"
	PrintJobStatusFailed     PrintJobStatus = "FAILED"
)

// PrintFormat represents the output format
type PrintFormat string

const (
	PrintFormatPDF  PrintFormat = "PDF"
	PrintFormatDOCX PrintFormat = "DOCX"
	PrintFormatHTML PrintFormat = "HTML"
)

// ContractPrintJob represents a contract printing job
type ContractPrintJob struct {
	ID           int64          `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ContractID   int64          `json:"contract_id"`
	Status       PrintJobStatus `json:"status"`
	Format       PrintFormat    `json:"format"`
	OutputPath   string         `json:"output_path,omitempty"`
	FileSize     int64          `json:"file_size,omitempty"`
	PageCount    int            `json:"page_count,omitempty"`
	QueuedAt     time.Time      `json:"queued_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	RetryCount   int            `json:"retry_count"`
	ErrorMessage string         `json:"error_message,omitempty"`
	RequestedBy  string         `json:"requested_by"`
}

// CreatePrintJobRequest represents the request to create a print job
type CreatePrintJobRequest struct {
	ContractID int64       `json:"contract_id"`
	Format     PrintFormat `json:"format"`
}

// PrintJobResponse represents the API response for a print job
type PrintJobResponse struct {
	ID          int64          `json:"id"`
	ContractID  int64          `json:"contract_id"`
	Status      PrintJobStatus `json:"status"`
	Format      PrintFormat    `json:"format"`
	FileSize    int64          `json:"file_size,omitempty"`
	PageCount   int            `json:"page_count,omitempty"`
	QueuedAt    time.Time      `json:"queued_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	RequestedBy string         `json:"requested_by"`
}

// ToResponse converts a ContractPrintJob to PrintJobResponse
func (j *ContractPrintJob) ToResponse() PrintJobResponse {
	return PrintJobResponse{
		ID:          j.ID,
		ContractID:  j.ContractID,
		Status:      j.Status,
		Format:      j.Format,
		FileSize:    j.FileSize,
		PageCount:   j.PageCount,
		QueuedAt:    j.QueuedAt,
		StartedAt:   j.StartedAt,
		CompletedAt: j.CompletedAt,
		RequestedBy: j.RequestedBy,
	}
}
