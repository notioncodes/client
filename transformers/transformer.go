package transformers

import (
	"context"
	"fmt"
	"sync"

	"github.com/notioncodes/types"
)

// TransformerPlugin defines the interface for all transformer plugins.
// Transformers process Notion objects and can convert them to different formats,
// extract data, or perform other transformations based on object type.
type TransformerPlugin interface {
	// Name returns the unique name of the transformer.
	Name() string

	// Version returns the version of the transformer.
	Version() string

	// SupportedTypes returns the object types this transformer can handle.
	SupportedTypes() []types.ObjectType

	// Priority returns the execution priority (higher numbers execute first).
	// This allows transformers to be chained in a specific order.
	Priority() int

	// Transform processes an object and returns the transformed result.
	// The context can be used for cancellation and timeouts.
	Transform(ctx context.Context, objectType types.ObjectType, object interface{}) (interface{}, error)

	// Initialize is called when the transformer is first loaded.
	// Use this for any setup or validation required by the transformer.
	Initialize(config map[string]interface{}) error

	// Cleanup is called when the transformer is being unloaded.
	// Use this to release resources or perform cleanup.
	Cleanup() error
}

// TransformerConfig holds configuration for transformer plugins.
type TransformerConfig struct {
	// Name of the transformer.
	Name string `json:"name" yaml:"name"`

	// Enabled determines if this transformer should be loaded.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Priority for execution order (higher numbers execute first).
	Priority int `json:"priority" yaml:"priority"`

	// ObjectTypes specifies which object types this transformer should handle.
	// If empty, it will handle all types supported by the transformer.
	ObjectTypes []string `json:"object_types" yaml:"object_types"`

	// Config contains transformer-specific configuration.
	Config map[string]interface{} `json:"config" yaml:"config"`
}

// TransformerRegistry manages transformer plugins with lazy loading and type-based routing.
// It provides a centralized way to register, configure, and execute transformers
// based on object types and priorities.
type TransformerRegistry struct {
	mu sync.RWMutex

	// Registered transformers by name.
	transformers map[string]TransformerPlugin

	// Transformer configurations.
	configs map[string]*TransformerConfig

	// Object type to transformer mappings for efficient routing.
	typeRoutes map[types.ObjectType][]string

	// Lazy-loaded transformers (name -> initializer function).
	lazyTransformers map[string]TransformerInitializer

	// Execution statistics.
	stats TransformerStats
}

// TransformerInitializer is a function that creates a transformer instance.
type TransformerInitializer func() TransformerPlugin

// TransformerStats provides statistics about transformer usage.
type TransformerStats struct {
	TotalTransformers int            `json:"total_transformers"`
	LoadedCount       int            `json:"loaded_count"`
	ExecutionCounts   map[string]int `json:"execution_counts"`
}

// NewTransformerRegistry creates a new transformer registry.
//
// Returns:
// - *TransformerRegistry: A new registry instance.
//
// Example:
//
//	registry := NewTransformerRegistry()
//	registry.RegisterLazy("markdown", func() TransformerPlugin {
//	    return &MarkdownTransformer{}
//	})
func NewTransformerRegistry() *TransformerRegistry {
	return &TransformerRegistry{
		transformers:     make(map[string]TransformerPlugin),
		configs:          make(map[string]*TransformerConfig),
		typeRoutes:       make(map[types.ObjectType][]string),
		lazyTransformers: make(map[string]TransformerInitializer),
		stats: TransformerStats{
			ExecutionCounts: make(map[string]int),
		},
	}
}

// RegisterLazy registers a transformer with lazy initialization.
//
// Arguments:
// - name: Unique name for the transformer.
// - initializer: Function to create the transformer instance.
//
// Example:
//
//	registry.RegisterLazy("markdown", func() TransformerPlugin {
//	    return &MarkdownTransformer{}
//	})
func (tr *TransformerRegistry) RegisterLazy(name string, initializer TransformerInitializer) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.lazyTransformers[name] = initializer
	tr.stats.TotalTransformers = len(tr.lazyTransformers)
}

