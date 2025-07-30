package client

import (
	"context"
	"net/url"
	"strconv"

	"github.com/notioncodes/types"
)

// BlockNamespace provides fluent access to block operations.
type BlockNamespace struct {
	registry *Registry
}

// Get retrieves a single block by ID.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to retrieve.
//
// Returns:
//   - Result[types.Block]: The block result with metadata.
func (ns *BlockNamespace) Get(ctx context.Context, blockID types.BlockID) Result[types.Block] {
	op, err := GetTyped[*Operator[types.Block]](ns.registry, "block")
	if err != nil {
		return Error[types.Block](err)
	}

	req := NewBlockGetRequest[types.Block](blockID)
	return Execute(op, ctx, req)
}

// GetMany retrieves multiple blocks concurrently by their IDs.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockIDs: Slice of block IDs to retrieve.
//
// Returns:
//   - <-chan Result[types.Block]: Channel of block results.
func (ns *BlockNamespace) GetMany(ctx context.Context, blockIDs []types.BlockID) <-chan Result[types.Block] {
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
