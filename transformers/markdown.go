package transformers

import (
	"context"
	"fmt"
	"strings"

	"github.com/notioncodes/types"
)

// MarkdownTransformer converts Notion objects to Markdown format.
// It supports pages, databases, and blocks with configurable output formats.
type MarkdownTransformer struct {
	config MarkdownConfig
}

// MarkdownConfig holds configuration for the Markdown transformer.
type MarkdownConfig struct {
	// OutputFormat specifies the Markdown variant to use.
	OutputFormat string `json:"output_format" yaml:"output_format"` // "github", "commonmark", "notion"

	// IncludeMetadata determines if object metadata should be included.
	IncludeMetadata bool `json:"include_metadata" yaml:"include_metadata"`

	// IncludeProperties determines if database properties should be included.
	IncludeProperties bool `json:"include_properties" yaml:"include_properties"`

	// CodeBlockLanguage sets the default language for code blocks.
	CodeBlockLanguage string `json:"code_block_language" yaml:"code_block_language"`

	// LinkFormat specifies how links should be formatted.
	LinkFormat string `json:"link_format" yaml:"link_format"` // "markdown", "html", "notion"
}

// MarkdownResult represents the output from the Markdown transformer.
type MarkdownResult struct {
	// Content is the generated Markdown content.
	Content string `json:"content"`

	// Metadata contains additional information about the transformation.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ObjectType is the type of the original object.
	ObjectType types.ObjectType `json:"object_type"`
}

// Name returns the transformer name.
func (mt *MarkdownTransformer) Name() string {
	return "markdown"
}

// Version returns the transformer version.
func (mt *MarkdownTransformer) Version() string {
	return "1.0.0"
}

// SupportedTypes returns the object types this transformer can handle.
func (mt *MarkdownTransformer) SupportedTypes() []types.ObjectType {
	return []types.ObjectType{
		types.ObjectTypePage,
		types.ObjectTypeDatabase,
		types.ObjectTypeBlock,
	}
}

// Priority returns the execution priority.
func (mt *MarkdownTransformer) Priority() int {
	return 10 // Medium priority
}

// Initialize sets up the transformer with the provided configuration.
func (mt *MarkdownTransformer) Initialize(config map[string]interface{}) error {
	// Set defaults.
	mt.config = MarkdownConfig{
		OutputFormat:      "github",
		IncludeMetadata:   true,
		IncludeProperties: true,
		CodeBlockLanguage: "",
		LinkFormat:        "markdown",
	}

	// Apply configuration overrides.
	if outputFormat, ok := config["output_format"].(string); ok {
		mt.config.OutputFormat = outputFormat
	}

	if includeMetadata, ok := config["include_metadata"].(bool); ok {
		mt.config.IncludeMetadata = includeMetadata
	}

	if includeProperties, ok := config["include_properties"].(bool); ok {
		mt.config.IncludeProperties = includeProperties
	}

	if codeBlockLanguage, ok := config["code_block_language"].(string); ok {
		mt.config.CodeBlockLanguage = codeBlockLanguage
	}

	if linkFormat, ok := config["link_format"].(string); ok {
		mt.config.LinkFormat = linkFormat
	}

	// Validate configuration.
	if !mt.isValidOutputFormat(mt.config.OutputFormat) {
		return fmt.Errorf("invalid output_format: %s", mt.config.OutputFormat)
	}

	if !mt.isValidLinkFormat(mt.config.LinkFormat) {
		return fmt.Errorf("invalid link_format: %s", mt.config.LinkFormat)
	}

	return nil
}

// Cleanup releases resources used by the transformer.
func (mt *MarkdownTransformer) Cleanup() error {
	// No cleanup needed for this transformer.
	return nil
}

// Transform converts a Notion object to Markdown format.
func (mt *MarkdownTransformer) Transform(ctx context.Context, objectType types.ObjectType, object interface{}) (interface{}, error) {
	result := &MarkdownResult{
		ObjectType: objectType,
		Metadata:   make(map[string]interface{}),
	}

	switch objectType {
	case types.ObjectTypePage:
		page, ok := object.(*types.Page)
		if !ok {
			return nil, fmt.Errorf("expected *types.Page, got %T", object)
		}
		content, err := mt.transformPage(ctx, page)
		if err != nil {
			return nil, err
		}
		result.Content = content
		result.Metadata["page_id"] = page.ID.String()
		result.Metadata["title"] = mt.extractPageTitle(page)

	case types.ObjectTypeDatabase:
		database, ok := object.(*types.Database)
		if !ok {
			return nil, fmt.Errorf("expected *types.Database, got %T", object)
		}
		content, err := mt.transformDatabase(ctx, database)
		if err != nil {
			return nil, err
		}
		result.Content = content
		result.Metadata["database_id"] = database.ID.String()
		result.Metadata["title"] = mt.extractDatabaseTitle(database)

	case types.ObjectTypeBlock:
		block, ok := object.(types.Block)
		if !ok {
			return nil, fmt.Errorf("expected *types.Block, got %T", object)
		}
		content, err := mt.transformBlock(ctx, block)
		if err != nil {
			return nil, err
		}
		result.Content = content
		result.Metadata["block_id"] = block.ID.String()
		result.Metadata["block_type"] = string(block.Type)

	default:
		return nil, fmt.Errorf("unsupported object type: %s", objectType)
	}

	return result, nil
}

