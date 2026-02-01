package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ErrEmptyResponse is returned when the API returns an empty data field
var ErrEmptyResponse = errors.New("empty response data from API")

// API path constants
const (
	paginationQueryFmt  = "%s?page=%d&limit=%d"
	apiErrorFmt         = "API error: %s"
	loginPath           = "/api/v1/auth/login"
	customersPath       = "/api/v1/customers"
	customerByIDPathFmt = "/api/v1/customers/%d"
	servicesPath        = "/api/v1/services"
	serviceByIDPathFmt  = "/api/v1/services/%d"
	contractsPath       = "/api/v1/contracts"
	contractByIDPathFmt = "/api/v1/contracts/%d"
	printJobsPath       = "/api/v1/print-jobs"
	defaultPageLimit    = 20
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response from the API
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         string `json:"user"`
	TenantID     string `json:"tenant_id,omitempty"`
}

// Login authenticates with the API and returns tokens
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	req := LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.doRequestWithContext(ctx, "POST", loginPath, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}

	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(resp.Data, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	// Auto-set the token for subsequent requests
	c.SetToken(loginResp.AccessToken)

	return &loginResp, nil
}

// ListOptions provides pagination options for list operations
type ListOptions struct {
	Page  int
	Limit int
}

// WithDefaults returns a copy of ListOptions with safe defaults applied
// Page defaults to 1 if <= 0, Limit defaults to 20 if <= 0
func (o *ListOptions) WithDefaults() ListOptions {
	if o == nil {
		return ListOptions{Page: 1, Limit: defaultPageLimit}
	}
	result := *o
	if result.Page <= 0 {
		result.Page = 1
	}
	if result.Limit <= 0 {
		result.Limit = defaultPageLimit
	}
	return result
}

