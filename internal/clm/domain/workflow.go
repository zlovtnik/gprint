package domain

import (
	"time"
)

// WorkflowType represents the type of workflow
type WorkflowType string

const (
	WorkflowTypeApproval    WorkflowType = "APPROVAL"
	WorkflowTypeReview      WorkflowType = "REVIEW"
	WorkflowTypeSignature   WorkflowType = "SIGNATURE"
	WorkflowTypeNegotiation WorkflowType = "NEGOTIATION"
)

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusPending    WorkflowStatus = "PENDING"
	WorkflowStatusInProgress WorkflowStatus = "IN_PROGRESS"
	WorkflowStatusCompleted  WorkflowStatus = "COMPLETED"
	WorkflowStatusRejected   WorkflowStatus = "REJECTED"
	WorkflowStatusCancelled  WorkflowStatus = "CANCELLED"
)

// StepType represents the type of workflow step
type StepType string

const (
	StepTypeApproval  StepType = "APPROVAL"
	StepTypeReview    StepType = "REVIEW"
	StepTypeSignature StepType = "SIGNATURE"
	StepTypeTask      StepType = "TASK"
)

// StepStatus represents the status of a workflow step
type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusApproved  StepStatus = "APPROVED"
	StepStatusRejected  StepStatus = "REJECTED"
	StepStatusSkipped   StepStatus = "SKIPPED"
	StepStatusCompleted StepStatus = "COMPLETED"
)