// transformPage converts a Page to Markdown.
func (mt *MarkdownTransformer) transformPage(_ context.Context, page *types.Page) (string, error) {
	var content strings.Builder

	// Add title.
	title := mt.extractPageTitle(page)
	if title.Text.Content != "" {
		content.WriteString(fmt.Sprintf("# %s\n\n", title.Text.Content))
	}

	// Add metadata if configured.
	if mt.config.IncludeMetadata {
		content.WriteString(mt.formatPageMetadata(page))
		content.WriteString("\n")
	}

	// Add properties if configured and available.
	if mt.config.IncludeProperties && page.Properties != nil {
		propertiesContent := mt.formatPageProperties(page.Properties)
		if propertiesContent != "" {
			content.WriteString("## Properties\n\n")
			content.WriteString(propertiesContent)
			content.WriteString("\n")
		}
	}

	// Note: In a real implementation, you would also transform the page's blocks/content.
	// This would require additional API calls to get the page's child blocks.
	content.WriteString("*[Page content would be rendered here from child blocks]*\n")

	return content.String(), nil
}

// transformDatabase converts a Database to Markdown.
func (mt *MarkdownTransformer) transformDatabase(ctx context.Context, database *types.Database) (string, error) {
	var content strings.Builder

	// Add title.
	title := mt.extractDatabaseTitle(database)
	if title != "" {
		content.WriteString(fmt.Sprintf("# %s\n\n", title))
	}

	// Add metadata if configured.
	if mt.config.IncludeMetadata {
		content.WriteString(mt.formatDatabaseMetadata(database))
		content.WriteString("\n")
	}

	// Add properties schema if configured.
	if mt.config.IncludeProperties && database.Properties != nil {
		content.WriteString("## Schema\n\n")
		content.WriteString(mt.formatDatabaseSchema(database.Properties))
		content.WriteString("\n")
	}

	// Note: In a real implementation, you would also query and format the database's pages.
	content.WriteString("*[Database entries would be rendered here]*\n")

	return content.String(), nil
}

// transformBlock converts a Block to Markdown.
func (mt *MarkdownTransformer) transformBlock(_ context.Context, block types.Block) (string, error) {
	switch block.Type {
	case types.BlockTypeParagraph:
		return mt.formatParagraphBlock(&block), nil
	case types.BlockTypeHeading1:
		return mt.formatHeadingBlock(&block, 1), nil
	case types.BlockTypeHeading2:
		return mt.formatHeadingBlock(&block, 2), nil
	case types.BlockTypeHeading3:
		return mt.formatHeadingBlock(&block, 3), nil
	case types.BlockTypeBulletedListItem:
		return mt.formatBulletListBlock(&block), nil
	case types.BlockTypeNumberedListItem:
		return mt.formatNumberedListBlock(&block), nil
	case types.BlockTypeCode:
		return mt.formatCodeBlock(&block), nil
	case types.BlockTypeQuote:
		return mt.formatQuoteBlock(&block), nil
	default:
		// For unknown block types, provide a generic representation.
		return fmt.Sprintf("*[%s block]*\n", block.Type), nil
	}
}

// Helper methods for formatting different elements.

func (mt *MarkdownTransformer) extractPageTitle(page *types.Page) types.RichText {
	return page.PropertyAccessor.GetTitleProperty("Name")[0]
}

func (mt *MarkdownTransformer) extractDatabaseTitle(database *types.Database) string {
	// Extract database title from rich text.
	if len(database.Title) > 0 {
		// In a real implementation, you would convert rich text to plain text.
		return "Database Title" // Placeholder
	}
	return "Untitled Database"
}

func (mt *MarkdownTransformer) formatPageMetadata(page *types.Page) string {
	var metadata strings.Builder

	metadata.WriteString("---\n")
	metadata.WriteString(fmt.Sprintf("id: %s\n", page.ID))
	metadata.WriteString(fmt.Sprintf("created_time: %s\n", page.CreatedTime.Format("2006-01-02T15:04:05Z")))
	metadata.WriteString(fmt.Sprintf("last_edited_time: %s\n", page.LastEditedTime.Format("2006-01-02T15:04:05Z")))
	if page.URL != "" {
		metadata.WriteString(fmt.Sprintf("url: %s\n", page.URL))
	}
	metadata.WriteString("---\n")

	return metadata.String()
}

