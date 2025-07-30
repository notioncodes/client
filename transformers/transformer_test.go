package transformers

import (
	"context"
	"testing"

	"github.com/notioncodes/types"
)

// TestTransformerRegistry tests the basic registry functionality.
// This test ensures that transformers can be registered and retrieved correctly.
func TestTransformerRegistry(t *testing.T) {
	registry := NewTransformerRegistry()

	// Test registration.
	mockTransformer := &MockTransformer{
		name:           "test",
		version:        "1.0.0",
		supportedTypes: []types.ObjectType{types.ObjectTypePage},
		priority:       10,
	}

	err := registry.Register(mockTransformer)
	if err != nil {
		t.Fatalf("Failed to register transformer: %v", err)
	}

	// Test retrieval.
	transformers := registry.ListTransformers()
	if len(transformers) != 1 {
		t.Errorf("Expected 1 transformer, got %d", len(transformers))
	}

	if transformers[0] != "test" {
		t.Errorf("Expected transformer name 'test', got '%s'", transformers[0])
	}

	// Test loaded status.
	if !registry.IsLoaded("test") {
		t.Error("Expected transformer to be loaded")
	}

	if registry.IsLoaded("nonexistent") {
		t.Error("Expected nonexistent transformer to not be loaded")
	}
}

// TestTransformerRegistryLazyLoading tests lazy loading functionality.
// This test ensures that transformers are only instantiated when first accessed.
func TestTransformerRegistryLazyLoading(t *testing.T) {
	registry := NewTransformerRegistry()

	created := false
	registry.RegisterLazy("lazy_test", func() TransformerPlugin {
		created = true
		return &MockTransformer{
			name:           "lazy_test",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage},
			priority:       5,
		}
	})

	// Should not be created yet.
	if created {
		t.Error("Transformer should not be created until first access")
	}

	// Should not be loaded yet.
	if registry.IsLoaded("lazy_test") {
		t.Error("Transformer should not be loaded until first access")
	}

	// Access the transformer - should trigger creation.
	ctx := context.Background()
	results, err := registry.Transform(ctx, types.ObjectTypePage, &types.Page{})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should be created now.
	if !created {
		t.Error("Transformer should be created after first access")
	}

	// Should be loaded now.
	if !registry.IsLoaded("lazy_test") {
		t.Error("Transformer should be loaded after first access")
	}

	// Should have results.
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

// TestTransformerRegistryConfigLoading tests configuration-based loading.
// This test ensures that transformers can be loaded and configured from config.
func TestTransformerRegistryConfigLoading(t *testing.T) {
	registry := NewTransformerRegistry()

	// Register a lazy transformer.
	registry.RegisterLazy("config_test", func() TransformerPlugin {
		return &MockTransformer{
			name:           "config_test",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage, types.ObjectTypeDatabase},
			priority:       15,
		}
	})

	// Create configuration.
	configs := map[string]*TransformerConfig{
		"config_test": {
			Name:        "config_test",
			Enabled:     true,
			Priority:    20,               // Override priority
			ObjectTypes: []string{"page"}, // Limit to pages only
			Config: map[string]interface{}{
				"test_option": "test_value",
			},
		},
		"disabled_test": {
			Name:    "disabled_test",
			Enabled: false, // Should not be loaded
		},
	}

	// Load from config.
	err := registry.LoadFromConfig(configs)
	if err != nil {
		t.Fatalf("Failed to load from config: %v", err)
	}

	// Should be loaded.
	if !registry.IsLoaded("config_test") {
		t.Error("Transformer should be loaded from config")
	}

	// Disabled transformer should not be loaded.
	if registry.IsLoaded("disabled_test") {
		t.Error("Disabled transformer should not be loaded")
	}

	// Test transformation with configured types.
	ctx := context.Background()

	// Should work for pages (configured type).
	results, err := registry.Transform(ctx, types.ObjectTypePage, &types.Page{})
	if err != nil {
		t.Fatalf("Transform failed for page: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for page, got %d", len(results))
	}

	// Should not work for databases (not in configured types).
	results, err = registry.Transform(ctx, types.ObjectTypeDatabase, &types.Database{})
	if err != nil {
		t.Fatalf("Transform failed for database: %v", err)
	}
	if len(results) != 0 {
		t.Error("Expected no results for database (not configured)")
	}
}

