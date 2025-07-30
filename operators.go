package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/notioncodes/types"
)

// RequestBody defines the constraint for request body types.
type RequestBody any

// RequestInterface defines the constraint for request types that can be executed.
type RequestInterface[B RequestBody] interface {
	GetPath() string
	GetMethod() string
	GetBody() B
	GetQuery() url.Values
	Validate() error
}

// PaginatedRequestInterface extends RequestInterface with pagination capabilities.
type PaginatedRequestInterface[B RequestBody] interface {
	RequestInterface[B]
	SetStartCursor(cursor *string)
	SetPageSize(pageSize *int)
}

// OperatorNamespace provides a namespace for operators.
type OperatorNamespace struct {
	Page     *Operator[types.Page]
	Database *Operator[types.Database]
	Block    *Operator[types.Block]
	User     *Operator[types.User]
}

// Operator handles operations that return a single resource.
type Operator[T any] struct {
	client *HTTPClient
	config *OperatorConfig
}

// PaginatedOperator handles operations that return paginated results.
type PaginatedOperator[T any] struct {
	client *HTTPClient
	config *OperatorConfig
}

// OperatorConfig contains configuration for operator behavior.
type OperatorConfig struct {
	// Concurrency is the number of concurrent operations to allow.
	Concurrency int

	// PageSize is the default page size for paginated operations.
	PageSize int

	// MaxPages is the maximum number of pages to retrieve (0 = unlimited).
	MaxPages int

	// Timeout is the timeout for individual operations.
	// Set to 0 to disable timeout (default).
	Timeout time.Duration

	// EnableCaching enables response caching.
	EnableCaching bool

	// CacheTTL is the cache time-to-live.
	CacheTTL time.Duration
}

// DefaultOperatorConfig returns a configuration with sensible defaults.
func DefaultOperatorConfig() *OperatorConfig {
	return &OperatorConfig{
		Concurrency:   10,
		PageSize:      100,
		MaxPages:      0, // unlimited
		Timeout:       0, // No timeout by default
		EnableCaching: false,
		CacheTTL:      5 * time.Minute,
	}
}

// NewOperator creates a new single resource operator.
//
// Arguments:
//   - client: HTTP client for making requests.
//   - config: Optional configuration (uses defaults if nil).
//
// Returns:
//   - *Operator[T]: A new single resource operator.
func NewOperator[T any](client *HTTPClient, config *OperatorConfig) *Operator[T] {
	if config == nil {
		config = DefaultOperatorConfig()
	}

	return &Operator[T]{
		client: client,
		config: config,
	}
}

// Execute performs a single resource operation.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - req: The operation request to execute.
//
// Returns:
//   - Result[T]: The result of the operation.
func Execute[T any, R RequestInterface[B], B RequestBody](op *Operator[T], ctx context.Context, req R) Result[T] {
	if err := req.Validate(); err != nil {
		return Error[T](&ValidationError{Message: err.Error()})
	}

	if op.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, op.config.Timeout)
		defer cancel()
	}

	startTime := time.Now()

	var result T
	var err error

	switch req.GetMethod() {
	case "GET":
		err = op.client.GetJSON(ctx, req.GetPath(), req.GetQuery(), &result)
	case "POST":
		err = op.client.PostJSON(ctx, req.GetPath(), req.GetBody(), &result)
	case "PATCH":
		err = op.client.PatchJSON(ctx, req.GetPath(), req.GetBody(), &result)
	case "DELETE":
		_, err = op.client.Delete(ctx, req.GetPath())
		// For DELETE operations, we don't expect a meaningful response body
	default:
		err = fmt.Errorf("unsupported HTTP method: %s", req.GetMethod())
	}

	metadata := &ResultMetadata{
		Duration:        time.Since(startTime),
		RequestDuration: time.Since(startTime),
	}

	if err != nil {
		return ErrorWithMetadata[T](err, metadata)
	}

	return SuccessWithMetadata(result, metadata)
}

// Stream performs a single resource operation and returns a channel.
func Stream[T any, R RequestInterface[B], B RequestBody](op *Operator[T], ctx context.Context, req R) <-chan Result[T] {
	resultCh := make(chan Result[T], 1)

	go func() {
		defer close(resultCh)
		result := Execute(op, ctx, req)

		select {
		case resultCh <- result:
		case <-ctx.Done():
		}
	}()

	return resultCh
}

// ExecuteConcurrent performs multiple single resource operations concurrently.
func ExecuteConcurrent[T any, R RequestInterface[B], B RequestBody](op *Operator[T], ctx context.Context, reqs []R) <-chan Result[T] {
	resultCh := make(chan Result[T], len(reqs))

	if len(reqs) == 0 {
		close(resultCh)
		return resultCh
	}

	// Use worker pool for concurrent execution
	config := DefaultWorkerPoolConfig()
	config.NumWorkers = op.config.Concurrency
	config.WorkBufferSize = len(reqs)
	config.ResultBufferSize = len(reqs)

	pool := NewWorkerPool[T](config)
	pool.Start(ctx)

	go func() {
		defer func() {
			pool.Stop()
			close(resultCh)
		}()

		// Submit all requests as work items
		for _, req := range reqs {
			req := req // Capture loop variable
			pool.Submit(ctx, func(ctx context.Context, _ int) (T, error) {
				result := Execute(op, ctx, req)
				if result.IsError() {
					var zero T
					return zero, result.Error
				}
				return result.Data, nil
			})
		}

		// Collect results
		for i := 0; i < len(reqs); i++ {
			select {
			case result := <-pool.Results():
				resultCh <- result
			case <-ctx.Done():
				resultCh <- Error[T](ctx.Err())
				return
			}
		}
	}()

	return resultCh
}