func (mt *MarkdownTransformer) formatDatabaseMetadata(database *types.Database) string {
	var metadata strings.Builder

	metadata.WriteString("---\n")
	metadata.WriteString(fmt.Sprintf("id: %s\n", database.ID))
	metadata.WriteString(fmt.Sprintf("created_time: %s\n", database.CreatedTime.Format("2006-01-02T15:04:05Z")))
	metadata.WriteString(fmt.Sprintf("last_edited_time: %s\n", database.LastEditedTime.Format("2006-01-02T15:04:05Z")))
	if database.URL != "" {
		metadata.WriteString(fmt.Sprintf("url: %s\n", database.URL))
	}
	metadata.WriteString("---\n")

	return metadata.String()
}

func (mt *MarkdownTransformer) formatPageProperties(properties map[string]types.Property) string {
	var content strings.Builder

	for name, prop := range properties {
		content.WriteString(fmt.Sprintf("- **%s**: ", name))

		// Format property value based on type.
		switch prop.Type {
		case types.PropertyTypeTitle, types.PropertyTypeRichText:
			content.WriteString("*[Rich text content]*")
		case types.PropertyTypeNumber:
			content.WriteString("*[Number value]*")
		case types.PropertyTypeSelect:
			content.WriteString("*[Select value]*")
		case types.PropertyTypeMultiSelect:
			content.WriteString("*[Multi-select values]*")
		case types.PropertyTypeDate:
			content.WriteString("*[Date value]*")
		case types.PropertyTypeCheckbox:
			content.WriteString("*[Checkbox value]*")
		case types.PropertyTypeURL:
			content.WriteString("*[URL value]*")
		case types.PropertyTypeEmail:
			content.WriteString("*[Email value]*")
		case types.PropertyTypePhoneNumber:
			content.WriteString("*[Phone number]*")
		default:
			content.WriteString("*[Unknown property type]*")
		}

		content.WriteString("\n")
	}

	return content.String()
}

func (mt *MarkdownTransformer) formatDatabaseSchema(properties map[string]types.DatabaseProperty) string {
	var content strings.Builder

	content.WriteString("| Property | Type | Configuration |\n")
	content.WriteString("|----------|------|---------------|\n")

	for name, prop := range properties {
		content.WriteString(fmt.Sprintf("| %s | %s | ", name, prop.Type))

		// Add configuration details based on property type.
		switch prop.Type {
		case types.PropertyTypeSelect:
			content.WriteString("Select options configured")
		case types.PropertyTypeMultiSelect:
			content.WriteString("Multi-select options configured")
		case types.PropertyTypeFormula:
			content.WriteString("Formula configured")
		case types.PropertyTypeRelation:
			content.WriteString("Relation configured")
		case types.PropertyTypeRollup:
			content.WriteString("Rollup configured")
		default:
			content.WriteString("-")
		}

		content.WriteString(" |\n")
	}

	return content.String()
}

func (mt *MarkdownTransformer) formatParagraphBlock(block *types.Block) string {
	// In a real implementation, you would extract and format rich text.
	return "*[Paragraph content]*\n\n"
}

func (mt *MarkdownTransformer) formatHeadingBlock(block *types.Block, level int) string {
	heading := strings.Repeat("#", level)
	return fmt.Sprintf("%s Heading\n\n", heading)
}

func (mt *MarkdownTransformer) formatBulletListBlock(block *types.Block) string {
	return "- List item\n"
}

func (mt *MarkdownTransformer) formatNumberedListBlock(block *types.Block) string {
	return "1. Numbered list item\n"
}

func (mt *MarkdownTransformer) formatCodeBlock(block *types.Block) string {
	language := mt.config.CodeBlockLanguage
	if language == "" {
		language = "text"
	}
	return fmt.Sprintf("```%s\n[Code content]\n```\n\n", language)
}

func (mt *MarkdownTransformer) formatQuoteBlock(block *types.Block) string {
	return "> Quote content\n\n"
}

func (mt *MarkdownTransformer) isValidOutputFormat(format string) bool {
	validFormats := []string{"github", "commonmark", "notion"}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

func (mt *MarkdownTransformer) isValidLinkFormat(format string) bool {
	validFormats := []string{"markdown", "html", "notion"}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

// Register the Markdown transformer with the global registry.
func init() {
	RegisterGlobal("markdown", func() TransformerPlugin {
		return &MarkdownTransformer{}
	})
}
