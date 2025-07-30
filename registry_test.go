package client

import (
	"sync"
	"testing"
	"time"

	"github.com/notioncodes/types"
)

// TestNewRegistry tests the creation of a new registry.
// This test ensures that the registry is properly initialized with core operators.
func TestNewRegistry(t *testing.T) {
	config := &Config{
		APIKey:      "test-key",
		Concurrency: 5,
		PageSize:    50,
		Timeout:     10 * time.Second,
	}
	httpClient := &HTTPClient{} // Mock client

	registry := NewRegistry(config, httpClient)

	if registry == nil {
		t.Fatal("expected registry to be created, got nil")
	}

	if registry.config != config {
		t.Error("expected registry config to match input config")
	}

	if registry.httpClient != httpClient {
		t.Error("expected registry httpClient to match input httpClient")
	}

	// Test that core operators are registered
	expectedOperators := []string{
		"page", "database", "block", "user",
		"page_paginated", "database_paginated", "block_paginated", "user_paginated",
		"search",
	}

	for _, opName := range expectedOperators {
		if !registry.HasOperator(opName) {
			t.Errorf("expected operator '%s' to be registered", opName)
		}
	}
}

// TestRegistryLazyInstantiation tests that operators are lazily instantiated and cached.
// This test ensures that instances are created only when first accessed and then cached.
func TestRegistryLazyInstantiation(t *testing.T) {
	registry := createTestRegistry()

	// Initially, no instances should be cached
	stats := registry.Stats()
	if stats.CachedCount != 0 {
		t.Errorf("expected 0 cached instances initially, got %d", stats.CachedCount)
	}

	// Get an operator - should trigger instantiation
	pageOp, err := registry.Get("page")
	if err != nil {
		t.Fatalf("failed to get page operator: %v", err)
	}

	if pageOp == nil {
		t.Fatal("expected page operator to be created, got nil")
	}

	// Should now have 1 cached instance
	stats = registry.Stats()
	if stats.CachedCount != 1 {
		t.Errorf("expected 1 cached instance after first access, got %d", stats.CachedCount)
	}

	// Get the same operator again - should return cached instance
	pageOp2, err := registry.Get("page")
	if err != nil {
		t.Fatalf("failed to get page operator second time: %v", err)
	}

	if pageOp != pageOp2 {
		t.Error("expected same instance to be returned from cache")
	}

	// Should still have 1 cached instance
	stats = registry.Stats()
	if stats.CachedCount != 1 {
		t.Errorf("expected 1 cached instance after second access, got %d", stats.CachedCount)
	}
}

// TestRegistryTypedAccess tests type-safe operator retrieval.
// This test ensures that GetTyped properly type-checks operator instances.
func TestRegistryTypedAccess(t *testing.T) {
	registry := createTestRegistry()

	// Test successful typed access
	pageOp, err := GetTyped[*Operator[types.Page]](registry, "page")
	if err != nil {
		t.Fatalf("failed to get typed page operator: %v", err)
	}

	if pageOp == nil {
		t.Fatal("expected typed page operator to be created, got nil")
	}

	// Test type assertion failure
	_, err = GetTyped[*PaginatedOperator[types.Page]](registry, "page")
	if err == nil {
		t.Error("expected type assertion error, got nil")
	}
}

// TestRegistryConcurrentAccess tests thread-safe concurrent access to the registry.
// This test ensures that the registry handles concurrent access properly with double-check locking.
func TestRegistryConcurrentAccess(t *testing.T) {
	registry := createTestRegistry()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	results := make(chan interface{}, numGoroutines*numOperations)

	// Launch multiple goroutines that access the same operator concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				op, err := registry.Get("page")
				if err != nil {
					t.Errorf("failed to get operator in goroutine: %v", err)
					continue
				}
				results <- op
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify all results are the same instance (cached)
	var firstInstance interface{}
	count := 0
	for result := range results {
		count++
		if firstInstance == nil {
			firstInstance = result
		} else if result != firstInstance {
			t.Error("expected all concurrent accesses to return same cached instance")
		}
	}

	expectedCount := numGoroutines * numOperations
	if count != expectedCount {
		t.Errorf("expected %d results, got %d", expectedCount, count)
	}

	// Should have exactly 1 cached instance despite concurrent access
	stats := registry.Stats()
	if stats.CachedCount != 1 {
		t.Errorf("expected 1 cached instance after concurrent access, got %d", stats.CachedCount)
	}
}

