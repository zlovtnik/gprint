package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ContractType represents the type of contract
type ContractType string

const (
	ContractTypeService   ContractType = "SERVICE"
	ContractTypeRecurring ContractType = "RECURRING"
	ContractTypeProject   ContractType = "PROJECT"
)

// ContractStatus represents the status of a contract
type ContractStatus string

const (
	ContractStatusDraft     ContractStatus = "DRAFT"
	ContractStatusPending   ContractStatus = "PENDING"
	ContractStatusActive    ContractStatus = "ACTIVE"
	ContractStatusSuspended ContractStatus = "SUSPENDED"
	ContractStatusCancelled ContractStatus = "CANCELLED"
	ContractStatusCompleted ContractStatus = "COMPLETED"
)

// BillingCycle represents the billing cycle
type BillingCycle string

const (
	BillingCycleMonthly   BillingCycle = "MONTHLY"
	BillingCycleQuarterly BillingCycle = "QUARTERLY"
	BillingCycleYearly    BillingCycle = "YEARLY"
	BillingCycleOnce      BillingCycle = "ONCE"
)

// Contract represents a service contract
type Contract struct {
	ID              int64           `json:"id"`
	TenantID        string          `json:"tenant_id"`
	ContractNumber  string          `json:"contract_number"`
	ContractType    ContractType    `json:"contract_type"`
	CustomerID      int64           `json:"customer_id"`
	Customer        *Customer       `json:"customer,omitempty"`
	StartDate       time.Time       `json:"start_date"`
	EndDate         *time.Time      `json:"end_date,omitempty"`
	DurationMonths  int             `json:"duration_months,omitempty"`
	AutoRenew       bool            `json:"auto_renew"`
	TotalValue      decimal.Decimal `json:"total_value"`
	PaymentTerms    string          `json:"payment_terms,omitempty"`
	BillingCycle    BillingCycle    `json:"billing_cycle"`
	Status          ContractStatus  `json:"status"`
	SignedAt        *time.Time      `json:"signed_at,omitempty"`
	SignedBy        string          `json:"signed_by,omitempty"`
	DocumentPath    string          `json:"document_path,omitempty"`
	DocumentHash    string          `json:"document_hash,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	TermsConditions string          `json:"terms_conditions,omitempty"`
	Items           []ContractItem  `json:"items,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CreatedBy       string          `json:"created_by,omitempty"`
	UpdatedBy       string          `json:"updated_by,omitempty"`
}

// ContractItemStatus represents the status of a contract item
type ContractItemStatus string

const (
	ContractItemStatusPending    ContractItemStatus = "PENDING"
	ContractItemStatusInProgress ContractItemStatus = "IN_PROGRESS"
	ContractItemStatusCompleted  ContractItemStatus = "COMPLETED"
	ContractItemStatusCancelled  ContractItemStatus = "CANCELLED"
)

