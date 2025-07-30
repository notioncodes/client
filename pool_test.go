package client

// import (
// 	"context"
// 	"errors"
// 	"sync/atomic"
// 	"testing"
// 	"time"
// )

// // TestWorkerPool_New tests worker pool creation.
// func TestWorkerPool_New(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	pool := NewWorkerPool[int](config)

// 	if pool == nil {
// 		t.Fatal("Expected pool to be created, got nil")
// 	}

// 	if len(pool.workers) != config.NumWorkers {
// 		t.Errorf("Expected %d workers, got %d", config.NumWorkers, len(pool.workers))
// 	}

// 	pool.Stop()
// }

// // TestWorkerPool_StartStop tests starting and stopping the pool.
// func TestWorkerPool_StartStop(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 3
// 	pool := NewWorkerPool[string](config)

// 	ctx := context.Background()

// 	// Test starting
// 	err := pool.Start(ctx)
// 	if err != nil {
// 		t.Fatalf("Failed to start pool: %v", err)
// 	}

// 	// Test double start (should error)
// 	err = pool.Start(ctx)
// 	if err == nil {
// 		t.Error("Expected error when starting already started pool")
// 	}

// 	// Test stopping
// 	err = pool.Stop()
// 	if err != nil {
// 		t.Fatalf("Failed to stop pool: %v", err)
// 	}

// 	// Test double stop (should error)
// 	err = pool.Stop()
// 	if err == nil {
// 		t.Error("Expected error when stopping already stopped pool")
// 	}
// }

// // TestWorkerPool_SubmitWork tests work submission and processing.
// func TestWorkerPool_SubmitWork(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 2
// 	config.EnableMetrics = true
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit a simple work item
// 	work := Work[int]{
// 		ID: "test-work-1",
// 		Handler: func(ctx context.Context, workerID int) (int, error) {
// 			return 42, nil
// 		},
// 		Context:    ctx,
// 		SubmitTime: time.Now(),
// 	}

// 	err := pool.SubmitWork(work)
// 	if err != nil {
// 		t.Fatalf("Failed to submit work: %v", err)
// 	}

// 	// Get the result
// 	select {
// 	case result := <-pool.Results():
// 		if result.IsError() {
// 			t.Fatalf("Expected successful result, got error: %v", result.Error)
// 		}
// 		if result.Data != 42 {
// 			t.Errorf("Expected result 42, got %d", result.Data)
// 		}
// 		if result.Metadata == nil {
// 			t.Error("Expected metadata to be present")
// 		}
// 		if result.Metadata.RequestID != "test-work-1" {
// 			t.Errorf("Expected request ID 'test-work-1', got %s", result.Metadata.RequestID)
// 		}
// 	case <-time.After(5 * time.Second):
// 		t.Fatal("Timeout waiting for result")
// 	}
// }

// // TestWorkerPool_Submit tests the convenience Submit method.
// func TestWorkerPool_Submit(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 2
// 	pool := NewWorkerPool[string](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit work using convenience method
// 	err := pool.Submit(ctx, func(ctx context.Context, workerID int) (string, error) {
// 		return "hello world", nil
// 	})
// 	if err != nil {
// 		t.Fatalf("Failed to submit work: %v", err)
// 	}

// 	// Get the result
// 	select {
// 	case result := <-pool.Results():
// 		if result.IsError() {
// 			t.Fatalf("Expected successful result, got error: %v", result.Error)
// 		}
// 		if result.Data != "hello world" {
// 			t.Errorf("Expected result 'hello world', got %s", result.Data)
// 		}
// 	case <-time.After(5 * time.Second):
// 		t.Fatal("Timeout waiting for result")
// 	}
// }

// // TestWorkerPool_ErrorHandling tests error handling in work processing.
// func TestWorkerPool_ErrorHandling(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 1
// 	config.EnableMetrics = true
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit work that returns an error
// 	expectedError := errors.New("test error")
// 	err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 		return 0, expectedError
// 	})
// 	if err != nil {
// 		t.Fatalf("Failed to submit work: %v", err)
// 	}