// WorkflowStep represents a step in a workflow (immutable)
type WorkflowStep struct {
	ID            WorkflowStepID `json:"id"`
	WorkflowID    WorkflowID     `json:"workflow_id"`
	StepNumber    int            `json:"step_number"`
	Name          string         `json:"name"`
	Type          StepType       `json:"type"`
	Status        StepStatus     `json:"status"`
	AssigneeID    *UserID        `json:"assignee_id,omitempty"`
	AssigneeType  string         `json:"assignee_type,omitempty"`
	IsOptional    bool           `json:"is_optional"`
	ActionDate    *time.Time     `json:"action_date,omitempty"`
	ActionBy      *UserID        `json:"action_by,omitempty"`
	ActionComment string         `json:"action_comment,omitempty"`
	DueDate       *time.Time     `json:"due_date,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// Approve marks the step as approved
func (s WorkflowStep) Approve(actionBy UserID, comment string) WorkflowStep {
	now := time.Now()
	s.Status = StepStatusApproved
	s.ActionDate = &now
	s.ActionBy = &actionBy
	s.ActionComment = comment
	return s
}

// Reject marks the step as rejected
func (s WorkflowStep) Reject(actionBy UserID, reason string) WorkflowStep {
	now := time.Now()
	s.Status = StepStatusRejected
	s.ActionDate = &now
	s.ActionBy = &actionBy
	s.ActionComment = reason
	return s
}

// Skip marks the step as skipped
func (s WorkflowStep) Skip(actionBy UserID, reason string) WorkflowStep {
	now := time.Now()
	s.Status = StepStatusSkipped
	s.ActionDate = &now
	s.ActionBy = &actionBy
	s.ActionComment = reason
	return s
}

// Complete marks the step as completed
func (s WorkflowStep) Complete(actionBy UserID, comment string) WorkflowStep {
	now := time.Now()
	s.Status = StepStatusCompleted
	s.ActionDate = &now
	s.ActionBy = &actionBy
	s.ActionComment = comment
	return s
}

// IsPending checks if the step is pending
func (s WorkflowStep) IsPending() bool {
	return s.Status == StepStatusPending
}

// IsComplete checks if the step is complete (approved, skipped, or completed)
func (s WorkflowStep) IsComplete() bool {
	return s.Status == StepStatusApproved || s.Status == StepStatusSkipped || s.Status == StepStatusCompleted
}

// WorkflowInstance represents a workflow instance (immutable)
type WorkflowInstance struct {
	ID           WorkflowID     `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ContractID   ContractID     `json:"contract_id"`
	WorkflowType string         `json:"workflow_type"`
	Status       WorkflowStatus `json:"status"`
	CurrentStep  int            `json:"current_step"`
	Steps        []WorkflowStep `json:"steps"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	CreatedBy    UserID         `json:"created_by"`
}

// Start marks the workflow as in progress
func (w WorkflowInstance) Start() WorkflowInstance {
	w.Status = WorkflowStatusInProgress
	return w
}

// Complete marks the workflow as completed
func (w WorkflowInstance) Complete() WorkflowInstance {
	now := time.Now()
	w.Status = WorkflowStatusCompleted
	w.CompletedAt = &now
	return w
}

// Reject marks the workflow as rejected
func (w WorkflowInstance) Reject() WorkflowInstance {
	now := time.Now()
	w.Status = WorkflowStatusRejected
	w.CompletedAt = &now
	return w
}

// Cancel marks the workflow as cancelled
func (w WorkflowInstance) Cancel() WorkflowInstance {
	now := time.Now()
	w.Status = WorkflowStatusCancelled
	w.CompletedAt = &now
	return w
}

// AdvanceToNextStep advances to the next step
func (w *WorkflowInstance) AdvanceToNextStep() {
	if w.CurrentStep < len(w.Steps) {
		w.CurrentStep++
	}
}

// GetCurrentStep returns the current step
func (w WorkflowInstance) GetCurrentStep() (WorkflowStep, bool) {
	for i := range w.Steps {
		if w.Steps[i].StepNumber == w.CurrentStep {
			return w.Steps[i], true
		}
	}
	return WorkflowStep{}, false
}

// UpdateStep updates a step in the workflow
func (w WorkflowInstance) UpdateStep(step WorkflowStep) WorkflowInstance {
	// Create a deep copy of the Steps slice
	oldSteps := w.Steps
	w.Steps = make([]WorkflowStep, len(oldSteps))
	copy(w.Steps, oldSteps)

	for i := range w.Steps {
		if w.Steps[i].ID == step.ID {
			w.Steps[i] = step
			break
		}
	}
	return w
}

// AllStepsComplete checks if all required steps are complete
func (w WorkflowInstance) AllStepsComplete() bool {
	for _, step := range w.Steps {
		if !step.IsOptional && !step.IsComplete() {
			return false
		}
	}
	return true
}

// HasRejectedSteps checks if any step is rejected
func (w WorkflowInstance) HasRejectedSteps() bool {
	for _, step := range w.Steps {
		if step.Status == StepStatusRejected {
			return true
		}
	}
	return false
}

// NewWorkflowInstance creates a new WorkflowInstance
func NewWorkflowInstance(
	tenantID string,
	contractID ContractID,
	workflowType string,
	steps []WorkflowStep,
	createdBy UserID,
) WorkflowInstance {
	return WorkflowInstance{
		ID:           NewWorkflowID(),
		TenantID:     tenantID,
		ContractID:   contractID,
		WorkflowType: workflowType,
		Status:       WorkflowStatusPending,
		CurrentStep:  1,
		Steps:        steps,
		StartedAt:    time.Now(),
		CreatedBy:    createdBy,
	}
}

// NewWorkflowStep creates a new WorkflowStep
func NewWorkflowStep(
	workflowID WorkflowID,
	stepNumber int,
	name string,
	stepType StepType,
	assigneeID *UserID,
	assigneeType string,
	isOptional bool,
) WorkflowStep {
	return WorkflowStep{
		ID:           NewWorkflowStepID(),
		WorkflowID:   workflowID,
		StepNumber:   stepNumber,
		Name:         name,
		Type:         stepType,
		Status:       StepStatusPending,
		AssigneeID:   assigneeID,
		AssigneeType: assigneeType,
		IsOptional:   isOptional,
		CreatedAt:    time.Now(),
	}
}
