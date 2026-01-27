package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
	"github.com/zlovtnik/gprint/internal/clm/repository"
	"github.com/zlovtnik/gprint/pkg/fp"
)

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo      *repository.WorkflowRepository
	auditRepo *repository.AuditRepository
}

// NewWorkflowService creates a new WorkflowService
func NewWorkflowService(repo *repository.WorkflowRepository, auditRepo *repository.AuditRepository) *WorkflowService {
	if repo == nil {
		panic("workflow repository is required")
	}
	return &WorkflowService{repo: repo, auditRepo: auditRepo}
}

// CreateWorkflowRequest represents a request to create a workflow instance
type CreateWorkflowRequest struct {
	ContractID   domain.ContractID
	WorkflowType string
	Steps        []CreateWorkflowStepRequest
}

// CreateWorkflowStepRequest represents a step in a workflow creation
type CreateWorkflowStepRequest struct {
	StepNumber   int
	Name         string
	Type         domain.StepType
	AssigneeID   *domain.UserID
	AssigneeType string
	IsOptional   bool
	DueDate      *time.Time
}

// Create creates a new workflow instance
func (s *WorkflowService) Create(ctx context.Context, tenantID string, createdBy domain.UserID, req CreateWorkflowRequest) fp.Result[domain.WorkflowInstance] {
	if req.ContractID.IsZero() {
		return fp.Failure[domain.WorkflowInstance](errors.New("contract is required"))
	}
	if req.WorkflowType == "" {
		return fp.Failure[domain.WorkflowInstance](errors.New("workflow type is required"))
	}
	if len(req.Steps) == 0 {
		return fp.Failure[domain.WorkflowInstance](errors.New("at least one step is required"))
	}

	// Validate step numbering: must start at 1, be unique, and contiguous
	stepNumbers := make(map[int]bool)
	for _, step := range req.Steps {
		if step.StepNumber < 1 {
			return fp.Failure[domain.WorkflowInstance](fmt.Errorf("step number must be >= 1, got %d", step.StepNumber))
		}
		if stepNumbers[step.StepNumber] {
			return fp.Failure[domain.WorkflowInstance](fmt.Errorf("duplicate step number: %d", step.StepNumber))
		}
		stepNumbers[step.StepNumber] = true
	}
	// Check for contiguous sequence starting at 1
	for i := 1; i <= len(req.Steps); i++ {
		if !stepNumbers[i] {
			return fp.Failure[domain.WorkflowInstance](fmt.Errorf("step numbers must be contiguous starting at 1, missing step %d", i))
		}
	}

	id := domain.WorkflowID(uuid.New())
	now := time.Now()

	workflow := domain.WorkflowInstance{
		ID:           id,
		TenantID:     tenantID,
		ContractID:   req.ContractID,
		WorkflowType: req.WorkflowType,
		Status:       domain.WorkflowStatusPending,
		CurrentStep:  1,
		StartedAt:    now,
		CreatedBy:    createdBy,
	}

	// Create steps
	for _, stepReq := range req.Steps {
		step := domain.WorkflowStep{
			ID:           domain.WorkflowStepID(uuid.New()),
			WorkflowID:   id,
			StepNumber:   stepReq.StepNumber,
			Name:         stepReq.Name,
			Type:         stepReq.Type,
			Status:       domain.StepStatusPending,
			AssigneeID:   stepReq.AssigneeID,
			AssigneeType: stepReq.AssigneeType,
			IsOptional:   stepReq.IsOptional,
			DueDate:      stepReq.DueDate,
			CreatedAt:    now,
		}
		workflow.Steps = append(workflow.Steps, step)
	}

	// Start the workflow
	workflow = workflow.Start()

	result := s.repo.CreateInstance(ctx, workflow)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "workflow", uuid.UUID(id).String(), domain.AuditActionCreate, nil, &workflow, createdBy, "")
	}

	return result
}

// FindByID retrieves a workflow by ID
func (s *WorkflowService) FindByID(ctx context.Context, tenantID string, id domain.WorkflowID) fp.Result[domain.WorkflowInstance] {
	return s.repo.FindByID(ctx, tenantID, id)
}

// FindByContract retrieves all workflows for a contract
func (s *WorkflowService) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[[]domain.WorkflowInstance] {
	return s.repo.FindByContract(ctx, tenantID, contractID)
}

// FindPendingApprovals retrieves pending approval steps for a user
func (s *WorkflowService) FindPendingApprovals(ctx context.Context, tenantID string, userID domain.UserID) fp.Result[[]domain.WorkflowStep] {
	return s.repo.FindPendingApprovals(ctx, tenantID, userID)
}

// ApproveStep approves a workflow step
func (s *WorkflowService) ApproveStep(ctx context.Context, tenantID string, stepID domain.WorkflowStepID, userID domain.UserID, comment string) fp.Result[domain.WorkflowStep] {
	stepResult := s.repo.FindStepByID(ctx, tenantID, stepID)
	if fp.IsFailure(stepResult) {
		return stepResult
	}
	step := fp.GetValue(stepResult)

	if !step.IsPending() {
		return fp.Failure[domain.WorkflowStep](errors.New("step is not pending"))
	}

	updated := step.Approve(userID, comment)
	result := s.repo.UpdateStep(ctx, tenantID, updated)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "workflow_step", uuid.UUID(stepID).String(), domain.AuditActionApprove, &step, &updated, userID, "")
	}

	// Advance workflow if needed
	if fp.IsSuccess(result) {
		if err := s.advanceWorkflow(ctx, tenantID, step.WorkflowID); err != nil {
			log.Printf("failed to advance workflow %s: %v", step.WorkflowID, err)
		}
	}

	return result
}

