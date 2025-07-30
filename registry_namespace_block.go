package client

import (
	"context"
	"net/url"
	"strconv"
	"sync"

	"github.com/notioncodes/types"
)

// BlockNamespace provides fluent access to block operations.
type BlockNamespace struct {
	registry *Registry
}

// GetBlockOptions configures what additional data to retrieve with blocks.
type GetBlockOptions struct {
	IncludeComments bool `json:"include_comments"`
}

// GetBlockResult contains a block and its associated data.
type GetBlockResult struct {
	Block    *types.Block     `json:"block"`
	Comments []*types.Comment `json:"comments,omitempty"`
}

// GetBlockResultWithError contains a block result and potential error.
type GetBlockResultWithError struct {
	GetBlockResult
	Error error `json:"-"`
}

// DefaultGetBlockOptions returns default options for block retrieval.
func DefaultGetBlockOptions() GetBlockOptions {
	return GetBlockOptions{
		IncludeComments: false,
	}
}

// Get retrieves a single block by ID with optional comments.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to retrieve.
//   - opts: Options for what additional data to retrieve.
//
// Returns:
//   - Result[GetBlockResult]: The block result with associated data.
//
// Example:
//
//	opts := GetBlockOptions{IncludeComments: true}
//	result := registry.Blocks().Get(ctx, blockID, opts)
//	if result.IsError() {
//	    return result.Error
//	}
//	fmt.Printf("Block has %d comments\n", len(result.Data.Comments))
func (ns *BlockNamespace) Get(ctx context.Context, blockID types.BlockID, opts GetBlockOptions) Result[GetBlockResult] {
	resultWithError := ns.getBlockWithData(ctx, blockID, opts)
	if resultWithError.Error != nil {
		return Error[GetBlockResult](resultWithError.Error)
	}
	return Success(resultWithError.GetBlockResult)
}

// GetSimple retrieves a single block by ID without additional data (backwards compatibility).
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to retrieve.
//
// Returns:
//   - Result[types.Block]: The block result.
func (ns *BlockNamespace) GetSimple(ctx context.Context, blockID types.BlockID) Result[types.Block] {
	op, err := GetTyped[*Operator[types.Block]](ns.registry, "block")
	if err != nil {
		return Error[types.Block](err)
	}

	req := NewBlockGetRequest[types.Block](blockID)
	return Execute(op, ctx, req)
}

// GetMany retrieves multiple blocks concurrently by their IDs with optional comments.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockIDs: Slice of block IDs to retrieve.
//   - opts: Options for what additional data to retrieve.
//
// Returns:
//   - <-chan Result[GetBlockResult]: Channel of block results with associated data.
//
// Example:
//
//	blockIDs := []types.BlockID{blockID1, blockID2, blockID3}
//	opts := GetBlockOptions{IncludeComments: true}
//	results := registry.Blocks().GetMany(ctx, blockIDs, opts)
//	for result := range results {
//	    if result.IsError() {
//	        log.Printf("Error: %v", result.Error)
//	        continue
//	    }
//	    fmt.Printf("Block has %d comments\n", len(result.Data.Comments))
//	}
func (ns *BlockNamespace) GetMany(ctx context.Context, blockIDs []types.BlockID, opts GetBlockOptions) <-chan Result[GetBlockResult] {
	resultCh := make(chan Result[GetBlockResult], len(blockIDs))

	go func() {
		defer close(resultCh)

		// Process blocks concurrently
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 10) // Limit concurrent requests

		for _, blockID := range blockIDs {
			wg.Add(1)
			go func(id types.BlockID) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				select {
				case <-ctx.Done():
					resultCh <- Error[GetBlockResult](ctx.Err())
					return
				default:
				}

				blockResultWithError := ns.getBlockWithData(ctx, id, opts)
				if blockResultWithError.Error != nil {
					resultCh <- Error[GetBlockResult](blockResultWithError.Error)
				} else {
					resultCh <- Success(blockResultWithError.GetBlockResult)
				}
			}(blockID)
		}

		wg.Wait()
	}()

	return resultCh
}