// Register registers an already-instantiated transformer.
//
// Arguments:
// - transformer: The transformer instance to register.
//
// Example:
//
//	transformer := &MarkdownTransformer{}
//	registry.Register(transformer)
func (tr *TransformerRegistry) Register(transformer TransformerPlugin) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	name := transformer.Name()
	tr.transformers[name] = transformer
	tr.stats.LoadedCount = len(tr.transformers)

	// Update type routing.
	tr.updateTypeRouting(name, transformer.SupportedTypes())

	return nil
}

// Configure sets the configuration for a transformer.
//
// Arguments:
// - name: Name of the transformer to configure.
// - config: Configuration for the transformer.
//
// Example:
//
//	config := &TransformerConfig{
//	    Name: "markdown",
//	    Enabled: true,
//	    Priority: 10,
//	    ObjectTypes: []types.ObjectType{types.ObjectTypePage},
//	    Config: map[string]interface{}{
//	        "output_format": "github_flavored",
//	    },
//	}
//	registry.Configure("markdown", config)
func (tr *TransformerRegistry) Configure(name string, config *TransformerConfig) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.configs[name] = config
}

// LoadFromConfig loads and configures transformers from a configuration map.
//
// Arguments:
// - configs: Map of transformer configurations.
//
// Returns:
// - error: Error if any transformer failed to load or configure.
//
// Example:
//
//	configs := map[string]*TransformerConfig{
//	    "markdown": {
//	        Name: "markdown",
//	        Enabled: true,
//	        Priority: 10,
//	        Config: map[string]interface{}{"format": "github"},
//	    },
//	}
//	err := registry.LoadFromConfig(configs)
func (tr *TransformerRegistry) LoadFromConfig(configs map[string]*TransformerConfig) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for name, config := range configs {
		if !config.Enabled {
			continue
		}

		tr.configs[name] = config

		// Load transformer if it's lazily registered.
		if initializer, exists := tr.lazyTransformers[name]; exists {
			transformer := initializer()

			// Initialize with config.
			if err := transformer.Initialize(config.Config); err != nil {
				return fmt.Errorf("failed to initialize transformer '%s': %w", name, err)
			}

			tr.transformers[name] = transformer
			tr.stats.LoadedCount = len(tr.transformers)

			// Update type routing based on config.
			supportedTypes := transformer.SupportedTypes()
			if len(config.ObjectTypes) > 0 {
				// Convert strings to ObjectTypes and use config-specified types.
				configTypes := make([]types.ObjectType, len(config.ObjectTypes))
				for i, objTypeStr := range config.ObjectTypes {
					configTypes[i] = types.ObjectType(objTypeStr)
				}
				supportedTypes = configTypes
			}
			tr.updateTypeRouting(name, supportedTypes)
		}
	}

	return nil
}

// Transform applies transformers to an object based on its type.
//
// Arguments:
// - ctx: Context for cancellation and timeouts.
// - objectType: The type of the object to transform.
// - object: The object to transform.
//
// Returns:
// - []interface{}: Results from all applicable transformers.
// - error: Error if any transformation failed.
//
// Example:
//
//	results, err := registry.Transform(ctx, types.ObjectTypePage, page)
//	if err != nil {
//	    log.Printf("Transformation failed: %v", err)
//	    return
//	}
//	for _, result := range results {
//	    fmt.Printf("Transformed result: %v\n", result)
//	}
func (tr *TransformerRegistry) Transform(ctx context.Context, objectType types.ObjectType, object interface{}) ([]interface{}, error) {
	tr.mu.RLock()
	transformerNames := tr.typeRoutes[objectType]
	tr.mu.RUnlock()

	// If no routes exist, check all lazy transformers to see if any support this type
	if len(transformerNames) == 0 {
		tr.mu.RLock()
		for name := range tr.lazyTransformers {
			transformerNames = append(transformerNames, name)
		}
		tr.mu.RUnlock()
	}

	// Get transformers and sort by priority.
	transformers := make([]TransformerPlugin, 0, len(transformerNames))
	for _, name := range transformerNames {
		if transformer, err := tr.getTransformer(name); err == nil {
			// Check if this transformer supports the requested object type
			if tr.transformerSupportsType(transformer, objectType) {
				transformers = append(transformers, transformer)
			}
		}
	}

	// Sort by priority (higher first).
	tr.sortTransformersByPriority(transformers)

	// Apply transformations.
	results := make([]interface{}, 0, len(transformers))
	for _, transformer := range transformers {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := transformer.Transform(ctx, objectType, object)
		if err != nil {
			return results, fmt.Errorf("transformer '%s' failed: %w", transformer.Name(), err)
		}

		if result != nil {
			results = append(results, result)
		}

		// Update stats.
		tr.mu.Lock()
		tr.stats.ExecutionCounts[transformer.Name()]++
		tr.mu.Unlock()
	}

	return results, nil
}

