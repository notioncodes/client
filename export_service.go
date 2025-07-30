package client

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/notioncodes/types"
)

// ExportDatabaseOptions represents the options for exporting a database.
type ExportDatabaseOptions struct {
	// If true, the pages that are linked to the database will be exported.
	IncludePages ExportPageOptions
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
	IncludeBlocks ExportBlockOptions
}

// ExportPageResult represents the result of exporting a page.
type ExportPageResult struct {
	// The page object itself.
	Page *types.Page
	// The blocks that are linked to the page.
	Blocks []*ExportBlockResult
	// The comments that are linked to the page.
	Comments []*ExportCommentResult
}

// ExportBlockOptions represents the options for exporting a block.
type ExportBlockOptions struct {
	// If true, the children of the block will be exported.
	IncludeChildren bool
	// If true, the comments of the block will be exported.
	IncludeComments ExportCommentOptions
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
	IncludeUser bool
}

// ExportCommentResult represents the result of exporting a comment.
type ExportCommentResult struct {
	// The comment object itself.
	Comment *types.Comment
	// The user object that is referenced by the comment.
	User *types.User
}

// ExportService handles comprehensive export of Notion content with support for
// downstream object retrieval (database->pages->blocks->children, comments->user).
type ExportService struct {
	client *Client
	config *ExportConfig
	result *ExportResult
	mu     sync.RWMutex

	// User cache for avoiding duplicate user fetches
	userCache map[types.UserID]*types.User
	userMu    sync.RWMutex
}

// NewExportService creates a new export service instance.
func NewExportService(client *Client, config *ExportConfig) (*ExportService, error) {
	if config == nil {
		config = DefaultExportConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid export config: %w", err)
	}

	return &ExportService{
		client:    client,
		config:    config,
		result:    NewExportResult(),
		userCache: make(map[types.UserID]*types.User),
	}, nil
}

// Export performs a comprehensive export of Notion content based on the configuration.
// This method handles the complete hierarchy: databases->pages->blocks->children.
func (s *ExportService) Export(ctx context.Context) (*ExportResult, error) {
	// Apply overall timeout if configured
	if s.config.Timeouts.Overall > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeouts.Overall)
		defer cancel()
	}

	s.result.Start = time.Now()
	defer func() {
		s.result.End = time.Now()
	}()

	// Step 1: Export databases if configured
	if slices.Contains(s.config.Content.Types, types.ObjectTypeDatabase) {
		if err := s.exportDatabases(ctx); err != nil && !s.config.Processing.ContinueOnError {
			return s.result, fmt.Errorf("database export failed: %w", err)
		}
	}

	// Step 2: Export pages (from databases or specific IDs) if configured
	if slices.Contains(s.config.Content.Types, types.ObjectTypePage) {
		if err := s.exportPages(ctx); err != nil && !s.config.Processing.ContinueOnError {
			return s.result, fmt.Errorf("page export failed: %w", err)
		}
	}

	// Step 3: Export blocks (from pages or specific IDs) if configured
	if slices.Contains(s.config.Content.Types, types.ObjectTypeBlock) {
		if err := s.exportBlocks(ctx); err != nil && !s.config.Processing.ContinueOnError {
			return s.result, fmt.Errorf("block export failed: %w", err)
		}
	}

	// Step 4: Export comments if configured
	if slices.Contains(s.config.Content.Types, types.ObjectTypeComment) {
		if err := s.exportComments(ctx); err != nil && !s.config.Processing.ContinueOnError {
			return s.result, fmt.Errorf("comment export failed: %w", err)
		}
	}

	// Step 5: Export users if configured
	if slices.Contains(s.config.Content.Types, types.ObjectTypeUser) {
		if err := s.exportUsers(ctx); err != nil && !s.config.Processing.ContinueOnError {
			return s.result, fmt.Errorf("user export failed: %w", err)
		}
	}

	return s.result, nil
}

