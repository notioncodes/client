package transformers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/notioncodes/types"
)

// TestConfigLoader tests the configuration loading functionality.
// This test ensures that configurations can be loaded from files correctly.
func TestConfigLoader(t *testing.T) {
	// Create a temporary directory for test files.
	tempDir, err := os.MkdirTemp("", "transformer_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	loader := NewConfigLoader([]string{tempDir})

	// Test loading default configuration.
	defaultConfigs := loader.LoadDefault()
	if len(defaultConfigs) == 0 {
		t.Error("Expected default configurations to be non-empty")
	}

	// Should contain markdown transformer.
	if _, exists := defaultConfigs["markdown"]; !exists {
		t.Error("Expected default configuration to contain markdown transformer")
	}

	// Should contain json transformer.
	if _, exists := defaultConfigs["json"]; !exists {
		t.Error("Expected default configuration to contain json transformer")
	}

	// Test saving and loading YAML configuration.
	yamlFile := filepath.Join(tempDir, "test_config.yaml")
	err = loader.SaveToFile(yamlFile, defaultConfigs)
	if err != nil {
		t.Fatalf("Failed to save YAML config: %v", err)
	}

	loadedYamlConfigs, err := loader.LoadFromFile("test_config.yaml")
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	if len(loadedYamlConfigs) != len(defaultConfigs) {
		t.Errorf("Expected %d configs, got %d", len(defaultConfigs), len(loadedYamlConfigs))
	}

	// Test saving and loading JSON configuration.
	jsonFile := filepath.Join(tempDir, "test_config.json")
	err = loader.SaveToFile(jsonFile, defaultConfigs)
	if err != nil {
		t.Fatalf("Failed to save JSON config: %v", err)
	}

	loadedJSONConfigs, err := loader.LoadFromFile("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	if len(loadedJSONConfigs) != len(defaultConfigs) {
		t.Errorf("Expected %d configs, got %d", len(defaultConfigs), len(loadedJSONConfigs))
	}
}

// TestConfigLoaderValidation tests configuration validation.
// This test ensures that invalid configurations are properly rejected.
func TestConfigLoaderValidation(t *testing.T) {
	loader := DefaultConfigLoader()

	// Test valid configuration.
	validConfig := &TransformerConfig{
		Name:        "test",
		Enabled:     true,
		Priority:    10,
		ObjectTypes: []string{"page", "database"},
		Config:      map[string]interface{}{},
	}

	err := loader.ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}

	// Test invalid configuration - empty name.
	invalidConfig := &TransformerConfig{
		Name:    "",
		Enabled: true,
	}

	err = loader.ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected empty name config to fail validation")
	}

	// Test invalid configuration - negative priority.
	invalidConfig = &TransformerConfig{
		Name:     "test",
		Enabled:  true,
		Priority: -1,
	}

	err = loader.ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected negative priority config to fail validation")
	}

	// Test invalid configuration - invalid object type.
	invalidConfig = &TransformerConfig{
		Name:        "test",
		Enabled:     true,
		ObjectTypes: []string{"invalid_type"},
	}

	err = loader.ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected invalid object type config to fail validation")
	}
}

// TestConfigLoaderExampleGeneration tests example configuration generation.
// This test ensures that example configurations can be generated correctly.
func TestConfigLoaderExampleGeneration(t *testing.T) {
	loader := DefaultConfigLoader()

	// Test YAML example generation.
	yamlExample, err := loader.GenerateExampleConfig("yaml")
	if err != nil {
		t.Fatalf("Failed to generate YAML example: %v", err)
	}

	if len(yamlExample) == 0 {
		t.Error("Expected non-empty YAML example")
	}

	// Should contain transformer names.
	yamlStr := string(yamlExample)
	if !contains(yamlStr, "markdown") {
		t.Error("Expected YAML example to contain 'markdown'")
	}

	if !contains(yamlStr, "json") {
		t.Error("Expected YAML example to contain 'json'")
	}

	// Test JSON example generation.
	jsonExample, err := loader.GenerateExampleConfig("json")
	if err != nil {
		t.Fatalf("Failed to generate JSON example: %v", err)
	}

	if len(jsonExample) == 0 {
		t.Error("Expected non-empty JSON example")
	}

	// Should contain transformer names.
	jsonStr := string(jsonExample)
	if !contains(jsonStr, "markdown") {
		t.Error("Expected JSON example to contain 'markdown'")
	}

	if !contains(jsonStr, "json") {
		t.Error("Expected JSON example to contain 'json'")
	}

	// Test unsupported format.
	_, err = loader.GenerateExampleConfig("xml")
	if err == nil {
		t.Error("Expected unsupported format to return error")
	}
}

