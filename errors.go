package client

import (
	"fmt"
	"net/http"
	"time"
)

// HTTPError represents an HTTP error response from the Notion API.
// This provides detailed information about API errors including status codes,
// error messages, and any additional context from the API response.
type HTTPError struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int

	// Status is the HTTP status text (e.g., "Bad Request").
	Status string

	// Message is the error message from the API response.
	Message string

	// Code is the specific error code from the Notion API (e.g., "invalid_request").
	Code string

	// RequestID is the unique request ID for debugging purposes.
	RequestID string

	// Details contains additional error details from the API response.
	Details map[string]interface{}

	// Response is the original HTTP response (optional, for advanced debugging).
	Response *http.Response
}

// Error returns the error message for HTTPError.
func (e *HTTPError) Error() string {
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("HTTP %d %s: %s (%s)", e.StatusCode, e.Status, e.Message, e.Code)
	} else if e.Message != "" {
		return fmt.Sprintf("HTTP %d %s: %s", e.StatusCode, e.Status, e.Message)
	} else {
		return fmt.Sprintf("HTTP %d %s", e.StatusCode, e.Status)
	}
}

// IsRetryable returns true if this HTTP error should trigger a retry.
// This is based on the HTTP status code and error type.
func (e *HTTPError) IsRetryable() bool {
	return isRetryableHTTPStatus(e.StatusCode)
}

// IsBadRequest returns true if this is a 400 Bad Request error.
func (e *HTTPError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// IsUnauthorized returns true if this is a 401 Unauthorized error.
func (e *HTTPError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if this is a 403 Forbidden error.
func (e *HTTPError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsNotFound returns true if this is a 404 Not Found error.
func (e *HTTPError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsRateLimited returns true if this is a 429 Too Many Requests error.
func (e *HTTPError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsServerError returns true if this is a 5xx server error.
func (e *HTTPError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// RateLimitError represents a rate limit error (HTTP 429).
// This includes information about when the client can retry the request.
type RateLimitError struct {
	// RetryAfter is the duration to wait before retrying.
	RetryAfter time.Duration

	// Limit is the rate limit threshold.
	Limit int

	// Remaining is the number of requests remaining.
	Remaining int

	// Reset is when the rate limit window resets.
	Reset time.Time

	// Message is the error message from the API.
	Message string

	// RequestID is the unique request ID for debugging.
	RequestID string
}

// Error returns the error message for RateLimitError.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %v: %s", e.RetryAfter, e.Message)
	}
	return fmt.Sprintf("rate limit exceeded: %s", e.Message)
}

// IsRetryable returns true since rate limit errors should always be retried.
func (e *RateLimitError) IsRetryable() bool {
	return true
}

// ValidationError represents a request validation error.
// This occurs when the request data doesn't meet the API's requirements.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string

	// Message is the validation error message.
	Message string

	// Code is the specific validation error code.
	Code string

	// Value is the invalid value that was provided.
	Value interface{}
}

// Error returns the error message for ValidationError.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// IsRetryable returns false since validation errors indicate client bugs.
func (e *ValidationError) IsRetryable() bool {
	return false
}

// TimeoutError represents a request timeout error.
// This occurs when a request takes longer than the configured timeout.
type TimeoutError struct {
	// Timeout is the configured timeout duration.
	Timeout time.Duration

	// Operation is a description of the operation that timed out.
	Operation string

	// Elapsed is how long the operation ran before timing out.
	Elapsed time.Duration
}

// Error returns the error message for TimeoutError.
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout after %v during %s (elapsed: %v)", e.Timeout, e.Operation, e.Elapsed)
}

// IsRetryable returns true since timeouts are often transient.
func (e *TimeoutError) IsRetryable() bool {
	return true
}

// NetworkError represents a network-level error.
// This includes connection failures, DNS errors, etc.
type NetworkError struct {
	// Operation is the network operation that failed.
	Operation string

	// Address is the network address involved.
	Address string

	// Underlying is the underlying network error.
	Underlying error
}

// Error returns the error message for NetworkError.
func (e *NetworkError) Error() string {
	if e.Address != "" {
		return fmt.Sprintf("network error during %s to %s: %v", e.Operation, e.Address, e.Underlying)
	}
	return fmt.Sprintf("network error during %s: %v", e.Operation, e.Underlying)
}

// IsRetryable returns true since network errors are often transient.
func (e *NetworkError) IsRetryable() bool {
	return true
}

// Unwrap returns the underlying error for error unwrapping.
func (e *NetworkError) Unwrap() error {
	return e.Underlying
}

// AuthenticationError represents an authentication failure.
// This occurs when the API key is invalid or missing.
type AuthenticationError struct {
	// Message is the authentication error message.
	Message string

	// Code is the specific authentication error code.
	Code string

	// RequestID is the unique request ID for debugging.
	RequestID string
}

// Error returns the error message for AuthenticationError.
func (e *AuthenticationError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("authentication error (%s): %s", e.Code, e.Message)
	}
	return fmt.Sprintf("authentication error: %s", e.Message)
}

// IsRetryable returns false since authentication errors require fixing the API key.
func (e *AuthenticationError) IsRetryable() bool {
	return false
}

// AuthorizationError represents an authorization failure.
// This occurs when the API key doesn't have permission for the requested operation.
type AuthorizationError struct {
	// Message is the authorization error message.
	Message string

	// Code is the specific authorization error code.
	Code string

	// Resource is the resource that access was denied to.
	Resource string

	// RequestID is the unique request ID for debugging.
	RequestID string
}

// Error returns the error message for AuthorizationError.
func (e *AuthorizationError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("authorization error for resource '%s': %s", e.Resource, e.Message)
	}
	return fmt.Sprintf("authorization error: %s", e.Message)
}

