package client

import "context"

// ToGoResult converts a Result[T] to Go's idiomatic (T, error) pattern.
// This provides compatibility with standard Go libraries and practices.
//
// Arguments:
//   - result: The Result[T] to convert.
//
// Returns:
//   - T: The data if successful, zero value if error.
//   - error: The error if failed, nil if successful.
//
// Example:
//
//	result := Execute(op, ctx, req)
//	page, err := ToGoResult(result)
//	if err != nil {
//	    return fmt.Errorf("failed to get page: %w", err)
//	}
//	fmt.Printf("Page title: %s\n", page.Title)
func ToGoResult[T any](result Result[T]) (T, error) {
	if result.IsError() {
		var zero T
		return zero, result.Error
	}
	return result.Data, nil
}

// FromGoResult converts Go's (T, error) pattern to Result[T].
// This is useful when wrapping standard Go functions.
//
// Arguments:
//   - data: The data value.
//   - err: The error value.
//
// Returns:
//   - Result[T]: A Result containing either the data or error.
//
// Example:
//
//	data, err := someStandardGoFunction()
//	result := FromGoResult(data, err)
//	return result
func FromGoResult[T any](data T, err error) Result[T] {
	if err != nil {
		return Error[T](err)
	}
	return Success(data)
}

// Must extracts the data from a Result, panicking if there's an error.
// Use only when you're certain the operation will succeed.
//
// Arguments:
//   - result: The Result[T] to extract data from.
//
// Returns:
//   - T: The data from the result.
//
// Panics:
//   - If the result contains an error.
//
// Example:
//
//	// Only use when you're absolutely sure it will succeed
//	page := Must(Execute(op, ctx, req))
//	fmt.Printf("Page: %+v\n", page)
func Must[T any](result Result[T]) T {
	if result.IsError() {
		panic(result.Error)
	}
	return result.Data
}

// MustWithContext is like Must but respects context cancellation.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - result: The Result[T] to extract data from.
//
// Returns:
//   - T: The data from the result.
//   - error: Context cancellation error or panic recovery.
//
// Example:
//
//	page, err := MustWithContext(ctx, Execute(op, ctx, req))
//	if err != nil {
//	    return fmt.Errorf("operation failed: %w", err)
//	}
func MustWithContext[T any](ctx context.Context, result Result[T]) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	default:
		if result.IsError() {
			var zero T
			return zero, result.Error
		}
		return result.Data, nil
	}
}

// TryResult attempts to extract data, returning a boolean success indicator.
// This provides a safe way to check results without error handling.
//
// Arguments:
//   - result: The Result[T] to check.
//
// Returns:
//   - T: The data if successful, zero value if error.
//   - bool: True if successful, false if error.
//
// Example:
//
//	result := Execute(op, ctx, req)
//	if page, ok := TryResult(result); ok {
//	    fmt.Printf("Got page: %s\n", page.Title)
//	} else {
//	    fmt.Println("Operation failed")
//	}
func TryResult[T any](result Result[T]) (T, bool) {
	if result.IsError() {
		var zero T
		return zero, false
	}
	return result.Data, true
}

// UnwrapOr extracts data from Result or returns a default value.
//
// Arguments:
//   - result: The Result[T] to unwrap.
//   - defaultValue: Value to return if result contains an error.
//
// Returns:
//   - T: The data from result or the default value.
//
// Example:
//
//	result := Execute(op, ctx, req)
//	page := UnwrapOr(result, &Page{Title: "Default Page"})
//	fmt.Printf("Page title: %s\n", page.Title)
func UnwrapOr[T any](result Result[T], defaultValue T) T {
	if result.IsError() {
		return defaultValue
	}
	return result.Data
}