// ContractItem represents a line item in a contract
type ContractItem struct {
	ID           int64              `json:"id"`
	TenantID     string             `json:"tenant_id"`
	ContractID   int64              `json:"contract_id"`
	ServiceID    int64              `json:"service_id"`
	Service      *Service           `json:"service,omitempty"`
	Quantity     decimal.Decimal    `json:"quantity"`
	UnitPrice    decimal.Decimal    `json:"unit_price"`
	DiscountPct  decimal.Decimal    `json:"discount_pct"`
	LineTotal    decimal.Decimal    `json:"line_total"`
	StartDate    *time.Time         `json:"start_date,omitempty"`
	EndDate      *time.Time         `json:"end_date,omitempty"`
	DeliveryDate *time.Time         `json:"delivery_date,omitempty"`
	Description  string             `json:"description,omitempty"`
	Status       ContractItemStatus `json:"status"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	Notes        string             `json:"notes,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// CreateContractRequest represents the request to create a contract
type CreateContractRequest struct {
	ContractNumber  string                      `json:"contract_number" validate:"required,max=50"`
	ContractType    ContractType                `json:"contract_type" validate:"required,oneof=SERVICE RECURRING PROJECT"`
	CustomerID      int64                       `json:"customer_id" validate:"required,gt=0"`
	StartDate       time.Time                   `json:"start_date" validate:"required"`
	EndDate         *time.Time                  `json:"end_date,omitempty"`
	DurationMonths  int                         `json:"duration_months,omitempty" validate:"omitempty,gte=0"`
	AutoRenew       bool                        `json:"auto_renew"`
	PaymentTerms    string                      `json:"payment_terms,omitempty"`
	BillingCycle    BillingCycle                `json:"billing_cycle,omitempty" validate:"omitempty,oneof=MONTHLY QUARTERLY YEARLY ONCE"`
	Notes           string                      `json:"notes,omitempty"`
	TermsConditions string                      `json:"terms_conditions,omitempty"`
	Items           []CreateContractItemRequest `json:"items,omitempty" validate:"dive"`
}

// CreateContractItemRequest represents the request to create a contract item
type CreateContractItemRequest struct {
	ServiceID    int64           `json:"service_id" validate:"required,gt=0"`
	Quantity     decimal.Decimal `json:"quantity" validate:"required"`
	UnitPrice    decimal.Decimal `json:"unit_price" validate:"required"`
	DiscountPct  decimal.Decimal `json:"discount_pct,omitempty"`
	StartDate    *time.Time      `json:"start_date,omitempty"`
	EndDate      *time.Time      `json:"end_date,omitempty"`
	DeliveryDate *time.Time      `json:"delivery_date,omitempty"`
	Description  string          `json:"description,omitempty"`
	Notes        string          `json:"notes,omitempty"`
}

// UpdateContractRequest represents the request to update a contract
type UpdateContractRequest struct {
	ContractType    *ContractType `json:"contract_type,omitempty"`
	StartDate       *time.Time    `json:"start_date,omitempty"`
	EndDate         *time.Time    `json:"end_date,omitempty"`
	DurationMonths  *int          `json:"duration_months,omitempty"`
	AutoRenew       *bool         `json:"auto_renew,omitempty"`
	PaymentTerms    string        `json:"payment_terms,omitempty"`
	BillingCycle    *BillingCycle `json:"billing_cycle,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	TermsConditions string        `json:"terms_conditions,omitempty"`
}

// UpdateContractStatusRequest represents the request to update contract status
type UpdateContractStatusRequest struct {
	Status ContractStatus `json:"status"`
}

// SignContractRequest represents the request to sign a contract
type SignContractRequest struct {
	SignedBy string `json:"signed_by"`
}

// ContractResponse represents the API response for a contract
type ContractResponse struct {
	ID             int64                  `json:"id"`
	ContractNumber string                 `json:"contract_number"`
	ContractType   ContractType           `json:"contract_type"`
	CustomerID     int64                  `json:"customer_id"`
	Customer       *CustomerResponse      `json:"customer,omitempty"`
	StartDate      time.Time              `json:"start_date"`
	EndDate        *time.Time             `json:"end_date,omitempty"`
	DurationMonths int                    `json:"duration_months,omitempty"`
	AutoRenew      bool                   `json:"auto_renew"`
	TotalValue     decimal.Decimal        `json:"total_value"`
	BillingCycle   BillingCycle           `json:"billing_cycle"`
	Status         ContractStatus         `json:"status"`
	SignedAt       *time.Time             `json:"signed_at,omitempty"`
	Items          []ContractItemResponse `json:"items,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ContractItemResponse represents the API response for a contract item
type ContractItemResponse struct {
	ID          int64              `json:"id"`
	ServiceID   int64              `json:"service_id"`
	Service     *ServiceResponse   `json:"service,omitempty"`
	Quantity    decimal.Decimal    `json:"quantity"`
	UnitPrice   decimal.Decimal    `json:"unit_price"`
	DiscountPct decimal.Decimal    `json:"discount_pct"`
	LineTotal   decimal.Decimal    `json:"line_total"`
	Status      ContractItemStatus `json:"status"`
	Description string             `json:"description,omitempty"`
}

// ToResponse converts a Contract to ContractResponse
// Returns empty ContractResponse if receiver is nil
func (c *Contract) ToResponse() ContractResponse {
	if c == nil {
		return ContractResponse{}
	}
	resp := ContractResponse{
		ID:             c.ID,
		ContractNumber: c.ContractNumber,
		ContractType:   c.ContractType,
		CustomerID:     c.CustomerID,
		StartDate:      c.StartDate,
		EndDate:        c.EndDate,
		DurationMonths: c.DurationMonths,
		AutoRenew:      c.AutoRenew,
		TotalValue:     c.TotalValue,
		BillingCycle:   c.BillingCycle,
		Status:         c.Status,
		SignedAt:       c.SignedAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}

	if c.Customer != nil {
		custResp := c.Customer.ToResponse()
		resp.Customer = &custResp
	}

	for _, item := range c.Items {
		resp.Items = append(resp.Items, item.ToResponse())
	}

	return resp
}

// ToResponse converts a ContractItem to ContractItemResponse
func (ci *ContractItem) ToResponse() ContractItemResponse {
	resp := ContractItemResponse{
		ID:          ci.ID,
		ServiceID:   ci.ServiceID,
		Quantity:    ci.Quantity,
		UnitPrice:   ci.UnitPrice,
		DiscountPct: ci.DiscountPct,
		LineTotal:   ci.LineTotal,
		Status:      ci.Status,
		Description: ci.Description,
	}

	if ci.Service != nil {
		svcResp := ci.Service.ToResponse()
		resp.Service = &svcResp
	}

	return resp
}