// NewPaginatedOperator creates a new paginated operator.
//
// Arguments:
//   - client: HTTP client for making requests.
//   - config: Optional configuration (uses defaults if nil).
//
// Returns:
//   - *PaginatedOperator[T]: A new paginated operator.
func NewPaginatedOperator[T any](client *HTTPClient, config *OperatorConfig) *PaginatedOperator[T] {
	if config == nil {
		config = DefaultOperatorConfig()
	}

	return &PaginatedOperator[T]{
		client: client,
		config: config,
	}
}

// ExecutePaginated performs a paginated operation and returns all results as a slice.
func ExecutePaginated[T any, R PaginatedRequestInterface[B], B RequestBody](po *PaginatedOperator[T], ctx context.Context, req R) Result[[]T] {
	results, err := CollectResults(ctx, StreamPaginated(po, ctx, req))
	if err != nil {
		return Error[[]T](err)
	}
	return Success(results)
}

// StreamPaginated performs a paginated operation and streams individual results.
func StreamPaginated[T any, R PaginatedRequestInterface[B], B RequestBody](po *PaginatedOperator[T], ctx context.Context, req R) <-chan Result[T] {
	resultCh := make(chan Result[T], po.config.PageSize)

	go func() {
		defer close(resultCh)

		if err := req.Validate(); err != nil {
			resultCh <- Error[T](&ValidationError{Message: err.Error()})
			return
		}

		if po.config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, po.config.Timeout)
			defer cancel()
		}

		var cursor *string
		pageNum := 1

		for {
			select {
			case <-ctx.Done():
				resultCh <- Error[T](ctx.Err())
				return
			default:
			}

			if po.config.MaxPages > 0 && pageNum > po.config.MaxPages {
				break
			}

			query := req.GetQuery()
			if query == nil {
				query = make(url.Values)
			}

			if cursor != nil {
				query.Set("start_cursor", *cursor)
			}

			query.Set("page_size", strconv.Itoa(po.config.PageSize))

			if cursor != nil {
				req.SetStartCursor(cursor)
			}
			req.SetPageSize(&po.config.PageSize)

			var response PaginationResponse[T]
			var err error

			switch req.GetMethod() {
			case "GET":
				err = po.client.GetJSON(ctx, req.GetPath(), query, &response)
			case "POST":
				err = po.client.PostJSON(ctx, req.GetPath(), req.GetBody(), &response)
			default:
				resultCh <- Error[T](fmt.Errorf("unsupported method for pagination: %s", req.GetMethod()))
				return
			}

			if err != nil {
				paginationErr := &PaginationError{
					Message:   err.Error(),
					Page:      pageNum,
					Operation: "paginated_stream",
				}
				if cursor != nil {
					paginationErr.Cursor = *cursor
				}
				resultCh <- Error[T](paginationErr)
				return
			}

			// Send results from this page
			for i, item := range response.Results {
				metadata := &ResultMetadata{
					FromStream:     true,
					StreamPosition: (pageNum-1)*po.config.PageSize + i + 1,
					PageInfo: &PageInfo{
						HasMore:    response.HasMore,
						NextCursor: response.NextCursor,
						PageSize:   len(response.Results),
						PageNumber: pageNum,
					},
				}

				result := SuccessWithMetadata(item, metadata)

				select {
				case resultCh <- result:
				case <-ctx.Done():
					resultCh <- Error[T](ctx.Err())
					return
				}
			}

			// Check if we have more pages
			if !response.HasMore || response.NextCursor == nil {
				break
			}

			cursor = response.NextCursor
			pageNum++
		}
	}()

	return resultCh
}

// ExecuteConcurrentPaginated performs multiple paginated operations concurrently.
func ExecuteConcurrentPaginated[T any, R PaginatedRequestInterface[B], B RequestBody](po *PaginatedOperator[T], ctx context.Context, reqs []R) <-chan Result[T] {
	resultCh := make(chan Result[T], po.config.PageSize*len(reqs))

	if len(reqs) == 0 {
		close(resultCh)
		return resultCh
	}

	go func() {
		defer close(resultCh)

		// Create a worker pool to handle concurrent stream processing
		config := DefaultWorkerPoolConfig()
		config.NumWorkers = po.config.Concurrency
		config.WorkBufferSize = len(reqs)
		config.ResultBufferSize = po.config.PageSize * len(reqs)

		pool := NewWorkerPool[[]T](config)
		pool.Start(ctx)

		// Submit each request as a work item that collects all results from its stream
		for _, req := range reqs {
			req := req // Capture loop variable
			pool.Submit(ctx, func(ctx context.Context, workerID int) ([]T, error) {
				// Collect all results from this request's stream
				results, err := CollectResults(ctx, StreamPaginated(po, ctx, req))
				if err != nil {
					return nil, err
				}
				return results, nil
			})
		}

		// Collect results from all workers and forward individual items
		for i := 0; i < len(reqs); i++ {
			select {
			case result := <-pool.Results():
				if result.IsError() {
					select {
					case resultCh <- Error[T](result.Error):
					case <-ctx.Done():
						pool.Stop()
						return
					}
				} else {
					// Forward each item from the collected results
					for _, item := range result.Data {
						select {
						case resultCh <- Success(item):
						case <-ctx.Done():
							pool.Stop()
							return
						}
					}
				}
			case <-ctx.Done():
				pool.Stop()
				return
			}
		}

		// Stop the pool after all results are processed
		pool.Stop()
	}()

	return resultCh
}
