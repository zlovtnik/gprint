package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/service"
)

// ServiceHandler handles service HTTP requests
type ServiceHandler struct {
	svc *service.ServiceService
}

// NewServiceHandler creates a new ServiceHandler
func NewServiceHandler(svc *service.ServiceService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

// List handles GET /api/v1/services
func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	params := parsePagination(r)
	search := parseSearchParams(r)

	services, total, err := h.svc.List(r.Context(), tenantID, params, search)
	if err != nil {
		log.Printf("failed to list services (tenant=%s): %v", tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list services")
		return
	}

	responses := make([]models.ServiceResponse, len(services))
	for i, s := range services {
		responses[i] = s.ToResponse()
	}

	result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// Get handles GET /api/v1/services/{id}
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid service ID")
		return
	}

	svc, err := h.svc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		log.Printf("failed to get service (id=%d, tenant=%s): %v", id, tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get service")
		return
	}
	if svc == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "service not found")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(svc.ToResponse()))
}

// Create handles POST /api/v1/services
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())

	var req models.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.Name == "" || req.ServiceCode == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and service_code are required")
		return
	}

	svc, err := h.svc.Create(r.Context(), tenantID, &req, user)
	if err != nil {
		log.Printf("failed to create service (tenant=%s): %v", tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create service")
		return
	}

	writeJSON(w, http.StatusCreated, models.SuccessResponse(svc.ToResponse()))
}

// Update handles PUT /api/v1/services/{id}
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid service ID")
		return
	}

	var req models.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	svc, err := h.svc.Update(r.Context(), tenantID, id, &req, user)
	if err != nil {
		log.Printf("failed to update service (id=%d, tenant=%s): %v", id, tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update service")
		return
	}
	if svc == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "service not found")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(svc.ToResponse()))
}

// Delete handles DELETE /api/v1/services/{id}
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid service ID")
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, id, user); err != nil {
		log.Printf("failed to delete service (id=%d, tenant=%s): %v", id, tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete service")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(nil))
}

// GetCategories handles GET /api/v1/services/categories
func (h *ServiceHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())

	categories, err := h.svc.GetCategories(r.Context(), tenantID)
	if err != nil {
		log.Printf("failed to get service categories (tenant=%s): %v", tenantID, err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get categories")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(categories))
}
