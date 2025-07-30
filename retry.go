package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RetryPolicy defines the strategy for retrying failed requests.
// This interface allows for custom retry logic while providing sensible defaults.
type RetryPolicy interface {
	// ShouldRetry determines if a request should be retried based on the error and attempt number.
	ShouldRetry(ctx context.Context, err error, attempt int) bool

	// BackoffDuration calculates how long to wait before the next retry attempt.
	BackoffDuration(attempt int) time.Duration
}

// DefaultRetryPolicy implements a production-ready retry policy with exponential backoff.
// This policy handles network errors, timeouts, and specific HTTP status codes intelligently.
type DefaultRetryPolicy struct {
	config RetryConfig
}

// NewDefaultRetryPolicy creates a new default retry policy with the given configuration.
//
// Arguments:
//   - config: Retry configuration specifying backoff behavior and limits.
//
// Returns:
//   - *DefaultRetryPolicy: A new default retry policy instance.
//
// Example:
//
//	config := RetryConfig{
//	    MaxRetries: 3,
//	    BaseBackoff: time.Second,
//	    MaxBackoff: 30 * time.Second,
//	    BackoffMultiplier: 2.0,
//	}
//	policy := NewDefaultRetryPolicy(config)
func NewDefaultRetryPolicy(config RetryConfig) *DefaultRetryPolicy {
	return &DefaultRetryPolicy{config: config}
}

// ShouldRetry determines if a request should be retried based on the error type and attempt count.
// This implementation retries on network errors, timeouts, and specific HTTP status codes.
//
// Arguments:
//   - ctx: Context for the operation (checked for cancellation).
//   - err: The error that occurred during the request.
//   - attempt: The current attempt number (1-based).
//
// Returns:
//   - bool: True if the request should be retried, false otherwise.
func (p *DefaultRetryPolicy) ShouldRetry(ctx context.Context, err error, attempt int) bool {
	// Don't retry if context is cancelled or deadline exceeded
	if ctx.Err() != nil {
		return false
	}

	// Don't retry if we've exceeded max attempts
	if attempt > p.config.MaxRetries {
		return false
	}

	// Check if this is a retryable error
	return isRetryableError(err)
}

// BackoffDuration calculates the backoff duration for a given attempt using exponential backoff with jitter.
// This helps distribute retry attempts and avoid thundering herd problems.
//
// Arguments:
//   - attempt: The attempt number (1-based).
//
// Returns:
//   - time.Duration: The duration to wait before the next attempt.
func (p *DefaultRetryPolicy) BackoffDuration(attempt int) time.Duration {
	if attempt <= 1 {
		return 0 // No backoff for first attempt
	}

	// Calculate exponential backoff: baseBackoff * multiplier^(attempt-2)
	backoff := float64(p.config.BaseBackoff) * math.Pow(p.config.BackoffMultiplier, float64(attempt-2))

	// Apply maximum backoff limit
	if backoff > float64(p.config.MaxBackoff) {
		backoff = float64(p.config.MaxBackoff)
	}

	// Add jitter to prevent thundering herd (±25% random variation)
	jitter := backoff * 0.25 * (rand.Float64()*2 - 1) // Random value between -0.25 and +0.25
	backoff += jitter

	// Ensure non-negative duration
	if backoff < 0 {
		backoff = float64(p.config.BaseBackoff)
	}

	return time.Duration(backoff)
}

// isRetryableError determines if an error should trigger a retry attempt.
// This function classifies errors based on their type and HTTP status codes.
//
// Arguments:
//   - err: The error to classify.
//
// Returns:
//   - bool: True if the error is retryable, false otherwise.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network-level errors are generally retryable
	if isNetworkError(err) {
		return true
	}

	// HTTP errors depend on status code
	if httpErr, ok := err.(*HTTPError); ok {
		return isRetryableHTTPStatus(httpErr.StatusCode)
	}

	// Rate limit errors are retryable
	if _, ok := err.(*RateLimitError); ok {
		return true
	}

	// Context errors are not retryable
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check error message for common retryable patterns
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"connection reset",
		"connection refused",
		"no such host",
		"timeout",
		"temporary failure",
		"server misbehaving",
		"i/o timeout",
		"network is unreachable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isNetworkError checks if an error is a network-level error.
