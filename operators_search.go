package client

import (
	"context"
	"net/url"

	"github.com/notioncodes/types"
)

// SearchOperator handles search operations with complex filtering and pagination.
type SearchOperator[T any] struct {
	client *HTTPClient
	config *OperatorConfig
}

// func (op *SearchOperator[T]) Query(ctx context.Context, req types.SearchRequest) (<-chan Result[T], error) {
// 	_, err := GetTyped[*SearchOperator[T]](op.registry, "search")
// 	if err != nil {
// 		resultCh := make(chan Result[T], 1)
// 		resultCh <- Error[T](err)
// 		close(resultCh)
// 		return resultCh
// 	}

// 	return op.Stream(ctx, req), nil
// }

// SearchRequest wraps types.SearchRequest to implement PaginatedRequestInterface.
type SearchRequest struct {
	types.SearchRequest
}

func (sr *SearchRequest) GetPath() string {
	return "/search"
}

func (sr *SearchRequest) GetMethod() string {
	return "POST"
}

func (sr *SearchRequest) GetBody() types.SearchRequest {
	return sr.SearchRequest
}

func (sr *SearchRequest) GetQuery() url.Values {
	return nil
}

func (sr *SearchRequest) Validate() error {
	// Search requests are generally always valid
	return nil
}

func (sr *SearchRequest) SetStartCursor(cursor *string) {
	sr.StartCursor = cursor
}

func (sr *SearchRequest) SetPageSize(pageSize *int) {
	sr.PageSize = pageSize
}

func NewSearchRequestWrapper(req types.SearchRequest) PaginatedRequestInterface[types.SearchRequest] {
	return &SearchRequest{SearchRequest: req}
}

// NewSearchOperator creates a new search operator.
//
// Arguments:
//   - client: HTTP client for making requests.
//   - config: Optional configuration (uses defaults if nil).
//
// Returns:
//   - *SearchOperator[T]: A new search operator.
func NewSearchOperator[T any](client *HTTPClient, config *OperatorConfig) *SearchOperator[T] {
	if config == nil {
		config = DefaultOperatorConfig()
	}

	return &SearchOperator[T]{
		client: client,
		config: config,
	}
}

// Execute performs a search operation and returns all results.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: The search request to execute.
//
// Returns:
//   - Result[[]T]: The result of the search operation.
func (so *SearchOperator[T]) Execute(ctx context.Context, req types.SearchRequest) Result[[]T] {
	paginatedOp := NewPaginatedOperator[T](so.client, so.config)
	return ExecutePaginated(paginatedOp, ctx, NewSearchRequestWrapper(req))
}

// Stream performs a search operation with pagination and streams results.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: The search request to execute.
//
// Returns:
//   - <-chan Result[T]: A channel of results.
func (so *SearchOperator[T]) Stream(ctx context.Context, req types.SearchRequest) <-chan Result[T] {
	// Search operations are essentially paginated POST requests
	paginatedOp := NewPaginatedOperator[T](so.client, so.config)
	return StreamPaginated(paginatedOp, ctx, NewSearchRequestWrapper(req))
}

// ExecuteConcurrent performs multiple search operations concurrently.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - reqs: The search requests to execute.
//
// Returns:
//   - <-chan Result[T]: A channel of results.
func (so *SearchOperator[T]) ExecuteConcurrent(ctx context.Context, reqs []types.SearchRequest) <-chan Result[T] {
	paginatedOp := NewPaginatedOperator[T](so.client, so.config)
	wrappers := make([]PaginatedRequestInterface[types.SearchRequest], len(reqs))
	for i, req := range reqs {
		wrappers[i] = NewSearchRequestWrapper(req)
	}
	return ExecuteConcurrentPaginated(paginatedOp, ctx, wrappers)
}
