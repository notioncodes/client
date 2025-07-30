package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// HTTPClient provides a high-performance HTTP client for the Notion API with built-in
// retry logic, rate limiting, metrics collection, and concurrent operation support.
type HTTPClient struct {
	config      *Config
	httpClient  *http.Client
	retryPolicy RetryPolicy
	breaker     *CircuitBreaker
	classifier  *ErrorClassifier
	metrics     *MetricsCollector
	baseHeaders map[string]string
	mu          sync.RWMutex
}

// NewHTTPClient creates a new HTTP client with the specified configuration.
// This client is optimized for the Notion API with proper authentication,
// retry logic, and performance characteristics.
//
// Arguments:
//   - config: Configuration for the HTTP client behavior and performance.
//
// Returns:
//   - *HTTPClient: A new HTTP client instance ready for API calls.
//   - error: Configuration error if the config is invalid.
//
// Example:
//
//	config := DefaultConfig()
//	config.APIKey = "your-api-key"
//	client, err := NewHTTPClient(config)
//	if err != nil {
//	    log.Fatalf("Failed to create client: %v", err)
//	}
//	defer client.Close()
func NewHTTPClient(config *Config) (*HTTPClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	httpClient := config.CreateHTTPClient()
	retryPolicy := NewDefaultRetryPolicy(config.GetRetryConfig())
	circuitBreaker := NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout)
	classifier := NewErrorClassifier()

	var metrics *MetricsCollector
	if config.EnableMetrics {
		metrics = NewMetricsCollector()
	}

	// Prepare base headers
	baseHeaders := map[string]string{
		"Authorization":  "Bearer " + config.APIKey,
		"Notion-Version": config.Version,
		"Content-Type":   "application/json",
		"User-Agent":     config.UserAgent,
	}

	// Add custom headers
	for key, value := range config.CustomHeaders {
		baseHeaders[key] = value
	}

	return &HTTPClient{
		config:      config,
		httpClient:  httpClient,
		retryPolicy: retryPolicy,
		breaker:     circuitBreaker,
		classifier:  classifier,
		metrics:     metrics,
		baseHeaders: baseHeaders,
	}, nil
}

// Request represents an HTTP request to be made to the Notion API.
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Body    interface{}
	Headers map[string]string
}

// Response represents the response from an HTTP request.
type Response[T any] struct {
	Data       T
	StatusCode int
	Headers    http.Header
	RawBody    []byte
}

// Get performs a GET request to the specified path with query parameters.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path (e.g., "/pages/123").
//   - query: Query parameters to include in the request.
//
// Returns:
//   - *http.Response: The raw HTTP response.
//   - error: Any error that occurred during the request.
//
// Example:
//
//	resp, err := client.Get(ctx, "/pages/123", nil)
//	if err != nil {
//	    log.Printf("GET failed: %v", err)
//	    return
//	}
//	defer resp.Body.Close()
func (c *HTTPClient) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	req := &Request{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	}
	return c.executeRequest(ctx, req)
}

// Post performs a POST request to the specified path with a JSON body.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - body: Request body to be serialized as JSON.
//
// Returns:
//   - *http.Response: The raw HTTP response.
//   - error: Any error that occurred during the request.
//
// Example:
//
//	createReq := &PageCreateRequest{...}
//	resp, err := client.Post(ctx, "/pages", createReq)
//	if err != nil {
//	    log.Printf("POST failed: %v", err)
//	    return
//	}
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	req := &Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	}
	return c.executeRequest(ctx, req)
}

// Patch performs a PATCH request to the specified path with a JSON body.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - body: Request body to be serialized as JSON.
//
// Returns:
//   - *http.Response: The raw HTTP response.
//   - error: Any error that occurred during the request.
//
// Example:
//
//	updateReq := &PageUpdateRequest{...}
//	resp, err := client.Patch(ctx, "/pages/123", updateReq)
//	if err != nil {
//	    log.Printf("PATCH failed: %v", err)
//	    return
//	}
func (c *HTTPClient) Patch(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	req := &Request{
		Method: http.MethodPatch,
		Path:   path,
		Body:   body,
	}
	return c.executeRequest(ctx, req)
}