// UnwrapOrElse extracts data from Result or computes a default value.
//
// Arguments:
//   - result: The Result[T] to unwrap.
//   - defaultFunc: Function to compute default value if result contains error.
//
// Returns:
//   - T: The data from result or the computed default value.
//
// Example:
//
//	result := Execute(op, ctx, req)
//	page := UnwrapOrElse(result, func(err error) *Page {
//	    log.Printf("Failed to get page: %v", err)
//	    return &Page{Title: "Error Page"}
//	})
func UnwrapOrElse[T any](result Result[T], defaultFunc func(error) T) T {
	if result.IsError() {
		return defaultFunc(result.Error)
	}
	return result.Data
}

// CollectGoResults collects streaming results into Go's ([]T, error) pattern.
// This stops at the first error, returning partial results and the error.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Channel of results to collect.
//
// Returns:
//   - []T: Slice of successful results up to the first error.
//   - error: The first error encountered, or nil if all succeeded.
//
// Example:
//
//	results, err := CollectGoResults(ctx, streamChannel)
//	if err != nil {
//	    log.Printf("Collection failed: %v", err)
//	    // results contains partial data up to the error
//	}
func CollectGoResults[T any](ctx context.Context, resultCh <-chan Result[T]) ([]T, error) {
	return CollectResults(ctx, resultCh)
}

// ExecuteGo wraps Execute to return Go's idiomatic (T, error) pattern.
//
// Arguments:
//   - op: The operator to use.
//   - ctx: Context for cancellation and timeouts.
//   - req: The request to execute.
//
// Returns:
//   - T: The result data.
//   - error: Any error that occurred.
//
// Example:
//
//	page, err := ExecuteGo(op, ctx, req)
//	if err != nil {
//	    return fmt.Errorf("failed to execute: %w", err)
//	}
//	return page, nil
func ExecuteGo[T any, R RequestInterface[B], B RequestBody](op *Operator[T], ctx context.Context, req R) (T, error) {
	result := Execute(op, ctx, req)
	return ToGoResult(result)
}

// StreamGo converts a Result channel to Go's idiomatic channel pattern.
// It returns separate channels for data and errors.
//
// Arguments:
//   - ctx: Context for cancellation.
//   - resultCh: Channel of Result[T] to convert.
//
// Returns:
//   - <-chan T: Channel of successful data.
//   - <-chan error: Channel of errors.
//
// Example:
//
//	dataCh, errCh := StreamGo(ctx, resultChannel)
//	for {
//	    select {
//	    case data, ok := <-dataCh:
//	        if !ok {
//	            return // Channel closed
//	        }
//	        processData(data)
//	    case err := <-errCh:
//	        if err != nil {
//	            log.Printf("Error: %v", err)
//	        }
//	    case <-ctx.Done():
//	        return
//	    }
//	}
func StreamGo[T any](ctx context.Context, resultCh <-chan Result[T]) (<-chan T, <-chan error) {
	dataCh := make(chan T)
	errCh := make(chan error)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		for {
			select {
			case result, ok := <-resultCh:
				if !ok {
					return
				}

				if result.IsError() {
					select {
					case errCh <- result.Error:
					case <-ctx.Done():
						return
					}
				} else {
					select {
					case dataCh <- result.Data:
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return dataCh, errCh
}

// WithErrorHandling wraps a Result-returning function to handle errors with a callback.
//
// Arguments:
//   - fn: Function that returns a Result[T].
//   - errorHandler: Function to handle errors.
//
// Returns:
//   - T: The data if successful, zero value if error.
//   - bool: True if successful, false if error was handled.
//
// Example:
//
//	page, ok := WithErrorHandling(
//	    func() Result[*Page] { return Execute(op, ctx, req) },
//	    func(err error) { log.Printf("Operation failed: %v", err) },
//	)
//	if ok {
//	    fmt.Printf("Got page: %s\n", page.Title)
//	}
func WithErrorHandling[T any](fn func() Result[T], errorHandler func(error)) (T, bool) {
	result := fn()
	if result.IsError() {
		errorHandler(result.Error)
		var zero T
		return zero, false
	}
	return result.Data, true
}
