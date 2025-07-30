package client

import (
	"fmt"
	"sync"

	"github.com/notioncodes/client/transformers"
	"github.com/notioncodes/types"
)

// Registry provides a type-safe, lazy-loading operator registry with caching.
// It enables ergonomic access to all Notion API operations with fluent interfaces
// and namespace grouping for better IDE experience.
type Registry struct {
	mu           sync.RWMutex
	config       *Config
	httpClient   *HTTPClient
	instances    map[string]interface{}
	initializers map[string]RegistryInitializer
	plugins      []PluginRegistrar
	transformers []transformers.TransformerPlugin
}

// RegistryInitializer defines how to create operator instances.
type RegistryInitializer func(httpClient *HTTPClient, config *OperatorConfig) interface{}

// PluginRegistrar allows plugins to register themselves with the registry.
type PluginRegistrar interface {
	RegisterWith(registry *Registry) error
	Name() string
	Version() string
}

// NewRegistry creates a new operator registry with lazy instantiation and caching.
//
// Arguments:
//   - config: Client configuration for operators.
//   - httpClient: HTTP client for making requests.
//
// Returns:
//   - *Registry: A new registry instance.
//
// Example:
//
//	registry := NewRegistry(config, httpClient)
//	pages := registry.Pages().Get(ctx, pageID)
func NewRegistry(config *Config, httpClient *HTTPClient) *Registry {
	registry := &Registry{
		config:       config,
		httpClient:   httpClient,
		instances:    make(map[string]interface{}),
		initializers: make(map[string]RegistryInitializer),
		plugins:      make([]PluginRegistrar, 0),
		transformers: make([]transformers.TransformerPlugin, 0),
	}

	// Register core operators.
	registry.registerCoreOperators()

	// Register transformers.
	registry.registerTransformers()

	return registry
}

// Search returns the search operations namespace.
func (r *Registry) Search() *SearchOperator[types.SearchResult] {
	operator, err := r.Get("search")
	if err != nil {
		panic(err)
	}
	return operator.(*SearchOperator[types.SearchResult])
}

// Databases returns the database operations namespace.
func (r *Registry) Databases() *DatabaseNamespace {
	return &DatabaseNamespace{registry: r}
}

// Pages returns the page operations namespace.
func (r *Registry) Pages() *PageNamespace {
	return &PageNamespace{registry: r}
}

// Blocks returns the block operations namespace.
func (r *Registry) Blocks() *BlockNamespace {
	return &BlockNamespace{registry: r}
}

// Users returns the user operations namespace.
func (r *Registry) Users() *UserNamespace {
	return &UserNamespace{registry: r}
}

// registerCoreOperators registers all core Notion API operators so they
// are available to the registry when called upon.
func (r *Registry) registerCoreOperators() {
	// Single resource operators.
	r.Register("page", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.Page](httpClient, config)
	})

	r.Register("database", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.Database](httpClient, config)
	})

	r.Register("block", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.Block](httpClient, config)
	})

	r.Register("user", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewOperator[types.User](httpClient, config)
	})

	// Paginated operators.
	r.Register("page_paginated", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewPaginatedOperator[types.Page](httpClient, config)
	})

	r.Register("database_paginated", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewPaginatedOperator[types.Database](httpClient, config)
	})

	r.Register("block_paginated", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewPaginatedOperator[types.Block](httpClient, config)
	})

	r.Register("user_paginated", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewPaginatedOperator[types.User](httpClient, config)
	})

	// Search operator.
	r.Register("search", func(httpClient *HTTPClient, config *OperatorConfig) interface{} {
		return NewSearchOperator[types.SearchResult](httpClient, config)
	})
}

func (r *Registry) registerTransformers() {
	r.transformers = append(r.transformers, &transformers.MarkdownTransformer{})
}

