package client

import (
	"testing"

	"github.com/notioncodes/types"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	apiKey string
	ids    struct {
		pages       []types.PageID
		database    types.DatabaseID
		block       types.BlockID
		syncedBlock types.BlockID
		property    types.PropertyID
		user        types.UserID
	}

	client *Client
}

func (c *ClientTestSuite) SetupTest() {
	// c.apiKey = os.Getenv("NOTION_API_KEY")
	c.apiKey = ""
	if c.apiKey == "" {
		c.T().Fatalf("NOTION_API_KEY environment variable not set")
	}

	c.ids = struct {
		pages       []types.PageID
		database    types.DatabaseID
		block       types.BlockID
		syncedBlock types.BlockID
		property    types.PropertyID
		user        types.UserID
	}{
		pages: func() []types.PageID {
			pages := []string{
				"22cd7342e57180d292f7c90f552b68f2",
				"22bd7342-e571-8064-b106-c67b700857f7",
			}
			ids := make([]types.PageID, len(pages))
			for i, page := range pages {
				id, err := types.ParsePageID(page)
				if err != nil {
					c.T().Fatalf("Expected no error parsing page ID, got: %v", err)
				}
				ids[i] = id
			}
			return ids
		}(),
		database: func() types.DatabaseID {
			id, err := types.ParseDatabaseID("22bd7342-e571-8009-80ca-dd739a33fe96")
			if err != nil {
				c.T().Fatalf("Expected no error parsing database ID, got: %v", err)
			}
			return id
		}(),
		block: func() types.BlockID {
			id, err := types.ParseBlockID("22cd7342-e571-8096-9127-c48e806cf469")
			if err != nil {
				c.T().Fatalf("Expected no error parsing block ID, got: %v", err)
			}
			return id
		}(),
		syncedBlock: func() types.BlockID {
			id, err := types.ParseBlockID("234d7342-e571-80c6-8a59-e869a1522417")
			if err != nil {
				c.T().Fatalf("Expected no error parsing synced block ID, got: %v", err)
			}
			return id
		}(),
		property: func() types.PropertyID {
			id, err := types.ParsePropertyID("234d7342-e571-8186-b047-fb21820f604b")
			if err != nil {
				c.T().Fatalf("Expected no error parsing property ID, got: %v", err)
			}
			return id
		}(),
		user: func() types.UserID {
			id, err := types.ParseUserID("16ad7342e57180c4a065c7a1015871d3")
			if err != nil {
				c.T().Fatalf("Expected no error parsing user ID, got: %v", err)
			}
			return id
		}(),
	}

	config := DefaultConfig()
	config.APIKey = c.apiKey

	var err error
	c.client, err = New(config)
	if err != nil {
		c.T().Fatalf("Expected no error creating client, got: %v", err)
	}

	// Test that registry is initialized and has core operators.
	if c.client.Registry == nil {
		c.T().Error("Expected registry to be initialized")
	}

	// Test that core operators are registered
	expectedOperators := []string{"page", "database", "block", "user", "search"}
	for _, opName := range expectedOperators {
		if !c.client.Registry.HasOperator(opName) {
			c.T().Errorf("Expected operator '%s' to be registered", opName)
		}
	}
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

// // TestDatabaseOperations_Query tests database querying with pagination.
// func TestDatabaseOperations_Query(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}

// 	client := createTestClient(t)
// 	defer client.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	databaseID := types.DatabaseID(getTestDatabaseID())
// 	if databaseID == "" {
// 		t.Skip("No test database ID provided")
// 	}

// 	results := client.Databases().Query(ctx, databaseID, nil, nil)

// 	var pageCount int
// 	for result := range results {
// 		if result.IsError() {
// 			t.Fatalf("Error querying database: %v", result.Error)
// 		}

// 		pageCount++

// 		// Verify page metadata
// 		if result.Metadata == nil {
// 			t.Error("Expected metadata to be present")
// 		}

// 		if result.Metadata.FromStream != true {
// 			t.Error("Expected FromStream to be true for paginated results")
// 		}

// 		if result.Metadata.PageInfo == nil {
// 			t.Error("Expected page info to be present")
// 		}

// 		// Break after a reasonable number of pages for testing
// 		if pageCount >= 10 {
// 			break
// 		}
// 	}

// 	if pageCount == 0 {
// 		t.Error("Expected at least one page from database query")
// 	}

// 	t.Logf("Retrieved %d pages from database", pageCount)
// }

// // TestSearchOperations_Query tests search functionality.
// func TestSearchOperations_Query(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}

// 	client := createTestClient(t)
// 	defer client.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	results := client.Search().Query(ctx, "test", nil, nil)

// 	var resultCount int
// 	for result := range results {
// 		if result.IsError() {
// 			t.Fatalf("Error searching: %v", result.Error)
// 		}

// 		resultCount++

// 		// Verify that we have either a page or database
// 		if result.Data.Page == nil && result.Data.Database == nil {
// 			t.Error("Expected either page or database in search result")
// 		}

// 		// Break after a reasonable number of results for testing
// 		if resultCount >= 5 {
// 			break
// 		}
// 	}

// 	t.Logf("Found %d search results", resultCount)
// }

// // TestClient_Metrics tests metrics collection.
// func TestClient_Metrics(t *testing.T) {
// 	config := DefaultConfig()
// 	config.APIKey = "test-api-key"
// 	config.EnableMetrics = true

// 	client, err := New(config)
// 	if err != nil {
// 		t.Fatalf("Expected no error creating client, got: %v", err)
// 	}
// 	defer client.Close()

// 	metrics := client.GetMetrics()
// 	if metrics == nil {
// 		t.Fatal("Expected metrics to be available when enabled")
// 	}

// 	// Initial metrics should be zero
// 	if metrics.TotalRequests != 0 {
// 		t.Errorf("Expected 0 initial requests, got %d", metrics.TotalRequests)
// 	}

// 	if metrics.TotalErrors != 0 {
// 		t.Errorf("Expected 0 initial errors, got %d", metrics.TotalErrors)
// 	}
// }

// // TestClient_ConcurrentSafety tests that the client is safe for concurrent use.
// func TestClient_ConcurrentSafety(t *testing.T) {
// 	client := createTestClient(t)
// 	defer client.Close()

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Run multiple goroutines that access client methods concurrently
// 	done := make(chan bool, 10)

// 	for i := 0; i < 10; i++ {
// 		go func() {
// 			defer func() { done <- true }()

// 			// Access different client methods
// 			client.Pages()
// 			client.Databases()
// 			client.Blocks()
// 			client.Users()
// 			client.Search()
// 			client.GetMetrics()
// 		}()
// 	}

// 	// Wait for all goroutines to complete
// 	for i := 0; i < 10; i++ {
// 		select {
// 		case <-done:
// 		case <-ctx.Done():
// 			t.Fatal("Timeout waiting for concurrent operations")
// 		}
// 	}
// }

// // createTestClient creates a client for integration tests.
// func createTestClient(t *testing.T) *Client {
// 	apiKey := os.Getenv("NOTION_API_KEY")
// 	if apiKey == "" {
// 		t.Skip("NOTION_API_KEY environment variable not set")
// 	}

// 	config := DefaultConfig()
// 	config.APIKey = apiKey
// 	config.EnableMetrics = true
// 	config.Concurrency = 5 // Lower concurrency for testing

// 	client, err := New(config)
// 	if err != nil {
// 		t.Fatalf("Failed to create test client: %v", err)
// 	}

// 	return client
// }

// // getTestPageID returns a test page ID from environment variables.
// func getTestPageID() string {
// 	return "22cd7342e57180d292f7c90f552b68f2"
// }

// // getTestPageIDs returns multiple test page IDs for concurrent testing.
// func getTestPageIDs() []types.PageID {
// 	pageID := getTestPageID()
// 	if pageID == "" {
// 		return nil
// 	}

// 	// For testing, we can use the same page ID multiple times
// 	// In a real test suite, you'd want different page IDs
// 	return []types.PageID{
// 		types.PageID(pageID),
// 		types.PageID(pageID),
// 		types.PageID(pageID),
// 	}
// }

// // getTestDatabaseID returns a test database ID from environment variables.
// func getTestDatabaseID() string {
// 	return os.Getenv("NOTION_TEST_DATABASE_ID")
// }

// // BenchmarkPageOperations_Get benchmarks single page retrieval.
// func BenchmarkPageOperations_Get(b *testing.B) {
// 	if testing.Short() {
// 		b.Skip("Skipping benchmark in short mode")
// 	}

// 	client := createBenchmarkClient(b)
// 	defer client.Close()

// 	pageID := types.PageID(getTestPageID())
// 	if pageID == "" {
// 		b.Skip("No test page ID provided")
// 	}

// 	ctx := context.Background()

// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			result := client.Pages().Get(ctx, pageID)
// 			if result.IsError() {
// 				b.Fatalf("Error in benchmark: %v", result.Error)
// 			}
// 		}
// 	})
// }

// // BenchmarkPageOperations_GetMany benchmarks concurrent page retrieval.
// func BenchmarkPageOperations_GetMany(b *testing.B) {
// 	if testing.Short() {
// 		b.Skip("Skipping benchmark in short mode")
// 	}

// 	client := createBenchmarkClient(b)
// 	defer client.Close()

// 	pageIDs := getTestPageIDs()
// 	if len(pageIDs) == 0 {
// 		b.Skip("No test page IDs provided")
// 	}

// 	ctx := context.Background()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		results := client.Pages().GetMany(ctx, pageIDs)

// 		// Consume all results
// 		for result := range results {
// 			if result.IsError() {
// 				b.Fatalf("Error in benchmark: %v", result.Error)
// 			}
// 		}
// 	}
// }

// // createBenchmarkClient creates a client optimized for benchmarking.
// func createBenchmarkClient(b *testing.B) *Client {
// 	apiKey := os.Getenv("NOTION_API_KEY")
// 	if apiKey == "" {
// 		b.Skip("NOTION_API_KEY environment variable not set")
// 	}

// 	config := DefaultConfig()
// 	config.APIKey = apiKey
// 	config.EnableMetrics = false // Disable metrics for cleaner benchmarks
// 	config.Concurrency = 20      // Higher concurrency for benchmarks

// 	client, err := New(config)
// 	if err != nil {
// 		b.Fatalf("Failed to create benchmark client: %v", err)
// 	}

// 	return client
// }