// exportDatabases exports databases based on configuration.
func (s *ExportService) exportDatabases(ctx context.Context) error {
	var databases []*types.Database
	var err error

	if len(s.config.Content.Databases.IDs) > 0 {
		// Export specific databases
		databases, err = s.getDatabasesByIDs(ctx, s.config.Content.Databases.IDs)
	} else {
		// Export all accessible databases
		databases, err = s.getAllDatabases(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch databases: %w", err)
	}

	// Process databases concurrently
	if err := s.processDatabasesConcurrently(ctx, databases); err != nil {
		return fmt.Errorf("failed to process databases: %w", err)
	}

	return nil
}

// exportPages exports pages based on configuration.
func (s *ExportService) exportPages(ctx context.Context) error {
	var pages []*types.Page
	var err error

	if len(s.config.Content.Pages.IDs) > 0 {
		// Export specific pages
		pages, err = s.getPagesByIDs(ctx, s.config.Content.Pages.IDs)
	} else if s.config.Content.Databases.IncludePages {
		// Export pages from databases (already handled in database export)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to fetch pages: %w", err)
	}

	// Process pages concurrently
	if err := s.processPagesConcurrently(ctx, pages); err != nil {
		return fmt.Errorf("failed to process pages: %w", err)
	}

	return nil
}

// exportBlocks exports blocks based on configuration.
func (s *ExportService) exportBlocks(ctx context.Context) error {
	var blocks []*types.Block
	var err error

	if len(s.config.Content.Blocks.IDs) > 0 {
		// Export specific blocks
		blocks, err = s.getBlocksByIDs(ctx, s.config.Content.Blocks.IDs)
	} else if s.config.Content.Pages.IncludeBlocks {
		// Export blocks from pages (already handled in page export)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to fetch blocks: %w", err)
	}

	// Process blocks concurrently
	if err := s.processBlocksConcurrently(ctx, blocks); err != nil {
		return fmt.Errorf("failed to process blocks: %w", err)
	}

	return nil
}

// exportComments exports comments based on configuration.
func (s *ExportService) exportComments(ctx context.Context) error {
	// Comments are typically exported per page/block during their processing
	// This method handles standalone comment export if needed

	// For now, comments are handled as part of page/block processing
	// Future enhancement: could add specific comment IDs to export config
	return nil
}

// exportUsers exports users based on configuration.
func (s *ExportService) exportUsers(ctx context.Context) error {
	// TODO: Implement when Users().List method is added
	// if s.config.Content.Users.IncludeAll {
	//     return s.exportAllUsers(ctx)
	// }

	if s.config.Content.Users.OnlyReferenced {
		return s.exportReferencedUsers(ctx)
	}

	return nil
}

// processDatabasesConcurrently processes databases with concurrent workers.
func (s *ExportService) processDatabasesConcurrently(ctx context.Context, databases []*types.Database) error {
	if len(databases) == 0 {
		return nil
	}

	// Create worker pool
	dbCh := make(chan *types.Database, len(databases))
	errCh := make(chan error, len(databases))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.config.Processing.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for db := range dbCh {
				if err := s.processSingleDatabase(ctx, db); err != nil {
					errCh <- fmt.Errorf("database %s: %w", db.ID, err)
				} else {
					s.recordSuccess(types.ObjectTypeDatabase, db.ID.String())
				}
			}
		}()
	}

	// Send databases to workers
	go func() {
		defer close(dbCh)
		for _, db := range databases {
			select {
			case dbCh <- db:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
		s.recordError(types.ObjectTypeDatabase, "", err)
	}

	if len(errors) > 0 && !s.config.Processing.ContinueOnError {
		return fmt.Errorf("database processing failed: %d errors occurred", len(errors))
	}

	return nil
}

// processSingleDatabase processes a single database and its downstream content.
func (s *ExportService) processSingleDatabase(ctx context.Context, database *types.Database) error {
	// Export pages from database if configured
	if s.config.Content.Databases.IncludePages {
		pages, err := s.getPagesFromDatabase(ctx, database.ID)
		if err != nil {
			return fmt.Errorf("failed to get pages from database: %w", err)
		}

		if err := s.processPagesConcurrently(ctx, pages); err != nil {
			return fmt.Errorf("failed to process database pages: %w", err)
		}
	}

	return nil
}

// processPagesConcurrently processes pages with concurrent workers.
func (s *ExportService) processPagesConcurrently(ctx context.Context, pages []*types.Page) error {
	if len(pages) == 0 {
		return nil
	}

	// Create worker pool
	pageCh := make(chan *types.Page, len(pages))
	errCh := make(chan error, len(pages))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.config.Processing.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for page := range pageCh {
				if err := s.processSinglePage(ctx, page); err != nil {
					errCh <- fmt.Errorf("page %s: %w", page.ID, err)
				} else {
					s.recordSuccess(types.ObjectTypePage, page.ID.String())
				}
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
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
		s.recordError(types.ObjectTypePage, "", err)
	}

	if len(errors) > 0 && !s.config.Processing.ContinueOnError {
		return fmt.Errorf("page processing failed: %d errors occurred", len(errors))
	}

	return nil
}

// processSinglePage processes a single page and its downstream content.
func (s *ExportService) processSinglePage(ctx context.Context, page *types.Page) error {
	// Export blocks from page if configured
	if s.config.Content.Pages.IncludeBlocks {
		blocks, err := s.getBlocksFromPage(ctx, page.ID)
		if err != nil {
			return fmt.Errorf("failed to get blocks from page: %w", err)
		}

		if err := s.processBlocksConcurrently(ctx, blocks); err != nil {
			return fmt.Errorf("failed to process page blocks: %w", err)
		}
	}

	// TODO: Export comments from page when Comments namespace is implemented
	// if s.config.Content.Pages.IncludeComments {
	//     comments, err := s.getCommentsFromPage(ctx, page.ID)
	//     if err != nil {
	//         return fmt.Errorf("failed to get comments from page: %w", err)
	//     }
	//     if err := s.processCommentsConcurrently(ctx, comments); err != nil {
	//         return fmt.Errorf("failed to process page comments: %w", err)
	//     }
	// }

	return nil
}

// processBlocksConcurrently processes blocks with concurrent workers.
func (s *ExportService) processBlocksConcurrently(ctx context.Context, blocks []*types.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// Process in batches to avoid overwhelming the system
	batchSize := s.config.Processing.BatchSize
	for i := 0; i < len(blocks); i += batchSize {
		end := i + batchSize
		if end > len(blocks) {
			end = len(blocks)
		}

		batch := blocks[i:end]

		// Process batch concurrently
		var wg sync.WaitGroup
		for _, block := range batch {
			wg.Add(1)
			go func(b *types.Block) {
				defer wg.Done()
				if err := s.processSingleBlock(ctx, b); err != nil {
					s.recordError(types.ObjectTypeBlock, b.ID.String(), err)
				} else {
					s.recordSuccess(types.ObjectTypeBlock, b.ID.String())
				}
			}(block)
		}
		wg.Wait()
	}

	return nil
}

// processSingleBlock processes a single block.
func (s *ExportService) processSingleBlock(ctx context.Context, block *types.Block) error {
	// Blocks are processed as part of the recursive children fetch
	// Additional processing can be added here if needed
	return nil
}

// TODO: Implement comment processing when Comments namespace is added
// processCommentsConcurrently processes comments with concurrent workers.
// func (s *ExportService) processCommentsConcurrently(ctx context.Context, comments []*types.Comment) error {
// 	return nil
// }

// processSingleComment processes a single comment and fetches associated user if needed.
// func (s *ExportService) processSingleComment(ctx context.Context, comment *types.Comment) error {
// 	return nil
// }

// recordSuccess records a successful export.
func (s *ExportService) recordSuccess(objectType types.ObjectType, objectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result.Success[objectType]++
}

// recordError records an export error.
func (s *ExportService) recordError(objectType types.ObjectType, objectID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result.Errors = append(s.result.Errors, ExportError{
		ObjectType: objectType,
		ObjectID:   objectID,
		Error:      err.Error(),
		Timestamp:  time.Now(),
	})
}

// GetResult returns the current export result.
func (s *ExportService) GetResult() *ExportResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.result
}

