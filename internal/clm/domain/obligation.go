package domain

import (
	"time"
)

// ObligationType represents the type of obligation
type ObligationType string

const (
	ObligationTypePayment     ObligationType = "PAYMENT"
	ObligationTypeDelivery    ObligationType = "DELIVERY"
	ObligationTypeService     ObligationType = "SERVICE"
	ObligationTypeReporting   ObligationType = "REPORTING"
	ObligationTypeCompliance  ObligationType = "COMPLIANCE"
	ObligationTypeMilestone   ObligationType = "MILESTONE"
	ObligationTypeRenewal     ObligationType = "RENEWAL"
	ObligationTypeTermination ObligationType = "TERMINATION"
)

// ObligationStatus represents the status of an obligation
type ObligationStatus string

const (
	ObligationStatusPending    ObligationStatus = "PENDING"
	ObligationStatusInProgress ObligationStatus = "IN_PROGRESS"
	ObligationStatusCompleted  ObligationStatus = "COMPLETED"
	ObligationStatusOverdue    ObligationStatus = "OVERDUE"
	ObligationStatusWaived     ObligationStatus = "WAIVED"
	ObligationStatusCancelled  ObligationStatus = "CANCELLED"
)

// ObligationFrequency represents the recurrence of an obligation
type ObligationFrequency string

const (
	ObligationFrequencyOnce      ObligationFrequency = "ONCE"
	ObligationFrequencyDaily     ObligationFrequency = "DAILY"
	ObligationFrequencyWeekly    ObligationFrequency = "WEEKLY"
	ObligationFrequencyMonthly   ObligationFrequency = "MONTHLY"
	ObligationFrequencyQuarterly ObligationFrequency = "QUARTERLY"
	ObligationFrequencyAnnually  ObligationFrequency = "ANNUALLY"
)

// Obligation represents a contractual obligation (immutable)
type Obligation struct {
	ID               ObligationID        `json:"id"`
	TenantID         string              `json:"tenant_id"`
	ContractID       ContractID          `json:"contract_id"`
	Type             ObligationType      `json:"type"`
	Title            string              `json:"title"`
	Description      string              `json:"description,omitempty"`
	ResponsibleParty PartyID             `json:"responsible_party"`
	Status           ObligationStatus    `json:"status"`
	DueDate          time.Time           `json:"due_date"`
	CompletedDate    *time.Time          `json:"completed_date,omitempty"`
	Amount           *Money              `json:"amount,omitempty"`
	Frequency        ObligationFrequency `json:"frequency"`
	ReminderDays     int                 `json:"reminder_days"`
	Notes            string              `json:"notes,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        *time.Time          `json:"updated_at,omitempty"`
	CreatedBy        UserID              `json:"created_by"`
	UpdatedBy        *UserID             `json:"updated_by,omitempty"`
}

// WithTitle returns a copy with updated title
func (o Obligation) WithTitle(title string, updatedBy UserID) Obligation {
	now := time.Now()
	o.Title = title
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// WithDescription returns a copy with updated description
func (o Obligation) WithDescription(description string, updatedBy UserID) Obligation {
	now := time.Now()
	o.Description = description
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// WithDueDate returns a copy with updated due date
func (o Obligation) WithDueDate(dueDate time.Time, updatedBy UserID) Obligation {
	now := time.Now()
	o.DueDate = dueDate
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// WithAmount returns a copy with updated amount
func (o Obligation) WithAmount(amount *Money, updatedBy UserID) Obligation {
	now := time.Now()
	o.Amount = amount
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// WithNotes returns a copy with updated notes
func (o Obligation) WithNotes(notes string, updatedBy UserID) Obligation {
	now := time.Now()
	o.Notes = notes
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// MarkInProgress marks the obligation as in progress
func (o Obligation) MarkInProgress(updatedBy UserID) Obligation {
	now := time.Now()
	o.Status = ObligationStatusInProgress
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// Complete marks the obligation as completed
func (o Obligation) Complete(updatedBy UserID) Obligation {
	now := time.Now()
	o.Status = ObligationStatusCompleted
	o.CompletedDate = &now
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// MarkOverdue marks the obligation as overdue
func (o Obligation) MarkOverdue(updatedBy UserID) Obligation {
	now := time.Now()
	o.Status = ObligationStatusOverdue
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// Waive waives the obligation
func (o Obligation) Waive(updatedBy UserID) Obligation {
	now := time.Now()
	o.Status = ObligationStatusWaived
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// Cancel cancels the obligation
func (o Obligation) Cancel(updatedBy UserID) Obligation {
	now := time.Now()
	o.Status = ObligationStatusCancelled
	o.UpdatedAt = &now
	o.UpdatedBy = &updatedBy
	return o
}

// IsOverdue checks if the obligation is overdue
func (o Obligation) IsOverdue() bool {
	if o.Status == ObligationStatusCompleted || o.Status == ObligationStatusWaived || o.Status == ObligationStatusCancelled {
		return false
	}
	return time.Now().After(o.DueDate)
}

// IsDueSoon checks if the obligation is due within the reminder period
func (o Obligation) IsDueSoon() bool {
	if o.Status != ObligationStatusPending && o.Status != ObligationStatusInProgress {
		return false
	}
	reminderDate := o.DueDate.AddDate(0, 0, -o.ReminderDays)
	return time.Now().After(reminderDate) && time.Now().Before(o.DueDate)
}

// DaysUntilDue returns the number of days until due
func (o Obligation) DaysUntilDue() int {
	return int(time.Until(o.DueDate).Hours() / 24)
}

// NewObligation creates a new Obligation
func NewObligation(
	tenantID string,
	contractID ContractID,
	obligationType ObligationType,
	title string,
	responsibleParty PartyID,
	dueDate time.Time,
	frequency ObligationFrequency,
	createdBy UserID,
) Obligation {
	return Obligation{
		ID:               NewObligationID(),
		TenantID:         tenantID,
		ContractID:       contractID,
		Type:             obligationType,
		Title:            title,
		ResponsibleParty: responsibleParty,
		Status:           ObligationStatusPending,
		DueDate:          dueDate,
		Frequency:        frequency,
		ReminderDays:     7,
		CreatedAt:        time.Now(),
		CreatedBy:        createdBy,
	}
}
