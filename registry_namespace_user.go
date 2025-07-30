package client

import (
	"context"

	"github.com/notioncodes/types"
)

// UserNamespace provides fluent access to user operations.
type UserNamespace struct {
	registry *Registry
}

// Get retrieves a single user by ID.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - userID: The ID of the user to retrieve.
//
// Returns:
//   - Result[types.User]: The user result with metadata.
func (ns *UserNamespace) Get(ctx context.Context, userID types.UserID) Result[types.User] {
	op, err := GetTyped[*Operator[types.User]](ns.registry, "user")
	if err != nil {
		return Error[types.User](err)
	}

	req := NewUserGetRequest[types.User](userID)
	return Execute(op, ctx, req)
}

// GetMany retrieves multiple users concurrently by their IDs.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - userIDs: Slice of user IDs to retrieve.
//
// Returns:
//   - <-chan Result[types.User]: Channel of user results.
func (ns *UserNamespace) GetMany(ctx context.Context, userIDs []types.UserID) <-chan Result[types.User] {
	op, err := GetTyped[*Operator[types.User]](ns.registry, "user")
	if err != nil {
		resultCh := make(chan Result[types.User], 1)
		resultCh <- Error[types.User](err)
		close(resultCh)
		return resultCh
	}

	reqs := make([]*GetRequest[types.User], len(userIDs))
	for i, userID := range userIDs {
		reqs[i] = NewUserGetRequest[types.User](userID)
	}

	return ExecuteConcurrent(op, ctx, reqs)
}
