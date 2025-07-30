package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/notioncodes/types"
)

// ExportDatabaseOptions represents the options for exporting a database.
type ExportDatabaseOptions struct {
	// If true, the pages that are linked to the database will be exported.
	Pages ExportPageOptions `json:"pages" validate:"bool"`
}

// ExportDatabaseResult represents the result of exporting a database.
type ExportDatabaseResult struct {
	// The database object itself.
	Database *types.Database
	// The pages that are linked to the database.
	Pages []*ExportPageResult
}

// ExportPageOptions represents the options for exporting a page.
type ExportPageOptions struct {
	// If true, the blocks that are linked to the page will be exported.
	Blocks ExportBlockOptions `json:"blocks" validate:"bool"`
}

// ExportPageResult represents the result of exporting a page.
type ExportPageResult struct {
	// The page object itself.
	Page *types.Page
	// The blocks that are linked to the page.
	Blocks []*ExportBlockResult
}

// ExportBlockOptions represents the options for exporting a block.
type ExportBlockOptions struct {
	// If true, the children of the block will be exported.
	Children bool `json:"children" validate:"bool"`
	// If true, the comments of the block will be exported.
	Comments ExportCommentOptions `json:"comments" validate:"bool"`
}

// ExportBlockResult represents the result of exporting a block.
type ExportBlockResult struct {
	// The block object itself.
	Block *types.Block
	// The children of the block.
	Children []*types.Block
	// The comments of the block.
	Comments []*ExportCommentResult
}

// ExportCommentOptions represents the options for exporting a comment.
type ExportCommentOptions struct {
	// If true, the complete user object will be fetched that is referenced by the comment.
	User bool `json:"user" validate:"bool"`
}

// ExportCommentResult represents the result of exporting a comment.
type ExportCommentResult struct {
	// The comment object itself.
	Comment *types.Comment
	// The user object that is referenced by the comment.
	User *types.User
}

// ExportService provides methods for exporting Notion objects with their dependencies.
// It supports exporting databases, pages, blocks, and comments with configurable options
// for including related data like children, comments, and user information.
//
// Key features:
//   - Hierarchical export: Database -> Pages -> Blocks -> Comments -> Users
//   - Configurable options for each object type
//   - User caching to avoid duplicate API calls
//   - Concurrent processing for better performance
//   - Graceful error handling (comments/users don't fail the entire export)
//
// Usage:
//
//	service := NewExportService(client)
//	result, err := service.ExportDatabase(ctx, databaseID, options)
type ExportService struct {
	client    *Client
	userCache map[types.UserID]*types.User
	userMu    sync.RWMutex
}

// NewExportService creates a new export service instance.
func NewExportService(client *Client) *ExportService {
	return &ExportService{
		client:    client,
		userCache: make(map[types.UserID]*types.User),
	}
}

// ExportDatabase exports a database with optional related data.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - databaseID: The ID of the database to export.
//   - opts: Options controlling what related data to include.
//
// Returns:
//   - *ExportDatabaseResult: The exported database with related data.
//   - error: Error if the export fails.
//
// Example:
//
//	opts := ExportDatabaseOptions{
//		IncludePages: ExportPageOptions{
//			IncludeBlocks: ExportBlockOptions{
//				IncludeChildren: true,
//				IncludeComments: ExportCommentOptions{IncludeUser: true},
//			},
//		},
//	}
//	result, err := exportService.ExportDatabase(ctx, databaseID, opts)
func (s *ExportService) ExportDatabase(ctx context.Context, databaseID types.DatabaseID, opts ExportDatabaseOptions) (*ExportDatabaseResult, error) {
	// Step 1: Get the database
	dbResult := s.client.Registry.Databases().Get(ctx, databaseID)
	if dbResult.IsError() {
		return nil, fmt.Errorf("failed to get database %s: %w", databaseID, dbResult.Error)
	}

	result := &ExportDatabaseResult{
		Database: &dbResult.Data,
		Pages:    []*ExportPageResult{},
	}

	// Step 2: Get pages if requested (check if any page-related options are enabled)
	if s.shouldIncludePages(opts.Pages) {
		pages, err := s.getDatabasePages(ctx, databaseID)
		if err != nil {
			return result, fmt.Errorf("failed to get database pages: %w", err)
		}

		// Export each page with its dependencies
		var wg sync.WaitGroup
		var mu sync.Mutex
		pageCh := make(chan *types.Page, len(pages))
		errCh := make(chan error, len(pages))

		// Start workers
		for i := 0; i < 5; i++ { // Concurrent workers
			wg.Add(1)
			go func() {
				defer wg.Done()
				for page := range pageCh {
					pageResult, err := s.ExportPage(ctx, page.ID, opts.Pages)
					if err != nil {
						errCh <- fmt.Errorf("failed to export page %s: %w", page.ID, err)
						return
					}
					mu.Lock()
					result.Pages = append(result.Pages, pageResult)
					mu.Unlock()
				}
			}()
		}

		// Send pages to workers
		go func() {
			defer close(pageCh)
			for _, page := range pages {
				select {
				case pageCh <- page:
				case <-ctx.Done():
					return
				}
			}
		}()

		// Wait for completion
		wg.Wait()
		close(errCh)

		// Check for errors
		for err := range errCh {
			return result, err
		}
	}

	return result, nil
}