func (ns *BlockNamespace) GetWithOptions(ctx context.Context, blockID types.BlockID, opts GetBlockOptions) Result[GetBlockResult] {
	blockResult := ns.GetSimple(ctx, blockID)
	if blockResult.IsError() {
		return Error[GetBlockResult](blockResult.Error)
	}

	result := GetBlockResult{
		Block:    &blockResult.Data,
		Comments: nil,
	}

	// Get comments for this block only if requested
	if opts.IncludeComments {
		comments, err := ns.getBlockComments(ctx, blockID)
		if err != nil {
			// If comment retrieval fails, still return the block without comments
			// Don't fail the entire request for comment errors
			result.Comments = []*types.Comment{}
		} else {
			result.Comments = comments
		}
	}

	return Success(result)
}

// GetManySimple retrieves multiple blocks concurrently without additional data (backwards compatibility).
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockIDs: Slice of block IDs to retrieve.
//
// Returns:
//   - <-chan Result[types.Block]: Channel of block results.
func (ns *BlockNamespace) GetManySimple(ctx context.Context, blockIDs []types.BlockID) <-chan Result[types.Block] {
	op, err := GetTyped[*Operator[types.Block]](ns.registry, "block")
	if err != nil {
		resultCh := make(chan Result[types.Block], 1)
		resultCh <- Error[types.Block](err)
		close(resultCh)
		return resultCh
	}

	reqs := make([]*GetRequest[types.Block], len(blockIDs))
	for i, blockID := range blockIDs {
		reqs[i] = NewBlockGetRequest[types.Block](blockID)
	}

	return ExecuteConcurrent(op, ctx, reqs)
}

// GetChildren retrieves all child blocks of a parent block with pagination support.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the parent block.
//
// Returns:
//   - <-chan Result[types.Block]: Channel of child block results.
func (ns *BlockNamespace) GetChildren(ctx context.Context, blockID types.BlockID) <-chan Result[types.Block] {
	paginatedOp := NewPaginatedOperator[types.Block](ns.registry.httpClient, DefaultOperatorConfig())

	req := &BlockChildrenListRequest{
		BlockID: blockID,
	}

	return StreamPaginated(paginatedOp, ctx, req)
}

// GetChildrenWithOptions retrieves all child blocks with optional comments.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the parent block.
//   - opts: Options for what additional data to retrieve.
//
// Returns:
//   - <-chan Result[GetBlockResult]: Channel of child block results with associated data.
func (ns *BlockNamespace) GetChildrenWithOptions(ctx context.Context, blockID types.BlockID, opts GetBlockOptions) <-chan Result[GetBlockResult] {
	resultCh := make(chan Result[GetBlockResult], 100)

	go func() {
		defer close(resultCh)

		// Get basic children first
		childrenCh := ns.GetChildren(ctx, blockID)

		// Process each child and potentially add comments
		for result := range childrenCh {
			if result.IsError() {
				resultCh <- Error[GetBlockResult](result.Error)
				continue
			}

			// If comments are not requested, just wrap the block
			if !opts.IncludeComments {
				blockResult := GetBlockResult{
					Block:    &result.Data,
					Comments: nil,
				}
				resultCh <- Success(blockResult)
				continue
			}

			// Get comments for this block
			comments, err := ns.getBlockComments(ctx, result.Data.ID)
			if err != nil {
				// If comment retrieval fails, still return the block without comments
				blockResult := GetBlockResult{
					Block:    &result.Data,
					Comments: []*types.Comment{},
				}
				resultCh <- Success(blockResult)
				continue
			}

			blockResult := GetBlockResult{
				Block:    &result.Data,
				Comments: comments,
			}
			resultCh <- Success(blockResult)
		}
	}()

	return resultCh
}

