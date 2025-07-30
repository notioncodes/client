// package client provides a high-performance, concurrent HTTP client for the Notion API.
// This package implements goroutine-based concurrency patterns with native pagination support,
// connection pooling, intelligent retry logic, and comprehensive observability.
//
// The client is designed for production use with features like:
//   - Concurrent request processing with configurable worker pools
//   - Native pagination with automatic cursor management
//   - Exponential backoff retry with circuit breaker patterns
//   - Real-time metrics and observability
//   - Context-based cancellation and timeouts
//   - Connection pooling and keep-alive optimization
//
// Example:
//
//	config := &Config{
//	    APIKey: "your-integration-token",
//	    BaseURL: "https://api.notion.com/v1",
//	    Timeout: 30 * time.Second,
//	    MaxRetries: 3,
//	    Concurrency: 10,
//	}
//
//	client := New(config)
//	defer client.Close()
//
//	// Stream paginated results
//	ctx := context.Background()
//	results := client.Stream(ctx, &SearchRequest{Query: "project"})
//	for result := range results {
//	    if result.Error != nil {
//	        log.Printf("Error: %v", result.Error)
//	        break
//	    }
//	    // Process result.Data
//	}
package client

import (
	"net/http"
	"time"
)

// Config contains all configuration options for the Notion API client.
// This configuration enables fine-tuning of performance, reliability, and observability
// characteristics for different deployment environments.
type Config struct {
	// APIKey is the Notion integration token for authentication.
	// This is required and should be kept secure.
	APIKey string

	// BaseURL is the base URL for the Notion API.
	// Defaults to "https://api.notion.com/v1" if not specified.
	BaseURL string

	// Version is the Notion API version to use.
	// Defaults to "2022-06-28" if not specified.
	Version string

	// Timeout is the HTTP request timeout duration.
	// Defaults to 30 seconds if not specified.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests.
	// Defaults to 3 if not specified.
	MaxRetries int

	// BaseBackoff is the initial backoff duration for retries.
	// Defaults to 1 second if not specified.
	BaseBackoff time.Duration

	// MaxBackoff is the maximum backoff duration for retries.
	// Defaults to 30 seconds if not specified.
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	// Defaults to 2.0 if not specified.
	BackoffMultiplier float64

	// Concurrency is the maximum number of concurrent requests allowed.
	// This controls the size of the worker pool for concurrent operations.
	// Defaults to 10 if not specified.
	Concurrency int

	// MaxIdleConns is the maximum number of idle HTTP connections to maintain.
	// Defaults to 100 if not specified.
	MaxIdleConns int

	// MaxIdleConnsPerHost is the maximum idle connections per host.
	// Defaults to 10 if not specified.
	MaxIdleConnsPerHost int

	// IdleConnTimeout is the maximum time an idle connection can remain open.
	// Defaults to 90 seconds if not specified.
	IdleConnTimeout time.Duration

	// PageSize is the default page size for paginated requests.
	// Defaults to 100 if not specified (Notion's maximum).
	PageSize int

	// RateLimitBuffer is the buffer time to add to rate limit delays.
	// This helps avoid hitting rate limits due to clock skew.
	// Defaults to 100 milliseconds if not specified.
	RateLimitBuffer time.Duration

	// EnableMetrics enables collection of request metrics and observability.
	// When enabled, the client will track request counts, durations, errors, etc.
	// Defaults to false.
	EnableMetrics bool

	// UserAgent is the User-Agent header to send with requests.
	// Defaults to "notion-go-client/1.0" if not specified.
	UserAgent string

	// CustomHeaders are additional headers to send with all requests.
	// These can be used for custom authentication, tracking, etc.
	CustomHeaders map[string]string

	// CircuitBreakerThreshold is the error rate threshold for circuit breaking.
	// When error rate exceeds this (0.0-1.0), the circuit breaker opens.
	// Defaults to 0.5 (50% error rate) if not specified.
	CircuitBreakerThreshold float64

	// CircuitBreakerTimeout is how long to keep the circuit breaker open.
	// Defaults to 60 seconds if not specified.
	CircuitBreakerTimeout time.Duration

	// BufferSize is the size of internal channels used for streaming.
	// Larger buffers can improve throughput but use more memory.
	// Defaults to 100 if not specified.
	BufferSize int

	// RequestDelay is the minimum delay between consecutive HTTP requests.
	// This provides throttling to prevent overwhelming the Notion API.
	// Set to 0 to disable throttling. Defaults to 0 (no throttling).
	RequestDelay time.Duration
}