// GetStats returns current transformer statistics.
//
// Returns:
// - TransformerStats: Current statistics.
func (tr *TransformerRegistry) GetStats() TransformerStats {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	// Create a copy to avoid race conditions.
	stats := TransformerStats{
		TotalTransformers: tr.stats.TotalTransformers,
		LoadedCount:       tr.stats.LoadedCount,
		ExecutionCounts:   make(map[string]int),
	}

	for name, count := range tr.stats.ExecutionCounts {
		stats.ExecutionCounts[name] = count
	}

	return stats
}

// ListTransformers returns a list of all registered transformer names.
//
// Returns:
// - []string: List of transformer names.
func (tr *TransformerRegistry) ListTransformers() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	names := make([]string, 0, len(tr.lazyTransformers)+len(tr.transformers))

	// Add lazy transformers.
	for name := range tr.lazyTransformers {
		names = append(names, name)
	}

	// Add loaded transformers not in lazy list.
	for name := range tr.transformers {
		if _, exists := tr.lazyTransformers[name]; !exists {
			names = append(names, name)
		}
	}

	return names
}

// IsLoaded checks if a transformer is currently loaded.
//
// Arguments:
// - name: Name of the transformer to check.
//
// Returns:
// - bool: True if the transformer is loaded.
func (tr *TransformerRegistry) IsLoaded(name string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	_, exists := tr.transformers[name]
	return exists
}

// Unload removes a transformer from the registry and calls its cleanup method.
//
// Arguments:
// - name: Name of the transformer to unload.
//
// Returns:
// - error: Error if cleanup failed.
func (tr *TransformerRegistry) Unload(name string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	transformer, exists := tr.transformers[name]
	if !exists {
		return fmt.Errorf("transformer '%s' is not loaded", name)
	}

	// Call cleanup.
	if err := transformer.Cleanup(); err != nil {
		return fmt.Errorf("cleanup failed for transformer '%s': %w", name, err)
	}

	// Remove from registry.
	delete(tr.transformers, name)
	tr.stats.LoadedCount = len(tr.transformers)

	// Update type routing.
	tr.removeFromTypeRouting(name)

	return nil
}

// Helper methods