// 	// Get the result
// 	select {
// 	case result := <-pool.Results():
// 		if !result.IsError() {
// 			t.Fatal("Expected error result, got success")
// 		}
// 		if result.Error.Error() != expectedError.Error() {
// 			t.Errorf("Expected error '%v', got '%v'", expectedError, result.Error)
// 		}
// 	case <-time.After(5 * time.Second):
// 		t.Fatal("Timeout waiting for result")
// 	}

// 	// Check metrics
// 	metrics := pool.GetMetrics()
// 	if metrics == nil {
// 		t.Fatal("Expected metrics to be available")
// 	}
// 	if metrics.TasksErrored == 0 {
// 		t.Error("Expected error count to be > 0")
// 	}
// }

// // TestWorkerPool_Timeout tests work item timeouts.
// func TestWorkerPool_Timeout(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 1
// 	config.WorkerTimeout = 100 * time.Millisecond // Very short timeout
// 	config.EnableMetrics = true
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit work that takes longer than the timeout
// 	err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 		select {
// 		case <-time.After(500 * time.Millisecond): // Longer than timeout
// 			return 42, nil
// 		case <-ctx.Done():
// 			return 0, ctx.Err()
// 		}
// 	})
// 	if err != nil {
// 		t.Fatalf("Failed to submit work: %v", err)
// 	}

// 	// Get the result
// 	select {
// 	case result := <-pool.Results():
// 		if !result.IsError() {
// 			t.Fatal("Expected timeout error, got success")
// 		}
// 		// Should be a timeout error
// 		if result.Error != context.DeadlineExceeded {
// 			t.Errorf("Expected DeadlineExceeded error, got %T: %v", result.Error, result.Error)
// 		}
// 	case <-time.After(2 * time.Second):
// 		t.Fatal("Timeout waiting for result")
// 	}

// 	// Check metrics
// 	metrics := pool.GetMetrics()
// 	if metrics != nil && metrics.TasksTimeout == 0 {
// 		t.Error("Expected timeout count to be > 0")
// 	}
// }

// // TestWorkerPool_ConcurrentExecution tests concurrent work processing.
// func TestWorkerPool_ConcurrentExecution(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 5
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	numTasks := 20
// 	var activeWorkers int64
// 	var maxConcurrency int64

// 	// Submit multiple tasks that track concurrency
// 	for i := 0; i < numTasks; i++ {
// 		taskID := i
// 		err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 			// Track concurrent execution
// 			current := atomic.AddInt64(&activeWorkers, 1)
// 			defer atomic.AddInt64(&activeWorkers, -1)

// 			// Update max concurrency seen
// 			for {
// 				max := atomic.LoadInt64(&maxConcurrency)
// 				if current <= max || atomic.CompareAndSwapInt64(&maxConcurrency, max, current) {
// 					break
// 				}
// 			}

// 			// Simulate some work
// 			time.Sleep(50 * time.Millisecond)
// 			return taskID, nil
// 		})
// 		if err != nil {
// 			t.Fatalf("Failed to submit task %d: %v", i, err)
// 		}
// 	}

// 	// Collect all results
// 	results := make([]Result[int], 0, numTasks)
// 	for i := 0; i < numTasks; i++ {
// 		select {
// 		case result := <-pool.Results():
// 			results = append(results, result)
// 		case <-time.After(10 * time.Second):
// 			t.Fatalf("Timeout waiting for result %d", i)
// 		}
// 	}

// 	// Verify all tasks completed successfully
// 	if len(results) != numTasks {
// 		t.Errorf("Expected %d results, got %d", numTasks, len(results))
// 	}

// 	for _, result := range results {
// 		if result.IsError() {
// 			t.Errorf("Task failed: %v", result.Error)
// 		}
// 	}

// 	// Verify we achieved some concurrency
// 	maxConcurrencySeen := atomic.LoadInt64(&maxConcurrency)
// 	if maxConcurrencySeen < 2 {
// 		t.Errorf("Expected some concurrency, max seen was %d", maxConcurrencySeen)
// 	}

// 	t.Logf("Max concurrency achieved: %d", maxConcurrencySeen)
// }

