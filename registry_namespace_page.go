package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/notioncodes/types"
)

// IDE-friendly namespaces with fluent interfaces for ergonomic access

// PageNamespace provides fluent access to page operations.
type PageNamespace struct {
	registry *Registry
}

// GetPageOptions configures what additional data to retrieve with pages.
type GetPageOptions struct {
	IncludeBlocks      bool `json:"include_blocks"`
	IncludeChildren    bool `json:"include_children"` // For recursive block retrieval
	IncludeComments    bool `json:"include_comments"` // For comments on blocks
	IncludeAttachments bool `json:"include_attachments"`
}

// DefaultGetPageOptions returns default options for page retrieval.
func DefaultGetPageOptions() GetPageOptions {
	return GetPageOptions{
		IncludeBlocks:      false,
		IncludeChildren:    false,
		IncludeComments:    false,
		IncludeAttachments: false,
	}
}

type GetPageResult struct {
	Page        *types.Page    `json:"page"`
	Blocks      []*types.Block `json:"blocks,omitempty"`
	Attachments []*types.File  `json:"attachments,omitempty"`
}

// GetPageResultWithError contains a page result and potential error.
type GetPageResultWithError struct {
	GetPageResult
	Error error `json:"-"`
}

// Get retrieves a single page by ID with optional blocks.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to retrieve.
//   - opts: Options for what additional data to retrieve.
//
// Returns:
//   - Result[GetPageResult]: The page result with associated data.
//
// Example:
//
//	opts := GetPageOptions{IncludeBlocks: true, IncludeChildren: true}
//	result := registry.Pages().Get(ctx, pageID, opts)
//	if result.IsError() {
//	    return result.Error
//	}
//	fmt.Printf("Page: %s, Blocks: %d\n", result.Data.Page.Title, len(result.Data.Blocks))
//
// Note: To retrieve comments on blocks, use BlockNamespace methods with GetBlockOptions{IncludeComments: true}
func (ns *PageNamespace) Get(ctx context.Context, pageID types.PageID, opts GetPageOptions) Result[GetPageResult] {
	resultWithError := ns.getPageWithData(ctx, pageID, opts)
	if resultWithError.Error != nil {
		return Error[GetPageResult](resultWithError.Error)
	}
	return Success(resultWithError.GetPageResult)
}

// GetSimple retrieves a single page by ID without additional data (backwards compatibility).
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to retrieve.
//
// Returns:
//   - Result[types.Page]: The page result.
func (ns *PageNamespace) GetSimple(ctx context.Context, pageID types.PageID) Result[types.Page] {
	op, err := GetTyped[*Operator[types.Page]](ns.registry, "page")
	if err != nil {
		return Error[types.Page](err)
	}

	req := NewPageGetRequest[types.Page](pageID)
	return Execute(op, ctx, req)
}

// GetMany retrieves multiple pages concurrently by their IDs with optional blocks.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageIDs: Slice of page IDs to retrieve.
//   - opts: Options for what additional data to retrieve.
//
// Returns:
//   - <-chan Result[GetPageResult]: Channel of page results with associated data.
//
// Example:
//
//	pageIDs := []types.PageID{pageID1, pageID2, pageID3}
//	opts := GetPageOptions{IncludeBlocks: true, IncludeChildren: true}
//	results := registry.Pages().GetMany(ctx, pageIDs, opts)
//	for result := range results {
//	    if result.IsError() {
//	        log.Printf("Error: %v", result.Error)
//	        continue
//	    }
//	    fmt.Printf("Page: %s, Blocks: %d\n", result.Data.Page.Title, len(result.Data.Blocks))
//	}
//
// Note: To retrieve comments on blocks, use BlockNamespace methods with GetBlockOptions{IncludeComments: true}
func (ns *PageNamespace) GetMany(ctx context.Context, pageIDs []types.PageID, opts GetPageOptions) <-chan Result[GetPageResult] {
	resultCh := make(chan Result[GetPageResult], len(pageIDs))

	go func() {
		defer close(resultCh)

		// Process pages concurrently
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 10) // Limit concurrent requests

		for _, pageID := range pageIDs {
			wg.Add(1)
			go func(id types.PageID) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				select {
				case <-ctx.Done():
					resultCh <- Error[GetPageResult](ctx.Err())
					return
				default:
				}

				pageResultWithError := ns.getPageWithData(ctx, id, opts)
				if pageResultWithError.Error != nil {
					resultCh <- Error[GetPageResult](pageResultWithError.Error)
				} else {
					resultCh <- Success(pageResultWithError.GetPageResult)
				}
			}(pageID)
		}

		wg.Wait()
	}()

	return resultCh
}

