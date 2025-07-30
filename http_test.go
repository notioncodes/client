package client

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"
// )

// // TestHTTPClient_New tests HTTP client creation.
// func TestHTTPClient_New(t *testing.T) {
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Expected no error creating HTTP client, got: %v", err)
// 	}

// 	if client == nil {
// 		t.Fatal("Expected HTTP client to be created, got nil")
// 	}

// 	client.Close()
// }

// // TestHTTPClient_GetJSON tests JSON GET requests.
// func TestHTTPClient_GetJSON(t *testing.T) {
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != "GET" {
// 			t.Errorf("Expected GET request, got %s", r.Method)
// 		}

// 		// Check headers
// 		if r.Header.Get("Authorization") != "Bearer test-api-key" {
// 			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
// 		}

// 		if r.Header.Get("Notion-Version") == "" {
// 			t.Error("Expected Notion-Version header")
// 		}

// 		// Return test data
// 		response := map[string]interface{}{
// 			"id":         "test-id",
// 			"object":     "page",
// 			"properties": map[string]interface{}{},
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make request
// 	ctx := context.Background()
// 	var result map[string]interface{}

// 	err = client.GetJSON(ctx, "/test", nil, &result)
// 	if err != nil {
// 		t.Fatalf("Expected no error, got: %v", err)
// 	}

// 	// Verify response
// 	if result["id"] != "test-id" {
// 		t.Errorf("Expected id 'test-id', got %v", result["id"])
// 	}
// }

// // TestHTTPClient_PostJSON tests JSON POST requests.
// func TestHTTPClient_PostJSON(t *testing.T) {
// 	expectedBody := map[string]interface{}{
// 		"properties": map[string]interface{}{
// 			"title": "Test Page",
// 		},
// 	}

// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != "POST" {
// 			t.Errorf("Expected POST request, got %s", r.Method)
// 		}

// 		// Check Content-Type
// 		if r.Header.Get("Content-Type") != "application/json" {
// 			t.Errorf("Expected application/json Content-Type, got %s", r.Header.Get("Content-Type"))
// 		}

// 		// Parse and verify body
// 		var body map[string]interface{}
// 		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
// 			t.Fatalf("Failed to parse request body: %v", err)
// 		}

// 		properties, ok := body["properties"].(map[string]interface{})
// 		if !ok {
// 			t.Error("Expected properties in request body")
// 		}

// 		if properties["title"] != "Test Page" {
// 			t.Errorf("Expected title 'Test Page', got %v", properties["title"])
// 		}

// 		// Return success response
// 		response := map[string]interface{}{
// 			"id":     "created-id",
// 			"object": "page",
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusCreated)
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make request
// 	ctx := context.Background()
// 	var result map[string]interface{}

// 	err = client.PostJSON(ctx, "/test", expectedBody, &result)
// 	if err != nil {
// 		t.Fatalf("Expected no error, got: %v", err)
// 	}

// 	// Verify response
// 	if result["id"] != "created-id" {
// 		t.Errorf("Expected id 'created-id', got %v", result["id"])
// 	}
// }

// // TestHTTPClient_ErrorHandling tests error response handling.
// func TestHTTPClient_ErrorHandling(t *testing.T) {
// 	// Create a test server that returns errors
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		errorResponse := map[string]interface{}{
// 			"object":  "error",
// 			"status":  400,
// 			"code":    "validation_error",
// 			"message": "Invalid request",
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusBadRequest)
// 		json.NewEncoder(w).Encode(errorResponse)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make request that should fail
// 	ctx := context.Background()
// 	var result map[string]interface{}

// 	err = client.GetJSON(ctx, "/test", nil, &result)
// 	if err == nil {
// 		t.Fatal("Expected error, got nil")
// 	}

// 	// Check that it's an HTTPError
// 	httpErr, ok := err.(*HTTPError)
// 	if !ok {
// 		t.Fatalf("Expected HTTPError, got %T", err)
// 	}

