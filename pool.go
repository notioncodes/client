package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of concurrent workers for processing operations.
// This provides controlled concurrency with graceful shutdown and load balancing.
type WorkerPool[T any] struct {
	config   *WorkerPoolConfig
	workers  []*Worker[T]
	workCh   chan Work[T]
	resultCh chan Result[T]
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	started  bool
	stopped  bool
	metrics  *PoolMetrics
}

// WorkerPoolConfig contains configuration for worker pool behavior.
type WorkerPoolConfig struct {
	// NumWorkers is the number of worker goroutines to create.
	NumWorkers int

	// WorkBufferSize is the size of the work queue buffer.
	WorkBufferSize int

	// ResultBufferSize is the size of the result queue buffer.
	ResultBufferSize int

	// WorkerTimeout is the maximum time a worker can spend on a single task.
	WorkerTimeout time.Duration

	// EnableMetrics enables collection of pool metrics.
	EnableMetrics bool

	// MaxQueueSize is the maximum number of items that can be queued.
	// If 0, the queue is unlimited (bounded only by WorkBufferSize).
	MaxQueueSize int

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	ShutdownTimeout time.Duration
}

// DefaultWorkerPoolConfig returns a worker pool configuration with sensible defaults.
//
// Returns:
//   - *WorkerPoolConfig: Default configuration for production use.
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		NumWorkers:       10,
		WorkBufferSize:   100,
		ResultBufferSize: 100,
		WorkerTimeout:    30 * time.Second,
		EnableMetrics:    true,
		MaxQueueSize:     1000,
		ShutdownTimeout:  30 * time.Second,
	}
}

// Work represents a unit of work to be processed by a worker.
type Work[T any] struct {
	// ID is a unique identifier for this work item.
	ID string

	// Handler is the function that processes this work item.
	Handler WorkHandler[T]

	// Context provides cancellation and timeout for this work item.
	Context context.Context

	// Priority can be used for work prioritization (higher values = higher priority).
	Priority int

	// Metadata contains additional information about this work item.
	Metadata map[string]interface{}

	// SubmitTime is when this work item was submitted.
	SubmitTime time.Time
}

// WorkHandler represents a function that processes work and returns a result.
type WorkHandler[T any] func(ctx context.Context, workerID int) (T, error)

// Worker represents a single worker in the pool.
type Worker[T any] struct {
	id       int
	pool     *WorkerPool[T]
	workCh   <-chan Work[T]
	resultCh chan<- Result[T]
	metrics  *WorkerMetrics
}

