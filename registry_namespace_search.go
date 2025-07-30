package client

import (
	"context"

	"github.com/notioncodes/types"
)

// SearchNamespace provides fluent access to search operations.
type SearchNamespace struct {
	registry *Registry
}

// Query performs a search query across the workspace.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: Search request.
//
// Returns:
//   - <-chan Result[SearchResult]: Channel of search results.
func (ns *SearchNamespace) Query(ctx context.Context, req SearchRequest) <-chan Result[types.SearchResult] {
	paginatedOp := NewPaginatedOperator[types.SearchResult](ns.registry.httpClient, DefaultOperatorConfig())

	r := &SearchRequest{
		SearchRequest: types.SearchRequest{
			Filter: &types.SearchFilter{
				Property: "object",
				Value:    "database",
			},
		},
	}

	return StreamPaginated(paginatedOp, ctx, r)
}

// QueryAll performs a search query and returns all results as a slice.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: Search request.
//
// Returns:
//   - Result[[]SearchResult]: All search results as a slice.
func (ns *SearchNamespace) QueryAll(ctx context.Context, req types.SearchRequest) Result[[]types.SearchResult] {
	// Check if search operator is available
	_, err := GetTyped[*SearchOperator[types.SearchResult]](ns.registry, "search")
	if err != nil {
		return Error[[]types.SearchResult](err)
	}

	searchOp := NewSearchOperator[types.SearchResult](ns.registry.httpClient, DefaultOperatorConfig())

	return searchOp.Execute(ctx, req)
}