// GetManySimple retrieves multiple pages concurrently without additional data (backwards compatibility).
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageIDs: Slice of page IDs to retrieve.
//
// Returns:
//   - <-chan Result[types.Page]: Channel of page results.
func (ns *PageNamespace) GetManySimple(ctx context.Context, pageIDs []types.PageID) <-chan Result[types.Page] {
	op, err := GetTyped[*Operator[types.Page]](ns.registry, "page")
	if err != nil {
		resultCh := make(chan Result[types.Page], 1)
		resultCh <- Error[types.Page](err)
		close(resultCh)
		return resultCh
	}

	reqs := make([]*GetRequest[types.Page], len(pageIDs))
	for i, pageID := range pageIDs {
		reqs[i] = NewPageGetRequest[types.Page](pageID)
	}

	return ExecuteConcurrent(op, ctx, reqs)
}

// getPageWithData is the central method that retrieves a page and its associated data.
// This method handles the core logic for fetching pages with blocks, comments, and attachments.
func (ns *PageNamespace) getPageWithData(ctx context.Context, pageID types.PageID, opts GetPageOptions) GetPageResultWithError {
	result := GetPageResultWithError{}

	// Step 1: Get the base page
	pageResult := ns.GetSimple(ctx, pageID)
	if pageResult.IsError() {
		result.Error = pageResult.Error
		return result
	}
	result.Page = &pageResult.Data

	// Use WaitGroup for concurrent data fetching
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Step 2: Get blocks if requested
	if opts.IncludeBlocks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			blocks, err := ns.getPageBlocks(ctx, pageID, opts.IncludeChildren, opts.IncludeComments)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if result.Error == nil {
					result.Error = fmt.Errorf("failed to get blocks: %w", err)
				}
			} else {
				result.Blocks = blocks
			}
		}()
	}

	// Comments are now handled at the block level when blocks are retrieved with IncludeComments option

	// Step 4: Get attachments if requested
	if opts.IncludeAttachments {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attachments, err := ns.getPageAttachments(ctx, pageID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if result.Error == nil {
					result.Error = fmt.Errorf("failed to get attachments: %w", err)
				}
			} else {
				result.Attachments = attachments
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return result
}

// getPageBlocks retrieves all blocks from a page with optional comments.
func (ns *PageNamespace) getPageBlocks(ctx context.Context, pageID types.PageID, includeChildren bool, includeComments bool) ([]*types.Block, error) {
	// Use BlockNamespace methods with appropriate options
	var ch <-chan Result[GetBlockResult]

	blockOpts := GetBlockOptions{
		IncludeComments: includeComments,
	}

	// TODO: Implement recursive children support when needed
	if includeChildren {
		// For now, just get direct children with options
		ch = ns.registry.Blocks().GetChildrenWithOptions(ctx, types.BlockID(pageID.String()), blockOpts)
	} else {
		ch = ns.registry.Blocks().GetChildrenWithOptions(ctx, types.BlockID(pageID.String()), blockOpts)
	}

	var blocks []*types.Block
	for blockResult := range ch {
		if blockResult.IsError() {
			return blocks, blockResult.Error
		}
		blocks = append(blocks, blockResult.Data.Block)
	}

	return blocks, nil
}

// getPageAttachments retrieves all attachments from a page.
// TODO: Implement when file attachment extraction is available
func (ns *PageNamespace) getPageAttachments(ctx context.Context, pageID types.PageID) ([]*types.File, error) {
	// File attachments would need to be extracted from blocks and page properties
	// This is a placeholder for when attachment extraction is implemented
	return []*types.File{}, nil
}
