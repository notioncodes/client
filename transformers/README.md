# Notion Transformers

A pluggable transformer system for the Notion API client that allows dynamic loading and configuration of transformers to process Notion objects.

## Features

- **Lazy Loading**: Transformers are only instantiated when first needed
- **Configuration-Based**: Load and configure transformers from YAML/JSON files
- **Type-Based Routing**: Automatically route objects to appropriate transformers based on type
- **Priority System**: Control execution order with configurable priorities
- **Plugin Architecture**: Easy to add custom transformers
- **Thread-Safe**: Concurrent access supported with proper synchronization
- **Statistics**: Track transformer usage and performance

## Quick Start

### Basic Usage

```go
// Create a transformer registry
registry := transformers.NewTransformerRegistry()

// Load configuration from file
loader := transformers.DefaultConfigLoader()
configs, err := loader.LoadFromFile("transformers.yaml")
if err != nil {
    log.Fatal(err)
}

// Apply configurations to registry
err = registry.LoadFromConfig(configs)
if err != nil {
    log.Fatal(err)
}

// Transform a Notion page
ctx := context.Background()
page := &types.Page{...}

results, err := registry.Transform(ctx, types.ObjectTypePage, page)
if err != nil {
    log.Fatal(err)
}

// Process results
for _, result := range results {
    fmt.Printf("Transformed: %+v\n", result)
}
```

### Using with the Client Registry

```go
// The transformer registry integrates with the main client registry
client, err := client.New(config)
if err != nil {
    log.Fatal(err)
}

// Access transformers through the client registry
transformerRegistry := client.registry.transformers

// Transform retrieved data
page := client.Pages().Get(ctx, pageID)
if !page.IsError() {
    results, err := transformerRegistry.Transform(ctx, types.ObjectTypePage, page.Data)
    // Process results...
}
```

## Built-in Transformers

### Markdown Transformer

Converts Notion objects to Markdown format with configurable options:

```yaml
markdown:
  name: markdown
  enabled: true
  priority: 10
  object_types: ["page", "database", "block"]
  config:
    output_format: "github"        # github, commonmark, notion
    include_metadata: true         # YAML frontmatter
    include_properties: true       # Page/database properties
    code_block_language: "go"      # Default code language
    link_format: "markdown"        # markdown, html, notion
```

**Output Example:**
```markdown
# Page Title

---
id: 22cd7342-e571-80d2-92f7-c90f552b68f2
created_time: 2023-10-01T10:00:00Z
last_edited_time: 2023-10-01T15:30:00Z
url: https://www.notion.so/Page-Title-22cd7342
---

## Properties

- **Status**: Done
- **Priority**: High
- **Tags**: [project, important]

*[Page content would be rendered here from child blocks]*
```

### JSON Transformer

Converts Notion objects to JSON format with filtering options:

```yaml
json:
  name: json
  enabled: true
  priority: 5
  object_types: ["page", "database", "block", "user"]
  config:
    pretty: true                   # Pretty-print output
    include_metadata: true         # Include timestamps, etc.
    include_properties: true       # Include properties
    compact_arrays: false          # Compact array representation
    exclude_fields: ["url"]        # Fields to exclude
```

**Output Example:**
```json
{
  "data": {
    "id": "22cd7342-e571-80d2-92f7-c90f552b68f2",
    "created_time": "2023-10-01T10:00:00Z",
    "last_edited_time": "2023-10-01T15:30:00Z",
    "properties": {
      "Status": {
        "type": "select",
        "select": {"name": "Done", "color": "green"}
      }
    }
  },
  "metadata": {
    "page_id": "22cd7342-e571-80d2-92f7-c90f552b68f2",
    "created_time": "2023-10-01T10:00:00.000Z",
    "last_edited_time": "2023-10-01T15:30:00.000Z"
  },
  "object_type": "page"
}
```

## Configuration

### Configuration File Format

Transformers can be configured using YAML or JSON files:

```yaml
# YAML configuration
transformer_name:
  name: transformer_name          # Unique transformer name
  enabled: true                   # Enable/disable transformer
  priority: 10                    # Execution priority (higher = first)
  object_types: ["page", "block"] # Supported object types
  config:                         # Transformer-specific configuration
    option1: value1
    option2: value2
```

```json
// JSON configuration
{
  "transformer_name": {
    "name": "transformer_name",
    "enabled": true,
    "priority": 10,
    "object_types": ["page", "block"],
    "config": {
      "option1": "value1",
      "option2": "value2"
    }
  }
}
```

### Configuration Loading

```go
// Load from specific file
loader := transformers.NewConfigLoader([]string{"./config", "."})
configs, err := loader.LoadFromFile("transformers.yaml")

// Load default configuration
configs := loader.LoadDefault()

// Generate example configuration
example, err := loader.GenerateExampleConfig("yaml")
fmt.Println(string(example))

// Save configuration to file
err = loader.SaveToFile("new_config.yaml", configs)
```

### Search Paths

The config loader searches for files in multiple directories:

