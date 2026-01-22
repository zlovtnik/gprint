package service

import (
	"context"

	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/repository"
)

// CustomerService handles customer business logic
type CustomerService struct {
	repo *repository.CustomerRepository
}

// NewCustomerService creates a new CustomerService
func NewCustomerService(repo *repository.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

// Create creates a new customer
func (s *CustomerService) Create(ctx context.Context, tenantID string, req *models.CreateCustomerRequest, createdBy string) (*models.Customer, error) {
	return s.repo.Create(ctx, tenantID, req, createdBy)
}

// GetByID retrieves a customer by ID
func (s *CustomerService) GetByID(ctx context.Context, tenantID string, id int64) (*models.Customer, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retrieves customers with pagination
func (s *CustomerService) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Customer, int, error) {
	return s.repo.List(ctx, tenantID, params, search)
}

// Update updates a customer
func (s *CustomerService) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateCustomerRequest, updatedBy string) (*models.Customer, error) {
	return s.repo.Update(ctx, tenantID, id, req, updatedBy)
}

// Delete soft-deletes a customer
func (s *CustomerService) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	// Check if customer exists first
	customer, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if customer == nil {
		return ErrCustomerNotFound
	}
	return s.repo.Delete(ctx, tenantID, id, deletedBy)
}