func (tr *TransformerRegistry) getTransformer(name string) (TransformerPlugin, error) {
	tr.mu.RLock()
	transformer, exists := tr.transformers[name]
	tr.mu.RUnlock()

	if exists {
		return transformer, nil
	}

	// Try lazy loading.
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Double-check pattern.
	if transformer, exists := tr.transformers[name]; exists {
		return transformer, nil
	}

	initializer, exists := tr.lazyTransformers[name]
	if !exists {
		return nil, fmt.Errorf("transformer '%s' not found", name)
	}

	// Create and initialize transformer.
	transformer = initializer()

	// Initialize with config if available.
	if config, exists := tr.configs[name]; exists {
		if err := transformer.Initialize(config.Config); err != nil {
			return nil, fmt.Errorf("failed to initialize transformer '%s': %w", name, err)
		}
	} else {
		// Initialize with empty config.
		if err := transformer.Initialize(make(map[string]interface{})); err != nil {
			return nil, fmt.Errorf("failed to initialize transformer '%s': %w", name, err)
		}
	}

	// Cache the transformer.
	tr.transformers[name] = transformer
	tr.stats.LoadedCount = len(tr.transformers)

	// Update type routing.
	supportedTypes := transformer.SupportedTypes()
	if config, exists := tr.configs[name]; exists && len(config.ObjectTypes) > 0 {
		// Convert strings to ObjectTypes.
		configTypes := make([]types.ObjectType, len(config.ObjectTypes))
		for i, objTypeStr := range config.ObjectTypes {
			configTypes[i] = types.ObjectType(objTypeStr)
		}
		supportedTypes = configTypes
	}
	tr.updateTypeRouting(name, supportedTypes)

	return transformer, nil
}

func (tr *TransformerRegistry) updateTypeRouting(name string, supportedTypes []types.ObjectType) {
	for _, objectType := range supportedTypes {
		// Add transformer to type routing if not already present.
		found := false
		for _, existingName := range tr.typeRoutes[objectType] {
			if existingName == name {
				found = true
				break
			}
		}
		if !found {
			tr.typeRoutes[objectType] = append(tr.typeRoutes[objectType], name)
		}
	}
}

func (tr *TransformerRegistry) removeFromTypeRouting(name string) {
	for objectType, transformerNames := range tr.typeRoutes {
		filtered := make([]string, 0, len(transformerNames))
		for _, transformerName := range transformerNames {
			if transformerName != name {
				filtered = append(filtered, transformerName)
			}
		}
		tr.typeRoutes[objectType] = filtered
	}
}

func (tr *TransformerRegistry) sortTransformersByPriority(transformers []TransformerPlugin) {
	// Simple bubble sort by priority (could be optimized with sort.Slice).
	n := len(transformers)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			priority1 := transformers[j].Priority()
			if config, exists := tr.configs[transformers[j].Name()]; exists {
				priority1 = config.Priority
			}

			priority2 := transformers[j+1].Priority()
			if config, exists := tr.configs[transformers[j+1].Name()]; exists {
				priority2 = config.Priority
			}

			if priority1 < priority2 {
				transformers[j], transformers[j+1] = transformers[j+1], transformers[j]
			}
		}
	}
}

// transformerSupportsType checks if a transformer supports the given object type.
// It considers both the transformer's native supported types and any configuration overrides.
func (tr *TransformerRegistry) transformerSupportsType(transformer TransformerPlugin, objectType types.ObjectType) bool {
	name := transformer.Name()
	
	// Check if there's a configuration override for supported types
	if config, exists := tr.configs[name]; exists && len(config.ObjectTypes) > 0 {
		for _, configType := range config.ObjectTypes {
			if types.ObjectType(configType) == objectType {
				return true
			}
		}
		return false // Configuration explicitly limits types
	}
	
	// Check transformer's native supported types
	supportedTypes := transformer.SupportedTypes()
	for _, supportedType := range supportedTypes {
		if supportedType == objectType {
			return true
		}
	}
	
	return false
}

// Global registry instance for convenience.
var globalRegistry = NewTransformerRegistry()

// RegisterGlobal registers a transformer with the global registry.
//
// Arguments:
// - name: Unique name for the transformer.
// - initializer: Function to create the transformer instance.
//
// Example:
//
//	func init() {
//	    RegisterGlobal("markdown", func() TransformerPlugin {
//	        return &MarkdownTransformer{}
//	    })
//	}
func RegisterGlobal(name string, initializer TransformerInitializer) {
	globalRegistry.RegisterLazy(name, initializer)
}

// GetGlobalRegistry returns the global transformer registry.
//
// Returns:
// - *TransformerRegistry: The global registry instance.
func GetGlobalRegistry() *TransformerRegistry {
	return globalRegistry
}