// PoolMetrics tracks metrics for the entire worker pool.
type PoolMetrics struct {
	WorkersStarted  int64         `json:"workers_started"`
	WorkersActive   int64         `json:"workers_active"`
	TasksQueued     int64         `json:"tasks_queued"`
	TasksProcessed  int64         `json:"tasks_processed"`
	TasksCompleted  int64         `json:"tasks_completed"`
	TasksErrored    int64         `json:"tasks_errored"`
	TasksTimeout    int64         `json:"tasks_timeout"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	AverageExecTime time.Duration `json:"average_exec_time"`
	TotalWaitTime   time.Duration
	TotalExecTime   time.Duration
	mu              sync.RWMutex
}

// WorkerMetrics tracks metrics for a specific worker.
type WorkerMetrics struct {
	WorkerID       int           `json:"worker_id"`
	TasksProcessed int64         `json:"tasks_processed"`
	TasksCompleted int64         `json:"tasks_completed"`
	TasksErrored   int64         `json:"tasks_errored"`
	TasksTimeout   int64         `json:"tasks_timeout"`
	TotalExecTime  time.Duration `json:"total_exec_time"`
	LastTaskTime   time.Time     `json:"last_task_time"`
	IsActive       bool          `json:"is_active"`
}

// NewWorkerPool creates a new worker pool with the specified configuration.
//
// Arguments:
//   - config: Configuration for the worker pool behavior.
//
// Returns:
//   - *WorkerPool[T]: A new worker pool instance.
//
// Example:
//
//	config := DefaultWorkerPoolConfig()
//	config.NumWorkers = 20
//	pool := NewWorkerPool[string](config)
//	pool.Start(ctx)
//	defer pool.Stop()
func NewWorkerPool[T any](config *WorkerPoolConfig) *WorkerPool[T] {
	ctx, cancel := context.WithCancel(context.Background())

	var metrics *PoolMetrics
	if config.EnableMetrics {
		metrics = &PoolMetrics{}
	}

	pool := &WorkerPool[T]{
		config:   config,
		workCh:   make(chan Work[T], config.WorkBufferSize),
		resultCh: make(chan Result[T], config.ResultBufferSize),
		ctx:      ctx,
		cancel:   cancel,
		metrics:  metrics,
	}

	// Create workers
	pool.workers = make([]*Worker[T], config.NumWorkers)
	for i := 0; i < config.NumWorkers; i++ {
		var workerMetrics *WorkerMetrics
		if config.EnableMetrics {
			workerMetrics = &WorkerMetrics{WorkerID: i}
		}

		pool.workers[i] = &Worker[T]{
			id:       i,
			pool:     pool,
			workCh:   pool.workCh,
			resultCh: pool.resultCh,
			metrics:  workerMetrics,
		}
	}

	return pool
}

// Start starts all workers in the pool.
//
// Arguments:
//   - ctx: Context for the worker pool lifecycle.
//
// Example:
//
//	ctx := context.Background()
//	pool.Start(ctx)
func (wp *WorkerPool[T]) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return fmt.Errorf("worker pool already started")
	}

	if wp.stopped {
		return fmt.Errorf("worker pool has been stopped")
	}

	// Start all workers
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.run(ctx, &wp.wg)
	}

	wp.started = true

	if wp.metrics != nil {
		atomic.StoreInt64(&wp.metrics.WorkersStarted, int64(len(wp.workers)))
	}

	return nil
}

// Submit submits work to the pool for processing.
//
// Arguments:
//   - ctx: Context for this work item (used for cancellation/timeout).
//   - handler: Function to execute for this work item.
//
// Returns:
//   - error: Error if the work could not be submitted.
//
// Example:
//
//	err := pool.Submit(ctx, func(ctx context.Context, workerID int) (string, error) {
//	    // Do some work...
//	    return "result", nil
//	})
//	if err != nil {
//	    log.Printf("Failed to submit work: %v", err)
//	}
func (wp *WorkerPool[T]) Submit(ctx context.Context, handler WorkHandler[T]) error {
	return wp.SubmitWork(Work[T]{
		ID:         generateWorkID(),
		Handler:    handler,
		Context:    ctx,
		SubmitTime: time.Now(),
	})
}

// SubmitWork submits a work item to the pool for processing.
//
// Arguments:
//   - work: The work item to submit.
//
// Returns:
//   - error: Error if the work could not be submitted.
//
// Example:
//
//	work := Work[string]{
//	    ID: "task-123",
//	    Handler: myHandler,
//	    Context: ctx,
//	    Priority: 10,
//	}
//	err := pool.SubmitWork(work)
func (wp *WorkerPool[T]) SubmitWork(work Work[T]) error {
	wp.mu.RLock()
	if !wp.started || wp.stopped {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool not running")
	}
	wp.mu.RUnlock()

	// Check queue size limit
	if wp.config.MaxQueueSize > 0 {
		if len(wp.workCh) >= wp.config.MaxQueueSize {
			return &ConcurrencyError{
				Operation: "submit_work",
				Reason:    "work queue is full",
			}
		}
	}

	select {
	case wp.workCh <- work:
		if wp.metrics != nil {
			atomic.AddInt64(&wp.metrics.TasksQueued, 1)
		}
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	case <-work.Context.Done():
		return work.Context.Err()
	}
}

// Results returns a channel that receives results from completed work items.
//
// Returns:
//   - <-chan Result[T]: Channel of results from processed work items.
//
// Example:
//
//	results := pool.Results()
//	for result := range results {
//	    if result.IsSuccess() {
//	        fmt.Printf("Work completed: %v\n", result.Data)
//	    } else {
//	        fmt.Printf("Work failed: %v\n", result.Error)
//	    }
//	}
func (wp *WorkerPool[T]) Results() <-chan Result[T] {
	return wp.resultCh
}

// Stop gracefully stops the worker pool and waits for all workers to finish.
//
// Example:
//
//	// Gracefully shutdown the pool
//	if err := pool.Stop(); err != nil {
//	    log.Printf("Error stopping pool: %v", err)
//	}
func (wp *WorkerPool[T]) Stop() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopped {
		return fmt.Errorf("worker pool already stopped")
	}

	wp.stopped = true

	// Close work channel to signal workers to stop
	close(wp.workCh)

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), wp.config.ShutdownTimeout)
	defer shutdownCancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers finished gracefully
	case <-shutdownCtx.Done():
		// Timeout reached, force cancellation
		wp.cancel()
		wp.wg.Wait() // Wait for forced shutdown
	}

	// Close result channel
	close(wp.resultCh)

	return nil
}

// GetMetrics returns current pool metrics if metrics are enabled.
//
// Returns:
//   - *PoolMetrics: Current metrics, or nil if metrics are disabled.
//
// Example:
//
//	if metrics := pool.GetMetrics(); metrics != nil {
//	    fmt.Printf("Tasks processed: %d\n", metrics.TasksProcessed)
//	}
func (wp *WorkerPool[T]) GetMetrics() *PoolMetrics {
	if wp.metrics == nil {
		return nil
	}

	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	// Calculate averages
	var avgWaitTime, avgExecTime time.Duration
	if wp.metrics.TasksCompleted > 0 {
		avgWaitTime = wp.metrics.TotalWaitTime / time.Duration(wp.metrics.TasksCompleted)
		avgExecTime = wp.metrics.TotalExecTime / time.Duration(wp.metrics.TasksCompleted)
	}

	return &PoolMetrics{
		WorkersStarted:  atomic.LoadInt64(&wp.metrics.WorkersStarted),
		WorkersActive:   atomic.LoadInt64(&wp.metrics.WorkersActive),
		TasksQueued:     atomic.LoadInt64(&wp.metrics.TasksQueued),
		TasksProcessed:  atomic.LoadInt64(&wp.metrics.TasksProcessed),
		TasksCompleted:  atomic.LoadInt64(&wp.metrics.TasksCompleted),
		TasksErrored:    atomic.LoadInt64(&wp.metrics.TasksErrored),
		TasksTimeout:    atomic.LoadInt64(&wp.metrics.TasksTimeout),
		AverageWaitTime: avgWaitTime,
		AverageExecTime: avgExecTime,
	}
}

// GetWorkerMetrics returns metrics for all workers if metrics are enabled.
//
// Returns:
//   - []*WorkerMetrics: Slice of worker metrics, or nil if metrics are disabled.
func (wp *WorkerPool[T]) GetWorkerMetrics() []*WorkerMetrics {
	if wp.metrics == nil {
		return nil
	}

	wp.mu.RLock()
	defer wp.mu.RUnlock()

	workerMetrics := make([]*WorkerMetrics, len(wp.workers))
	for i, worker := range wp.workers {
		if worker.metrics != nil {
			workerMetrics[i] = &WorkerMetrics{
				WorkerID:       worker.metrics.WorkerID,
				TasksProcessed: atomic.LoadInt64(&worker.metrics.TasksProcessed),
				TasksCompleted: atomic.LoadInt64(&worker.metrics.TasksCompleted),
				TasksErrored:   atomic.LoadInt64(&worker.metrics.TasksErrored),
				TasksTimeout:   atomic.LoadInt64(&worker.metrics.TasksTimeout),
				TotalExecTime:  worker.metrics.TotalExecTime,
				LastTaskTime:   worker.metrics.LastTaskTime,
				IsActive:       worker.metrics.IsActive,
			}
		}
	}

	return workerMetrics
}

// run is the main worker loop that processes work items.
func (w *Worker[T]) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if w.pool.metrics != nil {
		atomic.AddInt64(&w.pool.metrics.WorkersActive, 1)
		defer atomic.AddInt64(&w.pool.metrics.WorkersActive, -1)
	}

	for {
		select {
		case work, ok := <-w.workCh:
			if !ok {
				// Work channel closed, worker should exit
				return
			}

			w.processWork(ctx, work)

		case <-ctx.Done():
			return
		case <-w.pool.ctx.Done():
			return
		}
	}
}

// processWork processes a single work item.
func (w *Worker[T]) processWork(ctx context.Context, work Work[T]) {
	if w.metrics != nil {
		atomic.AddInt64(&w.metrics.TasksProcessed, 1)
		w.metrics.IsActive = true
		w.metrics.LastTaskTime = time.Now()
		defer func() { w.metrics.IsActive = false }()
	}

	if w.pool.metrics != nil {
		atomic.AddInt64(&w.pool.metrics.TasksProcessed, 1)
	}

	// Calculate wait time
	waitTime := time.Since(work.SubmitTime)

	// Create a timeout context for this work item
	workCtx := work.Context
	if w.pool.config.WorkerTimeout > 0 {
		var cancel context.CancelFunc
		workCtx, cancel = context.WithTimeout(work.Context, w.pool.config.WorkerTimeout)
		defer cancel()
	}

	// Execute the work
	startTime := time.Now()
	result, err := work.Handler(workCtx, w.id)
	execTime := time.Since(startTime)

	// Update metrics
	if w.metrics != nil {
		w.metrics.TotalExecTime += execTime
		if err != nil {
			if workCtx.Err() == context.DeadlineExceeded {
				atomic.AddInt64(&w.metrics.TasksTimeout, 1)
			} else {
				atomic.AddInt64(&w.metrics.TasksErrored, 1)
			}
		} else {
			atomic.AddInt64(&w.metrics.TasksCompleted, 1)
		}
	}

	if w.pool.metrics != nil {
		w.pool.metrics.mu.Lock()
		w.pool.metrics.TotalWaitTime += waitTime
		w.pool.metrics.TotalExecTime += execTime
		w.pool.metrics.mu.Unlock()

		if err != nil {
			if workCtx.Err() == context.DeadlineExceeded {
				atomic.AddInt64(&w.pool.metrics.TasksTimeout, 1)
			} else {
				atomic.AddInt64(&w.pool.metrics.TasksErrored, 1)
			}
		} else {
			atomic.AddInt64(&w.pool.metrics.TasksCompleted, 1)
		}
	}

	// Create result with metadata
	metadata := &ResultMetadata{
		RequestID:       work.ID,
		Duration:        execTime,
		RequestDuration: execTime,
	}

	var workResult Result[T]
	if err != nil {
		workResult = ErrorWithMetadata[T](err, metadata)
	} else {
		workResult = SuccessWithMetadata(result, metadata)
	}

	// Send result (non-blocking to prevent deadlock)
	select {
	case w.resultCh <- workResult:
	case <-ctx.Done():
	case <-w.pool.ctx.Done():
	default:
		// Result channel is full, could not send result
		// In a production system, you might want to log this
	}
}

// generateWorkID generates a unique identifier for work items.
func generateWorkID() string {
	return fmt.Sprintf("work-%d-%d", time.Now().UnixNano(), rand.Int63())
}

// ProcessConcurrently is a convenience function that processes a slice of items concurrently
// using a worker pool and returns all results.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - items: Slice of items to process.
//   - processor: Function to process each item.
//   - numWorkers: Number of concurrent workers to use.
//
// Returns:
//   - []Result[U]: Results from processing all items.
//
// Example:
//
//	items := []string{"a", "b", "c", "d", "e"}
//	results := ProcessConcurrently(ctx, items, func(ctx context.Context, item string, workerID int) (int, error) {
//	    return len(item), nil
//	}, 3)
//
//	for i, result := range results {
//	    if result.IsSuccess() {
//	        fmt.Printf("Item %s has length %d\n", items[i], result.Data)
//	    }
//	}
func ProcessConcurrently[T any, U any](
	ctx context.Context,
	items []T,
	processor func(context.Context, T, int) (U, error),
	numWorkers int,
) []Result[U] {
	if len(items) == 0 {
		return nil
	}

	// Create worker pool
	config := DefaultWorkerPoolConfig()
	config.NumWorkers = numWorkers
	config.WorkBufferSize = len(items)
	config.ResultBufferSize = len(items)

	pool := NewWorkerPool[U](config)
	pool.Start(ctx)
	defer pool.Stop()

	// Submit all work items
	for _, item := range items {
		item := item // Capture loop variable
		pool.Submit(ctx, func(ctx context.Context, workerID int) (U, error) {
			return processor(ctx, item, workerID)
		})
	}

	// Collect all results
	results := make([]Result[U], 0, len(items))
	for i := 0; i < len(items); i++ {
		select {
		case result := <-pool.Results():
			results = append(results, result)
		case <-ctx.Done():
			// Add error result for remaining items
			for j := i; j < len(items); j++ {
				results = append(results, Error[U](ctx.Err()))
			}
			return results
		}
	}

	return results
}

// BatchProcess processes items in batches using concurrent workers.
// This is useful when you want to process large datasets efficiently.
//
// Arguments:
//   - ctx: Context for cancellation and timeouts.
//   - items: Slice of items to process.
//   - batchSize: Number of items to process in each batch.
//   - processor: Function to process each batch.
//   - numWorkers: Number of concurrent workers to use.
//
// Returns:
//   - []Result[U]: Results from processing all batches.
//
// Example:
//
//	items := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
//	results := BatchProcess(ctx, items, 3, func(ctx context.Context, batch []string, workerID int) ([]int, error) {
//	    lengths := make([]int, len(batch))
//	    for i, item := range batch {
//	        lengths[i] = len(item)
//	    }
//	    return lengths, nil
//	}, 2)
func BatchProcess[T any, U any](
	ctx context.Context,
	items []T,
	batchSize int,
	processor func(context.Context, []T, int) (U, error),
	numWorkers int,
) []Result[U] {
	if len(items) == 0 || batchSize <= 0 {
		return nil
	}

	// Create batches
	var batches [][]T
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	// Process batches concurrently
	return ProcessConcurrently(ctx, batches, processor, numWorkers)
}
