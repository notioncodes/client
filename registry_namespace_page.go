package client

import (
	"context"

	"github.com/notioncodes/types"
)

// IDE-friendly namespaces with fluent interfaces for ergonomic access

// PageNamespace provides fluent access to page operations.
type PageNamespace struct {
	registry *Registry
}

// Get retrieves a single page by ID.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to retrieve.
//
// Returns:
//   - Result[types.Page]: The page result with metadata.
//
// Example:
//
//	page := registry.Pages().Get(ctx, pageID)
//	if page.IsError() {
//	    return page.Error
//	}
//	fmt.Println(page.Data.Title)
func (ns *PageNamespace) Get(ctx context.Context, pageID types.PageID) Result[types.Page] {
	op, err := GetTyped[*Operator[types.Page]](ns.registry, "page")
	if err != nil {
		return Error[types.Page](err)
	}

	req := NewPageGetRequest[types.Page](pageID)
	return Execute(op, ctx, req)
}

// GetMany retrieves multiple pages concurrently by their IDs.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageIDs: Slice of page IDs to retrieve.
//
// Returns:
//   - <-chan Result[types.Page]: Channel of page results.
//
// Example:
//
//	pageIDs := []types.PageID{pageID1, pageID2, pageID3}
//	results := registry.Pages().GetMany(ctx, pageIDs)
//	for result := range results {
//	    if result.IsError() {
//	        log.Printf("Error: %v", result.Error)
//	        continue
//	    }
//	    fmt.Printf("Page: %s\n", result.Data.Title)
//	}
func (ns *PageNamespace) GetMany(ctx context.Context, pageIDs []types.PageID) <-chan Result[types.Page] {
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