// ExportPage exports a page with optional related data.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to export.
//   - opts: Options controlling what related data to include.
//
// Returns:
//   - *ExportPageResult: The exported page with related data.
//   - error: Error if the export fails.
//
// Example:
//
//	opts := ExportPageOptions{
//		IncludeBlocks: ExportBlockOptions{
//			IncludeChildren: true,
//			IncludeComments: ExportCommentOptions{IncludeUser: true},
//		},
//	}
//	result, err := exportService.ExportPage(ctx, pageID, opts)
func (s *ExportService) ExportPage(ctx context.Context, pageID types.PageID, opts ExportPageOptions) (*ExportPageResult, error) {
	// Step 1: Get the page
	pageResult := s.client.Registry.Pages().GetSimple(ctx, pageID)
	if pageResult.IsError() {
		return nil, fmt.Errorf("failed to get page %s: %w", pageID, pageResult.Error)
	}

	result := &ExportPageResult{
		Page:   &pageResult.Data,
		Blocks: []*ExportBlockResult{},
	}

	// Step 2: Export blocks if requested (check if any block-related options are enabled)
	if s.shouldIncludeBlocks(opts.Blocks) {
		blocks, err := s.getPageBlocks(ctx, pageID)
		if err != nil {
			return result, fmt.Errorf("failed to get page blocks: %w", err)
		}

		// Export each block with its dependencies
		for _, block := range blocks {
			blockResult, err := s.ExportBlock(ctx, block.ID, opts.Blocks)
			if err != nil {
				return result, fmt.Errorf("failed to export block %s: %w", block.ID, err)
			}
			result.Blocks = append(result.Blocks, blockResult)
		}
	}

	return result, nil
}

// ExportBlock exports a block with optional related data.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to export.
//   - opts: Options controlling what related data to include.
//
// Returns:
//   - *ExportBlockResult: The exported block with related data.
//   - error: Error if the export fails.
//
// Example:
//
//	opts := ExportBlockOptions{
//		IncludeChildren: true,
//		IncludeComments: ExportCommentOptions{IncludeUser: true},
//	}
//	result, err := exportService.ExportBlock(ctx, blockID, opts)
func (s *ExportService) ExportBlock(ctx context.Context, blockID types.BlockID, opts ExportBlockOptions) (*ExportBlockResult, error) {
	// Step 1: Get the block
	blockResult := s.client.Registry.Blocks().GetSimple(ctx, blockID)
	if blockResult.IsError() {
		return nil, fmt.Errorf("failed to get block %s: %w", blockID, blockResult.Error)
	}

	result := &ExportBlockResult{
		Block:    &blockResult.Data,
		Children: []*types.Block{},
		Comments: []*ExportCommentResult{},
	}

	// Step 2: Get children if requested
	if opts.Children {
		children, err := s.getBlockChildren(ctx, blockID)
		if err != nil {
			return result, fmt.Errorf("failed to get block children: %w", err)
		}
		result.Children = children
	}

	// Step 3: Get comments if requested
	if s.shouldIncludeComments(opts.Comments) {
		comments, err := s.getBlockComments(ctx, blockID)
		if err != nil {
			// Don't fail for comment errors, just log and continue
			result.Comments = []*ExportCommentResult{}
		} else {
			// Export each comment with its dependencies
			for _, comment := range comments {
				commentResult, err := s.ExportComment(ctx, comment, opts.Comments)
				if err != nil {
					// Don't fail for individual comment errors
					continue
				}
				result.Comments = append(result.Comments, commentResult)
			}
		}
	}

	return result, nil
}