// RejectStep rejects a workflow step
func (s *WorkflowService) RejectStep(ctx context.Context, tenantID string, stepID domain.WorkflowStepID, userID domain.UserID, comment string) fp.Result[domain.WorkflowStep] {
	stepResult := s.repo.FindStepByID(ctx, tenantID, stepID)
	if fp.IsFailure(stepResult) {
		return stepResult
	}
	step := fp.GetValue(stepResult)

	if !step.IsPending() {
		return fp.Failure[domain.WorkflowStep](errors.New("step is not pending"))
	}

	updated := step.Reject(userID, comment)
	result := s.repo.UpdateStep(ctx, tenantID, updated)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "workflow_step", uuid.UUID(stepID).String(), domain.AuditActionReject, &step, &updated, userID, "")
	}

	// Mark workflow as rejected if a non-optional step is rejected
	if fp.IsSuccess(result) && !step.IsOptional {
		workflowResult := s.repo.FindByID(ctx, tenantID, step.WorkflowID)
		if fp.IsSuccess(workflowResult) {
			workflow := fp.GetValue(workflowResult)
			// Verify tenant ownership
			if workflow.TenantID != tenantID {
				log.Printf("tenant mismatch when rejecting workflow: expected %s, got %s", tenantID, workflow.TenantID)
			} else {
				rejected := workflow.Reject()
				updateResult := s.repo.UpdateInstance(ctx, rejected)
				if fp.IsFailure(updateResult) {
					log.Printf("failed to update workflow status to rejected: %v", fp.GetError(updateResult))
				}
			}
		}
	}

	return result
}

// SkipStep skips an optional workflow step
func (s *WorkflowService) SkipStep(ctx context.Context, tenantID string, stepID domain.WorkflowStepID, userID domain.UserID, comment string) fp.Result[domain.WorkflowStep] {
	stepResult := s.repo.FindStepByID(ctx, tenantID, stepID)
	if fp.IsFailure(stepResult) {
		return stepResult
	}
	step := fp.GetValue(stepResult)

	if !step.IsOptional {
		return fp.Failure[domain.WorkflowStep](errors.New("only optional steps can be skipped"))
	}
	if !step.IsPending() {
		return fp.Failure[domain.WorkflowStep](errors.New("step is not pending"))
	}

	updated := step.Skip(userID, comment)
	result := s.repo.UpdateStep(ctx, tenantID, updated)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "workflow_step", uuid.UUID(stepID).String(), domain.AuditActionUpdate, &step, &updated, userID, "")
	}

	// Advance workflow
	if fp.IsSuccess(result) {
		if err := s.advanceWorkflow(ctx, tenantID, step.WorkflowID); err != nil {
			log.Printf("failed to advance workflow %s: %v", step.WorkflowID, err)
		}
	}

	return result
}

// CancelWorkflow cancels a workflow
func (s *WorkflowService) CancelWorkflow(ctx context.Context, tenantID string, id domain.WorkflowID) fp.Result[domain.WorkflowInstance] {
	workflowResult := s.repo.FindByID(ctx, tenantID, id)
	if fp.IsFailure(workflowResult) {
		return workflowResult
	}
	workflow := fp.GetValue(workflowResult)

	if workflow.Status != domain.WorkflowStatusPending && workflow.Status != domain.WorkflowStatusInProgress {
		return fp.Failure[domain.WorkflowInstance](errors.New("workflow cannot be cancelled"))
	}

	updated := workflow.Cancel()
	result := s.repo.UpdateInstance(ctx, updated)

	if fp.IsSuccess(result) && s.auditRepo != nil {
		s.createAudit(ctx, tenantID, "workflow", uuid.UUID(id).String(), domain.AuditActionStatusChange, &workflow, &updated, domain.UserID(uuid.Nil), "")
	}

	return result
}

func (s *WorkflowService) advanceWorkflow(ctx context.Context, tenantID string, workflowID domain.WorkflowID) error {
	workflowResult := s.repo.FindByID(ctx, tenantID, workflowID)
	if fp.IsFailure(workflowResult) {
		return fp.GetError(workflowResult)
	}
	workflow := fp.GetValue(workflowResult)

	if workflow.AllStepsComplete() {
		completed := workflow.Complete()
		result := s.repo.UpdateInstance(ctx, completed)
		if fp.IsFailure(result) {
			return fp.GetError(result)
		}
	} else {
		oldStep := workflow.CurrentStep
		workflow.AdvanceToNextStep()
		if workflow.CurrentStep != oldStep {
			result := s.repo.UpdateInstance(ctx, workflow)
			if fp.IsFailure(result) {
				return fp.GetError(result)
			}
		}
	}
	return nil
}

func (s *WorkflowService) createAudit(ctx context.Context, tenantID, entityType, entityID string, action domain.AuditAction, oldVal, newVal interface{}, userID domain.UserID, userName string) {
	if userName == "" {
		userName = "system"
	}
	entry := domain.AuditEntry{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Category:   domain.AuditCategoryWorkflow,
		UserID:     userID,
		UserName:   userName,
		Timestamp:  time.Now(),
	}

	if oldVal != nil {
		entry.OldValues = map[string]interface{}{"data": oldVal}
	}
	if newVal != nil {
		entry.NewValues = map[string]interface{}{"data": newVal}
	}

	result := s.auditRepo.Create(ctx, entry)
	if fp.IsFailure(result) {
		log.Printf("audit create failed for %s/%s: %v", entityType, entityID, fp.GetError(result))
	}
}
