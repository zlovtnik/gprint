package models

import "time"

// PriceUnit represents the unit for pricing
type PriceUnit string

const (
	PriceUnitHour    PriceUnit = "HOUR"
	PriceUnitDay     PriceUnit = "DAY"
	PriceUnitMonth   PriceUnit = "MONTH"
	PriceUnitProject PriceUnit = "PROJECT"
	PriceUnitUnit    PriceUnit = "UNIT"
)

// Service represents a service in the catalog
type Service struct {
	ID                int64     `json:"id"`
	TenantID          string    `json:"tenant_id"`
	ServiceCode       string    `json:"service_code"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	Category          string    `json:"category,omitempty"`
	Subcategory       string    `json:"subcategory,omitempty"`
	UnitPrice         float64   `json:"unit_price"`
	Currency          string    `json:"currency"`
	PriceUnit         PriceUnit `json:"price_unit"`
	ServiceCodeFiscal string    `json:"service_code_fiscal,omitempty"`
	ISSRate           float64   `json:"iss_rate"`
	IRRFRate          float64   `json:"irrf_rate"`
	PISRate           float64   `json:"pis_rate"`
	COFINSRate        float64   `json:"cofins_rate"`
	CSLLRate          float64   `json:"csll_rate"`
	Active            bool      `json:"active"`
	Notes             string    `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         string    `json:"created_by,omitempty"`
	UpdatedBy         string    `json:"updated_by,omitempty"`
}

// CreateServiceRequest represents the request to create a service
type CreateServiceRequest struct {
	ServiceCode       string    `json:"service_code"`
	Name              string    `json:"name"`
	Description       string    `json:"description,omitempty"`
	Category          string    `json:"category,omitempty"`
	Subcategory       string    `json:"subcategory,omitempty"`
	UnitPrice         float64   `json:"unit_price"`
	Currency          string    `json:"currency,omitempty"`
	PriceUnit         PriceUnit `json:"price_unit,omitempty"`
	ServiceCodeFiscal string    `json:"service_code_fiscal,omitempty"`
	ISSRate           *float64  `json:"iss_rate,omitempty"`    // nil=not provided, 0=0% rate
	IRRFRate          *float64  `json:"irrf_rate,omitempty"`   // nil=not provided, 0=0% rate
	PISRate           *float64  `json:"pis_rate,omitempty"`    // nil=not provided, 0=0% rate
	COFINSRate        *float64  `json:"cofins_rate,omitempty"` // nil=not provided, 0=0% rate
	CSLLRate          *float64  `json:"csll_rate,omitempty"`   // nil=not provided, 0=0% rate
	Notes             string    `json:"notes,omitempty"`
}

// UpdateServiceRequest represents the request to update a service
type UpdateServiceRequest struct {
	Name              string    `json:"name,omitempty"`
	Description       string    `json:"description,omitempty"`
	Category          string    `json:"category,omitempty"`
	Subcategory       string    `json:"subcategory,omitempty"`
	UnitPrice         *float64  `json:"unit_price,omitempty"`
	Currency          string    `json:"currency,omitempty"`
	PriceUnit         PriceUnit `json:"price_unit,omitempty"`
	ServiceCodeFiscal string    `json:"service_code_fiscal,omitempty"`
	ISSRate           *float64  `json:"iss_rate,omitempty"`
	IRRFRate          *float64  `json:"irrf_rate,omitempty"`
	PISRate           *float64  `json:"pis_rate,omitempty"`
	COFINSRate        *float64  `json:"cofins_rate,omitempty"`
	CSLLRate          *float64  `json:"csll_rate,omitempty"`
	Active            *bool     `json:"active,omitempty"`
	Notes             string    `json:"notes,omitempty"`
}

// ServiceResponse represents the API response for a service
type ServiceResponse struct {
	ID          int64     `json:"id"`
	ServiceCode string    `json:"service_code"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	UnitPrice   float64   `json:"unit_price"`
	Currency    string    `json:"currency"`
	PriceUnit   PriceUnit `json:"price_unit"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a Service to ServiceResponse
func (s *Service) ToResponse() ServiceResponse {
	return ServiceResponse{
		ID:          s.ID,
		ServiceCode: s.ServiceCode,
		Name:        s.Name,
		Description: s.Description,
		Category:    s.Category,
		UnitPrice:   s.UnitPrice,
		Currency:    s.Currency,
		PriceUnit:   s.PriceUnit,
		Active:      s.Active,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