// Helper methods for data fetching

// getAllDatabases fetches all accessible databases.
func (s *ExportService) getAllDatabases(ctx context.Context) ([]*types.Database, error) {
	searchCtx := ctx
	var cancel context.CancelFunc
	if s.config.Timeouts.Runtime > 0 {
		searchCtx, cancel = context.WithTimeout(ctx, s.config.Timeouts.Runtime)
	}
	if cancel != nil {
		defer cancel()
	}

	ch := s.client.Registry.Search().Stream(searchCtx, types.SearchRequest{
		Query: "",
		Filter: &types.SearchFilter{
			Property: "object",
			Value:    "database",
		},
	})

	var databases []*types.Database
	for result := range ch {
		if result.Error != nil {
			return databases, result.Error
		}
		if result.Data.Database != nil {
			databases = append(databases, result.Data.Database)
		}
	}

	return databases, nil
}

// getDatabasesByIDs fetches specific databases by their IDs.
func (s *ExportService) getDatabasesByIDs(ctx context.Context, ids []types.DatabaseID) ([]*types.Database, error) {
	var databases []*types.Database
	for _, id := range ids {
		db, err := s.getDatabase(ctx, id)
		if err != nil {
			return databases, fmt.Errorf("failed to get database %s: %w", id, err)
		}
		databases = append(databases, db)
	}
	return databases, nil
}

