package router

import (
	"net/http"

	clmhandlers "github.com/zlovtnik/gprint/internal/clm/handlers"
)

// CLMRouter handles routing for CLM endpoints
type CLMRouter struct {
	mux *http.ServeMux

	partyHandler      *clmhandlers.PartyHandler
	contractHandler   *clmhandlers.ContractHandler
	obligationHandler *clmhandlers.ObligationHandler
	workflowHandler   *clmhandlers.WorkflowHandler
	documentHandler   *clmhandlers.DocumentHandler
	auditHandler      *clmhandlers.AuditHandler
}

// CLMHandlerSet contains all CLM handlers
type CLMHandlerSet struct {
	PartyHandler      *clmhandlers.PartyHandler
	ContractHandler   *clmhandlers.ContractHandler
	ObligationHandler *clmhandlers.ObligationHandler
	WorkflowHandler   *clmhandlers.WorkflowHandler
	DocumentHandler   *clmhandlers.DocumentHandler
	AuditHandler      *clmhandlers.AuditHandler
}

// NewCLMRouter creates a new CLMRouter with the provided handlers.
// Panics if mux is nil.
func NewCLMRouter(mux *http.ServeMux, handlers CLMHandlerSet) *CLMRouter {
	if mux == nil {
		panic("nil mux passed to NewCLMRouter")
	}
	return &CLMRouter{
		mux:               mux,
		partyHandler:      handlers.PartyHandler,
		contractHandler:   handlers.ContractHandler,
		obligationHandler: handlers.ObligationHandler,
		workflowHandler:   handlers.WorkflowHandler,
		documentHandler:   handlers.DocumentHandler,
		auditHandler:      handlers.AuditHandler,
	}
}

// RegisterRoutes registers all CLM routes with the mux
// Note: Auth middleware is applied globally by the main router
func (r *CLMRouter) RegisterRoutes() {
	// Party routes
	if r.partyHandler != nil {
		r.mux.HandleFunc("POST /api/v1/clm/parties", r.partyHandler.Create)
		r.mux.HandleFunc("GET /api/v1/clm/parties", r.partyHandler.List)
		r.mux.HandleFunc("GET /api/v1/clm/parties/{id}", r.partyHandler.Get)
		r.mux.HandleFunc("PUT /api/v1/clm/parties/{id}", r.partyHandler.Update)
		r.mux.HandleFunc("DELETE /api/v1/clm/parties/{id}", r.partyHandler.Delete)
	}

	// Contract routes
	if r.contractHandler != nil {
		r.mux.HandleFunc("POST /api/v1/clm/contracts", r.contractHandler.Create)
		r.mux.HandleFunc("GET /api/v1/clm/contracts", r.contractHandler.List)
		r.mux.HandleFunc("GET /api/v1/clm/contracts/expiring", r.contractHandler.FindExpiring)
		r.mux.HandleFunc("GET /api/v1/clm/contracts/{id}", r.contractHandler.Get)
		r.mux.HandleFunc("PUT /api/v1/clm/contracts/{id}", r.contractHandler.Update)
		r.mux.HandleFunc("DELETE /api/v1/clm/contracts/{id}", r.contractHandler.Delete)

		// Contract lifecycle actions
		r.mux.HandleFunc("POST /api/v1/clm/contracts/{id}/submit", r.contractHandler.Submit)
		r.mux.HandleFunc("POST /api/v1/clm/contracts/{id}/approve", r.contractHandler.Approve)
	}

	// Obligation routes
	if r.obligationHandler != nil {
		r.mux.HandleFunc("POST /api/v1/clm/obligations", r.obligationHandler.Create)
		r.mux.HandleFunc("GET /api/v1/clm/obligations/overdue", r.obligationHandler.FindOverdue)
		r.mux.HandleFunc("GET /api/v1/clm/obligations/{id}", r.obligationHandler.Get)
		r.mux.HandleFunc("PUT /api/v1/clm/obligations/{id}", r.obligationHandler.Update)
		r.mux.HandleFunc("POST /api/v1/clm/obligations/{id}/in-progress", r.obligationHandler.MarkInProgress)
		r.mux.HandleFunc("POST /api/v1/clm/obligations/{id}/complete", r.obligationHandler.Complete)
		r.mux.HandleFunc("POST /api/v1/clm/obligations/{id}/waive", r.obligationHandler.Waive)
	}

	// Nested routes under contracts
	if r.obligationHandler != nil {
		r.mux.HandleFunc("GET /api/v1/clm/contracts/{contract_id}/obligations", r.obligationHandler.ListByContract)
	}
	if r.documentHandler != nil {
		r.mux.HandleFunc("GET /api/v1/clm/contracts/{contract_id}/documents", r.documentHandler.ListByContract)
	}
	if r.workflowHandler != nil {
		r.mux.HandleFunc("GET /api/v1/clm/contracts/{contract_id}/workflows", r.workflowHandler.ListByContract)
	}

	// Workflow routes
	if r.workflowHandler != nil {
		r.mux.HandleFunc("GET /api/v1/clm/workflows/pending-approvals", r.workflowHandler.GetPendingApprovals)
		r.mux.HandleFunc("GET /api/v1/clm/workflows/{id}", r.workflowHandler.Get)
		r.mux.HandleFunc("POST /api/v1/clm/workflows/{id}/cancel", r.workflowHandler.CancelWorkflow)
		r.mux.HandleFunc("POST /api/v1/clm/workflow-steps/{id}/approve", r.workflowHandler.ApproveStep)
		r.mux.HandleFunc("POST /api/v1/clm/workflow-steps/{id}/reject", r.workflowHandler.RejectStep)
		r.mux.HandleFunc("POST /api/v1/clm/workflow-steps/{id}/skip", r.workflowHandler.SkipStep)
	}

	// Document routes
	if r.documentHandler != nil {
		r.mux.HandleFunc("POST /api/v1/clm/documents", r.documentHandler.Upload)
		r.mux.HandleFunc("GET /api/v1/clm/documents/{id}", r.documentHandler.Get)
		r.mux.HandleFunc("POST /api/v1/clm/documents/{id}/sign", r.documentHandler.Sign)
		r.mux.HandleFunc("DELETE /api/v1/clm/documents/{id}", r.documentHandler.Delete)

		// Template routes
		r.mux.HandleFunc("GET /api/v1/clm/templates", r.documentHandler.ListTemplates)
		r.mux.HandleFunc("GET /api/v1/clm/templates/{id}", r.documentHandler.GetTemplate)
	}

	// Audit routes
	if r.auditHandler != nil {
		r.mux.HandleFunc("GET /api/v1/clm/audit/{entity_type}/{entity_id}", r.auditHandler.ListByEntity)
		r.mux.HandleFunc("GET /api/v1/clm/audit/user/{user_id}", r.auditHandler.ListByUser)
	}
}
