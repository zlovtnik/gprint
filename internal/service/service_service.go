package service

import (
	"context"

	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/repository"
)

// ServiceService handles service business logic
type ServiceService struct {
	repo *repository.ServiceRepository
}

// NewServiceService creates a new ServiceService
func NewServiceService(repo *repository.ServiceRepository) *ServiceService {
	return &ServiceService{repo: repo}
}

// Create creates a new service
func (s *ServiceService) Create(ctx context.Context, tenantID string, req *models.CreateServiceRequest, createdBy string) (*models.Service, error) {
	return s.repo.Create(ctx, tenantID, req, createdBy)
}

// GetByID retrieves a service by ID
func (s *ServiceService) GetByID(ctx context.Context, tenantID string, id int64) (*models.Service, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retrieves services with pagination
func (s *ServiceService) List(ctx context.Context, tenantID string, params models.PaginationParams, search models.SearchParams) ([]models.Service, int, error) {
	return s.repo.List(ctx, tenantID, params, search)
}

// Update updates a service
func (s *ServiceService) Update(ctx context.Context, tenantID string, id int64, req *models.UpdateServiceRequest, updatedBy string) (*models.Service, error) {
	return s.repo.Update(ctx, tenantID, id, req, updatedBy)
}

// Delete soft-deletes a service
func (s *ServiceService) Delete(ctx context.Context, tenantID string, id int64, deletedBy string) error {
	return s.repo.Delete(ctx, tenantID, id, deletedBy)
}

// GetCategories retrieves distinct categories
func (s *ServiceService) GetCategories(ctx context.Context, tenantID string) ([]string, error) {
	return s.repo.GetCategories(ctx, tenantID)
}