// Delete performs a DELETE request to the specified path.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//
// Returns:
//   - *http.Response: The raw HTTP response.
//   - error: Any error that occurred during the request.
//
// Example:
//
//	resp, err := client.Delete(ctx, "/blocks/123")
//	if err != nil {
//	    log.Printf("DELETE failed: %v", err)
//	    return
//	}
func (c *HTTPClient) Delete(ctx context.Context, path string) (*http.Response, error) {
	req := &Request{
		Method: http.MethodDelete,
		Path:   path,
	}
	return c.executeRequest(ctx, req)
}

// GetJSON performs a GET request and unmarshals the response into the specified type.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - query: Query parameters to include in the request.
//   - result: Pointer to the struct to unmarshal the response into.
//
// Returns:
//   - error: Any error that occurred during the request or unmarshaling.
//
// Example:
//
//	var page types.Page
//	err := client.GetJSON(ctx, "/pages/123", nil, &page)
//	if err != nil {
//	    log.Printf("Failed to get page: %v", err)
//	    return
//	}
func (c *HTTPClient) GetJSON(ctx context.Context, path string, query url.Values, result interface{}) error {
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.unmarshalResponse(resp, result)
}

// PostJSON performs a POST request and unmarshals the response into the specified type.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - body: Request body to be serialized as JSON.
//   - result: Pointer to the struct to unmarshal the response into.
//
// Returns:
//   - error: Any error that occurred during the request or unmarshaling.
//
// Example:
//
//	createReq := &PageCreateRequest{...}
//	var page types.Page
//	err := client.PostJSON(ctx, "/pages", createReq, &page)
//	if err != nil {
//	    log.Printf("Failed to create page: %v", err)
//	    return
//	}
func (c *HTTPClient) PostJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.unmarshalResponse(resp, result)
}

// PatchJSON performs a PATCH request and unmarshals the response into the specified type.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - body: Request body to be serialized as JSON.
//   - result: Pointer to the struct to unmarshal the response into.
//
// Returns:
//   - error: Any error that occurred during the request or unmarshaling.
//
// Example:
//
//	updateReq := &PageUpdateRequest{...}
//	var page types.Page
//	err := client.PatchJSON(ctx, "/pages/123", updateReq, &page)
//	if err != nil {
//	    log.Printf("Failed to update page: %v", err)
//	    return
//	}
func (c *HTTPClient) PatchJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Patch(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.unmarshalResponse(resp, result)
}

// executeRequest executes an HTTP request with retry logic and error handling.
func (c *HTTPClient) executeRequest(ctx context.Context, req *Request) (*http.Response, error) {
	// Check circuit breaker
	canExecute, err := c.breaker.CanExecute()
	if !canExecute {
		if c.metrics != nil {
			c.metrics.RecordRequest("circuit_breaker_open", time.Duration(0), err)
		}
		return nil, err
	}

	startTime := time.Now()

	// Execute with retry logic
	result, err := ExecuteWithRetry(ctx, func(ctx context.Context, attempt int) (*http.Response, error) {
		return c.doRequest(ctx, req, attempt)
	}, c.retryPolicy)

	duration := time.Since(startTime)

	// Record metrics
	if c.metrics != nil {
		operation := fmt.Sprintf("%s %s", req.Method, req.Path)
		c.metrics.RecordRequest(operation, duration, err)
	}

	// Update circuit breaker
	if err != nil {
		c.breaker.RecordFailure()
	} else {
		c.breaker.RecordSuccess()
	}

	return result, err
}

// doRequest performs a single HTTP request without retry logic.
func (c *HTTPClient) doRequest(ctx context.Context, req *Request, attempt int) (*http.Response, error) {
	// Build URL
	url := c.config.BaseURL + req.Path
	if req.Query != nil && len(req.Query) > 0 {
		url += "?" + req.Query.Encode()
	}

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, c.classifier.WrapSerializationError("marshal", fmt.Sprintf("%T", req.Body), "", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.mu.RLock()
	for key, value := range c.baseHeaders {
		httpReq.Header.Set(key, value)
	}
	c.mu.RUnlock()

	// Set request-specific headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add attempt number for debugging
	httpReq.Header.Set("X-Retry-Attempt", strconv.Itoa(attempt))

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, c.classifier.WrapNetworkError(req.Method, url, err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, c.classifier.ClassifyHTTPError(resp, body)
	}

	return resp, nil
}

// unmarshalResponse unmarshals an HTTP response body into the specified result.
func (c *HTTPClient) unmarshalResponse(resp *http.Response, result interface{}) error {
	if result == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.classifier.WrapNetworkError("read_response", "", err)
	}

	if len(body) == 0 {
		return nil
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return c.classifier.WrapSerializationError("unmarshal", fmt.Sprintf("%T", result), "", err)
	}

	return nil
}