// GetChildrenRecursive retrieves all child blocks recursively, including nested children.
// This method efficiently handles blocks with has_children=true by recursively fetching
// their child blocks and flattening the result into a single channel.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the parent block.
//
// Returns:
//   - <-chan Result[types.Block]: Channel of all descendant block results.
func (ns *BlockNamespace) GetChildrenRecursive(ctx context.Context, blockID types.BlockID) <-chan Result[types.Block] {
	resultCh := make(chan Result[types.Block], 100)

	go func() {
		defer close(resultCh)
		ns.fetchChildrenRecursive(ctx, blockID, resultCh)
	}()

	return resultCh
}

// fetchChildrenRecursive is the internal recursive function that fetches children
func (ns *BlockNamespace) fetchChildrenRecursive(ctx context.Context, blockID types.BlockID, resultCh chan<- Result[types.Block]) {
	// Get immediate children
	childrenCh := ns.GetChildren(ctx, blockID)

	childCount := 0
	recursiveCount := 0
	for result := range childrenCh {
		if result.IsError() {
			resultCh <- result
			return
		}

		childCount++
		// Send the current block to the result channel
		resultCh <- result

		// If this block has children, recursively fetch them
		if result.Data.HasChildren {
			recursiveCount++
			ns.fetchChildrenRecursive(ctx, result.Data.ID, resultCh)
		}
	}
}

// BlockChildrenListRequest represents a request to list block children.
type BlockChildrenListRequest struct {
	BlockID     types.BlockID `json:"-"`
	StartCursor *string       `json:"-"`
	PageSize    *int          `json:"-"`
}

// GetPath returns the API path for the block children list request.
func (r *BlockChildrenListRequest) GetPath() string {
	return "/blocks/" + string(r.BlockID) + "/children"
}

// GetMethod returns the HTTP method for the block children list request.
func (r *BlockChildrenListRequest) GetMethod() string {
	return "GET"
}

// GetBody returns nil as this is a GET request.
func (r *BlockChildrenListRequest) GetBody() interface{} {
	return nil
}

// SetStartCursor sets the pagination start cursor.
func (r *BlockChildrenListRequest) SetStartCursor(cursor *string) {
	r.StartCursor = cursor
}

// SetPageSize sets the pagination page size.
func (r *BlockChildrenListRequest) SetPageSize(pageSize *int) {
	r.PageSize = pageSize
}

// GetQuery returns the query parameters for the block children list request.
func (r *BlockChildrenListRequest) GetQuery() url.Values {
	query := url.Values{}
	if r.StartCursor != nil {
		query.Set("start_cursor", *r.StartCursor)
	}
	if r.PageSize != nil {
		query.Set("page_size", strconv.Itoa(*r.PageSize))
	}
	return query
}

// Validate validates the block children list request.
func (r *BlockChildrenListRequest) Validate() error {
	return r.BlockID.Validate()
}

// getBlockWithData is the central method that retrieves a block and its associated data.
// This method handles the core logic for fetching blocks with comments.
func (ns *BlockNamespace) getBlockWithData(ctx context.Context, blockID types.BlockID, opts GetBlockOptions) GetBlockResultWithError {
	result := GetBlockResultWithError{}

	// Step 1: Get the base block
	blockResult := ns.GetSimple(ctx, blockID)
	if blockResult.IsError() {
		result.Error = blockResult.Error
		return result
	}
	result.Block = &blockResult.Data

	// Step 2: Get comments if requested
	if opts.IncludeComments {
		comments, err := ns.getBlockComments(ctx, blockID)
		if err != nil {
			// If comment retrieval fails, still return the block without comments
			// Don't fail the entire request for comment errors
			result.Comments = []*types.Comment{}
		} else {
			result.Comments = comments
		}
	}

	return result
}

// getBlockComments retrieves all comments from a block.
func (ns *BlockNamespace) getBlockComments(ctx context.Context, blockID types.BlockID) ([]*types.Comment, error) {
	return ns.registry.Comments().GetAllByBlock(ctx, blockID, &ListCommentsOptions{
		BlockID: &blockID,
	})
}
