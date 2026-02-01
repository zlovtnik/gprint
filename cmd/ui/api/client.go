package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// maxResponseBody is the maximum size of response body to read (10MB)
const maxResponseBody = 10 << 20

// ErrResponseTooLarge is returned when the response body exceeds maxResponseBody
var ErrResponseTooLarge = errors.New("response body too large")

// ErrInvalidBaseURL is returned when the base URL is empty or malformed
var ErrInvalidBaseURL = errors.New("invalid base URL: must be non-empty with scheme and host")

// Client is an HTTP client for the GPrint API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	mu         sync.RWMutex
	token      string
}

// NewClient creates a new API client.
// Returns an error if baseURL is empty or malformed (missing scheme/host).
func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		return nil, ErrInvalidBaseURL
	}

	// Validate URL has scheme and host
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidBaseURL
	}

	// Normalize BaseURL by trimming all trailing slashes to prevent double slashes
	normalizedURL := strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: normalizedURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SetToken sets the JWT token for authenticated requests
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// getToken returns the current JWT token in a thread-safe manner
func (c *Client) getToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// Response wraps API responses
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorResponse  `json:"error,omitempty"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Data       json.RawMessage `json:"data"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalCount int             `json:"total_count"`
	TotalPages int             `json:"total_pages"`
}

// ErrorString safely returns the error message from a Response.
// Returns empty string for successful responses.
func (r *Response) ErrorString() string {
	if r == nil {
		return "no response"
	}
	if r.Success {
		return ""
	}
	if r.Error == nil {
		return "unknown error"
	}
	if r.Error.Code != "" {
		return r.Error.Code + ": " + r.Error.Message
	}
	return r.Error.Message
}

// doRequest performs an HTTP request without context (uses background context)
func (c *Client) doRequest(method, path string, body interface{}) (*Response, error) {
	return c.doRequestWithContext(context.Background(), method, path, body)
}

// marshalBody converts body to JSON reader, returns nil if body is nil
func marshalBody(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewBuffer(jsonBody), nil
}

// readResponseBody reads and validates response body size
func readResponseBody(body io.ReadCloser) ([]byte, error) {
	respBody, err := io.ReadAll(io.LimitReader(body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if len(respBody) > maxResponseBody {
		return nil, ErrResponseTooLarge
	}
	return respBody, nil
}

// parseErrorResponse attempts to parse an error response from body
func parseErrorResponse(statusCode int, body []byte) error {
	var errResp Response
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		if errResp.Error.Code != "" {
			return fmt.Errorf("HTTP %d: %s: %s", statusCode, errResp.Error.Code, errResp.Error.Message)
		}
		return fmt.Errorf("HTTP %d: %s", statusCode, errResp.Error.Message)
	}
	// Fall back to truncated body (rune-safe to avoid splitting UTF-8 characters)
	errBody := string(body)
	runes := []rune(errBody)
	if len(runes) > 200 {
		errBody = string(runes[:200]) + "..."
	}
	return fmt.Errorf("HTTP %d: %s", statusCode, errBody)
}

// doRequestWithContext performs an HTTP request with context support
func (c *Client) doRequestWithContext(ctx context.Context, method, path string, body interface{}) (*Response, error) {
	reqBody, err := marshalBody(body)
	if err != nil {
		return nil, err
	}

	// Normalize path to ensure leading slash
	if path != "" && path[0] != '/' {
		path = "/" + path
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Always request JSON responses
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := c.getToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(resp.StatusCode, respBody)
	}

	// Handle 204 No Content and empty responses
	if resp.StatusCode == http.StatusNoContent || len(respBody) == 0 {
		return &Response{Success: true}, nil
	}

	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response (HTTP %d): %w", resp.StatusCode, err)
	}

	return &apiResp, nil
}

// Get performs a GET request
func (c *Client) Get(path string) (*Response, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

// GetWithContext performs a GET request with context support
func (c *Client) GetWithContext(ctx context.Context, path string) (*Response, error) {
	return c.doRequestWithContext(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) (*Response, error) {
	return c.doRequest(http.MethodPost, path, body)
}

// PostWithContext performs a POST request with context support
func (c *Client) PostWithContext(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.doRequestWithContext(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request
func (c *Client) Put(path string, body interface{}) (*Response, error) {
	return c.doRequest(http.MethodPut, path, body)
}

// PutWithContext performs a PUT request with context support
func (c *Client) PutWithContext(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.doRequestWithContext(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) (*Response, error) {
	return c.doRequest(http.MethodDelete, path, nil)
}

// DeleteWithContext performs a DELETE request with context support
func (c *Client) DeleteWithContext(ctx context.Context, path string) (*Response, error) {
	return c.doRequestWithContext(ctx, http.MethodDelete, path, nil)
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body interface{}) (*Response, error) {
	return c.doRequest(http.MethodPatch, path, body)
}

// PatchWithContext performs a PATCH request with context support
func (c *Client) PatchWithContext(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.doRequestWithContext(ctx, http.MethodPatch, path, body)
}
