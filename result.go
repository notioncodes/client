package client

import (
	"context"
	"time"
)

// Result represents the result of an asynchronous operation.
// This is the fundamental type used in all channel-based communication
// throughout the client, providing both success data and error information.
type Result[T any] struct {
	// Data contains the successful result of the operation.
	// This will be nil if Error is not nil.
	Data T

	// Error contains any error that occurred during the operation.
	// This will be nil if the operation succeeded.
	Error error

	// Metadata contains additional information about the operation.
	// This includes timing, retry information, and other observability data.
	Metadata *ResultMetadata
}

// ResultMetadata contains observability and debugging information about an operation.
type ResultMetadata struct {
	// RequestID is a unique identifier for this request, useful for tracing.
	RequestID string

	// Attempt is the number of attempts made (1-based, so 1 = first attempt).
	Attempt int

	// Duration is the total time taken for the operation including retries.
	Duration time.Duration

	// RequestDuration is the time taken for the actual HTTP request.
	RequestDuration time.Duration

	// RateLimited indicates if the request was rate limited.
	RateLimited bool

	// RateLimitDelay is the delay imposed due to rate limiting.
	RateLimitDelay time.Duration

	// Cached indicates if the result was served from cache.
	Cached bool

	// FromStream indicates if this result came from a streaming operation.
	FromStream bool

	// StreamPosition is the position in the stream (for paginated results).
	StreamPosition int

	// PageInfo contains pagination information if applicable.
	PageInfo *PageInfo
}

// PageInfo contains pagination metadata for streaming operations.
type PageInfo struct {
	// HasMore indicates if there are more pages available.
	HasMore bool

	// NextCursor is the cursor for the next page.
	NextCursor *string

	// PageSize is the size of the current page.
	PageSize int

	// PageNumber is the current page number (1-based).
	PageNumber int

	// TotalItems is the total number of items across all pages (if known).
	TotalItems *int
}

// IsSuccess returns true if the result represents a successful operation.
//
// Returns:
//   - bool: True if no error occurred, false otherwise.
//
// Example:
//
//	result := <-resultChannel
//	if result.IsSuccess() {
//	    // Process result.Data
//	} else {
//	    log.Printf("Operation failed: %v", result.Error)
//	}
func (r *Result[T]) IsSuccess() bool {
	return r.Error == nil
}

// IsError returns true if the result represents a failed operation.
//
// Returns:
//   - bool: True if an error occurred, false otherwise.
func (r *Result[T]) IsError() bool {
	return r.Error != nil
}

// WithMetadata returns a new result with the specified metadata.
// This is used internally to add observability information to results.
//
// Arguments:
//   - metadata: The metadata to attach to the result.
//
// Returns:
//   - Result[T]: A new result with the metadata attached.
func (r *Result[T]) WithMetadata(metadata *ResultMetadata) Result[T] {
	return Result[T]{
		Data:     r.Data,
		Error:    r.Error,
		Metadata: metadata,
	}
}

// Success creates a successful result with the given data.
// This is a convenience function for creating success results.
//
// Arguments:
//   - data: The successful result data.
//
// Returns:
//   - Result[T]: A successful result containing the data.
//
// Example:
//
//	result := Success(page)
//	return result
func Success[T any](data T) Result[T] {
	return Result[T]{Data: data}
}

// Error creates an error result with the given error.
// This is a convenience function for creating error results.
//
// Arguments:
//   - err: The error that occurred.
//
// Returns:
//   - Result[T]: An error result containing the error.
//
// Example:
//
//	result := Error[Page](fmt.Errorf("page not found"))
//	return result
func Error[T any](err error) Result[T] {
	var zero T
	return Result[T]{Data: zero, Error: err}
}

// SuccessWithMetadata creates a successful result with data and metadata.
//
// Arguments:
//   - data: The successful result data.
//   - metadata: The metadata to attach.
//
// Returns:
//   - Result[T]: A successful result with metadata.
func SuccessWithMetadata[T any](data T, metadata *ResultMetadata) Result[T] {
	return Result[T]{Data: data, Metadata: metadata}
}

// ErrorWithMetadata creates an error result with an error and metadata.
//
// Arguments:
//   - err: The error that occurred.
//   - metadata: The metadata to attach.
//
// Returns:
//   - Result[T]: An error result with metadata.
func ErrorWithMetadata[T any](err error, metadata *ResultMetadata) Result[T] {
	var zero T
	return Result[T]{Data: zero, Error: err, Metadata: metadata}
}

// StreamResult represents a single item in a stream of results.
// This extends Result with stream-specific information.
type StreamResult[T any] struct {
	Result[T]

	// Position is the position of this item in the stream.
	Position int

	// IsLast indicates if this is the last item in the stream.
	IsLast bool
}

