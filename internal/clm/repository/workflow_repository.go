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

// WorkflowRepository handles workflow persistence
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new WorkflowRepository
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// CreateInstance creates a new workflow instance with its steps
func (r *WorkflowRepository) CreateInstance(ctx context.Context, workflow domain.WorkflowInstance) fp.Result[domain.WorkflowInstance] {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}
	defer tx.Rollback()

	// Insert workflow instance
	query := `
		INSERT INTO clm_workflow_instances (
			id, tenant_id, contract_id, workflow_type, status,
			current_step, started_at, created_by
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8)`

	_, err = tx.ExecContext(ctx, query,
		uuid.UUID(workflow.ID).String(),
		workflow.TenantID,
		uuid.UUID(workflow.ContractID).String(),
		workflow.WorkflowType,
		string(workflow.Status),
		workflow.CurrentStep,
		workflow.StartedAt,
		uuid.UUID(workflow.CreatedBy).String(),
	)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}

	// Insert workflow steps
	for _, step := range workflow.Steps {
		stepQuery := `
			INSERT INTO clm_workflow_steps (
				id, workflow_id, step_number, name, step_type, status,
				assignee_id, assignee_type, is_optional, due_date, created_at
			) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11)`

		var assigneeID sql.NullString
		if step.AssigneeID != nil {
			assigneeID = sql.NullString{String: uuid.UUID(*step.AssigneeID).String(), Valid: true}
		}

		_, err = tx.ExecContext(ctx, stepQuery,
			uuid.UUID(step.ID).String(),
			uuid.UUID(workflow.ID).String(),
			step.StepNumber,
			step.Name,
			string(step.Type),
			string(step.Status),
			assigneeID,
			nullableString(step.AssigneeType),
			boolToInt(step.IsOptional),
			step.DueDate,
			step.CreatedAt,
		)
		if err != nil {
			return fp.Failure[domain.WorkflowInstance](err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}

	return fp.Success(workflow)
}

// FindByID retrieves a workflow instance by ID
func (r *WorkflowRepository) FindByID(ctx context.Context, tenantID string, id domain.WorkflowID) fp.Result[domain.WorkflowInstance] {
	query := `
		SELECT id, tenant_id, contract_id, workflow_type, status,
			current_step, started_at, completed_at, created_by
		FROM clm_workflow_instances
		WHERE tenant_id = :1 AND id = :2`

	row := r.db.QueryRowContext(ctx, query, tenantID, uuid.UUID(id).String())
	result := scanWorkflowInstance(row)
	if fp.IsFailure(result) {
		return result
	}

	workflow := fp.GetValue(result)

	// Load steps
	stepsResult := r.findWorkflowSteps(ctx, id)
	if fp.IsFailure(stepsResult) {
		return fp.Failure[domain.WorkflowInstance](fp.GetError(stepsResult))
	}
	workflow.Steps = fp.GetValue(stepsResult)

	return fp.Success(workflow)
}

func (r *WorkflowRepository) findWorkflowSteps(ctx context.Context, workflowID domain.WorkflowID) fp.Result[[]domain.WorkflowStep] {
	query := `
		SELECT id, workflow_id, step_number, name, step_type, status,
			assignee_id, assignee_type, is_optional, action_date,
			action_by, action_comment, due_date, created_at
		FROM clm_workflow_steps
		WHERE workflow_id = :1
		ORDER BY step_number`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(workflowID).String())
	if err != nil {
		return fp.Failure[[]domain.WorkflowStep](err)
	}
	defer rows.Close()

	var steps []domain.WorkflowStep
	for rows.Next() {
		result := scanWorkflowStepFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.WorkflowStep](fp.GetError(result))
		}
		steps = append(steps, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.WorkflowStep](err)
	}

	return fp.Success(steps)
}

// FindByContract retrieves all workflow instances for a contract
func (r *WorkflowRepository) FindByContract(ctx context.Context, tenantID string, contractID domain.ContractID) fp.Result[[]domain.WorkflowInstance] {
	query := `
		SELECT id, tenant_id, contract_id, workflow_type, status,
			current_step, started_at, completed_at, created_by
		FROM clm_workflow_instances
		WHERE tenant_id = :1 AND contract_id = :2
		ORDER BY started_at DESC`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.UUID(contractID).String())
	if err != nil {
		return fp.Failure[[]domain.WorkflowInstance](err)
	}
	defer rows.Close()

	var workflows []domain.WorkflowInstance
	var workflowIDs []domain.WorkflowID
	for rows.Next() {
		result := scanWorkflowInstanceFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.WorkflowInstance](fp.GetError(result))
		}
		workflow := fp.GetValue(result)
		workflows = append(workflows, workflow)
		workflowIDs = append(workflowIDs, workflow.ID)
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.WorkflowInstance](err)
	}

	// Batch load steps for all workflows
	if len(workflowIDs) > 0 {
		stepsMap, err := r.findWorkflowStepsBatch(ctx, workflowIDs)
		if err != nil {
			return fp.Failure[[]domain.WorkflowInstance](err)
		}
		for i := range workflows {
			workflows[i].Steps = stepsMap[workflows[i].ID]
		}
	}

	return fp.Success(workflows)
}

