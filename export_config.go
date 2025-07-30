package client

import (
	"time"

	"github.com/notioncodes/types"
)

// ExportConfig contains all configuration options for exporting Notion content.
// This configuration enables fine-tuning of what content is exported and how
// the export process behaves.
type ExportConfig struct {
	// Content specifies what types of content to export
	Content ContentConfig `json:"content" yaml:"content"`

	// Processing controls how the export process behaves
	Processing ProcessingConfig `json:"processing" yaml:"processing"`

	// Timeouts specifies timeout configurations for different operations
	Timeouts TimeoutConfig `json:"timeouts" yaml:"timeouts"`
}

// ContentConfig determines what content is exported and their relationships.
type ContentConfig struct {
	// Types specifies which object types to export
	Types []types.ObjectType `json:"types" yaml:"types"`

	// Databases configuration
	Databases DatabaseConfig `json:"databases" yaml:"databases"`

	// Pages configuration  
	Pages PageConfig `json:"pages" yaml:"pages"`

	// Blocks configuration
	Blocks BlockConfig `json:"blocks" yaml:"blocks"`

	// Comments configuration
	Comments CommentConfig `json:"comments" yaml:"comments"`

	// Users configuration
	Users UserConfig `json:"users" yaml:"users"`
}

// DatabaseConfig controls database export behavior.
type DatabaseConfig struct {
	// IDs specifies specific database IDs to export (if empty, exports all accessible)
	IDs []types.DatabaseID `json:"ids" yaml:"ids"`

	// IncludePages determines if pages from databases should be exported
	IncludePages bool `json:"include_pages" yaml:"include_pages"`

	// IncludeBlocks determines if blocks from database pages should be exported
	IncludeBlocks bool `json:"include_blocks" yaml:"include_blocks"`
}

// PageConfig controls page export behavior.
type PageConfig struct {
	// IDs specifies specific page IDs to export (if empty, exports from databases)
	IDs []types.PageID `json:"ids" yaml:"ids"`

	// IncludeBlocks determines if blocks from pages should be exported
	IncludeBlocks bool `json:"include_blocks" yaml:"include_blocks"`

	// IncludeComments determines if comments from pages should be exported
	IncludeComments bool `json:"include_comments" yaml:"include_comments"`

	// IncludeAttachments determines if attachments should be processed
	IncludeAttachments bool `json:"include_attachments" yaml:"include_attachments"`
}

// BlockConfig controls block export behavior.
type BlockConfig struct {
	// IDs specifies specific block IDs to export (if empty, exports from pages)
	IDs []types.BlockID `json:"ids" yaml:"ids"`

	// IncludeChildren determines if child blocks should be exported recursively
	IncludeChildren bool `json:"include_children" yaml:"include_children"`

	// MaxDepth limits the recursive depth of child blocks (0 = unlimited)
	MaxDepth int `json:"max_depth" yaml:"max_depth"`
}

// CommentConfig controls comment export behavior.
type CommentConfig struct {
	// IncludeUsers determines if user information should be fetched for comments
	IncludeUsers bool `json:"include_users" yaml:"include_users"`
}

// UserConfig controls user export behavior.
type UserConfig struct {
	// IncludeAll determines if all workspace users should be exported
	IncludeAll bool `json:"include_all" yaml:"include_all"`

	// OnlyReferenced determines if only users referenced in exported content should be included
	OnlyReferenced bool `json:"only_referenced" yaml:"only_referenced"`
}

// ProcessingConfig controls the export process behavior.
type ProcessingConfig struct {
	// Workers specifies the number of concurrent workers
	Workers int `json:"workers" yaml:"workers"`

	// ContinueOnError determines if export should continue when errors occur
	ContinueOnError bool `json:"continue_on_error" yaml:"continue_on_error"`

	// BatchSize specifies the size of batches for concurrent processing
	BatchSize int `json:"batch_size" yaml:"batch_size"`
}

// TimeoutConfig specifies timeout configurations for different operations.
type TimeoutConfig struct {
	// Overall timeout for the entire export operation
	Overall time.Duration `json:"overall" yaml:"overall"`

	// RuntimeTimeout for individual resource fetching operations
	Runtime time.Duration `json:"runtime" yaml:"runtime"`

	// RequestTimeout for individual HTTP requests
	Request time.Duration `json:"request" yaml:"request"`
}

// DefaultExportConfig returns a configuration with sensible defaults.
func DefaultExportConfig() *ExportConfig {
	return &ExportConfig{
		Content: ContentConfig{
			Types: []types.ObjectType{
				types.ObjectTypeDatabase,
				types.ObjectTypePage,
				types.ObjectTypeBlock,
			},
			Databases: DatabaseConfig{
				IDs:           nil, // Export all accessible databases
				IncludePages:  true,
				IncludeBlocks: true,
			},
			Pages: PageConfig{
				IDs:                nil, // Export from databases
				IncludeBlocks:      true,
				IncludeComments:    false,
				IncludeAttachments: false,
			},
			Blocks: BlockConfig{
				IDs:             nil, // Export from pages
				IncludeChildren: true,
				MaxDepth:        0, // Unlimited depth
			},
			Comments: CommentConfig{
				IncludeUsers: true,
			},
			Users: UserConfig{
				IncludeAll:     false,
				OnlyReferenced: true,
			},
		},
		Processing: ProcessingConfig{
			Workers:         10,
			ContinueOnError: true,
			BatchSize:       100,
		},
		Timeouts: TimeoutConfig{
			Overall: 30 * time.Minute,
			Runtime: 0, // No timeout by default
			Request: 0, // No timeout by default
		},
	}
}

// Validate ensures the export configuration has valid values.
func (c *ExportConfig) Validate() error {
	if c.Processing.Workers <= 0 {
		c.Processing.Workers = 10
	}

	if c.Processing.BatchSize <= 0 {
		c.Processing.BatchSize = 100
	}

	if len(c.Content.Types) == 0 {
		c.Content.Types = []types.ObjectType{
			types.ObjectTypeDatabase,
			types.ObjectTypePage,
			types.ObjectTypeBlock,
		}
	}

	return nil
}