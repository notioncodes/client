// Package transformers provides a system for loading and configuring transformers.
// It supports YAML and JSON configuration formats and provides a way to validate
// and generate example configurations.
package transformers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigLoader handles loading transformer configurations from files.
type ConfigLoader struct {
	searchPaths []string
}

// NewConfigLoader creates a new configuration loader.
//
// Arguments:
// - searchPaths: List of directories to search for configuration files.
//
// Returns:
// - *ConfigLoader: A new configuration loader instance.
//
// Example:
//
//	loader := NewConfigLoader([]string{".", "/etc/transformers", "~/.config/transformers"})
func NewConfigLoader(searchPaths []string) *ConfigLoader {
	return &ConfigLoader{
		searchPaths: searchPaths,
	}
}

// LoadFromFile loads transformer configurations from a file.
//
// Arguments:
// - filename: Path to the configuration file.
//
// Returns:
// - map[string]*TransformerConfig: Map of transformer configurations.
// - error: Error if the file could not be loaded or parsed.
//
// Example:
//
//	loader := NewConfigLoader([]string{"."})
//	configs, err := loader.LoadFromFile("transformers.yaml")
//	if err != nil {
//	    log.Printf("Failed to load config: %v", err)
//	    return
//	}
func (cl *ConfigLoader) LoadFromFile(filename string) (map[string]*TransformerConfig, error) {
	// Try to find the file in search paths.
	var filePath string
	var err error

	if filepath.IsAbs(filename) {
		filePath = filename
	} else {
		filePath, err = cl.findConfigFile(filename)
		if err != nil {
			return nil, fmt.Errorf("config file not found: %s", filename)
		}
	}

	// Open the file.
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", filePath, err)
	}
	defer file.Close()

	// Read the file content.
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	// Determine file format based on extension.
	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		return cl.parseYAML(content)
	case ".json":
		return cl.parseJSON(content)
	default:
		// Try YAML first, then JSON.
		if configs, err := cl.parseYAML(content); err == nil {
			return configs, nil
		}
		return cl.parseJSON(content)
	}
}

// LoadDefault loads the default transformer configuration.
//
// Returns:
// - map[string]*TransformerConfig: Default transformer configurations.
//
// Example:
//
//	loader := NewConfigLoader([]string{"."})
//	configs := loader.LoadDefault()
func (cl *ConfigLoader) LoadDefault() map[string]*TransformerConfig {
	return map[string]*TransformerConfig{
		"markdown": {
			Name:        "markdown",
			Enabled:     true,
			Priority:    10,
			ObjectTypes: []string{"page", "database", "block"},
			Config: map[string]interface{}{
				"output_format":      "github",
				"include_metadata":   true,
				"include_properties": true,
				"link_format":        "markdown",
			},
		},
		"json": {
			Name:        "json",
			Enabled:     true,
			Priority:    5,
			ObjectTypes: []string{"page", "database", "block", "user"},
			Config: map[string]interface{}{
				"pretty":             true,
				"include_metadata":   true,
				"include_properties": true,
				"compact_arrays":     false,
			},
		},
	}
}

// SaveToFile saves transformer configurations to a file.
//
// Arguments:
// - filename: Path to save the configuration file.
// - configs: Map of transformer configurations to save.
//
// Returns:
// - error: Error if the file could not be saved.
//
// Example:
//
//	loader := NewConfigLoader([]string{"."})
//	configs := loader.LoadDefault()
//	err := loader.SaveToFile("transformers.yaml", configs)
func (cl *ConfigLoader) SaveToFile(filename string, configs map[string]*TransformerConfig) error {
	// Determine format based on extension.
	ext := filepath.Ext(filename)
	var content []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		content, err = cl.marshalYAML(configs)
	case ".json":
		content, err = cl.marshalJSON(configs)
	default:
		// Default to YAML.
		content, err = cl.marshalYAML(configs)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist.
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the file.
	if err := os.WriteFile(filename, content, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}

	return nil
}