// maxInClauseSize is the maximum number of items in an IN clause to avoid DB limits
const maxInClauseSize = 1000

// findWorkflowStepsBatch loads steps for multiple workflows, chunking if necessary
func (r *WorkflowRepository) findWorkflowStepsBatch(ctx context.Context, workflowIDs []domain.WorkflowID) (map[domain.WorkflowID][]domain.WorkflowStep, error) {
	if len(workflowIDs) == 0 {
		return make(map[domain.WorkflowID][]domain.WorkflowStep), nil
	}

	stepsMap := make(map[domain.WorkflowID][]domain.WorkflowStep)

	// Process in chunks to avoid exceeding DB IN-clause limits
	for i := 0; i < len(workflowIDs); i += maxInClauseSize {
		end := i + maxInClauseSize
		if end > len(workflowIDs) {
			end = len(workflowIDs)
		}
		chunk := workflowIDs[i:end]

		chunkMap, err := r.findWorkflowStepsChunk(ctx, chunk)
		if err != nil {
			return nil, err
		}

		// Merge chunk results into main map
		for wfID, steps := range chunkMap {
			stepsMap[wfID] = append(stepsMap[wfID], steps...)
		}
	}

	return stepsMap, nil
}

// findWorkflowStepsChunk loads steps for a single chunk of workflow IDs
func (r *WorkflowRepository) findWorkflowStepsChunk(ctx context.Context, workflowIDs []domain.WorkflowID) (map[domain.WorkflowID][]domain.WorkflowStep, error) {
	// Build IN clause with placeholders
	placeholders := ""
	args := make([]interface{}, len(workflowIDs))
	for i, id := range workflowIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf(":%d", i+1)
		args[i] = uuid.UUID(id).String()
	}

	query := fmt.Sprintf(`
		SELECT id, workflow_id, step_number, name, step_type, status,
			assignee_id, assignee_type, is_optional, action_date,
			action_by, action_comment, due_date, created_at
		FROM clm_workflow_steps
		WHERE workflow_id IN (%s)
		ORDER BY workflow_id, step_number`, placeholders)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stepsMap := make(map[domain.WorkflowID][]domain.WorkflowStep)
	for rows.Next() {
		result := scanWorkflowStepFromRows(rows)
		if fp.IsFailure(result) {
			return nil, fp.GetError(result)
		}
		step := fp.GetValue(result)
		stepsMap[step.WorkflowID] = append(stepsMap[step.WorkflowID], step)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stepsMap, nil
}

// FindPendingApprovals retrieves pending approvals for a user
func (r *WorkflowRepository) FindPendingApprovals(ctx context.Context, tenantID string, userID domain.UserID) fp.Result[[]domain.WorkflowStep] {
	query := `
		SELECT s.id, s.workflow_id, s.step_number, s.name, s.step_type, s.status,
			s.assignee_id, s.assignee_type, s.is_optional, s.action_date,
			s.action_by, s.action_comment, s.due_date, s.created_at
		FROM clm_workflow_steps s
		JOIN clm_workflow_instances w ON s.workflow_id = w.id
		WHERE w.tenant_id = :1 AND s.assignee_id = :2 AND s.status = 'PENDING'
			AND w.status NOT IN ('CANCELLED', 'COMPLETED')
		ORDER BY s.due_date`

	rows, err := r.db.QueryContext(ctx, query, tenantID, uuid.UUID(userID).String())
	if err != nil {
		return fp.Failure[[]domain.WorkflowStep](err)
	}
	defer rows.Close()

	var steps []domain.WorkflowStep
	for rows.Next() {
		result := scanWorkflowStepFromRows(rows)
		if fp.IsFailure(result) {
			return fp.Failure[[]domain.WorkflowStep](fp.GetError(result))
		}
		steps = append(steps, fp.GetValue(result))
	}

	if err := rows.Err(); err != nil {
		return fp.Failure[[]domain.WorkflowStep](err)
	}

	return fp.Success(steps)
}