// // TestWorkerPool_QueueLimit tests work queue size limits.
// func TestWorkerPool_QueueLimit(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 1
// 	config.WorkBufferSize = 2
// 	config.MaxQueueSize = 2
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit work that blocks workers
// 	blockCh := make(chan struct{})
// 	for i := 0; i < 3; i++ { // More than queue size
// 		err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 			<-blockCh // Block until we signal
// 			return 42, nil
// 		})
// 		if i < 2 {
// 			// First two should succeed
// 			if err != nil {
// 				t.Errorf("Expected task %d to succeed, got error: %v", i, err)
// 			}
// 		} else {
// 			// Third should fail due to queue limit
// 			if err == nil {
// 				t.Error("Expected queue full error for third task")
// 			}
// 			if concErr, ok := err.(*ConcurrencyError); !ok {
// 				t.Errorf("Expected ConcurrencyError, got %T", err)
// 			} else if concErr.Operation != "submit_work" {
// 				t.Errorf("Expected submit_work operation, got %s", concErr.Operation)
// 			}
// 		}
// 	}

// 	// Unblock workers
// 	close(blockCh)

// 	// Collect results from successful submissions
// 	for i := 0; i < 2; i++ {
// 		select {
// 		case result := <-pool.Results():
// 			if result.IsError() {
// 				t.Errorf("Task %d failed: %v", i, result.Error)
// 			}
// 		case <-time.After(5 * time.Second):
// 			t.Fatalf("Timeout waiting for result %d", i)
// 		}
// 	}
// }

// // TestWorkerPool_Metrics tests metrics collection.
// func TestWorkerPool_Metrics(t *testing.T) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 2
// 	config.EnableMetrics = true
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	// Submit some work
// 	numTasks := 5
// 	for i := 0; i < numTasks; i++ {
// 		err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 			time.Sleep(10 * time.Millisecond) // Small delay for metrics
// 			return 42, nil
// 		})
// 		if err != nil {
// 			t.Fatalf("Failed to submit task %d: %v", i, err)
// 		}
// 	}

// 	// Collect all results
// 	for i := 0; i < numTasks; i++ {
// 		select {
// 		case result := <-pool.Results():
// 			if result.IsError() {
// 				t.Errorf("Task %d failed: %v", i, result.Error)
// 			}
// 		case <-time.After(5 * time.Second):
// 			t.Fatalf("Timeout waiting for result %d", i)
// 		}
// 	}

// 	// Check metrics
// 	metrics := pool.GetMetrics()
// 	if metrics == nil {
// 		t.Fatal("Expected metrics to be available")
// 	}

// 	if metrics.WorkersStarted != int64(config.NumWorkers) {
// 		t.Errorf("Expected %d workers started, got %d", config.NumWorkers, metrics.WorkersStarted)
// 	}

// 	if metrics.TasksProcessed < int64(numTasks) {
// 		t.Errorf("Expected at least %d tasks processed, got %d", numTasks, metrics.TasksProcessed)
// 	}

// 	if metrics.TasksCompleted != int64(numTasks) {
// 		t.Errorf("Expected %d tasks completed, got %d", numTasks, metrics.TasksCompleted)
// 	}

// 	if metrics.AverageExecTime <= 0 {
// 		t.Error("Expected positive average execution time")
// 	}

// 	// Check worker metrics
// 	workerMetrics := pool.GetWorkerMetrics()
// 	if len(workerMetrics) != config.NumWorkers {
// 		t.Errorf("Expected %d worker metrics, got %d", config.NumWorkers, len(workerMetrics))
// 	}

// 	for i, wm := range workerMetrics {
// 		if wm == nil {
// 			t.Errorf("Worker %d metrics is nil", i)
// 			continue
// 		}
// 		if wm.WorkerID != i {
// 			t.Errorf("Expected worker ID %d, got %d", i, wm.WorkerID)
// 		}
// 	}
// }

// // TestProcessConcurrently tests the convenience function for concurrent processing.
// func TestProcessConcurrently(t *testing.T) {
// 	ctx := context.Background()

// 	// Test data
// 	items := []string{"hello", "world", "test", "concurrent"}