// Register adds a new operator initializer to the registry.
//
// Arguments:
//   - name: Unique name for the operator.
//   - initializer: Function to create the operator instance.
//
// Example:
//
//	registry.Register("custom_page", func(client *HTTPClient, config *OperatorConfig) interface{} {
//	    return NewCustomPageOperator(client, config)
//	})
func (r *Registry) Register(name string, initializer RegistryInitializer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initializers[name] = initializer
}

// Get retrieves an operator instance, creating and caching it if necessary.
//
// Arguments:
//   - name: Name of the operator to retrieve.
//
// Returns:
//   - interface{}: The operator instance.
//   - error: Error if operator not found or creation failed.
//
// Example:
//
//	op, err := registry.Get("page")
//	if err != nil {
//	    return err
//	}
//	pageOp := op.(*Operator[types.Page])
func (r *Registry) Get(name string) (interface{}, error) {
	// Fast path: try read lock
	r.mu.RLock()
	instance, found := r.instances[name]
	r.mu.RUnlock()
	if found {
		return instance, nil
	}

	// Slow path: acquire write lock
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check to avoid race after acquiring write lock
	if instance, found := r.instances[name]; found {
		return instance, nil
	}

	initializer, ok := r.initializers[name]
	if !ok {
		return nil, fmt.Errorf("operator '%s' not registered", name)
	}

	instance = initializer(r.httpClient, &OperatorConfig{
		Concurrency:   r.config.Concurrency,
		PageSize:      r.config.PageSize,
		Timeout:       r.config.Timeout,
		EnableCaching: false, // TODO: pull from r.config
		CacheTTL:      0,     // TODO: pull from r.config
	})

	r.instances[name] = instance
	return instance, nil
}

// GetTyped retrieves a typed operator instance.
//
// Arguments:
//   - name: Name of the operator to retrieve.
//
// Returns:
//   - T: The typed operator instance.
//   - error: Error if operator not found or type assertion failed.
//
// Example:
//
//	pageOp, err := GetTyped[*Operator[types.Page]](registry, "page")
//	if err != nil {
//	    return err
//	}
func GetTyped[T any](registry *Registry, name string) (T, error) {
	var zero T
	instance, err := registry.Get(name)
	if err != nil {
		return zero, err
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("operator '%s' is not of type %T, got %T",
			name, zero, instance)
	}

	return typed, nil
}

// Clear removes all cached instances, forcing re-creation on next access.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances = make(map[string]interface{})
}

// ClearOperator removes a specific cached instance.
//
// Arguments:
//   - name: Name of the operator to clear from cache.
func (r *Registry) ClearOperator(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.instances, name)
}

// ListRegistered returns all registered operator names.
//
// Returns:
//   - []string: List of registered operator names.
func (r *Registry) ListRegistered() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.initializers))
	for name := range r.initializers {
		names = append(names, name)
	}
	return names
}

// ListCached returns all currently cached operator names.
//
// Returns:
//   - []string: List of cached operator names.
func (r *Registry) ListCached() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.instances))
	for name := range r.instances {
		names = append(names, name)
	}
	return names
}

// Stats returns registry statistics.
//
// Returns:
//   - RegistryStats: Statistics about the registry.
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RegistryStats{
		RegisteredCount: len(r.initializers),
		CachedCount:     len(r.instances),
		PluginsCount:    len(r.plugins),
	}
}

// RegistryStats provides statistics about the registry.
type RegistryStats struct {
	RegisteredCount int `json:"registered_count"`
	CachedCount     int `json:"cached_count"`
	PluginsCount    int `json:"plugins_count"`
}

// HasOperator checks if an operator is registered.
//
// Arguments:
//   - name: Name of the operator to check.
//
// Returns:
//   - bool: True if the operator is registered.
func (r *Registry) HasOperator(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.initializers[name]
	return exists
}

// ValidateConfiguration validates the registry configuration.
//
// Returns:
//   - error: Validation error if configuration is invalid.
func (r *Registry) ValidateConfiguration() error {
	if r.config == nil {
		return fmt.Errorf("registry configuration cannot be nil")
	}
	if r.httpClient == nil {
		return fmt.Errorf("HTTP client cannot be nil")
	}
	return nil
}