// IsRetryable returns false since authorization errors require permission changes.
func (e *AuthorizationError) IsRetryable() bool {
	return false
}

// ConcurrencyError represents an error in concurrent operations.
// This can occur when worker pools are overloaded or channels are closed unexpectedly.
type ConcurrencyError struct {
	// Operation is the concurrent operation that failed.
	Operation string

	// Reason is the reason for the failure.
	Reason string

	// WorkerID is the ID of the worker that encountered the error (if applicable).
	WorkerID int
}

// Error returns the error message for ConcurrencyError.
func (e *ConcurrencyError) Error() string {
	if e.WorkerID > 0 {
		return fmt.Sprintf("concurrency error in %s (worker %d): %s", e.Operation, e.WorkerID, e.Reason)
	}
	return fmt.Sprintf("concurrency error in %s: %s", e.Operation, e.Reason)
}

// IsRetryable returns true since concurrency errors might be transient.
func (e *ConcurrencyError) IsRetryable() bool {
	return true
}

// PaginationError represents an error during paginated operations.
// This can occur when pagination cursors are invalid or when streaming fails.
type PaginationError struct {
	// Message is the pagination error message.
	Message string

	// Cursor is the pagination cursor that caused the error.
	Cursor string

	// Page is the page number where the error occurred.
	Page int

	// Operation is the pagination operation that failed.
	Operation string
}

// Error returns the error message for PaginationError.
func (e *PaginationError) Error() string {
	if e.Page > 0 {
		return fmt.Sprintf("pagination error on page %d during %s: %s", e.Page, e.Operation, e.Message)
	}
	return fmt.Sprintf("pagination error during %s: %s", e.Operation, e.Message)
}

// IsRetryable returns true since pagination errors might be transient.
func (e *PaginationError) IsRetryable() bool {
	return true
}

// SerializationError represents an error during JSON serialization/deserialization.
// This can occur when API responses don't match expected formats.
type SerializationError struct {
	// Operation indicates whether this was during marshaling or unmarshaling.
	Operation string // "marshal" or "unmarshal"

	// Type is the Go type involved in the serialization.
	Type string

	// Field is the specific field that caused the error (if known).
	Field string

	// Underlying is the underlying serialization error.
	Underlying error
}

// Error returns the error message for SerializationError.
func (e *SerializationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("serialization error during %s of %s.%s: %v", e.Operation, e.Type, e.Field, e.Underlying)
	}
	return fmt.Sprintf("serialization error during %s of %s: %v", e.Operation, e.Type, e.Underlying)
}

// IsRetryable returns false since serialization errors indicate data format issues.
func (e *SerializationError) IsRetryable() bool {
	return false
}

// Unwrap returns the underlying error for error unwrapping.
func (e *SerializationError) Unwrap() error {
	return e.Underlying
}

// ErrorClassifier provides methods for classifying and handling different types of errors.
// This is used throughout the client to make intelligent decisions about retry behavior.
type ErrorClassifier struct{}

// NewErrorClassifier creates a new error classifier.
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// IsRetryable determines if an error should trigger a retry attempt.
// This centralizes the retry logic for all error types.
//
// Arguments:
//   - err: The error to classify.
//
// Returns:
//   - bool: True if the error is retryable, false otherwise.
func (ec *ErrorClassifier) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for errors that implement IsRetryable method
	if retryable, ok := err.(interface{ IsRetryable() bool }); ok {
		return retryable.IsRetryable()
	}

	// Fall back to generic error classification
	return isRetryableError(err)
}

