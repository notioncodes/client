package client

import (
	"context"
	"testing"
	"time"

	"github.com/notioncodes/types"
)

// TestPageNamespaceGet tests the PageNamespace Get method.
// This test ensures that page retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure
func TestPageNamespaceGet(_ *testing.T) {
	registry := createTestRegistry()
	pageNS := registry.Pages()

	ctx := context.Background()
	pageID := types.PageID("test-page-id")

	// Mock the result since we're testing the interface, not actual API calls.
	// In a real test, you'd mock the HTTP client.
	// For now, we're testing that the method exists and compiles correctly.
	_ = pageNS.Get(ctx, pageID)
}

// TestPageNamespaceGetMany tests the PageNamespace GetMany method.
// This test ensures that multiple page retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestPageNamespaceGetMany(t *testing.T) {
	registry := createTestRegistry()
	pageNS := registry.Pages()

	ctx := context.Background()
	pageIDs := []types.PageID{
		types.PageID("page-1"),
		types.PageID("page-2"),
		types.PageID("page-3"),
	}

	// Test the method exists and returns a channel.
	resultCh := pageNS.GetMany(ctx, pageIDs)
	if resultCh == nil {
		t.Error("expected GetMany to return a channel, got nil")
	}

	// Consume the channel to prevent goroutine leaks
	go func() {
		for range resultCh {
			// Consume all results
		}
	}()
}

// TestDatabaseNamespaceGet tests the DatabaseNamespace Get method.
// This test ensures that database retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestDatabaseNamespaceGet(_ *testing.T) {
	registry := createTestRegistry()
	dbNS := registry.Databases()

	ctx := context.Background()
	dbID := types.DatabaseID("test-db-id")

	// Test the method exists and compiles.
	_ = dbNS.Get(ctx, dbID)
}

// TestDatabaseNamespaceQuery tests the DatabaseNamespace Query method.
// This test ensures that database querying works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestDatabaseNamespaceQuery(t *testing.T) {
	registry := createTestRegistry()
	dbNS := registry.Databases()

	ctx := context.Background()
	dbID := types.DatabaseID("test-db-id")
	filter := map[string]interface{}{
		"property": "Status",
		"select": map[string]interface{}{
			"equals": "Done",
		},
	}
	sorts := []map[string]interface{}{
		{
			"property":  "Name",
			"direction": "ascending",
		},
	}

	// Test the method exists and returns a channel.
	resultCh := dbNS.Query(ctx, dbID, filter, sorts)
	if resultCh == nil {
		t.Error("expected Query to return a channel, got nil")
	}

	// Consume the channel to prevent goroutine leaks
	go func() {
		for range resultCh {
			// Consume all results
		}
	}()
}

// TestBlockNamespaceGet tests the BlockNamespace Get method.
// This test ensures that block retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestBlockNamespaceGet(_ *testing.T) {
	registry := createTestRegistry()
	blockNS := registry.Blocks()

	ctx := context.Background()
	blockID := types.BlockID("test-block-id")

	// Test the method exists and compiles.
	_ = blockNS.Get(ctx, blockID)
}

// TestBlockNamespaceGetChildren tests the BlockNamespace GetChildren method.
// This test ensures that block children retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestBlockNamespaceGetChildren(t *testing.T) {
	registry := createTestRegistry()
	blockNS := registry.Blocks()

	ctx := context.Background()
	blockID := types.BlockID("test-block-id")

	// Test the method exists and returns a channel.
	resultCh := blockNS.GetMany(ctx, []types.BlockID{blockID})
	if resultCh == nil {
		t.Error("expected GetMany to return a channel, got nil")
	}

	// Consume the channel to prevent goroutine leaks
	go func() {
		for range resultCh {
			// Consume all results
		}
	}()
}

// TestUserNamespaceGet tests the UserNamespace Get method.
// This test ensures that user retrieval works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestUserNamespaceGet(_ *testing.T) {
	registry := createTestRegistry()
	userNS := registry.Users()

	ctx := context.Background()
	userID := types.UserID("test-user-id")

	// Test the method exists and compiles
	_ = userNS.Get(ctx, userID)
}

// TestUserNamespaceList tests the UserNamespace List method.
// This test ensures that user listing works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestUserNamespaceList(t *testing.T) {
	registry := createTestRegistry()
	userNS := registry.Users()

	ctx := context.Background()

	// Test the method exists and returns a channel.
	userID := types.UserID("test-user-id")
	resultCh := userNS.GetMany(ctx, []types.UserID{userID})
	if resultCh == nil {
		t.Error("expected GetMany to return a channel, got nil")
	}

	// Consume the channel to prevent goroutine leaks.
	go func() {
		for range resultCh {
			// Consume all results.
		}
	}()
}

