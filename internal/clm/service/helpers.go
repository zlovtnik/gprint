package service

import (
	"github.com/google/uuid"
	"github.com/zlovtnik/gprint/internal/clm/domain"
)

// ParsePartyID parses a string to a PartyID
func ParsePartyID(s string) (domain.PartyID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.PartyID{}, err
	}
	return domain.PartyID(u), nil
}

// ParseContractID parses a string to a ContractID
func ParseContractID(s string) (domain.ContractID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.ContractID{}, err
	}
	return domain.ContractID(u), nil
}

// ParseContractTypeID parses a string to a ContractTypeID
func ParseContractTypeID(s string) (domain.ContractTypeID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.ContractTypeID{}, err
	}
	return domain.ContractTypeID(u), nil
}

// ParseObligationID parses a string to an ObligationID
func ParseObligationID(s string) (domain.ObligationID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.ObligationID{}, err
	}
	return domain.ObligationID(u), nil
}

// ParseWorkflowID parses a string to a WorkflowID
func ParseWorkflowID(s string) (domain.WorkflowID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.WorkflowID{}, err
	}
	return domain.WorkflowID(u), nil
}

// ParseWorkflowStepID parses a string to a WorkflowStepID
func ParseWorkflowStepID(s string) (domain.WorkflowStepID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.WorkflowStepID{}, err
	}
	return domain.WorkflowStepID(u), nil
}

// ParseDocumentID parses a string to a DocumentID
func ParseDocumentID(s string) (domain.DocumentID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.DocumentID{}, err
	}
	return domain.DocumentID(u), nil
}

// ParseTemplateID parses a string to a TemplateID
func ParseTemplateID(s string) (domain.TemplateID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.TemplateID{}, err
	}
	return domain.TemplateID(u), nil
}

// ParseUserID parses a string to a UserID
func ParseUserID(s string) (domain.UserID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return domain.UserID{}, err
	}
	return domain.UserID(u), nil
}