// DefaultConfig returns a configuration with sensible defaults for production use.
// Applications should customize the APIKey and optionally other fields as needed.
//
// Returns:
//   - *Config: A configuration with production-ready default values.
//
// Example:
//
//	config := DefaultConfig()
//	config.APIKey = "your-integration-token"
//	config.Concurrency = 20  // Increase for high-throughput scenarios
//	client := New(config)
func DefaultConfig() *Config {
	return &Config{
		BaseURL:                 "https://api.notion.com/v1",
		Version:                 "2022-06-28",
		Timeout:                 30 * time.Second,
		MaxRetries:              3,
		BaseBackoff:             1 * time.Second,
		MaxBackoff:              30 * time.Second,
		BackoffMultiplier:       2.0,
		Concurrency:             10,
		MaxIdleConns:            100,
		MaxIdleConnsPerHost:     10,
		IdleConnTimeout:         90 * time.Second,
		PageSize:                100,
		RateLimitBuffer:         100 * time.Millisecond,
		EnableMetrics:           false,
		UserAgent:               "notion-go-client/1.0",
		CustomHeaders:           make(map[string]string),
		CircuitBreakerThreshold: 0.5,
		CircuitBreakerTimeout:   60 * time.Second,
		BufferSize:              100,
		RequestDelay:            0, // No throttling by default
	}
}

// Validate ensures the configuration has valid values and sets defaults where needed.
// This method should be called before creating a client to ensure proper initialization.
//
// Returns:
//   - error: Validation error if required fields are missing or invalid, nil if valid.
//
// Example:
//
//	config := &Config{APIKey: os.Getenv("NOTION_API_KEY")}
//	if err := config.Validate(); err != nil {
//	    log.Fatalf("Invalid config: %v", err)
//	}
//	client := New(config)
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return &ConfigError{Field: "APIKey", Message: "API key is required"}
	}

	// Set defaults for empty values
	if c.BaseURL == "" {
		c.BaseURL = "https://api.notion.com/v1"
	}

	if c.Version == "" {
		c.Version = "2022-06-28"
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRetries < 0 {
		return &ConfigError{Field: "MaxRetries", Message: "max retries cannot be negative"}
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.BaseBackoff == 0 {
		c.BaseBackoff = 1 * time.Second
	}

	if c.MaxBackoff == 0 {
		c.MaxBackoff = 30 * time.Second
	}

	if c.BackoffMultiplier <= 1.0 {
		c.BackoffMultiplier = 2.0
	}

	if c.Concurrency <= 0 {
		c.Concurrency = 10
	}

	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 100
	}

	if c.MaxIdleConnsPerHost <= 0 {
		c.MaxIdleConnsPerHost = 10
	}

	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = 90 * time.Second
	}

	if c.PageSize <= 0 || c.PageSize > 100 {
		c.PageSize = 100 // Notion's maximum page size
	}

	if c.RateLimitBuffer == 0 {
		c.RateLimitBuffer = 100 * time.Millisecond
	}

	if c.UserAgent == "" {
		c.UserAgent = "notion-go-client/1.0"
	}

	if c.CustomHeaders == nil {
		c.CustomHeaders = make(map[string]string)
	}

	if c.CircuitBreakerThreshold <= 0 || c.CircuitBreakerThreshold > 1.0 {
		c.CircuitBreakerThreshold = 0.5
	}

	if c.CircuitBreakerTimeout == 0 {
		c.CircuitBreakerTimeout = 60 * time.Second
	}

	if c.BufferSize <= 0 {
		c.BufferSize = 100
	}

	// RequestDelay can be 0 (no throttling) or positive
	if c.RequestDelay < 0 {
		return &ConfigError{Field: "RequestDelay", Message: "request delay cannot be negative"}
	}

	return nil
}

// CreateHTTPClient creates an optimized HTTP client based on the configuration.
// This client includes connection pooling, timeouts, and other performance optimizations.
//
// Returns:
//   - *http.Client: Configured HTTP client optimized for the Notion API.
//
// Example:
//
//	config := DefaultConfig()
//	httpClient := config.CreateHTTPClient()
func (c *Config) CreateHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   c.Timeout,
	}
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

// Error returns the error message for ConfigError.
func (e *ConfigError) Error() string {
	return "config error in field '" + e.Field + "': " + e.Message
}

// RetryConfig contains configuration for retry behavior.
// This is used internally by the client for request retry logic.
type RetryConfig struct {
	MaxRetries        int
	BaseBackoff       time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RateLimitBuffer   time.Duration
}

// GetRetryConfig extracts retry configuration from the main config.
// This is used internally by retry mechanisms.
//
// Returns:
//   - RetryConfig: Retry configuration extracted from the main config.
func (c *Config) GetRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        c.MaxRetries,
		BaseBackoff:       c.BaseBackoff,
		MaxBackoff:        c.MaxBackoff,
		BackoffMultiplier: c.BackoffMultiplier,
		RateLimitBuffer:   c.RateLimitBuffer,
	}
}