// TestConfigLoaderSearchPaths tests configuration file search functionality.
// This test ensures that configuration files are found in the correct search paths.
func TestConfigLoaderSearchPaths(t *testing.T) {
	// Create temporary directories for testing search paths.
	tempDir1, err := os.MkdirTemp("", "config_search_1")
	if err != nil {
		t.Fatalf("Failed to create temp directory 1: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "config_search_2")
	if err != nil {
		t.Fatalf("Failed to create temp directory 2: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Create config file in second directory.
	configContent := `
markdown:
  name: markdown
  enabled: true
  priority: 10
  config:
    output_format: github
`
	configFile := filepath.Join(tempDir2, "search_test.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create loader with search paths (first directory doesn't have the file).
	loader := NewConfigLoader([]string{tempDir1, tempDir2})

	// Should find the file in the second directory.
	configs, err := loader.LoadFromFile("search_test.yaml")
	if err != nil {
		t.Fatalf("Failed to load config from search paths: %v", err)
	}

	if len(configs) == 0 {
		t.Error("Expected to load at least one configuration")
	}

	if _, exists := configs["markdown"]; !exists {
		t.Error("Expected to load markdown configuration")
	}
}

// TestIntegrationWithRegistry tests integration between config loader and registry.
// This test ensures that configurations can be loaded and applied to a registry.
func TestIntegrationWithRegistry(t *testing.T) {
	// Create a temporary directory for test files.
	tempDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test configuration file.
	configContent := `
markdown:
  name: markdown
  enabled: true
  priority: 15
  object_types: ["page"]
  config:
    output_format: github
    include_metadata: false

json:
  name: json
  enabled: false
  priority: 5

custom:
  name: custom
  enabled: true
  priority: 20
  object_types: ["database"]
  config:
    custom_option: test_value
`

	configFile := filepath.Join(tempDir, "integration_test.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load configuration.
	loader := NewConfigLoader([]string{tempDir})
	configs, err := loader.LoadFromFile("integration_test.yaml")
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create registry and register mock transformers.
	registry := NewTransformerRegistry()

	registry.RegisterLazy("markdown", func() TransformerPlugin {
		return &MockTransformer{
			name:           "markdown",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage, types.ObjectTypeDatabase},
			priority:       10,
		}
	})

	registry.RegisterLazy("json", func() TransformerPlugin {
		return &MockTransformer{
			name:           "json",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypePage},
			priority:       5,
		}
	})

	registry.RegisterLazy("custom", func() TransformerPlugin {
		return &MockTransformer{
			name:           "custom",
			version:        "1.0.0",
			supportedTypes: []types.ObjectType{types.ObjectTypeDatabase},
			priority:       20,
		}
	})

	// Load configurations into registry.
	err = registry.LoadFromConfig(configs)
	if err != nil {
		t.Fatalf("Failed to load configs into registry: %v", err)
	}

	// Test that enabled transformers are loaded.
	if !registry.IsLoaded("markdown") {
		t.Error("Expected markdown transformer to be loaded")
	}

	if !registry.IsLoaded("custom") {
		t.Error("Expected custom transformer to be loaded")
	}

	// Test that disabled transformers are not loaded.
	if registry.IsLoaded("json") {
		t.Error("Expected json transformer to not be loaded (disabled)")
	}

	// Test transformation with configured object types.
	ctx := context.Background()

	// Markdown should work for pages.
	results, err := registry.Transform(ctx, types.ObjectTypePage, &types.Page{})
	if err != nil {
		t.Fatalf("Failed to transform page: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for page, got %d", len(results))
	}

	// Custom should work for databases.
	results, err = registry.Transform(ctx, types.ObjectTypeDatabase, &types.Database{})
	if err != nil {
		t.Fatalf("Failed to transform database: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for database, got %d", len(results))
	}
}