// findConfigFile searches for a configuration file in the search paths.
func (cl *ConfigLoader) findConfigFile(filename string) (string, error) {
	for _, searchPath := range cl.searchPaths {
		fullPath := filepath.Join(searchPath, filename)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("file not found in search paths")
}

// parseYAML parses YAML configuration content.
func (cl *ConfigLoader) parseYAML(content []byte) (map[string]*TransformerConfig, error) {
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(content, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return cl.convertToTransformerConfigs(configMap)
}

// parseJSON parses JSON configuration content.
func (cl *ConfigLoader) parseJSON(content []byte) (map[string]*TransformerConfig, error) {
	var configMap map[string]interface{}
	if err := json.Unmarshal(content, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return cl.convertToTransformerConfigs(configMap)
}

// marshalYAML marshals configurations to YAML format.
func (cl *ConfigLoader) marshalYAML(configs map[string]*TransformerConfig) ([]byte, error) {
	return yaml.Marshal(configs)
}

// marshalJSON marshals configurations to JSON format.
func (cl *ConfigLoader) marshalJSON(configs map[string]*TransformerConfig) ([]byte, error) {
	return json.MarshalIndent(configs, "", "  ")
}

// convertToTransformerConfigs converts a generic map to TransformerConfig structs.
func (cl *ConfigLoader) convertToTransformerConfigs(configMap map[string]interface{}) (map[string]*TransformerConfig, error) {
	configs := make(map[string]*TransformerConfig)

	for name, configData := range configMap {
		configDataMap, ok := configData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid config format for transformer '%s'", name)
		}

		config := &TransformerConfig{
			Name: name,
		}

		// Parse enabled flag.
		if enabled, ok := configDataMap["enabled"].(bool); ok {
			config.Enabled = enabled
		} else {
			config.Enabled = true // Default to enabled.
		}

		// Parse priority.
		if priority, ok := configDataMap["priority"].(int); ok {
			config.Priority = priority
		} else if priorityFloat, ok := configDataMap["priority"].(float64); ok {
			config.Priority = int(priorityFloat)
		}

		// Parse object types.
		if objectTypes, ok := configDataMap["object_types"].([]interface{}); ok {
			config.ObjectTypes = make([]string, len(objectTypes))
			for i, objType := range objectTypes {
				if objTypeStr, ok := objType.(string); ok {
					config.ObjectTypes[i] = objTypeStr
				}
			}
		}

		// Parse transformer-specific config.
		if transformerConfig, ok := configDataMap["config"].(map[string]interface{}); ok {
			config.Config = transformerConfig
		} else {
			config.Config = make(map[string]interface{})
		}

		configs[name] = config
	}

	return configs, nil
}

// GenerateExampleConfig generates an example configuration file.
//
// Arguments:
// - format: Output format ("yaml" or "json").
//
// Returns:
// - []byte: The example configuration content.
// - error: Error if generation failed.
//
// Example:
//
//	loader := NewConfigLoader([]string{"."})
//	example, err := loader.GenerateExampleConfig("yaml")
//	if err != nil {
//	    log.Printf("Failed to generate example: %v", err)
//	    return
//	}
//	fmt.Println(string(example))
func (cl *ConfigLoader) GenerateExampleConfig(format string) ([]byte, error) {
	exampleConfig := map[string]*TransformerConfig{
		"markdown": {
			Name:        "markdown",
			Enabled:     true,
			Priority:    10,
			ObjectTypes: []string{"page", "database", "block"},
			Config: map[string]interface{}{
				"output_format":       "github",
				"include_metadata":    true,
				"include_properties":  true,
				"code_block_language": "",
				"link_format":         "markdown",
			},
		},
		"json": {
			Name:        "json",
			Enabled:     true,
			Priority:    5,
			ObjectTypes: []string{"page", "database", "block", "user"},
			Config: map[string]interface{}{
				"pretty":             true,
				"include_metadata":   true,
				"include_properties": true,
				"compact_arrays":     false,
				"exclude_fields":     []string{},
			},
		},
		"custom_transformer": {
			Name:        "custom_transformer",
			Enabled:     false,
			Priority:    1,
			ObjectTypes: []string{"page"},
			Config: map[string]interface{}{
				"custom_option1": "value1",
				"custom_option2": true,
				"custom_option3": 42,
			},
		},
	}

	switch format {
	case "yaml", "yml":
		return cl.marshalYAML(exampleConfig)
	case "json":
		return cl.marshalJSON(exampleConfig)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ValidateConfig validates a transformer configuration.
//
// Arguments:
// - config: The configuration to validate.
//
// Returns:
// - error: Validation error if the configuration is invalid.
//
// Example:
//
//	loader := NewConfigLoader([]string{"."})
//	config := &TransformerConfig{Name: "test", Enabled: true}
//	if err := loader.ValidateConfig(config); err != nil {
//	    log.Printf("Invalid config: %v", err)
//	}
func (cl *ConfigLoader) ValidateConfig(config *TransformerConfig) error {
	if config.Name == "" {
		return fmt.Errorf("transformer name cannot be empty")
	}

	if config.Priority < 0 {
		return fmt.Errorf("transformer priority cannot be negative")
	}

	// Validate object types.
	validObjectTypes := []string{"page", "database", "block", "user"}
	for _, objType := range config.ObjectTypes {
		valid := false
		for _, validType := range validObjectTypes {
			if objType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid object type: %s", objType)
		}
	}

	return nil
}

// DefaultConfigLoader returns a config loader with default search paths.
//
// Returns:
// - *ConfigLoader: A config loader with common search paths.
func DefaultConfigLoader() *ConfigLoader {
	searchPaths := []string{
		".", // Current directory
		"./config",
		"./transformers",
		"/etc/notion-transformers",
	}

	// Add home directory path if available.
	if homeDir, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(homeDir, ".config", "notion-transformers"))
	}

	return NewConfigLoader(searchPaths)
}
