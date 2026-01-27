package domain

import (
	"time"
)

// ContractType represents a category of contract
type ContractType struct {
	ID                  ContractTypeID `json:"id"`
	TenantID            string         `json:"tenant_id"`
	Name                string         `json:"name"`
	Code                string         `json:"code"`
	Description         string         `json:"description,omitempty"`
	DefaultDurationDays int            `json:"default_duration_days"`
	RequiresApproval    bool           `json:"requires_approval"`
	ApprovalLevels      int            `json:"approval_levels"`
	IsActive            bool           `json:"is_active"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           *time.Time     `json:"updated_at,omitempty"`
}

// ContractParty represents a party's involvement in a contract
type ContractParty struct {
	PartyID   PartyID    `json:"party_id"`
	Role      PartyRole  `json:"role"`
	IsPrimary bool       `json:"is_primary"`
	SignedAt  *time.Time `json:"signed_at,omitempty"`
	SignedBy  *UserID    `json:"signed_by,omitempty"`
}

// Contract represents a contract (immutable)
type Contract struct {
	ID               ContractID      `json:"id"`
	TenantID         string          `json:"tenant_id"`
	ContractNumber   string          `json:"contract_number"`
	ExternalRef      string          `json:"external_ref,omitempty"`
	Title            string          `json:"title"`
	Description      string          `json:"description,omitempty"`
	ContractType     ContractType    `json:"contract_type"`
	Status           ContractStatus  `json:"status"`
	Parties          []ContractParty `json:"parties"`
	Value            Money           `json:"value"`
	EffectiveDate    *time.Time      `json:"effective_date,omitempty"`
	ExpirationDate   *time.Time      `json:"expiration_date,omitempty"`
	TerminationDate  *time.Time      `json:"termination_date,omitempty"`
	AutoRenew        bool            `json:"auto_renew"`
	RenewalTermDays  int             `json:"renewal_term_days"`
	NoticePeriodDays int             `json:"notice_period_days"`
	Terms            string          `json:"terms,omitempty"`
	Notes            string          `json:"notes,omitempty"`
	Version          int             `json:"version"`
	PreviousVersion  *ContractID     `json:"previous_version,omitempty"`
	IsDeleted        bool            `json:"is_deleted"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        *time.Time      `json:"updated_at,omitempty"`
	CreatedBy        UserID          `json:"created_by"`
	UpdatedBy        *UserID         `json:"updated_by,omitempty"`
}

// WithTitle returns a copy with updated title
func (c Contract) WithTitle(title string) Contract {
	now := time.Now()
	c.Title = title
	c.UpdatedAt = &now
	return c
}

// WithDescription returns a copy with updated description
func (c Contract) WithDescription(description string) Contract {
	now := time.Now()
	c.Description = description
	c.UpdatedAt = &now
	return c
}

// WithValue returns a copy with updated value
func (c Contract) WithValue(value Money) Contract {
	now := time.Now()
	c.Value = value
	c.UpdatedAt = &now
	return c
}

// WithDates returns a copy with updated dates
func (c Contract) WithDates(effective, expiration *time.Time) Contract {
	now := time.Now()
	c.EffectiveDate = effective
	c.ExpirationDate = expiration
	c.UpdatedAt = &now
	return c
}

// WithRenewalTerms returns a copy with updated renewal terms
func (c Contract) WithRenewalTerms(autoRenew bool, termDays, noticeDays int) Contract {
	now := time.Now()
	c.AutoRenew = autoRenew
	c.RenewalTermDays = termDays
	c.NoticePeriodDays = noticeDays
	c.UpdatedAt = &now
	return c
}

// WithTerms returns a copy with updated terms
func (c Contract) WithTerms(terms string) Contract {
	now := time.Now()
	c.Terms = terms
	c.UpdatedAt = &now
	return c
}

// WithNotes returns a copy with updated notes
func (c Contract) WithNotes(notes string) Contract {
	now := time.Now()
	c.Notes = notes
	c.UpdatedAt = &now
	return c
}

