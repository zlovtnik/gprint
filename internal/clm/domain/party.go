package domain

import (
	"time"
)

// PartyType represents the type of party
type PartyType string

const (
	PartyTypeIndividual  PartyType = "INDIVIDUAL"
	PartyTypeCorporation PartyType = "CORPORATION"
	PartyTypeGovernment  PartyType = "GOVERNMENT"
	PartyTypeNonProfit   PartyType = "NON_PROFIT"
	PartyTypePartnership PartyType = "PARTNERSHIP"
)

// PartyRole represents the role a party plays in a contract
type PartyRole string

const (
	PartyRoleOwner        PartyRole = "OWNER"
	PartyRoleCounterparty PartyRole = "COUNTERPARTY"
	PartyRoleGuarantor    PartyRole = "GUARANTOR"
	PartyRoleWitness      PartyRole = "WITNESS"
	PartyRoleBeneficiary  PartyRole = "BENEFICIARY"
)

// RiskLevel represents the risk assessment level
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// Address represents a postal address
type Address struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// Party represents a party in a contract (immutable)
type Party struct {
	ID             PartyID    `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Type           PartyType  `json:"type"`
	Name           string     `json:"name"`
	LegalName      string     `json:"legal_name"`
	TaxID          string     `json:"tax_id,omitempty"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone,omitempty"`
	Address        Address    `json:"address"`
	BillingAddress *Address   `json:"billing_address,omitempty"`
	RiskLevel      RiskLevel  `json:"risk_level"`
	RiskScore      int        `json:"risk_score"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	CreatedBy      UserID     `json:"created_by"`
	UpdatedBy      *UserID    `json:"updated_by,omitempty"`
}

// WithName returns a copy with updated name
func (p Party) WithName(name string) Party {
	now := time.Now()
	p.Name = name
	p.UpdatedAt = &now
	return p
}

// WithLegalName returns a copy with updated legal name
func (p Party) WithLegalName(legalName string) Party {
	now := time.Now()
	p.LegalName = legalName
	p.UpdatedAt = &now
	return p
}

// WithEmail returns a copy with updated email
func (p Party) WithEmail(email string) Party {
	now := time.Now()
	p.Email = email
	p.UpdatedAt = &now
	return p
}

// WithPhone returns a copy with updated phone
func (p Party) WithPhone(phone string) Party {
	now := time.Now()
	p.Phone = phone
	p.UpdatedAt = &now
	return p
}

// WithAddress returns a copy with updated address
func (p Party) WithAddress(address Address) Party {
	now := time.Now()
	p.Address = address
	p.UpdatedAt = &now
	return p
}

// WithBillingAddress returns a copy with updated billing address
func (p Party) WithBillingAddress(address *Address) Party {
	now := time.Now()
	p.BillingAddress = address
	p.UpdatedAt = &now
	return p
}

// WithRiskAssessment returns a copy with updated risk assessment
func (p Party) WithRiskAssessment(level RiskLevel, score int) Party {
	now := time.Now()
	p.RiskLevel = level
	p.RiskScore = score
	p.UpdatedAt = &now
	return p
}

// Deactivate returns a copy with IsActive set to false
func (p Party) Deactivate(updatedBy UserID) Party {
	now := time.Now()
	p.IsActive = false
	p.UpdatedAt = &now
	p.UpdatedBy = &updatedBy
	return p
}

// Activate returns a copy with IsActive set to true
func (p Party) Activate(updatedBy UserID) Party {
	now := time.Now()
	p.IsActive = true
	p.UpdatedAt = &now
	p.UpdatedBy = &updatedBy
	return p
}

// NewParty creates a new Party
func NewParty(tenantID string, partyType PartyType, name, legalName, email string, address Address, createdBy UserID) Party {
	return Party{
		ID:        NewPartyID(),
		TenantID:  tenantID,
		Type:      partyType,
		Name:      name,
		LegalName: legalName,
		Email:     email,
		Address:   address,
		RiskLevel: RiskLevelLow,
		RiskScore: 0,
		IsActive:  true,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}
}