// SetHeader sets a header that will be included in all requests.
// This is thread-safe and can be called concurrently.
//
// Arguments:
//   - key: Header name.
//   - value: Header value.
//
// Example:
//
//	client.SetHeader("X-Custom-Header", "custom-value")
func (c *HTTPClient) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseHeaders[key] = value
}

// RemoveHeader removes a header from all future requests.
// This is thread-safe and can be called concurrently.
//
// Arguments:
//   - key: Header name to remove.
//
// Example:
//
//	client.RemoveHeader("X-Custom-Header")
func (c *HTTPClient) RemoveHeader(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.baseHeaders, key)
}

// GetMetrics returns the current metrics if metrics collection is enabled.
//
// Returns:
//   - *Metrics: Current metrics data, or nil if metrics are disabled.
//
// Example:
//
//	if metrics := client.GetMetrics(); metrics != nil {
//	    fmt.Printf("Total requests: %d\n", metrics.TotalRequests)
//	}
func (c *HTTPClient) GetMetrics() *Metrics {
	if c.metrics == nil {
		return nil
	}
	return c.metrics.GetMetrics()
}

// ResetMetrics resets all collected metrics.
// This is useful for periodic metrics reporting.
//
// Example:
//
//	metrics := client.GetMetrics()
//	// Report metrics...
//	client.ResetMetrics()
func (c *HTTPClient) ResetMetrics() {
	if c.metrics != nil {
		c.metrics.Reset()
	}
}