// AddParty returns a copy with added party
func (c Contract) AddParty(party ContractParty) Contract {
	now := time.Now()
	// Create a copy of the Parties slice to preserve immutability
	c.Parties = append(make([]ContractParty, 0, len(c.Parties)+1), c.Parties...)
	c.Parties = append(c.Parties, party)
	c.UpdatedAt = &now
	return c
}

// TransitionTo attempts to transition to a new status
func (c Contract) TransitionTo(status ContractStatus, updatedBy UserID, at time.Time) (Contract, error) {
	if !c.Status.CanTransitionTo(status) {
		return c, NewDomainError("invalid status transition from %s to %s", c.Status, status)
	}
	c.Status = status
	c.UpdatedAt = &at
	c.UpdatedBy = &updatedBy
	return c, nil
}

// Submit submits the contract for approval
func (c Contract) Submit(updatedBy UserID) (Contract, error) {
	now := time.Now()
	return c.TransitionTo(ContractStatusPending, updatedBy, now)
}

// Approve approves the contract
func (c Contract) Approve(updatedBy UserID) (Contract, error) {
	now := time.Now()
	return c.TransitionTo(ContractStatusApproved, updatedBy, now)
}

// Reject rejects the contract
func (c Contract) Reject(updatedBy UserID) (Contract, error) {
	now := time.Now()
	return c.TransitionTo(ContractStatusRejected, updatedBy, now)
}

// Execute executes the contract
func (c Contract) Execute(updatedBy UserID) (Contract, error) {
	now := time.Now()
	return c.TransitionTo(ContractStatusExecuted, updatedBy, now)
}

// Activate activates the contract
func (c Contract) Activate(updatedBy UserID) (Contract, error) {
	now := time.Now()
	return c.TransitionTo(ContractStatusActive, updatedBy, now)
}

// Terminate terminates the contract
func (c Contract) Terminate(updatedBy UserID) (Contract, error) {
	now := time.Now()
	contract, err := c.TransitionTo(ContractStatusTerminated, updatedBy, now)
	if err != nil {
		return c, err
	}
	contract.TerminationDate = &now
	return contract, nil
}

// SoftDelete marks the contract as deleted
func (c Contract) SoftDelete(updatedBy UserID) Contract {
	now := time.Now()
	c.IsDeleted = true
	c.UpdatedAt = &now
	c.UpdatedBy = &updatedBy
	return c
}

// IncrementVersion returns a copy with incremented version
func (c Contract) IncrementVersion(previousID ContractID, updatedBy UserID) Contract {
	now := time.Now()
	c.Version++
	c.PreviousVersion = &previousID
	c.UpdatedAt = &now
	c.UpdatedBy = &updatedBy
	return c
}

// NewContract creates a new Contract
func NewContract(
	tenantID string,
	contractNumber string,
	title string,
	contractType ContractType,
	value Money,
	createdBy UserID,
) Contract {
	return Contract{
		ID:             NewContractID(),
		TenantID:       tenantID,
		ContractNumber: contractNumber,
		Title:          title,
		ContractType:   contractType,
		Status:         ContractStatusDraft,
		Parties:        []ContractParty{},
		Value:          value,
		Version:        1,
		IsDeleted:      false,
		CreatedAt:      time.Now(),
		CreatedBy:      createdBy,
	}
}

// IsExpiringSoon checks if the contract is expiring within the given days
func (c Contract) IsExpiringSoon(days int) bool {
	if c.ExpirationDate == nil {
		return false
	}
	threshold := time.Now().AddDate(0, 0, days)
	return c.ExpirationDate.Before(threshold) && c.ExpirationDate.After(time.Now())
}

// DaysUntilExpiration returns the number of days until expiration
func (c Contract) DaysUntilExpiration() int {
	if c.ExpirationDate == nil {
		return -1
	}
	return int(time.Until(*c.ExpirationDate).Hours() / 24)
}