// TestTransformerRegistryStats tests statistics collection.
// This test ensures that the registry properly tracks transformer statistics.
func TestTransformerRegistryStats(t *testing.T) {
	registry := NewTransformerRegistry()

	// Register transformers.
	registry.RegisterLazy("stats_test1", func() TransformerPlugin {
		return &MockTransformer{
			name:           "stats_test1",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage},
			priority:       10,
		}
	})

	registry.RegisterLazy("stats_test2", func() TransformerPlugin {
		return &MockTransformer{
			name:           "stats_test2",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage},
			priority:       5,
		}
	})

	// Check initial stats.
	stats := registry.GetStats()
	if stats.TotalTransformers != 2 {
		t.Errorf("Expected 2 total transformers, got %d", stats.TotalTransformers)
	}
	if stats.LoadedCount != 0 {
		t.Errorf("Expected 0 loaded transformers initially, got %d", stats.LoadedCount)
	}

	// Trigger loading by transforming.
	ctx := context.Background()
	_, err := registry.Transform(ctx, types.ObjectTypePage, &types.Page{})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Check updated stats.
	stats = registry.GetStats()
	if stats.LoadedCount != 2 {
		t.Errorf("Expected 2 loaded transformers after transform, got %d", stats.LoadedCount)
	}
	if stats.ExecutionCounts["stats_test1"] != 1 {
		t.Errorf("Expected 1 execution for stats_test1, got %d", stats.ExecutionCounts["stats_test1"])
	}
	if stats.ExecutionCounts["stats_test2"] != 1 {
		t.Errorf("Expected 1 execution for stats_test2, got %d", stats.ExecutionCounts["stats_test2"])
	}
}

// TestTransformerRegistryUnload tests transformer unloading.
// This test ensures that transformers can be properly unloaded and cleaned up.
func TestTransformerRegistryUnload(t *testing.T) {
	registry := NewTransformerRegistry()

	mockTransformer := &MockTransformer{
		name:           "unload_test",
		version:        "1.0.0",
		supportedTypes: []types.ObjectType{types.ObjectTypePage},
		priority:       10,
	}

	// Register and load transformer.
	err := registry.Register(mockTransformer)
	if err != nil {
		t.Fatalf("Failed to register transformer: %v", err)
	}

	// Should be loaded.
	if !registry.IsLoaded("unload_test") {
		t.Error("Transformer should be loaded")
	}

	// Unload transformer.
	err = registry.Unload("unload_test")
	if err != nil {
		t.Fatalf("Failed to unload transformer: %v", err)
	}

	// Should not be loaded anymore.
	if registry.IsLoaded("unload_test") {
		t.Error("Transformer should not be loaded after unloading")
	}

	// Cleanup should have been called.
	if !mockTransformer.cleanupCalled {
		t.Error("Cleanup should have been called during unloading")
	}

	// Should not be able to transform anymore.
	ctx := context.Background()
	results, err := registry.Transform(ctx, types.ObjectTypePage, &types.Page{})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results after unloading, got %d", len(results))
	}
}

// Helper functions and mock types for testing.

// MockTransformer is a mock transformer for testing.
type MockTransformer struct {
	name           string
	version        string
	supportedTypes []types.ObjectType
	priority       int
	initialized    bool
	cleanupCalled  bool
	config         map[string]interface{}
}

func (mt *MockTransformer) Name() string {
	return mt.name
}

func (mt *MockTransformer) Version() string {
	return mt.version
}

func (mt *MockTransformer) SupportedTypes() []types.ObjectType {
	return mt.supportedTypes
}

func (mt *MockTransformer) Priority() int {
	return mt.priority
}

func (mt *MockTransformer) Initialize(config map[string]interface{}) error {
	mt.initialized = true
	mt.config = config
	return nil
}

func (mt *MockTransformer) Cleanup() error {
	mt.cleanupCalled = true
	return nil
}

func (mt *MockTransformer) Transform(ctx context.Context, objectType types.ObjectType, object interface{}) (interface{}, error) {
	return map[string]interface{}{
		"transformer": mt.name,
		"object_type": string(objectType),
		"transformed": true,
	}, nil
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
