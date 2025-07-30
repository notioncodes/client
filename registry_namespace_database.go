package client

import (
	"context"

	"github.com/notioncodes/types"
)

// DatabaseNamespace provides fluent access to database operations.
type DatabaseNamespace struct {
	registry *Registry
}

// Get retrieves a single database by ID.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - databaseID: The ID of the database to retrieve.
//
// Returns:
//   - Result[types.Database]: The database result with metadata.
func (ns *DatabaseNamespace) Get(ctx context.Context, databaseID types.DatabaseID) Result[types.Database] {
	op, err := GetTyped[*Operator[types.Database]](ns.registry, "database")
	if err != nil {
		return Error[types.Database](err)
	}

	req := NewDatabaseGetRequest[types.Database](databaseID)
	return Execute(op, ctx, req)
}

// Query queries a database with filtering and sorting.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - databaseID: The ID of the database to query.
//   - filter: Optional filter criteria.
//   - sorts: Optional sorting criteria.
//
// Returns:
//   - <-chan Result[types.Page]: Channel of page results from the database.
func (ns *DatabaseNamespace) Query(ctx context.Context, databaseID types.DatabaseID, filter, sorts interface{}) <-chan Result[types.Page] {
	paginatedOp := NewPaginatedOperator[types.Page](ns.registry.httpClient, DefaultOperatorConfig())

	req := &DatabaseQueryRequest{
		DatabaseID: databaseID,
		Body: DatabaseQueryRequestBody{
			Filter: filter,
			Sorts:  sorts,
		},
	}

	return StreamPaginated(paginatedOp, ctx, req)
}