// ListResult wraps a list of items with pagination metadata
type ListResult[T any] struct {
	Items      []T
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// Customer represents a customer
type Customer struct {
	ID           int64     `json:"id"`
	CustomerCode string    `json:"customer_code"`
	CustomerType string    `json:"customer_type"`
	Name         string    `json:"name"`
	TradeName    string    `json:"trade_name,omitempty"`
	TaxID        string    `json:"tax_id,omitempty"`
	Email        string    `json:"email,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

// Service represents a service
type Service struct {
	ID          int64           `json:"id"`
	ServiceCode string          `json:"service_code"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Currency    string          `json:"currency"`
	PriceUnit   string          `json:"price_unit"`
	Active      bool            `json:"active"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Contract represents a contract
type Contract struct {
	ID             int64           `json:"id"`
	ContractNumber string          `json:"contract_number"`
	ContractType   string          `json:"contract_type"`
	CustomerID     int64           `json:"customer_id"`
	StartDate      time.Time       `json:"start_date"`
	EndDate        *time.Time      `json:"end_date,omitempty"`
	TotalValue     decimal.Decimal `json:"total_value"`
	BillingCycle   string          `json:"billing_cycle"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

// PrintJob represents a print job
type PrintJob struct {
	ID          int64      `json:"id"`
	ContractID  int64      `json:"contract_id"`
	Status      string     `json:"status"`
	Format      string     `json:"format"`
	FileSize    int64      `json:"file_size,omitempty"`
	PageCount   int        `json:"page_count,omitempty"`
	QueuedAt    time.Time  `json:"queued_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	RequestedBy string     `json:"requested_by"`
}

// CreateCustomerRequest is the request payload for creating a customer
type CreateCustomerRequest struct {
	CustomerCode string            `json:"customer_code"`
	CustomerType string            `json:"customer_type"`
	Name         string            `json:"name"`
	TradeName    string            `json:"trade_name,omitempty"`
	TaxID        string            `json:"tax_id,omitempty"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Address      map[string]string `json:"address,omitempty"`
}

// UpdateCustomerRequest is the request payload for updating a customer
type UpdateCustomerRequest struct {
	CustomerCode string            `json:"customer_code,omitempty"`
	CustomerType string            `json:"customer_type,omitempty"`
	Name         string            `json:"name,omitempty"`
	TradeName    string            `json:"trade_name,omitempty"`
	TaxID        string            `json:"tax_id,omitempty"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	Address      map[string]string `json:"address,omitempty"`
}

// CreateServiceRequest is the request payload for creating a service
type CreateServiceRequest struct {
	ServiceCode string          `json:"service_code"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    string          `json:"category,omitempty"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	PriceUnit   string          `json:"price_unit"`
	Currency    string          `json:"currency"`
}

// UpdateServiceRequest is the request payload for updating a service
type UpdateServiceRequest struct {
	ServiceCode string           `json:"service_code,omitempty"`
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Category    string           `json:"category,omitempty"`
	UnitPrice   *decimal.Decimal `json:"unit_price,omitempty"`
	PriceUnit   string           `json:"price_unit,omitempty"`
	Currency    string           `json:"currency,omitempty"`
}

// CreateContractRequest is the request payload for creating a contract
type CreateContractRequest struct {
	ContractNumber string `json:"contract_number"`
	CustomerID     int64  `json:"customer_id"`
	ContractType   string `json:"contract_type"`
	BillingCycle   string `json:"billing_cycle"`
	TotalValue     string `json:"total_value"`
}

// UpdateContractRequest is the request payload for updating a contract
type UpdateContractRequest struct {
	ContractNumber string `json:"contract_number,omitempty"`
	CustomerID     *int64 `json:"customer_id,omitempty"`
	ContractType   string `json:"contract_type,omitempty"`
	BillingCycle   string `json:"billing_cycle,omitempty"`
	TotalValue     string `json:"total_value,omitempty"`
}

// listItems is a generic helper for fetching paginated lists
func listItems[T any](c *Client, basePath string, opts *ListOptions) (*ListResult[T], error) {
	return listItemsWithContext[T](context.Background(), c, basePath, opts)
}

// listItemsWithContext is a generic helper for fetching paginated lists with context support
func listItemsWithContext[T any](ctx context.Context, c *Client, basePath string, opts *ListOptions) (*ListResult[T], error) {
	normalized := opts.WithDefaults()
	path := fmt.Sprintf(paginationQueryFmt, basePath, normalized.Page, normalized.Limit)

	resp, err := c.GetWithContext(ctx, path)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var paginated PaginatedResponse
	if err := json.Unmarshal(resp.Data, &paginated); err != nil {
		return nil, fmt.Errorf("failed to parse paginated response: %w", err)
	}

	var items []T
	if err := json.Unmarshal(paginated.Data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}
	return &ListResult[T]{
		Items:      items,
		Total:      paginated.TotalCount,
		Page:       paginated.Page,
		PageSize:   paginated.PageSize,
		TotalPages: paginated.TotalPages,
	}, nil
}

// ListCustomers fetches customers with pagination support
func (c *Client) ListCustomers(opts *ListOptions) (*ListResult[Customer], error) {
	return listItems[Customer](c, customersPath, opts)
}

// ListCustomersWithContext fetches customers with context and pagination support
func (c *Client) ListCustomersWithContext(ctx context.Context, opts *ListOptions) (*ListResult[Customer], error) {
	return listItemsWithContext[Customer](ctx, c, customersPath, opts)
}

// GetCustomer fetches a customer by ID
func (c *Client) GetCustomer(id int64) (*Customer, error) {
	resp, err := c.Get(fmt.Sprintf(customerByIDPathFmt, id))
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var customer Customer
	if err := json.Unmarshal(resp.Data, &customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

// CreateCustomer creates a new customer
func (c *Client) CreateCustomer(req *CreateCustomerRequest) (*Customer, error) {
	return c.CreateCustomerWithContext(context.Background(), req)
}

// CreateCustomerWithContext creates a new customer with context support
func (c *Client) CreateCustomerWithContext(ctx context.Context, req *CreateCustomerRequest) (*Customer, error) {
	resp, err := c.doRequestWithContext(ctx, "POST", customersPath, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var customer Customer
	if err := json.Unmarshal(resp.Data, &customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

// UpdateCustomer updates a customer
func (c *Client) UpdateCustomer(id int64, req *UpdateCustomerRequest) (*Customer, error) {
	return c.UpdateCustomerWithContext(context.Background(), id, req)
}

// UpdateCustomerWithContext updates a customer with context support
func (c *Client) UpdateCustomerWithContext(ctx context.Context, id int64, req *UpdateCustomerRequest) (*Customer, error) {
	resp, err := c.doRequestWithContext(ctx, "PUT", fmt.Sprintf(customerByIDPathFmt, id), req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var customer Customer
	if err := json.Unmarshal(resp.Data, &customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

// DeleteCustomer deletes a customer
func (c *Client) DeleteCustomer(id int64) error {
	return c.DeleteCustomerWithContext(context.Background(), id)
}

// DeleteCustomerWithContext deletes a customer with context support
func (c *Client) DeleteCustomerWithContext(ctx context.Context, id int64) error {
	resp, err := c.doRequestWithContext(ctx, "DELETE", fmt.Sprintf(customerByIDPathFmt, id), nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// ListServices fetches services with pagination support
func (c *Client) ListServices(opts *ListOptions) (*ListResult[Service], error) {
	return listItems[Service](c, servicesPath, opts)
}

// ListServicesWithContext fetches services with context and pagination support
func (c *Client) ListServicesWithContext(ctx context.Context, opts *ListOptions) (*ListResult[Service], error) {
	return listItemsWithContext[Service](ctx, c, servicesPath, opts)
}

// GetService fetches a service by ID
func (c *Client) GetService(id int64) (*Service, error) {
	resp, err := c.Get(fmt.Sprintf(serviceByIDPathFmt, id))
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

// CreateService creates a new service
func (c *Client) CreateService(req *CreateServiceRequest) (*Service, error) {
	return c.CreateServiceWithContext(context.Background(), req)
}

// CreateServiceWithContext creates a new service with context support
func (c *Client) CreateServiceWithContext(ctx context.Context, req *CreateServiceRequest) (*Service, error) {
	resp, err := c.doRequestWithContext(ctx, "POST", servicesPath, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

// UpdateService updates a service
func (c *Client) UpdateService(id int64, req *UpdateServiceRequest) (*Service, error) {
	return c.UpdateServiceWithContext(context.Background(), id, req)
}

// UpdateServiceWithContext updates a service with context support
func (c *Client) UpdateServiceWithContext(ctx context.Context, id int64, req *UpdateServiceRequest) (*Service, error) {
	resp, err := c.doRequestWithContext(ctx, "PUT", fmt.Sprintf(serviceByIDPathFmt, id), req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

// DeleteService deletes a service
func (c *Client) DeleteService(id int64) error {
	return c.DeleteServiceWithContext(context.Background(), id)
}

// DeleteServiceWithContext deletes a service with context support
func (c *Client) DeleteServiceWithContext(ctx context.Context, id int64) error {
	resp, err := c.doRequestWithContext(ctx, "DELETE", fmt.Sprintf(serviceByIDPathFmt, id), nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// ListContracts fetches contracts with pagination support
func (c *Client) ListContracts(opts *ListOptions) (*ListResult[Contract], error) {
	return listItems[Contract](c, contractsPath, opts)
}

// ListContractsWithContext fetches contracts with context and pagination support
func (c *Client) ListContractsWithContext(ctx context.Context, opts *ListOptions) (*ListResult[Contract], error) {
	return listItemsWithContext[Contract](ctx, c, contractsPath, opts)
}

// GetContract fetches a contract by ID
func (c *Client) GetContract(id int64) (*Contract, error) {
	resp, err := c.Get(fmt.Sprintf(contractByIDPathFmt, id))
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var contract Contract
	if err := json.Unmarshal(resp.Data, &contract); err != nil {
		return nil, err
	}
	return &contract, nil
}

// CreateContract creates a new contract
func (c *Client) CreateContract(req *CreateContractRequest) (*Contract, error) {
	return c.CreateContractWithContext(context.Background(), req)
}

// CreateContractWithContext creates a new contract with context support
func (c *Client) CreateContractWithContext(ctx context.Context, req *CreateContractRequest) (*Contract, error) {
	resp, err := c.doRequestWithContext(ctx, "POST", contractsPath, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var contract Contract
	if err := json.Unmarshal(resp.Data, &contract); err != nil {
		return nil, err
	}
	return &contract, nil
}

// UpdateContract updates a contract
func (c *Client) UpdateContract(id int64, req *UpdateContractRequest) (*Contract, error) {
	return c.UpdateContractWithContext(context.Background(), id, req)
}

// UpdateContractWithContext updates a contract with context support
func (c *Client) UpdateContractWithContext(ctx context.Context, id int64, req *UpdateContractRequest) (*Contract, error) {
	resp, err := c.doRequestWithContext(ctx, "PUT", fmt.Sprintf(contractByIDPathFmt, id), req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var contract Contract
	if err := json.Unmarshal(resp.Data, &contract); err != nil {
		return nil, err
	}
	return &contract, nil
}

// UpdateContractStatus updates a contract's status
func (c *Client) UpdateContractStatus(id int64, status string) error {
	resp, err := c.Patch(fmt.Sprintf(contractByIDPathFmt+"/status", id), map[string]string{"status": status})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// ListPrintJobs fetches print jobs with pagination support
func (c *Client) ListPrintJobs(opts *ListOptions) (*ListResult[PrintJob], error) {
	return listItems[PrintJob](c, printJobsPath, opts)
}

// ListPrintJobsWithContext fetches print jobs with context and pagination support
func (c *Client) ListPrintJobsWithContext(ctx context.Context, opts *ListOptions) (*ListResult[PrintJob], error) {
	return listItemsWithContext[PrintJob](ctx, c, printJobsPath, opts)
}

// CreatePrintJob creates a print job for a contract
func (c *Client) CreatePrintJob(contractID int64, format string) (*PrintJob, error) {
	return c.CreatePrintJobWithContext(context.Background(), contractID, format)
}

// CreatePrintJobWithContext creates a print job for a contract with context support
func (c *Client) CreatePrintJobWithContext(ctx context.Context, contractID int64, format string) (*PrintJob, error) {
	resp, err := c.doRequestWithContext(ctx, "POST", fmt.Sprintf(contractByIDPathFmt+"/print", contractID), map[string]string{"format": format})
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var job PrintJob
	if err := json.Unmarshal(resp.Data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// GenerateContract triggers contract generation
func (c *Client) GenerateContract(contractID int64) error {
	return c.GenerateContractWithContext(context.Background(), contractID)
}

// GenerateContractWithContext triggers contract generation with context support
func (c *Client) GenerateContractWithContext(ctx context.Context, contractID int64) error {
	resp, err := c.doRequestWithContext(ctx, "POST", fmt.Sprintf(contractByIDPathFmt+"/generate", contractID), nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// SignContract signs a contract
func (c *Client) SignContract(id int64, signedBy string) error {
	return c.SignContractWithContext(context.Background(), id, signedBy)
}

// SignContractWithContext signs a contract with context support
func (c *Client) SignContractWithContext(ctx context.Context, id int64, signedBy string) error {
	resp, err := c.doRequestWithContext(ctx, "POST", fmt.Sprintf(contractByIDPathFmt+"/sign", id), map[string]string{"signed_by": signedBy})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}
