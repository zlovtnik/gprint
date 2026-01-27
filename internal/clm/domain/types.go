package domain

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ID type aliases for type safety
type PartyID uuid.UUID
type ContractID uuid.UUID
type ContractTypeID uuid.UUID
type ObligationID uuid.UUID
type WorkflowID uuid.UUID
type WorkflowStepID uuid.UUID
type DocumentID uuid.UUID
type TemplateID uuid.UUID
type UserID uuid.UUID

// String methods for ID types
func (id PartyID) String() string        { return uuid.UUID(id).String() }
func (id ContractID) String() string     { return uuid.UUID(id).String() }
func (id ContractTypeID) String() string { return uuid.UUID(id).String() }
func (id ObligationID) String() string   { return uuid.UUID(id).String() }
func (id WorkflowID) String() string     { return uuid.UUID(id).String() }
func (id WorkflowStepID) String() string { return uuid.UUID(id).String() }
func (id DocumentID) String() string     { return uuid.UUID(id).String() }
func (id TemplateID) String() string     { return uuid.UUID(id).String() }
func (id UserID) String() string         { return uuid.UUID(id).String() }

// IsZero methods for ID types
func (id PartyID) IsZero() bool        { return uuid.UUID(id) == uuid.Nil }
func (id ContractID) IsZero() bool     { return uuid.UUID(id) == uuid.Nil }
func (id ContractTypeID) IsZero() bool { return uuid.UUID(id) == uuid.Nil }
func (id ObligationID) IsZero() bool   { return uuid.UUID(id) == uuid.Nil }
func (id WorkflowID) IsZero() bool     { return uuid.UUID(id) == uuid.Nil }
func (id WorkflowStepID) IsZero() bool { return uuid.UUID(id) == uuid.Nil }
func (id DocumentID) IsZero() bool     { return uuid.UUID(id) == uuid.Nil }
func (id TemplateID) IsZero() bool     { return uuid.UUID(id) == uuid.Nil }
func (id UserID) IsZero() bool         { return uuid.UUID(id) == uuid.Nil }

// NewPartyID creates a new PartyID
func NewPartyID() PartyID { return PartyID(uuid.New()) }

// NewContractID creates a new ContractID
func NewContractID() ContractID { return ContractID(uuid.New()) }

// NewContractTypeID creates a new ContractTypeID
func NewContractTypeID() ContractTypeID { return ContractTypeID(uuid.New()) }

// NewObligationID creates a new ObligationID
func NewObligationID() ObligationID { return ObligationID(uuid.New()) }

// NewWorkflowID creates a new WorkflowID
func NewWorkflowID() WorkflowID { return WorkflowID(uuid.New()) }

// NewWorkflowStepID creates a new WorkflowStepID
func NewWorkflowStepID() WorkflowStepID { return WorkflowStepID(uuid.New()) }

// NewDocumentID creates a new DocumentID
func NewDocumentID() DocumentID { return DocumentID(uuid.New()) }

// NewTemplateID creates a new TemplateID
func NewTemplateID() TemplateID { return TemplateID(uuid.New()) }

// NewUserID creates a new UserID
func NewUserID() UserID { return UserID(uuid.New()) }

// Money represents a monetary value with currency
type Money struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// NewMoney creates a new Money value
func NewMoney(amount decimal.Decimal, currency string) Money {
	return Money{Amount: amount, Currency: currency}
}

// Add adds two money values (must be same currency)
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot add money with different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// Subtract subtracts two money values (must be same currency)
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot subtract money with different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Sub(other.Amount), Currency: m.Currency}, nil
}

// Multiply multiplies money by a factor
func (m Money) Multiply(factor decimal.Decimal) Money {
	return Money{Amount: m.Amount.Mul(factor), Currency: m.Currency}
}

// IsPositive checks if the amount is positive
func (m Money) IsPositive() bool {
	return m.Amount.IsPositive()
}

// IsZero checks if the amount is zero
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// Contract Status constants
type ContractStatus string

const (
	ContractStatusDraft      ContractStatus = "DRAFT"
	ContractStatusPending    ContractStatus = "PENDING_APPROVAL"
	ContractStatusApproved   ContractStatus = "APPROVED"
	ContractStatusRejected   ContractStatus = "REJECTED"
	ContractStatusExecuted   ContractStatus = "EXECUTED"
	ContractStatusActive     ContractStatus = "ACTIVE"
	ContractStatusExpired    ContractStatus = "EXPIRED"
	ContractStatusTerminated ContractStatus = "TERMINATED"
	ContractStatusSuspended  ContractStatus = "SUSPENDED"
	ContractStatusRenewing   ContractStatus = "RENEWING"
)

// ValidTransitions defines valid status transitions
var ValidTransitions = map[ContractStatus][]ContractStatus{
	ContractStatusDraft:      {ContractStatusPending},
	ContractStatusPending:    {ContractStatusApproved, ContractStatusRejected, ContractStatusDraft},
	ContractStatusApproved:   {ContractStatusExecuted, ContractStatusDraft},
	ContractStatusRejected:   {ContractStatusDraft},
	ContractStatusExecuted:   {ContractStatusActive},
	ContractStatusActive:     {ContractStatusExpired, ContractStatusTerminated, ContractStatusSuspended, ContractStatusRenewing},
	ContractStatusSuspended:  {ContractStatusActive, ContractStatusTerminated},
	ContractStatusRenewing:   {ContractStatusActive, ContractStatusExpired},
	ContractStatusExpired:    {},
	ContractStatusTerminated: {},
}

// CanTransitionTo checks if a transition is valid
func (s ContractStatus) CanTransitionTo(target ContractStatus) bool {
	valid, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, v := range valid {
		if v == target {
			return true
		}
	}
	return false
}