// UpdateStep updates a workflow step with tenant isolation
func (r *WorkflowRepository) UpdateStep(ctx context.Context, tenantID string, step domain.WorkflowStep) fp.Result[domain.WorkflowStep] {
	query := `
		UPDATE clm_workflow_steps s SET
			status = :1, action_date = :2, action_by = :3, action_comment = :4
		WHERE s.id = :5
			AND EXISTS (SELECT 1 FROM clm_workflow_instances w WHERE w.id = s.workflow_id AND w.tenant_id = :6)`

	var actionBy sql.NullString
	if step.ActionBy != nil {
		actionBy = sql.NullString{String: uuid.UUID(*step.ActionBy).String(), Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		string(step.Status),
		step.ActionDate,
		actionBy,
		nullableString(step.ActionComment),
		uuid.UUID(step.ID).String(),
		tenantID,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[domain.WorkflowStep](sql.ErrNoRows)
	}

	return fp.Success(step)
}

// UpdateInstance updates a workflow instance
func (r *WorkflowRepository) UpdateInstance(ctx context.Context, workflow domain.WorkflowInstance) fp.Result[domain.WorkflowInstance] {
	query := `
		UPDATE clm_workflow_instances SET
			status = :1, current_step = :2, completed_at = :3
		WHERE id = :4 AND tenant_id = :5`

	result, err := r.db.ExecContext(ctx, query,
		string(workflow.Status),
		workflow.CurrentStep,
		workflow.CompletedAt,
		uuid.UUID(workflow.ID).String(),
		workflow.TenantID,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[domain.WorkflowInstance](sql.ErrNoRows)
	}

	return fp.Success(workflow)
}

// FindStepByID retrieves a workflow step by ID with tenant isolation
func (r *WorkflowRepository) FindStepByID(ctx context.Context, tenantID string, id domain.WorkflowStepID) fp.Result[domain.WorkflowStep] {
	query := `
		SELECT s.id, s.workflow_id, s.step_number, s.name, s.step_type, s.status,
			s.assignee_id, s.assignee_type, s.is_optional, s.action_date,
			s.action_by, s.action_comment, s.due_date, s.created_at
		FROM clm_workflow_steps s
		JOIN clm_workflow_instances w ON s.workflow_id = w.id
		WHERE s.id = :1 AND w.tenant_id = :2`

	row := r.db.QueryRowContext(ctx, query, uuid.UUID(id).String(), tenantID)
	return scanWorkflowStep(row)
}

func scanWorkflowInstance(row *sql.Row) fp.Result[domain.WorkflowInstance] {
	var workflow domain.WorkflowInstance
	var idStr, contractIDStr, createdByStr string
	var completedAt sql.NullTime
	var status string

	err := row.Scan(
		&idStr, &workflow.TenantID, &contractIDStr, &workflow.WorkflowType,
		&status, &workflow.CurrentStep, &workflow.StartedAt,
		&completedAt, &createdByStr,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse workflow id %q: %w", idStr, err))
	}
	workflow.ID = domain.WorkflowID(id)

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse contract_id %q: %w", contractIDStr, err))
	}
	workflow.ContractID = domain.ContractID(contractID)

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	workflow.CreatedBy = domain.UserID(createdBy)

	workflow.Status = domain.WorkflowStatus(status)

	if completedAt.Valid {
		workflow.CompletedAt = &completedAt.Time
	}

	return fp.Success(workflow)
}

func scanWorkflowInstanceFromRows(rows *sql.Rows) fp.Result[domain.WorkflowInstance] {
	var workflow domain.WorkflowInstance
	var idStr, contractIDStr, createdByStr string
	var completedAt sql.NullTime
	var status string

	err := rows.Scan(
		&idStr, &workflow.TenantID, &contractIDStr, &workflow.WorkflowType,
		&status, &workflow.CurrentStep, &workflow.StartedAt,
		&completedAt, &createdByStr,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse workflow id %q: %w", idStr, err))
	}
	workflow.ID = domain.WorkflowID(id)

	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse contract_id %q: %w", contractIDStr, err))
	}
	workflow.ContractID = domain.ContractID(contractID)

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return fp.Failure[domain.WorkflowInstance](fmt.Errorf("parse created_by %q: %w", createdByStr, err))
	}
	workflow.CreatedBy = domain.UserID(createdBy)

	workflow.Status = domain.WorkflowStatus(status)

	if completedAt.Valid {
		workflow.CompletedAt = &completedAt.Time
	}

	return fp.Success(workflow)
}