// CollectResults collects all results from a channel into a slice.
// This is useful when you want to gather all streaming results at once.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Channel of results to collect.
//
// Returns:
//   - []T: Slice of all successful results.
//   - error: The first error encountered, or nil if all succeeded.
//
// Example:
//
//	results, err := CollectResults(ctx, resultChannel)
//	if err != nil {
//	    log.Printf("Collection failed: %v", err)
//	    return
//	}
//	fmt.Printf("Collected %d results\n", len(results))
func CollectResults[T any](ctx context.Context, resultCh <-chan Result[T]) ([]T, error) {
	var results []T

	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				return results, nil
			}

			if result.IsError() {
				return results, result.Error
			}

			results = append(results, result.Data)

		case <-ctx.Done():
			return results, ctx.Err()
		}
	}
}

// CollectAllResults collects all results from a channel, including errors.
// This returns both successful results and all errors encountered.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Channel of results to collect.
//
// Returns:
//   - []T: Slice of all successful results.
//   - []error: Slice of all errors encountered.
//
// Example:
//
//	results, errors := CollectAllResults(ctx, resultChannel)
//	fmt.Printf("Got %d results and %d errors\n", len(results), len(errors))
func CollectAllResults[T any](ctx context.Context, resultCh <-chan Result[T]) ([]T, []error) {
	var results []T
	var errors []error

	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				return results, errors
			}

			if result.IsError() {
				errors = append(errors, result.Error)
			} else {
				results = append(results, result.Data)
			}

		case <-ctx.Done():
			return results, errors
		}
	}
}

// MapResults transforms a stream of results using a mapping function.
// This creates a new channel with transformed results.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Input channel of results.
//   - mapper: Function to transform each successful result.
//
// Returns:
//   - <-chan Result[U]: Channel of transformed results.
//
// Example:
//
//	// Transform pages to their titles
//	titleCh := MapResults(ctx, pageCh, func(page Page) string {
//	    return page.GetTitle()
//	})
func MapResults[T, U any](ctx context.Context, resultCh <-chan Result[T], mapper func(T) U) <-chan Result[U] {
	outputCh := make(chan Result[U])

	go func() {
		defer close(outputCh)

		for {
			select {
			case result, ok := <-resultCh:
				if !ok {
					return
				}

				if result.IsError() {
					outputCh <- Error[U](result.Error)
				} else {
					mapped := mapper(result.Data)
					outputCh <- SuccessWithMetadata(mapped, result.Metadata)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh
}

// FilterResults filters a stream of results using a predicate function.
// Only results that pass the predicate are forwarded to the output channel.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Input channel of results.
//   - predicate: Function to test each successful result.
//
// Returns:
//   - <-chan Result[T]: Channel of filtered results.
//
// Example:
//
//	// Filter to only archived pages
//	archivedCh := FilterResults(ctx, pageCh, func(page Page) bool {
//	    return page.Archived
//	})
func FilterResults[T any](ctx context.Context, resultCh <-chan Result[T], predicate func(T) bool) <-chan Result[T] {
	outputCh := make(chan Result[T])

	go func() {
		defer close(outputCh)

		for {
			select {
			case result, ok := <-resultCh:
				if !ok {
					return
				}

				if result.IsError() || predicate(result.Data) {
					outputCh <- result
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh
}

// BatchResults groups results into batches of a specified size.
// This is useful for processing results in chunks.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Input channel of results.
//   - batchSize: Maximum size of each batch.
//
// Returns:
//   - <-chan Result[[]T]: Channel of batched results.
//
// Example:
//
//	// Process pages in batches of 10
//	batchCh := BatchResults(ctx, pageCh, 10)
//	for batch := range batchCh {
//	    if batch.IsSuccess() {
//	        processBatch(batch.Data) // []Page with up to 10 items
//	    }
//	}
func BatchResults[T any](ctx context.Context, resultCh <-chan Result[T], batchSize int) <-chan Result[[]T] {
	outputCh := make(chan Result[[]T])

	go func() {
		defer close(outputCh)

		var batch []T

		for {
			select {
			case result, ok := <-resultCh:
				if !ok {
					// Send final batch if not empty
					if len(batch) > 0 {
						outputCh <- Success(batch)
					}
					return
				}

				if result.IsError() {
					// Send current batch if not empty, then send error
					if len(batch) > 0 {
						outputCh <- Success(batch)
						batch = nil
					}
					outputCh <- Error[[]T](result.Error)
					continue
				}

				batch = append(batch, result.Data)

				if len(batch) >= batchSize {
					outputCh <- Success(batch)
					batch = nil
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh
}