- Current directory (`.`)
- `./config`
- `./transformers`
- `/etc/notion-transformers`
- `~/.config/notion-transformers`

## Creating Custom Transformers

### Implement the TransformerPlugin Interface

```go
type MyTransformer struct {
    config MyConfig
}

type MyConfig struct {
    OutputFormat string `json:"output_format"`
    CustomOption bool   `json:"custom_option"`
}

func (mt *MyTransformer) Name() string {
    return "my_transformer"
}

func (mt *MyTransformer) Version() string {
    return "1.0.0"
}

func (mt *MyTransformer) SupportedTypes() []types.ObjectType {
    return []types.ObjectType{types.ObjectTypePage}
}

func (mt *MyTransformer) Priority() int {
    return 10
}

func (mt *MyTransformer) Initialize(config map[string]interface{}) error {
    // Set defaults
    mt.config = MyConfig{
        OutputFormat: "default",
        CustomOption: false,
    }
    
    // Apply configuration
    if format, ok := config["output_format"].(string); ok {
        mt.config.OutputFormat = format
    }
    
    if option, ok := config["custom_option"].(bool); ok {
        mt.config.CustomOption = option
    }
    
    return nil
}

func (mt *MyTransformer) Cleanup() error {
    // Release resources
    return nil
}

func (mt *MyTransformer) Transform(ctx context.Context, objectType types.ObjectType, object interface{}) (interface{}, error) {
    // Transform the object
    switch objectType {
    case types.ObjectTypePage:
        page := object.(*types.Page)
        return mt.transformPage(ctx, page)
    default:
        return nil, fmt.Errorf("unsupported object type: %s", objectType)
    }
}

func (mt *MyTransformer) transformPage(ctx context.Context, page *types.Page) (interface{}, error) {
    // Your transformation logic here
    return map[string]interface{}{
        "transformed_page": page.ID,
        "format": mt.config.OutputFormat,
    }, nil
}
```

### Register the Transformer

```go
// Register globally (typically in init())
func init() {
    transformers.RegisterGlobal("my_transformer", func() transformers.TransformerPlugin {
        return &MyTransformer{}
    })
}

// Or register with a specific registry
registry := transformers.NewTransformerRegistry()
registry.RegisterLazy("my_transformer", func() transformers.TransformerPlugin {
    return &MyTransformer{}
})
```

### Configure the Transformer

```yaml
my_transformer:
  name: my_transformer
  enabled: true
  priority: 15
  object_types: ["page"]
  config:
    output_format: "custom"
    custom_option: true
```

## Advanced Usage

### Priority and Execution Order

Transformers are executed in priority order (higher numbers first):

```yaml
high_priority:
  priority: 20    # Executes first

medium_priority:
  priority: 10    # Executes second

low_priority:
  priority: 5     # Executes last
```

### Object Type Filtering

Limit transformers to specific object types:

```yaml
page_only:
  object_types: ["page"]         # Only processes pages

multi_type:
  object_types: ["page", "database", "block"]  # Multiple types

all_types:
  object_types: []               # All supported types (default)
```

### Statistics and Monitoring

Track transformer usage:

```go
// Get statistics
stats := registry.GetStats()
fmt.Printf("Total transformers: %d\n", stats.TotalTransformers)
fmt.Printf("Loaded transformers: %d\n", stats.LoadedCount)

for name, count := range stats.ExecutionCounts {
    fmt.Printf("%s executed %d times\n", name, count)
}

// List registered transformers
transformers := registry.ListTransformers()
fmt.Printf("Available transformers: %v\n", transformers)

// Check if transformer is loaded
if registry.IsLoaded("markdown") {
    fmt.Println("Markdown transformer is loaded")
}
```

### Error Handling and Context

Transformers support proper error handling and context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := registry.Transform(ctx, types.ObjectTypePage, page)
if err != nil {
    if err == context.DeadlineExceeded {
        log.Println("Transformation timed out")
    } else {
        log.Printf("Transformation failed: %v", err)
    }
    return
}
```

## Testing

The transformer system includes comprehensive tests:

```bash
# Run all transformer tests
go test ./transformers/...

# Run specific test
go test ./transformers/ -run TestTransformerRegistry

# Run with verbose output
go test -v ./transformers/...

# Run with coverage
go test -cover ./transformers/...
```

## Configuration Examples

See [`example_config.yaml`](./example_config.yaml) for a complete configuration example with all built-in transformers and their options.

## Integration with Client

The transformer system integrates seamlessly with the main Notion client:

```go
// Transformers are automatically registered in the client registry
client, err := client.New(config)
if err != nil {
    log.Fatal(err)
}

// Configure transformers (typically at startup)
transformerConfigs := map[string]*transformers.TransformerConfig{
    "markdown": {
        Name:    "markdown",
        Enabled: true,
        Priority: 10,
        Config: map[string]interface{}{
            "output_format": "github",
            "include_metadata": true,
        },
    },
}

// Apply to the client's transformer registry
err = client.registry.transformers.LoadFromConfig(transformerConfigs)
if err != nil {
    log.Fatal(err)
}
```