// Close closes the HTTP client and releases resources.
// This should be called when the client is no longer needed.
//
// Example:
//
//	client := NewHTTPClient(config)
//	defer client.Close()
func (c *HTTPClient) Close() error {
	// Close the underlying HTTP client's transport
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// PaginationRequest represents a request for paginated data.
type PaginationRequest struct {
	// StartCursor is the cursor to start pagination from.
	StartCursor *string

	// PageSize is the number of items to return per page.
	PageSize *int

	// Filter contains filtering criteria for the request.
	Filter interface{}

	// Sorts contains sorting criteria for the request.
	Sorts interface{}
}

// PaginationResponse represents a paginated response from the API.
type PaginationResponse[T any] struct {
	// Results contains the items in this page.
	Results []T `json:"results"`

	// NextCursor is the cursor for the next page (nil if no more pages).
	NextCursor *string `json:"next_cursor"`

	// HasMore indicates if there are more pages available.
	HasMore bool `json:"has_more"`

	// Object is the type of the response object.
	Object string `json:"object"`

	// Type is the specific type of the paginated response.
	Type string `json:"type,omitempty"`
}

// GetPaginated performs a paginated GET request and returns a channel of results.
// This handles pagination automatically and streams results as they become available.
//
// Arguments:
//   - client: The HTTPClient to use for requests.
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - req: Pagination request parameters.
//
// Returns:
//   - <-chan Result[T]: Channel of paginated results.
//
// Example:
//
//	req := &PaginationRequest{PageSize: &50}
//	results := GetPaginated[types.Page](client, ctx, "/search", req)
//	for result := range results {
//	    if result.IsError() {
//	        log.Printf("Error: %v", result.Error)
//	        break
//	    }
//	    // Process result.Data
//	}
func GetPaginated[T any](c *HTTPClient, ctx context.Context, path string, req *PaginationRequest) <-chan Result[T] {
	resultCh := make(chan Result[T], c.config.BufferSize)

	go func() {
		defer close(resultCh)

		cursor := req.StartCursor
		pageNum := 1

		for {
			select {
			case <-ctx.Done():
				resultCh <- Error[T](ctx.Err())
				return
			default:
			}

			// Build query parameters
			query := make(url.Values)
			if cursor != nil {
				query.Set("start_cursor", *cursor)
			}

			pageSize := c.config.PageSize
			if req.PageSize != nil && *req.PageSize > 0 {
				pageSize = *req.PageSize
			}
			query.Set("page_size", strconv.Itoa(pageSize))

			// Make request
			var response PaginationResponse[T]
			err := c.GetJSON(ctx, path, query, &response)
			if err != nil {
				// Wrap with pagination context
				paginationErr := &PaginationError{
					Message:   err.Error(),
					Cursor:    "",
					Page:      pageNum,
					Operation: "get_paginated",
				}
				if cursor != nil {
					paginationErr.Cursor = *cursor
				}
				resultCh <- Error[T](paginationErr)
				return
			}

			// Send results
			for i, item := range response.Results {
				metadata := &ResultMetadata{
					FromStream:     true,
					StreamPosition: (pageNum-1)*pageSize + i + 1,
					PageInfo: &PageInfo{
						HasMore:    response.HasMore,
						NextCursor: response.NextCursor,
						PageSize:   len(response.Results),
						PageNumber: pageNum,
					},
				}

				result := SuccessWithMetadata(item, metadata)

				select {
				case resultCh <- result:
				case <-ctx.Done():
					resultCh <- Error[T](ctx.Err())
					return
				}
			}

			// Check if we have more pages
			if !response.HasMore || response.NextCursor == nil {
				break
			}

			cursor = response.NextCursor
			pageNum++
		}
	}()

	return resultCh
}

// PostPaginated performs a paginated POST request (useful for search operations).
//
// Arguments:
//   - client: The HTTPClient to use for requests.
//   - ctx: Context for cancellation and timeouts.
//   - path: API endpoint path.
//   - body: Request body containing search criteria and pagination parameters.
//
// Returns:
//   - <-chan Result[T]: Channel of paginated results.
//
// Example:
//
//	searchReq := &SearchRequest{Query: "project", PageSize: 50}
//	results := PostPaginated[types.Page](client, ctx, "/search", searchReq)
//	for result := range results {
//	    if result.IsSuccess() {
//	        fmt.Printf("Found page: %s\n", result.Data.GetTitle())
//	    }
//	}
func PostPaginated[T any](c *HTTPClient, ctx context.Context, path string, body interface{}) <-chan Result[T] {
	resultCh := make(chan Result[T], c.config.BufferSize)

	go func() {
		defer close(resultCh)

		var cursor *string
		pageNum := 1

		for {
			select {
			case <-ctx.Done():
				resultCh <- Error[T](ctx.Err())
				return
			default:
			}

			// Update cursor in request body if it's a map
			if bodyMap, ok := body.(map[string]interface{}); ok {
				if cursor != nil {
					bodyMap["start_cursor"] = *cursor
				}
			}

			// Make request
			var response PaginationResponse[T]
			err := c.PostJSON(ctx, path, body, &response)
			if err != nil {
				paginationErr := &PaginationError{
					Message:   err.Error(),
					Cursor:    "",
					Page:      pageNum,
					Operation: "post_paginated",
				}
				if cursor != nil {
					paginationErr.Cursor = *cursor
				}
				resultCh <- Error[T](paginationErr)
				return
			}

			// Send results
			for i, item := range response.Results {
				metadata := &ResultMetadata{
					FromStream:     true,
					StreamPosition: (pageNum-1)*len(response.Results) + i + 1,
					PageInfo: &PageInfo{
						HasMore:    response.HasMore,
						NextCursor: response.NextCursor,
						PageSize:   len(response.Results),
						PageNumber: pageNum,
					},
				}

				result := SuccessWithMetadata(item, metadata)

				select {
				case resultCh <- result:
				case <-ctx.Done():
					resultCh <- Error[T](ctx.Err())
					return
				}
			}

			// Check if we have more pages
			if !response.HasMore || response.NextCursor == nil {
				break
			}

			cursor = response.NextCursor
			pageNum++
		}
	}()

	return resultCh
}