// ClassifyHTTPError creates a specific error type based on an HTTP response.
// This converts generic HTTP errors into more specific error types.
//
// Arguments:
//   - resp: The HTTP response containing the error.
//   - body: The response body (if available).
//
// Returns:
//   - error: A specific error type based on the HTTP response.
func (ec *ErrorClassifier) ClassifyHTTPError(resp *http.Response, body []byte) error {
	if resp == nil {
		return fmt.Errorf("nil HTTP response")
	}

	// Extract request ID for debugging
	requestID := resp.Header.Get("X-Request-Id")

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		rateLimitInfo := ExtractRateLimitInfo(resp)
		rateLimitErr := &RateLimitError{
			RequestID: requestID,
			Message:   "rate limit exceeded",
		}

		if rateLimitInfo != nil {
			rateLimitErr.RetryAfter = rateLimitInfo.RetryAfter
			rateLimitErr.Limit = rateLimitInfo.Limit
			rateLimitErr.Remaining = rateLimitInfo.Remaining
			rateLimitErr.Reset = rateLimitInfo.Reset
		}

		return rateLimitErr
	}

	// Handle authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		return &AuthenticationError{
			Message:   "invalid or missing API key",
			Code:      "unauthorized",
			RequestID: requestID,
		}
	}

	// Handle authorization errors
	if resp.StatusCode == http.StatusForbidden {
		return &AuthorizationError{
			Message:   "insufficient permissions",
			Code:      "forbidden",
			RequestID: requestID,
		}
	}

	// Try to parse error details from response body
	message := "unknown error"
	code := ""

	if len(body) > 0 {
		// In a real implementation, you would parse the JSON error response here
		// For now, we'll use the raw body as the message
		message = string(body)
	}

	// Create generic HTTP error
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Message:    message,
		Code:       code,
		RequestID:  requestID,
		Response:   resp,
	}
}

// WrapNetworkError wraps a network error with additional context.
//
// Arguments:
//   - operation: Description of the network operation.
//   - address: The network address involved.
//   - err: The underlying network error.
//
// Returns:
//   - error: A NetworkError with the provided context.
func (ec *ErrorClassifier) WrapNetworkError(operation, address string, err error) error {
	return &NetworkError{
		Operation:  operation,
		Address:    address,
		Underlying: err,
	}
}

// WrapTimeoutError creates a timeout error with context.
//
// Arguments:
//   - operation: Description of the operation that timed out.
//   - timeout: The configured timeout duration.
//   - elapsed: How long the operation ran before timing out.
//
// Returns:
//   - error: A TimeoutError with the provided context.
func (ec *ErrorClassifier) WrapTimeoutError(operation string, timeout, elapsed time.Duration) error {
	return &TimeoutError{
		Timeout:   timeout,
		Operation: operation,
		Elapsed:   elapsed,
	}
}

// WrapSerializationError creates a serialization error with context.
//
// Arguments:
//   - operation: "marshal" or "unmarshal".
//   - typeName: The Go type involved.
//   - field: The specific field (if known).
//   - err: The underlying serialization error.
//
// Returns:
//   - error: A SerializationError with the provided context.
func (ec *ErrorClassifier) WrapSerializationError(operation, typeName, field string, err error) error {
	return &SerializationError{
		Operation:  operation,
		Type:       typeName,
		Field:      field,
		Underlying: err,
	}
}

// IsTemporary checks if an error is temporary and likely to resolve itself.
// This is useful for determining retry strategies.
//
// Arguments:
//   - err: The error to check.
//
// Returns:
//   - bool: True if the error is likely temporary, false otherwise.
func (ec *ErrorClassifier) IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific temporary error types
	switch err.(type) {
	case *TimeoutError, *NetworkError, *RateLimitError:
		return true
	case *HTTPError:
		httpErr := err.(*HTTPError)
		return httpErr.IsServerError()
	}

	// Check if error implements Temporary method
	if temp, ok := err.(interface{ Temporary() bool }); ok {
		return temp.Temporary()
	}

	return false
}

// IsPermanent checks if an error is permanent and won't resolve with retries.
// This is the opposite of IsTemporary and helps with error handling decisions.
//
// Arguments:
//   - err: The error to check.
//
// Returns:
//   - bool: True if the error is permanent, false otherwise.
func (ec *ErrorClassifier) IsPermanent(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific permanent error types
	switch err.(type) {
	case *AuthenticationError, *AuthorizationError, *ValidationError, *SerializationError:
		return true
	case *HTTPError:
		httpErr := err.(*HTTPError)
		return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && !httpErr.IsRateLimited()
	}

	return false
}
