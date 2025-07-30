package transformers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/notioncodes/types"
)

// JSONTransformer converts Notion objects to JSON format.
// It supports configurable output formats and field selection.
type JSONTransformer struct {
	config JSONConfig
}

// JSONConfig holds configuration for the JSON transformer.
type JSONConfig struct {
	// Pretty determines if the JSON should be pretty-printed.
	Pretty bool `json:"pretty" yaml:"pretty"`

	// IncludeMetadata determines if object metadata should be included.
	IncludeMetadata bool `json:"include_metadata" yaml:"include_metadata"`

	// IncludeProperties determines if properties should be included.
	IncludeProperties bool `json:"include_properties" yaml:"include_properties"`

	// CompactArrays determines if arrays should be compacted.
	CompactArrays bool `json:"compact_arrays" yaml:"compact_arrays"`

	// ExcludeFields is a list of fields to exclude from the output.
	ExcludeFields []string `json:"exclude_fields" yaml:"exclude_fields"`
}

// JSONResult represents the output from the JSON transformer.
type JSONResult struct {
	// Data is the JSON-serialized content.
	Data string `json:"data"`

	// Metadata contains additional information about the transformation.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ObjectType is the type of the original object.
	ObjectType types.ObjectType `json:"object_type"`
}

// Name returns the transformer name.
func (jt *JSONTransformer) Name() string {
	return "json"
}

// Version returns the transformer version.
func (jt *JSONTransformer) Version() string {
	return "1.0.0"
}

// SupportedTypes returns the object types this transformer can handle.
func (jt *JSONTransformer) SupportedTypes() []types.ObjectType {
	return []types.ObjectType{
		types.ObjectTypePage,
		types.ObjectTypeDatabase,
		types.ObjectTypeBlock,
		types.ObjectTypeUser,
	}
}

// Priority returns the execution priority.
func (jt *JSONTransformer) Priority() int {
	return 5 // Lower priority than markdown
}

// Initialize sets up the transformer with the provided configuration.
func (jt *JSONTransformer) Initialize(config map[string]interface{}) error {
	// Set defaults.
	jt.config = JSONConfig{
		Pretty:            true,
		IncludeMetadata:   true,
		IncludeProperties: true,
		CompactArrays:     false,
		ExcludeFields:     []string{},
	}

	// Apply configuration overrides.
	if pretty, ok := config["pretty"].(bool); ok {
		jt.config.Pretty = pretty
	}

	if includeMetadata, ok := config["include_metadata"].(bool); ok {
		jt.config.IncludeMetadata = includeMetadata
	}

	if includeProperties, ok := config["include_properties"].(bool); ok {
		jt.config.IncludeProperties = includeProperties
	}

	if compactArrays, ok := config["compact_arrays"].(bool); ok {
		jt.config.CompactArrays = compactArrays
	}

	if excludeFields, ok := config["exclude_fields"].([]interface{}); ok {
		jt.config.ExcludeFields = make([]string, len(excludeFields))
		for i, field := range excludeFields {
			if fieldStr, ok := field.(string); ok {
				jt.config.ExcludeFields[i] = fieldStr
			}
		}
	}

	return nil
}

// Cleanup releases resources used by the transformer.
func (jt *JSONTransformer) Cleanup() error {
	// No cleanup needed for this transformer.
	return nil
}

// Transform converts a Notion object to JSON format.
func (jt *JSONTransformer) Transform(ctx context.Context, objectType types.ObjectType, object interface{}) (interface{}, error) {
	// Create a filtered version of the object based on configuration.
	filteredObject := jt.filterObject(object)

	// Serialize to JSON.
	var jsonData []byte
	var err error

	if jt.config.Pretty {
		jsonData, err = json.MarshalIndent(filteredObject, "", "  ")
	} else {
		jsonData, err = json.Marshal(filteredObject)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to serialize object to JSON: %w", err)
	}

	result := &JSONResult{
		Data:       string(jsonData),
		ObjectType: objectType,
		Metadata:   make(map[string]interface{}),
	}

	// Add metadata based on object type.
	switch objectType {
	case types.ObjectTypePage:
		if page, ok := object.(*types.Page); ok {
			result.Metadata["page_id"] = page.ID.String()
			result.Metadata["created_time"] = page.CreatedTime
			result.Metadata["last_edited_time"] = page.LastEditedTime
		}
	case types.ObjectTypeDatabase:
		if database, ok := object.(*types.Database); ok {
			result.Metadata["database_id"] = database.ID.String()
			result.Metadata["created_time"] = database.CreatedTime
			result.Metadata["last_edited_time"] = database.LastEditedTime
		}
	case types.ObjectTypeBlock:
		if block, ok := object.(*types.Block); ok {
			result.Metadata["block_id"] = block.ID.String()
			result.Metadata["block_type"] = string(block.Type)
			result.Metadata["created_time"] = block.CreatedTime
			result.Metadata["last_edited_time"] = block.LastEditedTime
		}
	case types.ObjectTypeUser:
		if user, ok := object.(*types.User); ok {
			result.Metadata["user_id"] = user.ID.String()
			result.Metadata["user_type"] = string(user.Type)
		}
	}

	return result, nil
}

// filterObject applies the configuration filters to the object.
func (jt *JSONTransformer) filterObject(object interface{}) interface{} {
	if len(jt.config.ExcludeFields) == 0 && jt.config.IncludeMetadata && jt.config.IncludeProperties {
		return object // No filtering needed.
	}

	// Convert to map for manipulation.
	objectMap := make(map[string]interface{})
	jsonData, err := json.Marshal(object)
	if err != nil {
		return object // Return original if can't marshal.
	}

	if err := json.Unmarshal(jsonData, &objectMap); err != nil {
		return object // Return original if can't unmarshal.
	}

	// Apply exclusion filters.
	for _, excludeField := range jt.config.ExcludeFields {
		delete(objectMap, excludeField)
	}

	// Filter metadata if configured.
	if !jt.config.IncludeMetadata {
		delete(objectMap, "created_time")
		delete(objectMap, "created_by")
		delete(objectMap, "last_edited_time")
		delete(objectMap, "last_edited_by")
		delete(objectMap, "url")
	}

	// Filter properties if configured.
	if !jt.config.IncludeProperties {
		delete(objectMap, "properties")
	}

	return objectMap
}

// Register the JSON transformer with the global registry.
func init() {
	RegisterGlobal("json", func() TransformerPlugin {
		return &JSONTransformer{}
	})
}