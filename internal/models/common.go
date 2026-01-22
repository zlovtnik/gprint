package models

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// DefaultPagination returns default pagination settings
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:     1,
		PageSize: 20,
	}
}

// Offset calculates the offset for database queries
func (p PaginationParams) Offset() int {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * p.PageSize
}

// Limit returns the page size
func (p PaginationParams) Limit() int {
	return p.PageSize
}

// PaginatedResponse wraps paginated results
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](data []T, page, pageSize, totalCount int) PaginatedResponse[T] {
	if pageSize <= 0 {
		pageSize = 20 // fallback to default
	}
	totalPages := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPages++
	}
	// Ensure data is never nil (serialize as [] instead of null)
	dataOrEmpty := data
	if dataOrEmpty == nil {
		dataOrEmpty = make([]T, 0)
	}
	return PaginatedResponse[T]{
		Data:       dataOrEmpty,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// SuccessResponse creates a success response
func SuccessResponse(data any) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
	}
}

// ErrorResponse creates an error response
func ErrorResponse(code, message string, details any) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// SearchParams holds search parameters
type SearchParams struct {
	Query   string `json:"query"`
	Field   string `json:"field"`
	SortBy  string `json:"sort_by"`
	SortDir string `json:"sort_dir"`
	Active  *bool  `json:"active,omitempty"`
}