// 	if httpErr.StatusCode != 400 {
// 		t.Errorf("Expected status code 400, got %d", httpErr.StatusCode)
// 	}

// 	if httpErr.Code != "validation_error" {
// 		t.Errorf("Expected code 'validation_error', got %s", httpErr.Code)
// 	}
// }

// // TestHTTPClient_RateLimitHandling tests rate limit response handling.
// func TestHTTPClient_RateLimitHandling(t *testing.T) {
// 	// Create a test server that returns rate limit error
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("X-RateLimit-Limit", "1000")
// 		w.Header().Set("X-RateLimit-Remaining", "0")
// 		w.Header().Set("Retry-After", "60")

// 		errorResponse := map[string]interface{}{
// 			"object":  "error",
// 			"status":  429,
// 			"code":    "rate_limited",
// 			"message": "Rate limited",
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusTooManyRequests)
// 		json.NewEncoder(w).Encode(errorResponse)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make request that should be rate limited
// 	ctx := context.Background()
// 	var result map[string]interface{}

// 	err = client.GetJSON(ctx, "/test", nil, &result)
// 	if err == nil {
// 		t.Fatal("Expected error, got nil")
// 	}

// 	// Check that it's a RateLimitError
// 	rateLimitErr, ok := err.(*RateLimitError)
// 	if !ok {
// 		t.Fatalf("Expected RateLimitError, got %T", err)
// 	}

// 	if rateLimitErr.RetryAfter != 60*time.Second {
// 		t.Errorf("Expected retry after 60s, got %v", rateLimitErr.RetryAfter)
// 	}
// }

// // TestHTTPClient_Timeout tests request timeout handling.
// func TestHTTPClient_Timeout(t *testing.T) {
// 	// Create a test server that hangs
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		time.Sleep(2 * time.Second) // Longer than our timeout
// 		w.WriteHeader(http.StatusOK)
// 	}))
// 	defer server.Close()

// 	// Create client with short timeout
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL
// 	config.Timeout = 100 * time.Millisecond // Very short timeout

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make request that should timeout
// 	ctx := context.Background()
// 	var result map[string]interface{}

// 	start := time.Now()
// 	err = client.GetJSON(ctx, "/test", nil, &result)
// 	duration := time.Since(start)

// 	if err == nil {
// 		t.Fatal("Expected timeout error, got nil")
// 	}

// 	// Should have timed out quickly
// 	if duration > 1*time.Second {
// 		t.Errorf("Request took too long: %v", duration)
// 	}

// 	// Check that it's a timeout error
// 	if _, ok := err.(*TimeoutError); !ok {
// 		t.Logf("Expected TimeoutError, got %T: %v", err, err)
// 		// Context timeout is also acceptable
// 		if ctx.Err() == nil {
// 			t.Errorf("Expected timeout-related error, got %T", err)
// 		}
// 	}
// }

// // TestHTTPClient_Pagination tests paginated requests.
// func TestHTTPClient_Pagination(t *testing.T) {
// 	page := 0

// 	// Create a test server that returns paginated data
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		page++

// 		var response map[string]interface{}

// 		if page == 1 {
// 			// First page
// 			response = map[string]interface{}{
// 				"object": "list",
// 				"results": []map[string]interface{}{
// 					{"id": "item-1"},
// 					{"id": "item-2"},
// 				},
// 				"has_more":    true,
// 				"next_cursor": "cursor-2",
// 			}
// 		} else {
// 			// Second page
// 			response = map[string]interface{}{
// 				"object": "list",
// 				"results": []map[string]interface{}{
// 					{"id": "item-3"},
// 					{"id": "item-4"},
// 				},
// 				"has_more":    false,
// 				"next_cursor": nil,
// 			}
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Test paginated request
// 	ctx := context.Background()
// 	pageSize := 2
// 	req := &PaginationRequest{
// 		PageSize: &pageSize,
// 	}