// getDatabase fetches a single database by ID.
func (s *ExportService) getDatabase(ctx context.Context, databaseID types.DatabaseID) (*types.Database, error) {
	result, err := ToGoResult(s.client.Registry.Databases().Get(ctx, databaseID))
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// getPagesFromDatabase fetches all pages from a database.
func (s *ExportService) getPagesFromDatabase(ctx context.Context, databaseID types.DatabaseID) ([]*types.Page, error) {
	queryCtx := ctx
	var cancel context.CancelFunc
	if s.config.Timeouts.Runtime > 0 {
		queryCtx, cancel = context.WithTimeout(ctx, s.config.Timeouts.Runtime)
	}
	if cancel != nil {
		defer cancel()
	}

	ch := s.client.Registry.Databases().Query(queryCtx, databaseID, nil, nil)

	var pages []*types.Page
	for result := range ch {
		if result.Error != nil {
			return pages, result.Error
		}
		pages = append(pages, &result.Data)
	}

	return pages, nil
}

// getPagesByIDs fetches specific pages by their IDs.
func (s *ExportService) getPagesByIDs(ctx context.Context, ids []types.PageID) ([]*types.Page, error) {
	var pages []*types.Page
	for _, id := range ids {
		page, err := s.getPage(ctx, id)
		if err != nil {
			return pages, fmt.Errorf("failed to get page %s: %w", id, err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// getPage fetches a single page by ID.
func (s *ExportService) getPage(ctx context.Context, pageID types.PageID) (*types.Page, error) {
	result := s.client.Registry.Pages().GetSimple(ctx, pageID)
	if result.IsError() {
		return nil, result.Error
	}
	return &result.Data, nil
}

// getBlocksFromPage fetches all blocks from a page.
func (s *ExportService) getBlocksFromPage(ctx context.Context, pageID types.PageID) ([]*types.Block, error) {
	var ch <-chan Result[types.Block]

	if s.config.Content.Blocks.IncludeChildren {
		ch = s.client.Registry.Blocks().GetChildrenRecursive(ctx, types.BlockID(pageID.String()))
	} else {
		ch = s.client.Registry.Blocks().GetChildren(ctx, types.BlockID(pageID.String()))
	}

	var blocks []*types.Block
	for result := range ch {
		if result.Error != nil {
			return blocks, result.Error
		}
		blocks = append(blocks, &result.Data)
	}

	return blocks, nil
}

// getBlocksByIDs fetches specific blocks by their IDs.
func (s *ExportService) getBlocksByIDs(ctx context.Context, ids []types.BlockID) ([]*types.Block, error) {
	var blocks []*types.Block
	for _, id := range ids {
		block, err := s.getBlock(ctx, id)
		if err != nil {
			return blocks, fmt.Errorf("failed to get block %s: %w", id, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// getBlock fetches a single block by ID.
func (s *ExportService) getBlock(ctx context.Context, blockID types.BlockID) (*types.Block, error) {
	result := s.client.Registry.Blocks().GetSimple(ctx, blockID)
	if result.IsError() {
		return nil, result.Error
	}
	return &result.Data, nil
}

// TODO: Implement when Comments namespace is added to client registry
// getCommentsFromPage fetches all comments from a page.
// func (s *ExportService) getCommentsFromPage(ctx context.Context, pageID types.PageID) ([]*types.Comment, error) {
// 	return nil, nil
// }

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

// TODO: Implement when Users().List method is added to client registry
// exportAllUsers exports all workspace users.
// func (s *ExportService) exportAllUsers(ctx context.Context) error {
// 	return nil
// }

// exportReferencedUsers exports users that were referenced during the export.
func (s *ExportService) exportReferencedUsers(ctx context.Context) error {
	// Process cached users from the userCache
	s.userMu.RLock()
	defer s.userMu.RUnlock()

	for _, user := range s.userCache {
		s.recordSuccess(types.ObjectTypeUser, user.ID.String())
	}

	return nil
}
