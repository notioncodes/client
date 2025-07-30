package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/notioncodes/types"
)

// CommentNamespace provides fluent access to comment operations.
type CommentNamespace struct {
	registry *Registry
}

type CommentListRequest struct {
	Body CommentListRequestBody
}

type CommentListRequestBody struct {
	BlockID     *types.BlockID `json:"block_id,omitempty"`
	StartCursor *string        `json:"start_cursor,omitempty"`
	PageSize    *int           `json:"page_size,omitempty"`
}

// ListOptions configures comment listing options.
type ListCommentsOptions struct {
	// Block ID to get comments for a specific block
	BlockID *types.BlockID `json:"block_id,omitempty"`

	// Page ID to get comments for a specific page
	PageID *types.PageID `json:"page_id,omitempty"`

	// Pagination options
	StartCursor *string `json:"start_cursor,omitempty"`
	PageSize    *int    `json:"page_size,omitempty"`
}

// CreateCommentOptions configures comment creation.
type CreateCommentOptions struct {
	// Parent specifies where to attach the comment
	Parent *types.CommentParent `json:"parent,omitempty"`

	// DiscussionID to add comment to existing thread
	DiscussionID *types.DiscussionID `json:"discussion_id,omitempty"`

	// RichText content of the comment
	RichText []types.RichText `json:"rich_text"`

	// Attachments for the comment
	Attachments []types.CommentAttachment `json:"attachments,omitempty"`

	// DisplayName for custom display
	DisplayName *types.CommentDisplayName `json:"display_name,omitempty"`
}

// List retrieves comments from a page or block.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - opts: Options for comment listing (page/block ID, pagination).
//
// Returns:
//   - <-chan Result[types.Comment]: Channel of comment results.
//
// Example:
//
//	opts := &ListCommentsOptions{
//	    PageID: &pageID,
//	    PageSize: &50,
//	}
//	results := registry.Comments().List(ctx, opts)
//	for result := range results {
//	    if result.IsError() {
//	        log.Printf("Error: %v", result.Error)
//	        continue
//	    }
//	    fmt.Printf("Comment: %s\n", result.Data.GetPlainText())
//	}
func (ns *CommentNamespace) List(ctx context.Context, opts *ListCommentsOptions) <-chan Result[types.Comment] {
	if opts == nil {
		opts = &ListCommentsOptions{}
	}

	// Build query parameters
	query := make(url.Values)

	if opts.BlockID != nil {
		query.Set("block_id", opts.BlockID.String())
	}

	if opts.PageID != nil {
		query.Set("page_id", opts.PageID.String())
	}

	paginatedOp := NewPaginatedOperator[types.Comment](ns.registry.httpClient, DefaultOperatorConfig())

	// Create pagination request
	paginationReq := &CommentListRequest{
		Body: CommentListRequestBody{
			BlockID:     opts.BlockID,
			StartCursor: opts.StartCursor,
			PageSize:    opts.PageSize,
		},
	}

	// Use the generic pagination handler
	return StreamPaginated(paginatedOp, ctx, paginationReq)
}

func (r *CommentListRequest) GetPath() string {
	return "/comments"
}

func (r *CommentListRequest) GetQuery() url.Values {
	query := make(url.Values)

	if r.Body.BlockID != nil {
		query.Set("block_id", r.Body.BlockID.String())
	}

	if r.Body.StartCursor != nil {
		query.Set("start_cursor", *r.Body.StartCursor)
	}

	if r.Body.PageSize != nil {
		query.Set("page_size", strconv.Itoa(*r.Body.PageSize))
	} else {
		query.Set("page_size", "100")
	}

	return query
}

func (r *CommentListRequest) SetStartCursor(cursor *string) {
	r.Body.StartCursor = cursor
}

func (r *CommentListRequest) SetPageSize(pageSize *int) {
	r.Body.PageSize = pageSize
}

func (r *CommentListRequest) GetMethod() string {
	return "GET"
}

func (r *CommentListRequest) GetBody() interface{} {
	return nil
}

