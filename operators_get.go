package client

import (
	"context"
	"fmt"
	"net/url"

	"github.com/notioncodes/types"
)

// GetRequestBody represents an empty request body for GET requests.
type GetRequestBody struct{}

// GetRequest represents a generic request to get a single resource by ID.
// This unified request type works for all Notion resource types.
//
// Arguments:
//   - T: The type of resource to retrieve.
type GetRequest[T any] struct {
	ResourceType   types.ResourceType `json:"resource_type"`
	ResourceID     string             `json:"resource_id"`
	ParentID       string             `json:"parent_id,omitempty"`       // For properties (database ID)
	AdditionalPath string             `json:"additional_path,omitempty"` // For complex paths
}

// GetResourceID returns the resource ID.
//
// Returns:
//   - string: The resource ID.
func (gr *GetRequest[T]) GetResourceID() string {
	return gr.ResourceID
}

// GetResourceType returns the resource type as a string.
//
// Returns:
//   - string: The resource type as a string.
func (gr *GetRequest[T]) GetResourceType() string {
	return string(gr.ResourceType)
}

// GetPath returns the API path for the resource.
//
// Returns:
//   - string: The API path for the resource.
func (gr *GetRequest[T]) GetPath() string {
	switch gr.ResourceType {
	case types.ResourceTypePage:
		return "/pages/" + gr.ResourceID
	case types.ResourceTypeDatabase:
		return "/databases/" + gr.ResourceID
	case types.ResourceTypeBlock:
		return "/blocks/" + gr.ResourceID
	case types.ResourceTypeUser:
		return "/users/" + gr.ResourceID
	case types.ResourceTypeProperty:
		if gr.ParentID == "" {
			return fmt.Sprintf("/properties/%s", gr.ResourceID)
		}
		return fmt.Sprintf("/databases/%s/properties/%s", gr.ParentID, gr.ResourceID)
	default:
		return "/" + string(gr.ResourceType) + "s/" + gr.ResourceID
	}
}

// GetMethod returns the HTTP method (always GET for resource retrieval).
//
// Returns:
//   - string: The HTTP method.
func (gr *GetRequest[T]) GetMethod() string {
	return "GET"
}

// GetBody returns the request body (nil for GET requests).
//
// Returns:
//   - GetRequestBody: The request body.
func (gr *GetRequest[T]) GetBody() GetRequestBody {
	return GetRequestBody{}
}

// GetQuery returns query parameters for the request.
//
// Returns:
//   - url.Values: Query parameters for the request.
func (gr *GetRequest[T]) GetQuery() url.Values {
	return nil
}

// Validate ensures the request is valid.
//
// Returns:
//   - error: An error if the request is invalid.
func (gr *GetRequest[T]) Validate() error {
	if gr.ResourceID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}

	switch gr.ResourceType {
	case types.ResourceTypePage:
		pageID := types.PageID(gr.ResourceID)
		return pageID.Validate()
	case types.ResourceTypeDatabase:
		dbID := types.DatabaseID(gr.ResourceID)
		return dbID.Validate()
	case types.ResourceTypeBlock:
		blockID := types.BlockID(gr.ResourceID)
		return blockID.Validate()
	case types.ResourceTypeUser:
		userID := types.UserID(gr.ResourceID)
		return userID.Validate()
	case types.ResourceTypeProperty:
		if gr.ParentID == "" {
			return fmt.Errorf("parent ID (database ID) is required for property requests")
		}
		dbID := types.DatabaseID(gr.ParentID)
		if err := dbID.Validate(); err != nil {
			return fmt.Errorf("invalid parent database ID: %w", err)
		}
		propID := types.PropertyID(gr.ResourceID)
		return propID.Validate()
	default:
		return fmt.Errorf("unsupported resource type: %s", gr.ResourceType)
	}
}

// NewGetRequest creates a new generic get request for any resource type.
//
// Arguments:
//   - resourceType: The type of resource to retrieve.
//   - resourceID: The ID of the resource to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new generic get request.
func NewGetRequest[T any](resourceType types.ResourceType, resourceID string) *GetRequest[T] {
	return &GetRequest[T]{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// NewGetRequestWithParent creates a new generic get request with a parent ID.
// This is useful for resources like properties that need a parent database ID.
//
// Arguments:
//   - resourceType: The type of resource to retrieve.
//   - resourceID: The ID of the resource to retrieve.
//   - parentID: The ID of the parent resource.
//
// Returns:
//   - *GetRequest[T]: A new generic get request with parent ID.
func NewGetRequestWithParent[T any](resourceType types.ResourceType, resourceID, parentID string) *GetRequest[T] {
	return &GetRequest[T]{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ParentID:     parentID,
	}
}

// NewPageGetRequest creates a get request for a page.
//
// Arguments:
//   - pageID: The ID of the page to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new get request for a page.
func NewPageGetRequest[T any](pageID types.PageID) *GetRequest[T] {
	return NewGetRequest[T](types.ResourceTypePage, pageID.String())
}

// NewDatabaseGetRequest creates a get request for a database.
//
// Arguments:
//   - databaseID: The ID of the database to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new get request for a database.
func NewDatabaseGetRequest[T any](databaseID types.DatabaseID) *GetRequest[T] {
	return NewGetRequest[T](types.ResourceTypeDatabase, databaseID.String())
}

// NewBlockGetRequest creates a get request for a block.
//
// Arguments:
//   - blockID: The ID of the block to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new get request for a block.
func NewBlockGetRequest[T any](blockID types.BlockID) *GetRequest[T] {
	return NewGetRequest[T](types.ResourceTypeBlock, blockID.String())
}

// NewUserGetRequest creates a get request for a user.
//
// Arguments:
//   - userID: The ID of the user to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new get request for a user.
func NewUserGetRequest[T any](userID types.UserID) *GetRequest[T] {
	return NewGetRequest[T](types.ResourceTypeUser, userID.String())
}

// NewPropertyGetRequest creates a get request for a database property.
//
// Arguments:
//   - databaseID: The ID of the database containing the property.
//   - propertyID: The ID of the property to retrieve.
//
// Returns:
//   - *GetRequest[T]: A new get request for a database property.
func NewPropertyGetRequest[T any](databaseID types.DatabaseID, propertyID types.PropertyID) *GetRequest[T] {
	return NewGetRequestWithParent[T](types.ResourceTypeProperty, propertyID.String(), databaseID.String())
}

// GetOperator provides a unified interface for all get operations.
// This replaces the need for separate operators for each resource type.
type GetOperator[T any] struct {
	operator *Operator[T]
}

// NewGetOperator creates a new get operator for any resource type.
//
// Arguments:
//   - client: HTTP client for making requests.
//   - config: Optional configuration (uses defaults if nil).
//
// Returns:
//   - *GetOperator[T]: A new get operator.
func NewGetOperator[T any](client *HTTPClient, config *OperatorConfig) *GetOperator[T] {
	return &GetOperator[T]{
		operator: NewOperator[T](client, config),
	}
}

// Get retrieves a single resource using the generic get request.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: The get request to execute.
//
// Returns:
//   - Result[T]: The result of the get operation.
func (go_op *GetOperator[T]) Get(ctx context.Context, req *GetRequest[T]) Result[T] {
	return Execute(go_op.operator, ctx, req)
}

// GetMany retrieves multiple resources concurrently.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - reqs: The get requests to execute.
//
// Returns:
//   - <-chan Result[T]: A channel of results.
func (go_op *GetOperator[T]) GetMany(ctx context.Context, reqs []*GetRequest[T]) <-chan Result[T] {
	return ExecuteConcurrent(go_op.operator, ctx, reqs)
}