func scanWorkflowStep(row *sql.Row) fp.Result[domain.WorkflowStep] {
	var step domain.WorkflowStep
	var idStr, workflowIDStr string
	var stepType, status string
	var assigneeID, assigneeType, actionComment sql.NullString
	var actionDate, dueDate sql.NullTime
	var actionByStr sql.NullString
	var isOptional int

	err := row.Scan(
		&idStr, &workflowIDStr, &step.StepNumber, &step.Name,
		&stepType, &status, &assigneeID, &assigneeType,
		&isOptional, &actionDate, &actionByStr, &actionComment,
		&dueDate, &step.CreatedAt,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse step id %q: %w", idStr, err))
	}
	step.ID = domain.WorkflowStepID(id)

	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse workflow_id %q: %w", workflowIDStr, err))
	}
	step.WorkflowID = domain.WorkflowID(workflowID)

	step.Type = domain.StepType(stepType)
	step.Status = domain.StepStatus(status)
	step.IsOptional = isOptional == 1

	if assigneeID.Valid {
		aID, err := uuid.Parse(assigneeID.String)
		if err != nil {
			return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse assignee_id %q: %w", assigneeID.String, err))
		}
		step.AssigneeID = (*domain.UserID)(&aID)
	}
	if assigneeType.Valid {
		step.AssigneeType = assigneeType.String
	}
	if actionDate.Valid {
		step.ActionDate = &actionDate.Time
	}
	if actionByStr.Valid {
		aBy, err := uuid.Parse(actionByStr.String)
		if err != nil {
			return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse action_by %q: %w", actionByStr.String, err))
		}
		step.ActionBy = (*domain.UserID)(&aBy)
	}
	if actionComment.Valid {
		step.ActionComment = actionComment.String
	}
	if dueDate.Valid {
		step.DueDate = &dueDate.Time
	}

	return fp.Success(step)
}

func scanWorkflowStepFromRows(rows *sql.Rows) fp.Result[domain.WorkflowStep] {
	var step domain.WorkflowStep
	var idStr, workflowIDStr string
	var stepType, status string
	var assigneeID, assigneeType, actionComment sql.NullString
	var actionDate, dueDate sql.NullTime
	var actionByStr sql.NullString
	var isOptional int

	err := rows.Scan(
		&idStr, &workflowIDStr, &step.StepNumber, &step.Name,
		&stepType, &status, &assigneeID, &assigneeType,
		&isOptional, &actionDate, &actionByStr, &actionComment,
		&dueDate, &step.CreatedAt,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse step id %q: %w", idStr, err))
	}
	step.ID = domain.WorkflowStepID(id)

	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse workflow_id %q: %w", workflowIDStr, err))
	}
	step.WorkflowID = domain.WorkflowID(workflowID)

	step.Type = domain.StepType(stepType)
	step.Status = domain.StepStatus(status)
	step.IsOptional = isOptional == 1

	if assigneeID.Valid {
		aID, err := uuid.Parse(assigneeID.String)
		if err != nil {
			return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse assignee_id %q: %w", assigneeID.String, err))
		}
		step.AssigneeID = (*domain.UserID)(&aID)
	}
	if assigneeType.Valid {
		step.AssigneeType = assigneeType.String
	}
	if actionDate.Valid {
		step.ActionDate = &actionDate.Time
	}
	if actionByStr.Valid {
		aBy, err := uuid.Parse(actionByStr.String)
		if err != nil {
			return fp.Failure[domain.WorkflowStep](fmt.Errorf("parse action_by %q: %w", actionByStr.String, err))
		}
		step.ActionBy = (*domain.UserID)(&aBy)
	}
	if actionComment.Valid {
		step.ActionComment = actionComment.String
	}
	if dueDate.Valid {
		step.DueDate = &dueDate.Time
	}

	return fp.Success(step)
}

// MarkStepComplete atomically marks a step as complete with tenant isolation
func (r *WorkflowRepository) MarkStepComplete(ctx context.Context, tenantID string, stepID domain.WorkflowStepID, actionBy domain.UserID, comment string) fp.Result[domain.WorkflowStep] {
	now := time.Now()

	// Atomic UPDATE with tenant isolation check
	query := `
		UPDATE clm_workflow_steps s SET
			status = :1, action_date = :2, action_by = :3, action_comment = :4
		WHERE s.id = :5 AND EXISTS (
			SELECT 1 FROM clm_workflow_instances w 
			WHERE w.id = s.workflow_id AND w.tenant_id = :6
		)`

	result, err := r.db.ExecContext(ctx, query,
		string(domain.StepStatusApproved),
		now,
		uuid.UUID(actionBy).String(),
		comment,
		uuid.UUID(stepID).String(),
		tenantID,
	)
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fp.Failure[domain.WorkflowStep](err)
	}
	if rowsAffected == 0 {
		return fp.Failure[domain.WorkflowStep](sql.ErrNoRows)
	}

	// Fetch the updated step to return
	return r.FindStepByID(ctx, tenantID, stepID)
}
