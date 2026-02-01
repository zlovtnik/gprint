package models

import "time"

// CustomerType represents the type of customer
type CustomerType string

const (
	CustomerTypeIndividual CustomerType = "INDIVIDUAL"
	CustomerTypeCompany    CustomerType = "COMPANY"
)

// Customer represents a customer entity
type Customer struct {
	ID           int64        `json:"id"`
	TenantID     string       `json:"tenant_id"`
	CustomerCode string       `json:"customer_code"`
	CustomerType CustomerType `json:"customer_type"`
	Name         string       `json:"name"`
	TradeName    string       `json:"trade_name,omitempty"`
	TaxID        string       `json:"tax_id,omitempty"`
	StateReg     string       `json:"state_reg,omitempty"`
	MunicipalReg string       `json:"municipal_reg,omitempty"`
	Email        string       `json:"email,omitempty"`
	Phone        string       `json:"phone,omitempty"`
	Mobile       string       `json:"mobile,omitempty"`
	Address      *Address     `json:"address,omitempty"`
	Active       bool         `json:"active"`
	Notes        string       `json:"notes,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	CreatedBy    string       `json:"created_by,omitempty"`
	UpdatedBy    string       `json:"updated_by,omitempty"`
}

// Address represents a physical address
type Address struct {
	Street   string `json:"street,omitempty"`
	Number   string `json:"number,omitempty"`
	Comp     string `json:"comp,omitempty"`
	District string `json:"district,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	Zip      string `json:"zip,omitempty"`
	Country  string `json:"country,omitempty"`
}

// AddressInput represents a request payload for address updates.
// Pointer fields allow distinguishing "not provided" from "set to empty".
type AddressInput struct {
	Street   *string `json:"street,omitempty"`
	Number   *string `json:"number,omitempty"`
	Comp     *string `json:"comp,omitempty"`
	District *string `json:"district,omitempty"`
	City     *string `json:"city,omitempty"`
	State    *string `json:"state,omitempty"`
	Zip      *string `json:"zip,omitempty"`
	Country  *string `json:"country,omitempty"`
}

// CreateCustomerRequest represents the request to create a customer
type CreateCustomerRequest struct {
	CustomerCode string        `json:"customer_code"`
	CustomerType CustomerType  `json:"customer_type"`
	Name         string        `json:"name"`
	TradeName    *string       `json:"trade_name,omitempty"`
	TaxID        *string       `json:"tax_id,omitempty"`
	StateReg     *string       `json:"state_reg,omitempty"`
	MunicipalReg *string       `json:"municipal_reg,omitempty"`
	Email        *string       `json:"email,omitempty"`
	Phone        *string       `json:"phone,omitempty"`
	Mobile       *string       `json:"mobile,omitempty"`
	Address      *AddressInput `json:"address,omitempty"` // nil = no address
	Notes        *string       `json:"notes,omitempty"`
}

// UpdateCustomerRequest represents the request to update a customer
type UpdateCustomerRequest struct {
	CustomerType *CustomerType `json:"customer_type,omitempty"`
	Name         *string       `json:"name,omitempty"`
	TradeName    *string       `json:"trade_name,omitempty"`
	TaxID        *string       `json:"tax_id,omitempty"`
	StateReg     *string       `json:"state_reg,omitempty"`
	MunicipalReg *string       `json:"municipal_reg,omitempty"`
	Email        *string       `json:"email,omitempty"`
	Phone        *string       `json:"phone,omitempty"`
	Mobile       *string       `json:"mobile,omitempty"`
	Address      *AddressInput `json:"address,omitempty"` // nil = no change to address
	Active       *bool         `json:"active,omitempty"`
	Notes        *string       `json:"notes,omitempty"`
}

// CustomerResponse represents the API response for a customer
type CustomerResponse struct {
	ID           int64        `json:"id"`
	CustomerCode string       `json:"customer_code"`
	CustomerType CustomerType `json:"customer_type"`
	Name         string       `json:"name"`
	TradeName    string       `json:"trade_name,omitempty"`
	TaxID        string       `json:"tax_id,omitempty"`
	Email        string       `json:"email,omitempty"`
	Phone        string       `json:"phone,omitempty"`
	Mobile       string       `json:"mobile,omitempty"`
	Address      Address      `json:"address"`
	Active       bool         `json:"active"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ToResponse converts a Customer to CustomerResponse
func (c *Customer) ToResponse() CustomerResponse {
	if c == nil {
		return CustomerResponse{}
	}
	resp := CustomerResponse{
		ID:           c.ID,
		CustomerCode: c.CustomerCode,
		CustomerType: c.CustomerType,
		Name:         c.Name,
		TradeName:    c.TradeName,
		TaxID:        c.TaxID,
		Email:        c.Email,
		Phone:        c.Phone,
		Mobile:       c.Mobile,
		Active:       c.Active,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
	if c.Address != nil {
		resp.Address = *c.Address
	}
	return resp
}