func (r *CommentListRequest) Validate() error {
	return nil
}

// ListByPage retrieves all comments for a specific page.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to get comments for.
//
// Returns:
//   - <-chan Result[types.Comment]: Channel of comment results.
//
// Example:
//
//	results := registry.Comments().ListByPage(ctx, pageID)
//	for result := range results {
//	    if result.IsSuccess() {
//	        fmt.Printf("Page comment: %s\n", result.Data.GetPlainText())
//	    }
//	}
func (ns *CommentNamespace) ListByPage(ctx context.Context, pageID types.PageID) <-chan Result[types.Comment] {
	return ns.List(ctx, &ListCommentsOptions{
		PageID: &pageID,
	})
}

// ListByBlock retrieves all comments for a specific block.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to get comments for.
//
// Returns:
//   - <-chan Result[types.Comment]: Channel of comment results.
//
// Example:
//
//	results := registry.Comments().ListByBlock(ctx, blockID)
//	for result := range results {
//	    if result.IsSuccess() {
//	        fmt.Printf("Block comment: %s\n", result.Data.GetPlainText())
//	    }
//	}
func (ns *CommentNamespace) ListByBlock(ctx context.Context, blockID types.BlockID) <-chan Result[types.Comment] {
	return ns.List(ctx, &ListCommentsOptions{
		BlockID: &blockID,
	})
}

// Create creates a new comment.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - opts: Options for comment creation.
//
// Returns:
//   - Result[types.Comment]: The created comment result.
//
// Example:
//
//	parent := &types.CommentParent{
//	    Type: types.CommentParentTypePage,
//	    PageID: &pageID,
//	}
//	richText := []types.RichText{*types.NewTextRichText("Great work!", nil)}
//	opts := &CreateCommentOptions{
//	    Parent: parent,
//	    RichText: richText,
//	}
//	result := registry.Comments().Create(ctx, opts)
//	if result.IsSuccess() {
//	    fmt.Printf("Created comment: %s\n", result.Data.ID)
//	}
func (ns *CommentNamespace) Create(ctx context.Context, opts *CreateCommentOptions) Result[types.Comment] {
	if opts == nil {
		return Error[types.Comment](fmt.Errorf("create options are required"))
	}

	// Validate that either parent or discussion_id is provided
	if opts.Parent == nil && opts.DiscussionID == nil {
		return Error[types.Comment](fmt.Errorf("either parent or discussion_id must be provided"))
	}

	if opts.Parent != nil && opts.DiscussionID != nil {
		return Error[types.Comment](fmt.Errorf("parent and discussion_id cannot both be provided"))
	}

	// Validate rich text is provided
	if len(opts.RichText) == 0 {
		return Error[types.Comment](fmt.Errorf("rich_text content is required"))
	}

	// Create the request body
	requestBody := map[string]interface{}{
		"rich_text": opts.RichText,
	}

	if opts.Parent != nil {
		requestBody["parent"] = opts.Parent
	}

	if opts.DiscussionID != nil {
		requestBody["discussion_id"] = opts.DiscussionID
	}

	if len(opts.Attachments) > 0 {
		requestBody["attachments"] = opts.Attachments
	}

	if opts.DisplayName != nil {
		requestBody["display_name"] = opts.DisplayName
	}

	var comment types.Comment
	err := ns.registry.httpClient.PostJSON(ctx, "/comments", requestBody, &comment)
	if err != nil {
		return Error[types.Comment](err)
	}

	return Success(comment)
}

// CreateOnPage creates a comment on a specific page.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to comment on.
//   - richText: The rich text content of the comment.
//
// Returns:
//   - Result[types.Comment]: The created comment result.
//
// Example:
//
//	richText := []types.RichText{*types.NewTextRichText("This page is awesome!", nil)}
//	result := registry.Comments().CreateOnPage(ctx, pageID, richText)
//	if result.IsSuccess() {
//	    fmt.Printf("Comment created on page: %s\n", result.Data.ID)
//	}
func (ns *CommentNamespace) CreateOnPage(ctx context.Context, pageID types.PageID, richText []types.RichText) Result[types.Comment] {
	parent := &types.CommentParent{
		Type:   types.CommentParentTypePage,
		PageID: &pageID,
	}

	return ns.Create(ctx, &CreateCommentOptions{
		Parent:   parent,
		RichText: richText,
	})
}