// 	results := GetPaginated[map[string]interface{}](client, ctx, "/test", req)

// 	var items []map[string]interface{}
// 	for result := range results {
// 		if result.IsError() {
// 			t.Fatalf("Expected no error, got: %v", result.Error)
// 		}
// 		items = append(items, result.Data)
// 	}

// 	// Should have received 4 items total
// 	if len(items) != 4 {
// 		t.Errorf("Expected 4 items, got %d", len(items))
// 	}

// 	// Verify item IDs
// 	expectedIDs := []string{"item-1", "item-2", "item-3", "item-4"}
// 	for i, item := range items {
// 		if item["id"] != expectedIDs[i] {
// 			t.Errorf("Expected item %d to have id %s, got %v", i, expectedIDs[i], item["id"])
// 		}
// 	}
// }

// // TestHTTPClient_ConcurrentRequests tests concurrent request handling.
// func TestHTTPClient_ConcurrentRequests(t *testing.T) {
// 	requestCount := 0

// 	// Create a test server that counts requests
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		requestCount++

// 		response := map[string]interface{}{
// 			"id": fmt.Sprintf("item-%d", requestCount),
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL
// 	config.Concurrency = 5

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	// Make concurrent requests
// 	ctx := context.Background()
// 	numRequests := 10
// 	results := make(chan error, numRequests)

// 	for i := 0; i < numRequests; i++ {
// 		go func() {
// 			var result map[string]interface{}
// 			err := client.GetJSON(ctx, "/test", nil, &result)
// 			results <- err
// 		}()
// 	}

// 	// Wait for all requests to complete
// 	for i := 0; i < numRequests; i++ {
// 		select {
// 		case err := <-results:
// 			if err != nil {
// 				t.Errorf("Request %d failed: %v", i, err)
// 			}
// 		case <-time.After(5 * time.Second):
// 			t.Fatal("Timeout waiting for concurrent requests")
// 		}
// 	}

// 	// All requests should have been processed
// 	if requestCount != numRequests {
// 		t.Errorf("Expected %d requests, got %d", numRequests, requestCount)
// 	}
// }

// // BenchmarkHTTPClient_GetJSON benchmarks JSON GET requests.
// func BenchmarkHTTPClient_GetJSON(b *testing.B) {
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		response := map[string]interface{}{
// 			"id":     "test-id",
// 			"object": "page",
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL
// 	config.EnableMetrics = false // Disable metrics for cleaner benchmarks

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		b.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	ctx := context.Background()

// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			var result map[string]interface{}
// 			err := client.GetJSON(ctx, "/test", nil, &result)
// 			if err != nil {
// 				b.Fatalf("Request failed: %v", err)
// 			}
// 		}
// 	})
// }

// // BenchmarkHTTPClient_PostJSON benchmarks JSON POST requests.
// func BenchmarkHTTPClient_PostJSON(b *testing.B) {
// 	// Create a test server
// 	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		response := map[string]interface{}{
// 			"id":     "created-id",
// 			"object": "page",
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(response)
// 	}))
// 	defer server.Close()

// 	// Create client with test server URL
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.BaseURL = server.URL
// 	config.EnableMetrics = false // Disable metrics for cleaner benchmarks

// 	client, err := NewHTTPClient(config)
// 	if err != nil {
// 		b.Fatalf("Failed to create HTTP client: %v", err)
// 	}
// 	defer client.Close()

// 	ctx := context.Background()
// 	requestBody := map[string]interface{}{
// 		"properties": map[string]interface{}{
// 			"title": "Test Page",
// 		},
// 	}

// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			var result map[string]interface{}
// 			err := client.PostJSON(ctx, "/test", requestBody, &result)
// 			if err != nil {
// 				b.Fatalf("Request failed: %v", err)
// 			}
// 		}
// 	})
// }