// These errors are typically transient and worth retrying.
//
// Arguments:
//   - err: The error to check.
//
// Returns:
//   - bool: True if this is a network error, false otherwise.
func isNetworkError(err error) bool {
	errMsg := err.Error()

	// Common network error patterns
	networkPatterns := []string{
		"connection reset by peer",
		"connection refused",
		"no such host",
		"network is unreachable",
		"connection timeout",
		"dial tcp",
		"EOF",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isRetryableHTTPStatus determines if an HTTP status code should trigger a retry.
// This follows common best practices for HTTP client retry logic.
//
// Arguments:
//   - statusCode: The HTTP status code to check.
//
// Returns:
//   - bool: True if the status code is retryable, false otherwise.
func isRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests: // 429 - Rate limited
		return true
	case http.StatusInternalServerError: // 500 - Server error
		return true
	case http.StatusBadGateway: // 502 - Bad gateway
		return true
	case http.StatusServiceUnavailable: // 503 - Service unavailable
		return true
	case http.StatusGatewayTimeout: // 504 - Gateway timeout
		return true
	case http.StatusInsufficientStorage: // 507 - Insufficient storage
		return true
	default:
		// 4xx errors (except 429) are generally not retryable as they indicate client errors
		// 2xx and 3xx are successful and don't need retry
		return false
	}
}

// RetryableOperation represents an operation that can be retried.
// This is used by the retry mechanism to execute operations with retry logic.
type RetryableOperation[T any] func(ctx context.Context, attempt int) (T, error)

// ExecuteWithRetry executes an operation with retry logic according to the specified policy.
// This is the core retry mechanism used throughout the client.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - operation: The operation to execute with retry logic.
//   - policy: The retry policy to use for determining retry behavior.
//
// Returns:
//   - T: The result of the successful operation.
//   - error: The final error if all retry attempts failed.
//
// Example:
//
//	result, err := ExecuteWithRetry(ctx, func(ctx context.Context, attempt int) (*http.Response, error) {
//	    return http.Get("https://api.notion.com/v1/pages/123")
//	}, retryPolicy)
func ExecuteWithRetry[T any](ctx context.Context, operation RetryableOperation[T], policy RetryPolicy) (T, error) {
	var lastErr error
	var result T

	for attempt := 1; ; attempt++ {
		// Execute the operation
		result, err := operation(ctx, attempt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !policy.ShouldRetry(ctx, err, attempt) {
			break
		}

		// Calculate backoff duration
		backoffDuration := policy.BackoffDuration(attempt)

		// Handle rate limiting with specific delay
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			if rateLimitErr.RetryAfter > backoffDuration {
				backoffDuration = rateLimitErr.RetryAfter
			}
		}

		// Wait for backoff duration
		if backoffDuration > 0 {
			select {
			case <-time.After(backoffDuration):
				// Continue to next attempt
			case <-ctx.Done():
				return result, ctx.Err()
			}
		}
	}

	return result, lastErr
}

// RateLimitInfo extracts rate limit information from HTTP response headers.
// This is used to implement intelligent rate limiting and retry behavior.
type RateLimitInfo struct {
	// Limit is the maximum number of requests allowed in the time window.
	Limit int

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// Reset is the time when the rate limit window resets.
	Reset time.Time

	// RetryAfter is the duration to wait before making another request.
	RetryAfter time.Duration
}