// TestRegistryNamespaces tests the fluent interface namespaces.
// This test ensures that namespace methods work correctly and provide type-safe access.
func TestRegistryNamespaces(t *testing.T) {
	registry := createTestRegistry()

	// Test Pages namespace
	pageNS := registry.Pages()
	if pageNS == nil {
		t.Fatal("expected Pages namespace to be created, got nil")
	}
	if pageNS.registry != registry {
		t.Error("expected Pages namespace to reference the registry")
	}

	// Test Databases namespace
	dbNS := registry.Databases()
	if dbNS == nil {
		t.Fatal("expected Databases namespace to be created, got nil")
	}
	if dbNS.registry != registry {
		t.Error("expected Databases namespace to reference the registry")
	}

	// Test Blocks namespace
	blockNS := registry.Blocks()
	if blockNS == nil {
		t.Fatal("expected Blocks namespace to be created, got nil")
	}
	if blockNS.registry != registry {
		t.Error("expected Blocks namespace to reference the registry")
	}

	// Test Users namespace
	userNS := registry.Users()
	if userNS == nil {
		t.Fatal("expected Users namespace to be created, got nil")
	}
	if userNS.registry != registry {
		t.Error("expected Users namespace to reference the registry")
	}

	// Test Search namespace
	searchNS := registry.Search()
	if searchNS == nil {
		t.Fatal("expected Search namespace to be created, got nil")
	}
	if searchNS.registry != registry {
		t.Error("expected Search namespace to reference the registry")
	}
}

// TestRegistryOperatorManagement tests operator registration and management.
// This test ensures that operators can be registered, cleared, and managed properly.
func TestRegistryOperatorManagement(t *testing.T) {
	registry := createTestRegistry()

	// Test custom operator registration
	registry.Register("custom", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.Page](httpClient, config)
	})

	if !registry.HasOperator("custom") {
		t.Error("expected custom operator to be registered")
	}

	// Test listing registered operators
	registered := registry.ListRegistered()
	found := false
	for _, name := range registered {
		if name == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom' to be in registered operators list")
	}

	// Get the custom operator to cache it
	_, err := registry.Get("custom")
	if err != nil {
		t.Fatalf("failed to get custom operator: %v", err)
	}

	// Test listing cached operators
	cached := registry.ListCached()
	found = false
	for _, name := range cached {
		if name == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom' to be in cached operators list")
	}

	// Test clearing specific operator
	registry.ClearOperator("custom")
	cached = registry.ListCached()
	for _, name := range cached {
		if name == "custom" {
			t.Error("expected 'custom' to be removed from cache")
		}
	}

	// Test clearing all operators
	registry.Get("page") // Cache another operator
	registry.Clear()
	stats := registry.Stats()
	if stats.CachedCount != 0 {
		t.Errorf("expected 0 cached instances after Clear(), got %d", stats.CachedCount)
	}
}

// TestRegistryValidation tests registry configuration validation.
// This test ensures that the registry properly validates its configuration.
func TestRegistryValidation(t *testing.T) {
	// Test valid configuration
	registry := createTestRegistry()
	err := registry.ValidateConfiguration()
	if err != nil {
		t.Errorf("expected valid configuration to pass validation, got: %v", err)
	}

	// Test nil config
	registry.config = nil
	err = registry.ValidateConfiguration()
	if err == nil {
		t.Error("expected nil config to fail validation")
	}

	// Test nil httpClient
	registry.config = &Config{}
	registry.httpClient = nil
	err = registry.ValidateConfiguration()
	if err == nil {
		t.Error("expected nil httpClient to fail validation")
	}
}

// Helper functions for testing

// createTestRegistry creates a registry for testing.
func createTestRegistry() *Registry {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.Concurrency = 5
	config.PageSize = 50
	config.Timeout = 10 * time.Second

	httpClient, err := NewHTTPClient(config)
	if err != nil {
		panic(err) // In tests, this is acceptable
	}
	return NewRegistry(config, httpClient)
}

// TestPlugin is a mock plugin for testing.
type TestPlugin struct {
	name    string
	version string
}

// RegisterWith registers the test plugin with a registry.
func (tp *TestPlugin) RegisterWith(registry *Registry) error {
	registry.Register("test-custom", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.Page](httpClient, config)
	})
	return nil
}

// Name returns the plugin name.
func (tp *TestPlugin) Name() string {
	return tp.name
}

// Version returns the plugin version.
func (tp *TestPlugin) Version() string {
	return tp.version
}