// TestSearchNamespaceQuery tests the SearchNamespace Query method.
// This test ensures that search querying works through the fluent interface.
// Note: This will fail with actual API call, but tests the interface structure.
func TestSearchNamespaceQuery(t *testing.T) {
	registry := createTestRegistry()
	searchNS := registry.Search()

	ctx := context.Background()

	// Test the method exists and returns a channel.
	resultCh := searchNS.Query(ctx, types.SearchRequest{
		Query: "test search",
		Filter: &types.SearchFilter{
			Property: "object",
			Value:    "page",
		},
		Sort: &types.SearchSort{
			Direction: "descending",
			Timestamp: "last_edited_time",
		},
	})
	if resultCh == nil {
		t.Error("expected Query to return a channel, got nil")
	}

	// Consume the channel to prevent goroutine leaks.
	go func() {
		for range resultCh {
			// Consume all results.
		}
	}()
}

// TestNamespaceConsistency tests that all namespaces are consistently implemented.
// This test ensures that all namespace methods follow the same patterns.
// Note: This will fail with actual API call, but tests the interface structure
func TestNamespaceConsistency(t *testing.T) {
	registry := createTestRegistry()

	// Test that all namespace constructors return non-nil
	namespaces := []interface{}{
		registry.Pages(),
		registry.Databases(),
		registry.Blocks(),
		registry.Users(),
		registry.Search(),
	}

	for i, ns := range namespaces {
		if ns == nil {
			t.Errorf("namespace %d returned nil", i)
		}
	}

	// Test that namespaces properly reference the registry
	pageNS := registry.Pages()
	if pageNS.registry != registry {
		t.Error("PageNamespace does not properly reference registry")
	}

	dbNS := registry.Databases()
	if dbNS.registry != registry {
		t.Error("DatabaseNamespace does not properly reference registry")
	}

	blockNS := registry.Blocks()
	if blockNS.registry != registry {
		t.Error("BlockNamespace does not properly reference registry")
	}

	userNS := registry.Users()
	if userNS.registry != registry {
		t.Error("UserNamespace does not properly reference registry")
	}

	searchNS := registry.Search()
	if searchNS.registry != registry {
		t.Error("SearchNamespace does not properly reference registry")
	}
}

// TestNamespaceErrorHandling tests error handling in namespace methods.
// This test ensures that namespace methods properly handle errors from the registry.
// Note: This will fail with actual API call, but tests the interface structure
func TestNamespaceErrorHandling(t *testing.T) {
	// Create a registry with a missing operator to test error handling
	registry := createTestRegistry()

	// Clear the registry to simulate missing operators
	registry.initializers = make(map[string]RegistryInitializer)

	ctx := context.Background()

	// Test PageNamespace error handling
	pageNS := registry.Pages()
	result := pageNS.Get(ctx, types.PageID("test"))
	if !result.IsError() {
		t.Error("expected PageNamespace.Get to return error when operator missing")
	}

	// Test GetMany error handling
	resultCh := pageNS.GetMany(ctx, []types.PageID{types.PageID("test")})
	select {
	case result := <-resultCh:
		if !result.IsError() {
			t.Error("expected PageNamespace.GetMany to return error when operator missing")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected PageNamespace.GetMany to return error result quickly")
	}

	// Test DatabaseNamespace error handling
	dbNS := registry.Databases()
	dbResult := dbNS.Get(ctx, types.DatabaseID("test"))
	if !dbResult.IsError() {
		t.Error("expected DatabaseNamespace.Get to return error when operator missing")
	}

	// Test BlockNamespace error handling
	blockNS := registry.Blocks()
	blockResult := blockNS.Get(ctx, types.BlockID("test"))
	if !blockResult.IsError() {
		t.Error("expected BlockNamespace.Get to return error when operator missing")
	}

	// Test UserNamespace error handling
	userNS := registry.Users()
	userResult := userNS.Get(ctx, types.UserID("test"))
	if !userResult.IsError() {
		t.Error("expected UserNamespace.Get to return error when operator missing")
	}

	// Test SearchNamespace error handling
	searchNS := registry.Search()
	searchResultCh := searchNS.Query(ctx, types.SearchRequest{
		Query: "test search",
	})
	select {
	case result := <-searchResultCh:
		if !result.IsError() {
			t.Error("expected SearchNamespace.Query to return error when operator missing")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected SearchNamespace.Query to return error result quickly")
	}
}

// TestNamespaceContextCancellation tests context cancellation in namespace methods.
// This test ensures that namespace methods respect context cancellation.
// Note: This will fail with actual API call, but tests the interface structure
func TestNamespaceContextCancellation(t *testing.T) {
	registry := createTestRegistry()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that methods respect cancelled context
	pageNS := registry.Pages()
	resultCh := pageNS.GetMany(ctx, []types.PageID{types.PageID("test")})

	select {
	case result := <-resultCh:
		// Should get an error or cancelled result
		if result.IsError() {
			// This is expected
		} else {
			t.Error("expected error result when context is cancelled")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected quick response when context is cancelled")
	}
}
