package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// ═══════════════════════════════════════════════════════════════════════════
// GENERIC CRUD HELPERS - Reduce boilerplate for entity operations
// ═══════════════════════════════════════════════════════════════════════════

// GetByID fetches a single entity by ID using a GET request.
// T is the entity type to unmarshal to.
func GetByID[T any](c *Client, pathFmt string, id int64) (*T, error) {
	return GetByIDWithContext[T](context.Background(), c, pathFmt, id)
}

// GetByIDWithContext fetches a single entity by ID with context support.
func GetByIDWithContext[T any](ctx context.Context, c *Client, pathFmt string, id int64) (*T, error) {
	resp, err := c.GetWithContext(ctx, fmt.Sprintf(pathFmt, id))
	if err != nil {
		return nil, err
	}
	return parseResponseData[T](resp)
}

// CreateEntity creates a new entity via POST request.
func CreateEntity[T any, R any](c *Client, path string, req *R) (*T, error) {
	return CreateEntityWithContext[T](context.Background(), c, path, req)
}

// CreateEntityWithContext creates a new entity with context support.
func CreateEntityWithContext[T any, R any](ctx context.Context, c *Client, path string, req *R) (*T, error) {
	resp, err := c.doRequestWithContext(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}
	return parseResponseData[T](resp)
}

// UpdateEntity updates an entity via PUT request.
func UpdateEntity[T any, R any](c *Client, pathFmt string, id int64, req *R) (*T, error) {
	return UpdateEntityWithContext[T](context.Background(), c, pathFmt, id, req)
}

// UpdateEntityWithContext updates an entity with context support.
func UpdateEntityWithContext[T any, R any](ctx context.Context, c *Client, pathFmt string, id int64, req *R) (*T, error) {
	resp, err := c.doRequestWithContext(ctx, "PUT", fmt.Sprintf(pathFmt, id), req)
	if err != nil {
		return nil, err
	}
	return parseResponseData[T](resp)
}

// DeleteEntity deletes an entity via DELETE request.
func DeleteEntity(c *Client, pathFmt string, id int64) error {
	return DeleteEntityWithContext(context.Background(), c, pathFmt, id)
}

// DeleteEntityWithContext deletes an entity with context support.
func DeleteEntityWithContext(ctx context.Context, c *Client, pathFmt string, id int64) error {
	resp, err := c.doRequestWithContext(ctx, "DELETE", fmt.Sprintf(pathFmt, id), nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// PostAction performs a POST action on an entity (e.g., sign, generate).
func PostAction(c *Client, pathFmt string, id int64, action string, payload any) error {
	return PostActionWithContext(context.Background(), c, pathFmt, id, action, payload)
}

// PostActionWithContext performs a POST action with context support.
func PostActionWithContext(ctx context.Context, c *Client, pathFmt string, id int64, action string, payload any) error {
	path := fmt.Sprintf(pathFmt+"/%s", id, action)
	resp, err := c.doRequestWithContext(ctx, "POST", path, payload)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	return nil
}

// PostActionWithResult performs a POST action and returns a result entity.
func PostActionWithResult[T any](c *Client, pathFmt string, id int64, action string, payload any) (*T, error) {
	return PostActionWithResultContext[T](context.Background(), c, pathFmt, id, action, payload)
}

// PostActionWithResultContext performs a POST action with result and context support.
func PostActionWithResultContext[T any](ctx context.Context, c *Client, pathFmt string, id int64, action string, payload any) (*T, error) {
	path := fmt.Sprintf(pathFmt+"/%s", id, action)
	resp, err := c.doRequestWithContext(ctx, "POST", path, payload)
	if err != nil {
		return nil, err
	}
	return parseResponseData[T](resp)
}

// ═══════════════════════════════════════════════════════════════════════════
// RESPONSE PARSING HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// parseResponseData handles common response parsing logic.
func parseResponseData[T any](resp *Response) (*T, error) {
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var result T
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// parseResponseList parses a list response (for non-paginated endpoints).
func parseResponseList[T any](resp *Response) ([]T, error) {
	if !resp.Success {
		return nil, fmt.Errorf(apiErrorFmt, resp.ErrorString())
	}
	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	var items []T
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return items, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ENTITY CRUD FACTORY - Generate CRUD functions for an entity type
// ═══════════════════════════════════════════════════════════════════════════

// EntityConfig defines paths for entity CRUD operations.
type EntityConfig struct {
	BasePath    string // e.g., "/api/v1/customers"
	ByIDPathFmt string // e.g., "/api/v1/customers/%d"
}

// EntityCRUD provides CRUD operations for a specific entity type.
type EntityCRUD[T any, CreateReq any, UpdateReq any] struct {
	client *Client
	config EntityConfig
}

// NewEntityCRUD creates a new EntityCRUD instance.
func NewEntityCRUD[T any, CreateReq any, UpdateReq any](client *Client, config EntityConfig) *EntityCRUD[T, CreateReq, UpdateReq] {
	return &EntityCRUD[T, CreateReq, UpdateReq]{
		client: client,
		config: config,
	}
}

// List fetches entities with pagination.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) List(opts *ListOptions) (*ListResult[T], error) {
	return listItems[T](e.client, e.config.BasePath, opts)
}

// ListWithContext fetches entities with pagination and context.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) ListWithContext(ctx context.Context, opts *ListOptions) (*ListResult[T], error) {
	return listItemsWithContext[T](ctx, e.client, e.config.BasePath, opts)
}

// Get fetches an entity by ID.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) Get(id int64) (*T, error) {
	return GetByID[T](e.client, e.config.ByIDPathFmt, id)
}

// GetWithContext fetches an entity by ID with context.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) GetWithContext(ctx context.Context, id int64) (*T, error) {
	return GetByIDWithContext[T](ctx, e.client, e.config.ByIDPathFmt, id)
}

// Create creates a new entity.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) Create(req *CreateReq) (*T, error) {
	return CreateEntity[T](e.client, e.config.BasePath, req)
}

// CreateWithContext creates a new entity with context.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) CreateWithContext(ctx context.Context, req *CreateReq) (*T, error) {
	return CreateEntityWithContext[T](ctx, e.client, e.config.BasePath, req)
}

// Update updates an entity.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) Update(id int64, req *UpdateReq) (*T, error) {
	return UpdateEntity[T](e.client, e.config.ByIDPathFmt, id, req)
}

// UpdateWithContext updates an entity with context.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) UpdateWithContext(ctx context.Context, id int64, req *UpdateReq) (*T, error) {
	return UpdateEntityWithContext[T](ctx, e.client, e.config.ByIDPathFmt, id, req)
}

// Delete deletes an entity.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) Delete(id int64) error {
	return DeleteEntity(e.client, e.config.ByIDPathFmt, id)
}

// DeleteWithContext deletes an entity with context.
func (e *EntityCRUD[T, CreateReq, UpdateReq]) DeleteWithContext(ctx context.Context, id int64) error {
	return DeleteEntityWithContext(ctx, e.client, e.config.ByIDPathFmt, id)
}