// ExportComment exports a comment with optional related data.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - comment: The comment to export.
//   - opts: Options controlling what related data to include.
//
// Returns:
//   - *ExportCommentResult: The exported comment with related data.
//   - error: Error if the export fails.
//
// Example:
//
//	opts := ExportCommentOptions{IncludeUser: true}
//	result, err := exportService.ExportComment(ctx, comment, opts)
func (s *ExportService) ExportComment(ctx context.Context, comment *types.Comment, opts ExportCommentOptions) (*ExportCommentResult, error) {
	result := &ExportCommentResult{
		Comment: comment,
		User:    nil,
	}

	// Get user if requested
	if opts.User && comment.CreatedBy != nil {
		user, err := s.getUserCached(ctx, comment.CreatedBy.ID)
		if err != nil {
			// Don't fail for user errors, just continue without user data
			result.User = nil
		} else {
			result.User = user
		}
	}

	return result, nil
}

// Helper methods

// shouldIncludePages determines if pages should be included based on options.
func (s *ExportService) shouldIncludePages(opts ExportPageOptions) bool {
	return s.shouldIncludeBlocks(opts.Blocks)
}

// shouldIncludeBlocks determines if blocks should be included based on options.
func (s *ExportService) shouldIncludeBlocks(opts ExportBlockOptions) bool {
	return opts.Children || s.shouldIncludeComments(opts.Comments)
}

// shouldIncludeComments determines if comments should be included based on options.
// This method checks if the ExportCommentOptions indicates that comments should be retrieved.
// Currently, this means checking if any comment-related option is set.
func (s *ExportService) shouldIncludeComments(opts ExportCommentOptions) bool {
	// We include comments if IncludeUser is true (indicating user wants comment data)
	// In the future, we could add more flags like IncludeReplies, etc.
	return opts.User
}

// getDatabasePages retrieves all pages from a database.
func (s *ExportService) getDatabasePages(ctx context.Context, databaseID types.DatabaseID) ([]*types.Page, error) {
	ch := s.client.Registry.Databases().Query(ctx, databaseID, nil, nil)

	var pages []*types.Page
	for result := range ch {
		if result.IsError() {
			return pages, result.Error
		}
		pages = append(pages, &result.Data)
	}

	return pages, nil
}

// getPageBlocks retrieves all blocks from a page.
func (s *ExportService) getPageBlocks(ctx context.Context, pageID types.PageID) ([]*types.Block, error) {
	ch := s.client.Registry.Blocks().GetChildren(ctx, types.BlockID(pageID.String()))

	var blocks []*types.Block
	for result := range ch {
		if result.IsError() {
			return blocks, result.Error
		}
		blocks = append(blocks, &result.Data)
	}

	return blocks, nil
}

// getBlockChildren retrieves all children of a block.
func (s *ExportService) getBlockChildren(ctx context.Context, blockID types.BlockID) ([]*types.Block, error) {
	ch := s.client.Registry.Blocks().GetChildrenRecursive(ctx, blockID)

	var children []*types.Block
	for result := range ch {
		if result.IsError() {
			return children, result.Error
		}
		children = append(children, &result.Data)
	}

	return children, nil
}

// getBlockComments retrieves all comments from a block.
func (s *ExportService) getBlockComments(ctx context.Context, blockID types.BlockID) ([]*types.Comment, error) {
	return s.client.Registry.Comments().GetAllByBlock(ctx, blockID, &ListCommentsOptions{
		BlockID: &blockID,
	})
}

// getUserCached fetches a user by ID with caching to avoid duplicate requests.
func (s *ExportService) getUserCached(ctx context.Context, userID types.UserID) (*types.User, error) {
	// Check cache first
	s.userMu.RLock()
	if user, exists := s.userCache[userID]; exists {
		s.userMu.RUnlock()
		return user, nil
	}
	s.userMu.RUnlock()

	// Fetch user
	result := s.client.Registry.Users().Get(ctx, userID)
	if result.IsError() {
		return nil, result.Error
	}

	user := &result.Data

	// Cache the user
	s.userMu.Lock()
	s.userCache[userID] = user
	s.userMu.Unlock()

	return user, nil
}

// ClearUserCache clears the user cache.
func (s *ExportService) ClearUserCache() {
	s.userMu.Lock()
	defer s.userMu.Unlock()
	s.userCache = make(map[types.UserID]*types.User)
}

// GetCachedUserCount returns the number of users in the cache.
func (s *ExportService) GetCachedUserCount() int {
	s.userMu.RLock()
	defer s.userMu.RUnlock()
	return len(s.userCache)
}
