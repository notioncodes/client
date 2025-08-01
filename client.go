package client

import (
	"fmt"
	"time"
)

// Client is the main entry point for the Notion API client.
// It provides high-level operations for working with Notion resources
// using concurrent, streaming patterns with built-in retry logic and observability.
type Client struct {
	config   *Config
	Registry *Registry
}

// NewClient creates a new Notion API client with the specified configuration.
//
// Arguments:
//   - config: Configuration for the client behavior and performance characteristics.
//
// Returns:
//   - *Client: A new client instance ready for API operations.
//   - error: Configuration or initialization error.
func NewClient(config *Config) (*Client, error) {
	HttpClient, err := NewHTTPClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	registry := NewRegistry(config, HttpClient)

	return &Client{
		config:   config,
		Registry: registry,
	}, nil
}

// Pages returns the page operations namespace.
func (c *Client) Pages() *PageNamespace {
	return c.Registry.Pages()
}

// Blocks returns the block operations namespace.
func (c *Client) Blocks() *BlockNamespace {
	return c.Registry.Blocks()
}

// Users returns the user operations namespace.
func (c *Client) Users() *UserNamespace {
	return c.Registry.Users()
}

// Comments returns the comment operations namespace.
//
// Returns:
//   - *CommentNamespace: The comment operations namespace.
func (c *Client) Comments() *CommentNamespace {
	return c.Registry.Comments()
}

// GetMetrics returns current client metrics if metrics collection is enabled.
//
// Returns:
//   - *Metrics: Current metrics data, or nil if metrics are disabled.
//
// Example:
//
//	metrics := client.GetMetrics()
//	if metrics != nil {
//	    fmt.Printf("Success rate: %.2f%%\n", metrics.SuccessRate*100)
//	}
func (c *Client) GetMetrics() *Metrics {
	return c.Registry.httpClient.GetMetrics()
}

// GetThrottleStats returns throttling statistics for the client.
// This provides insight into how often requests are being delayed due to throttling.
//
// Returns:
//   - int64: Number of times requests were throttled.
//   - time.Duration: Total time spent waiting due to throttling.
//
// Example:
//
//	count, totalWait := client.GetThrottleStats()
//	fmt.Printf("Throttled %d requests for total of %v\n", count, totalWait)
func (c *Client) GetThrottleStats() (int64, time.Duration) {
	return c.Registry.httpClient.GetThrottleStats()
}

// Close gracefully closes the client and releases resources.
// This should be called when the client is no longer needed.
//
// Example:
//
//	client, err := New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
func (c *Client) Close() error {
	return c.Registry.httpClient.Close()
}