// CreateOnBlock creates a comment on a specific block.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to comment on.
//   - richText: The rich text content of the comment.
//
// Returns:
//   - Result[types.Comment]: The created comment result.
//
// Example:
//
//	richText := []types.RichText{*types.NewTextRichText("Interesting point!", nil)}
//	result := registry.Comments().CreateOnBlock(ctx, blockID, richText)
//	if result.IsSuccess() {
//	    fmt.Printf("Comment created on block: %s\n", result.Data.ID)
//	}
func (ns *CommentNamespace) CreateOnBlock(ctx context.Context, blockID types.BlockID, richText []types.RichText) Result[types.Comment] {
	parent := &types.CommentParent{
		Type:    types.CommentParentTypeBlock,
		BlockID: &blockID,
	}

	return ns.Create(ctx, &CreateCommentOptions{
		Parent:   parent,
		RichText: richText,
	})
}

// CreateInDiscussion creates a comment in an existing discussion thread.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - discussionID: The ID of the discussion thread.
//   - richText: The rich text content of the comment.
//
// Returns:
//   - Result[types.Comment]: The created comment result.
//
// Example:
//
//	richText := []types.RichText{*types.NewTextRichText("I agree with this!", nil)}
//	result := registry.Comments().CreateInDiscussion(ctx, discussionID, richText)
//	if result.IsSuccess() {
//	    fmt.Printf("Reply created: %s\n", result.Data.ID)
//	}
func (ns *CommentNamespace) CreateInDiscussion(ctx context.Context, discussionID types.DiscussionID, richText []types.RichText) Result[types.Comment] {
	return ns.Create(ctx, &CreateCommentOptions{
		DiscussionID: &discussionID,
		RichText:     richText,
	})
}

// GetAll retrieves all comments as a slice (convenience method).
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - opts: Options for comment listing.
//
// Returns:
//   - []*types.Comment: Slice of all comments.
//   - error: Any error that occurred during retrieval.
//
// Example:
//
//	comments, err := registry.Comments().GetAll(ctx, &ListCommentsOptions{
//	    PageID: &pageID,
//	})
//	if err != nil {
//	    log.Printf("Error: %v", err)
//	    return
//	}
//	fmt.Printf("Found %d comments\n", len(comments))
func (ns *CommentNamespace) GetAll(ctx context.Context, opts *ListCommentsOptions) ([]*types.Comment, error) {
	var comments []*types.Comment

	results := ns.List(ctx, opts)
	for result := range results {
		if result.IsError() {
			return comments, result.Error
		}
		comments = append(comments, &result.Data)
	}

	return comments, nil
}

// GetAllByPage retrieves all comments for a page as a slice.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - pageID: The ID of the page to get comments for.
//
// Returns:
//   - []*types.Comment: Slice of all page comments.
//   - error: Any error that occurred during retrieval.
func (ns *CommentNamespace) GetAllByPage(ctx context.Context, pageID types.PageID) ([]*types.Comment, error) {
	return ns.GetAll(ctx, &ListCommentsOptions{
		PageID: &pageID,
	})
}

// GetAllByBlock retrieves all comments for a block as a slice.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - blockID: The ID of the block to get comments for.
//
// Returns:
//   - []*types.Comment: Slice of all block comments.
//   - error: Any error that occurred during retrieval.
func (ns *CommentNamespace) GetAllByBlock(ctx context.Context, blockID types.BlockID, opts *ListCommentsOptions) ([]*types.Comment, error) {
	var comments []*types.Comment

	results := ns.List(ctx, opts)
	for result := range results {
		if result.IsError() {
			return comments, result.Error
		}
		comments = append(comments, &result.Data)
	}

	return comments, nil
}