// ExtractRateLimitInfo extracts rate limit information from HTTP response headers.
// This follows common HTTP rate limiting header conventions.
//
// Arguments:
//   - resp: The HTTP response to extract rate limit information from.
//
// Returns:
//   - *RateLimitInfo: Rate limit information, or nil if not available.
//
// Example:
//
//	if rateLimitInfo := ExtractRateLimitInfo(resp); rateLimitInfo != nil {
//	    fmt.Printf("Rate limit: %d/%d, resets at %v\n",
//	        rateLimitInfo.Remaining, rateLimitInfo.Limit, rateLimitInfo.Reset)
//	}
func ExtractRateLimitInfo(resp *http.Response) *RateLimitInfo {
	if resp == nil {
		return nil
	}

	info := &RateLimitInfo{}

	// Extract rate limit values from headers
	if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			info.Limit = limit
		}
	}

	if remainingStr := resp.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			info.Remaining = remaining
		}
	}

	if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			info.Reset = time.Unix(resetUnix, 0)
		}
	}

	if retryAfterStr := resp.Header.Get("Retry-After"); retryAfterStr != "" {
		if retryAfterSecs, err := strconv.Atoi(retryAfterStr); err == nil {
			info.RetryAfter = time.Duration(retryAfterSecs) * time.Second
		}
	}

	// If no rate limit headers found, return nil
	if info.Limit == 0 && info.Remaining == 0 && info.Reset.IsZero() && info.RetryAfter == 0 {
		return nil
	}

	return info
}

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures.
// When the error rate exceeds a threshold, the circuit opens and fails fast.
type CircuitBreaker struct {
	threshold       float64       // Error rate threshold (0.0-1.0)
	timeout         time.Duration // How long to keep circuit open
	requestCount    int           // Total requests in current window
	errorCount      int           // Error count in current window
	lastFailureTime time.Time     // Time of last failure
	state           CircuitState  // Current circuit state
	windowStart     time.Time     // Start of current measurement window
	mu              struct {
		// Embedded struct to provide fine-grained locking
		sync.RWMutex
		requests int
		errors   int
	}
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed means requests are allowed through normally.
	CircuitClosed CircuitState = iota

	// CircuitOpen means requests are failing fast to prevent cascading failures.
	CircuitOpen

	// CircuitHalfOpen means we're testing if the service has recovered.
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker with the specified configuration.
//
// Arguments:
//   - threshold: Error rate threshold (0.0-1.0) for opening the circuit.
//   - timeout: Duration to keep the circuit open before trying again.
//
// Returns:
//   - *CircuitBreaker: A new circuit breaker instance.
//
// Example:
//
//	// Open circuit when 50% of requests fail, stay open for 60 seconds
//	cb := NewCircuitBreaker(0.5, 60*time.Second)
func NewCircuitBreaker(threshold float64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:   threshold,
		timeout:     timeout,
		state:       CircuitClosed,
		windowStart: time.Now(),
	}
}

// CanExecute checks if a request can be executed based on the circuit breaker state.
//
// Returns:
//   - bool: True if the request can proceed, false if it should fail fast.
//   - error: Error if the circuit is open, nil otherwise.
func (cb *CircuitBreaker) CanExecute() (bool, error) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true, nil

	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Transition to half-open to test if service recovered
			cb.state = CircuitHalfOpen
			return true, nil
		}
		return false, &CircuitBreakerError{State: "open"}

	case CircuitHalfOpen:
		return true, nil

	default:
		return false, &CircuitBreakerError{State: "unknown"}
	}
}

// RecordSuccess records a successful request execution.
// This may transition the circuit from half-open to closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.requestCount++

	if cb.state == CircuitHalfOpen {
		// Success in half-open state transitions to closed
		cb.state = CircuitClosed
		cb.resetCounters()
	}
}

// RecordFailure records a failed request execution.
// This may open the circuit if the error rate exceeds the threshold.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.requestCount++
	cb.errorCount++
	cb.lastFailureTime = time.Now()

	// Check if we should open the circuit
	if cb.shouldOpenCircuit() {
		cb.state = CircuitOpen
	}
}

// shouldOpenCircuit determines if the circuit should be opened based on current metrics.
func (cb *CircuitBreaker) shouldOpenCircuit() bool {
	if cb.requestCount < 10 { // Minimum requests before considering circuit opening
		return false
	}

	errorRate := float64(cb.errorCount) / float64(cb.requestCount)
	return errorRate >= cb.threshold
}

// resetCounters resets the request and error counters.
func (cb *CircuitBreaker) resetCounters() {
	cb.requestCount = 0
	cb.errorCount = 0
	cb.windowStart = time.Now()
}

// CircuitBreakerError represents an error when the circuit breaker is open.
type CircuitBreakerError struct {
	State string
}

// Error returns the error message for CircuitBreakerError.
func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker is %s", e.State)
}