// 	// Process items concurrently (calculate lengths)
// 	results := ProcessConcurrently(ctx, items, func(ctx context.Context, item string, workerID int) (int, error) {
// 		// Simulate some work
// 		time.Sleep(10 * time.Millisecond)
// 		return len(item), nil
// 	}, 2)

// 	// Verify results
// 	if len(results) != len(items) {
// 		t.Errorf("Expected %d results, got %d", len(items), len(results))
// 	}

// 	expectedLengths := []int{5, 5, 4, 10} // lengths of "hello", "world", "test", "concurrent"

// 	for i, result := range results {
// 		if result.IsError() {
// 			t.Errorf("Result %d failed: %v", i, result.Error)
// 			continue
// 		}

// 		// Note: Results may not be in order due to concurrent processing
// 		// For this test, we'll just verify all expected lengths are present
// 		found := false
// 		for _, expected := range expectedLengths {
// 			if result.Data == expected {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			t.Errorf("Unexpected result length: %d", result.Data)
// 		}
// 	}
// }

// // TestBatchProcess tests batch processing functionality.
// func TestBatchProcess(t *testing.T) {
// 	ctx := context.Background()

// 	// Test data
// 	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// 	// Process items in batches of 3
// 	results := BatchProcess(ctx, items, 3, func(ctx context.Context, batch []int, workerID int) (int, error) {
// 		// Sum the batch
// 		sum := 0
// 		for _, item := range batch {
// 			sum += item
// 		}
// 		return sum, nil
// 	}, 2)

// 	// Should have 4 batches: [1,2,3], [4,5,6], [7,8,9], [10]
// 	expectedResults := []int{6, 15, 24, 10} // sums of each batch

// 	if len(results) != len(expectedResults) {
// 		t.Errorf("Expected %d results, got %d", len(expectedResults), len(results))
// 	}

// 	// Collect all sums
// 	var sums []int
// 	for _, result := range results {
// 		if result.IsError() {
// 			t.Errorf("Batch processing failed: %v", result.Error)
// 			continue
// 		}
// 		sums = append(sums, result.Data)
// 	}

// 	// Verify total sum is correct (regardless of order)
// 	totalSum := 0
// 	for _, sum := range sums {
// 		totalSum += sum
// 	}

// 	expectedTotal := 55 // sum of 1 to 10
// 	if totalSum != expectedTotal {
// 		t.Errorf("Expected total sum %d, got %d", expectedTotal, totalSum)
// 	}
// }

// // BenchmarkWorkerPool_Submit benchmarks work submission and processing.
// func BenchmarkWorkerPool_Submit(b *testing.B) {
// 	config := DefaultWorkerPoolConfig()
// 	config.NumWorkers = 10
// 	config.EnableMetrics = false // Disable metrics for cleaner benchmarks
// 	pool := NewWorkerPool[int](config)

// 	ctx := context.Background()
// 	pool.Start(ctx)
// 	defer pool.Stop()

// 	b.ResetTimer()
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			err := pool.Submit(ctx, func(ctx context.Context, workerID int) (int, error) {
// 				return 42, nil
// 			})
// 			if err != nil {
// 				b.Fatalf("Failed to submit work: %v", err)
// 			}

// 			// Consume the result
// 			select {
// 			case result := <-pool.Results():
// 				if result.IsError() {
// 					b.Fatalf("Work failed: %v", result.Error)
// 				}
// 			case <-time.After(5 * time.Second):
// 				b.Fatal("Timeout waiting for result")
// 			}
// 		}
// 	})
// }

// // BenchmarkProcessConcurrently benchmarks the ProcessConcurrently function.
// func BenchmarkProcessConcurrently(b *testing.B) {
// 	items := make([]int, 100)
// 	for i := range items {
// 		items[i] = i
// 	}

// 	ctx := context.Background()

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		results := ProcessConcurrently(ctx, items, func(ctx context.Context, item int, workerID int) (int, error) {
// 			return item * 2, nil
// 		}, 10)

// 		// Verify all results are successful
// 		for _, result := range results {
// 			if result.IsError() {
// 				b.Fatalf("Processing failed: %v", result.Error)
// 			}
// 		}
// 	}
// }
