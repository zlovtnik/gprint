package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/zlovtnik/gprint/internal/middleware"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/internal/service"
)

// CustomerHandler handles customer HTTP requests
type CustomerHandler struct {
	svc *service.CustomerService
}

// NewCustomerHandler creates a new CustomerHandler
func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

// List handles GET /api/v1/customers
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	params := parsePagination(r)
	search := parseSearchParams(r)

	customers, total, err := h.svc.List(r.Context(), tenantID, params, search)
	if err != nil {
		log.Printf("failed to list customers: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	responses := make([]models.CustomerResponse, len(customers))
	for i, c := range customers {
		responses[i] = c.ToResponse()
	}

	result := models.NewPaginatedResponse(responses, params.Page, params.PageSize, total)
	writeJSON(w, http.StatusOK, models.SuccessResponse(result))
}

// Get handles GET /api/v1/customers/{id}
func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidCustomerID)
		return
	}

	customer, err := h.svc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		log.Printf("failed to retrieve customer (id=%d, tenant=%s): %v", id, tenantID, err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgFailedToRetrieveCustomer)
		return
	}
	if customer == nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgCustomerNotFound)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(customer.ToResponse()))
}

// Create handles POST /api/v1/customers
func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())

	var req models.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	if req.Name == "" || req.CustomerCode == "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidationErr, "name and customer_code are required")
		return
	}

	customer, err := h.svc.Create(r.Context(), tenantID, &req, user)
	if err != nil {
		log.Printf("failed to create customer: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, models.SuccessResponse(customer.ToResponse()))
}

// Update handles PUT /api/v1/customers/{id}
func (h *CustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidCustomerID)
		return
	}

	var req models.UpdateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, MsgInvalidRequestBody)
		return
	}

	customer, err := h.svc.Update(r.Context(), tenantID, id, &req, user)
	if err != nil {
		log.Printf("failed to update customer: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}
	if customer == nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgCustomerNotFound)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(customer.ToResponse()))
}

// Delete handles DELETE /api/v1/customers/{id}
func (h *CustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r.Context())
	user := middleware.GetUser(r.Context())
	id, err := parseIDFromPath(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidID, MsgInvalidCustomerID)
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, id, user); err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, MsgCustomerNotFound)
			return
		}
		log.Printf("failed to delete customer: %v", err)
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, MsgInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(nil))
}

// Helper functions

func parsePagination(r *http.Request) models.PaginationParams {
	params := models.DefaultPagination()

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			params.PageSize = ps
		}
	}

	return params
}

func parseSearchParams(r *http.Request) models.SearchParams {
	params := models.SearchParams{
		Query:   r.URL.Query().Get("q"),
		Field:   r.URL.Query().Get("field"),
		SortBy:  r.URL.Query().Get("sort_by"),
		SortDir: r.URL.Query().Get("sort_dir"),
	}

	if active := r.URL.Query().Get("active"); active != "" {
		b := strings.ToLower(active) == "true" || active == "1"
		params.Active = &b
	}

	return params
}

func parseIDFromPath(r *http.Request, name string) (int64, error) {
	idStr := r.PathValue(name)
	return strconv.ParseInt(idStr, 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Headers already sent, log the error
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorResponse(code, message, nil))
}
