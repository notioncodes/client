package client

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and tracks various metrics about HTTP client operations.
// This provides observability into client performance, error rates, and usage patterns.
type MetricsCollector struct {
	mu sync.RWMutex

	// Atomic counters for high-frequency operations
	totalRequests      int64
	totalErrors        int64
	totalSuccesses     int64
	totalRetries       int64
	circuitBreakerHits int64
	rateLimitHits      int64
	throttleWaits      int64
	throttleWaitTime   int64 // Total time spent waiting for throttling (nanoseconds)

	// Protected by mutex for complex operations
	operationStats    map[string]*OperationStats
	errorStats        map[string]int64
	statusStats       map[int]int64
	pathStats         map[string]int64
	methodStats       map[string]int64
	responseTimeStats *ResponseTimeStats
	totalBytesIn      int64
	totalBytesOut     int64
	startTime         time.Time
}

// OperationStats tracks statistics for a specific operation (e.g., "GET /pages").
type OperationStats struct {
	Count         int64         `json:"count"`
	Errors        int64         `json:"errors"`
	TotalDuration time.Duration `json:"total_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	LastCall      time.Time     `json:"last_call"`
}

// ResponseTimeStats tracks response time distribution.
type ResponseTimeStats struct {
	P50 time.Duration `json:"p50"`
	P90 time.Duration `json:"p90"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`

	samples []time.Duration
	dirty   bool
}

// Metrics represents a snapshot of all collected metrics.
type Metrics struct {
	// Basic counters
	TotalRequests      int64 `json:"total_requests"`
	TotalErrors        int64 `json:"total_errors"`
	TotalSuccesses     int64 `json:"total_successes"`
	TotalRetries       int64 `json:"total_retries"`
	CircuitBreakerHits int64 `json:"circuit_breaker_hits"`
	RateLimitHits      int64 `json:"rate_limit_hits"`
	ThrottleWaits      int64 `json:"throttle_waits"`
	ThrottleWaitTime   time.Duration `json:"throttle_wait_time"`

	// Derived metrics
	ErrorRate         float64 `json:"error_rate"`
	SuccessRate       float64 `json:"success_rate"`
	AverageRetries    float64 `json:"average_retries"`
	RequestsPerSecond float64 `json:"requests_per_second"`

	// Operation-specific stats
	Operations map[string]*OperationStats `json:"operations"`

	// Request breakdown by various dimensions
	ErrorBreakdown  map[string]int64 `json:"error_breakdown"`
	StatusBreakdown map[int]int64    `json:"status_breakdown"`
	PathBreakdown   map[string]int64 `json:"path_breakdown"`
	MethodBreakdown map[string]int64 `json:"method_breakdown"`

	// Byte tracking
	TotalBytesIn  int64 `json:"total_bytes_in"`
	TotalBytesOut int64 `json:"total_bytes_out"`

	// Response time statistics
	ResponseTimes *ResponseTimeStats `json:"response_times"`

	// Collection metadata
	CollectionStart time.Time     `json:"collection_start"`
	CollectionTime  time.Time     `json:"collection_time"`
	Uptime          time.Duration `json:"uptime"`
}

// NewMetricsCollector creates a new metrics collector.
//
// Returns:
//   - *MetricsCollector: A new metrics collector instance.
//
// Example:
//
//	collector := NewMetricsCollector()
//	// Use collector to track metrics...
//	metrics := collector.GetMetrics()
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		operationStats:    make(map[string]*OperationStats),
		errorStats:        make(map[string]int64),
		statusStats:       make(map[int]int64),
		pathStats:         make(map[string]int64),
		methodStats:       make(map[string]int64),
		responseTimeStats: &ResponseTimeStats{samples: make([]time.Duration, 0, 1000)},
		startTime:         time.Now(),
	}
}

// RecordRequest records metrics for a completed request with comprehensive tracking.
//
// Arguments:
//   - operation: Description of the operation (e.g., "GET /pages").
//   - method: HTTP method (GET, POST, etc.).
//   - path: Request path.
//   - statusCode: HTTP status code (0 if error occurred before response).
//   - duration: How long the operation took.
//   - bytesIn: Response bytes received.
//   - bytesOut: Request bytes sent.
//   - retryCount: Number of retries attempted.
//   - err: Error that occurred (nil if successful).
//
// Example:
//
//	start := time.Now()
//	resp, err := httpClient.Get(ctx, "/pages", nil)
//	collector.RecordRequest("GET /pages", "GET", "/pages", resp.StatusCode, time.Since(start), bytesIn, bytesOut, 0, err)
func (mc *MetricsCollector) RecordRequest(operation, method, path string, statusCode int, duration time.Duration, bytesIn, bytesOut, retryCount int64, err error) {
	atomic.AddInt64(&mc.totalRequests, 1)
	atomic.AddInt64(&mc.totalRetries, retryCount)
	atomic.AddInt64(&mc.totalBytesIn, bytesIn)
	atomic.AddInt64(&mc.totalBytesOut, bytesOut)

	if err != nil {
		atomic.AddInt64(&mc.totalErrors, 1)

		// Record error type
		mc.mu.Lock()
		errorType := getErrorType(err)
		mc.errorStats[errorType]++

		// Handle specific error types
		switch err.(type) {
		case *RateLimitError:
			atomic.AddInt64(&mc.rateLimitHits, 1)
		case *CircuitBreakerError:
			atomic.AddInt64(&mc.circuitBreakerHits, 1)
		}
		mc.mu.Unlock()
	} else {
		atomic.AddInt64(&mc.totalSuccesses, 1)
	}

	// Record operation-specific stats
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Track by method, path, and status code
	mc.methodStats[method]++
	mc.pathStats[path]++
	if statusCode > 0 {
		mc.statusStats[statusCode]++
	}

	stats, exists := mc.operationStats[operation]
	if !exists {
		stats = &OperationStats{
			MinDuration: duration,
			MaxDuration: duration,
		}
		mc.operationStats[operation] = stats
	}

	stats.Count++
	stats.TotalDuration += duration
	stats.LastCall = time.Now()

	if err != nil {
		stats.Errors++
	}

	if duration < stats.MinDuration {
		stats.MinDuration = duration
	}
	if duration > stats.MaxDuration {
		stats.MaxDuration = duration
	}

	// Record response time for percentile calculations
	mc.responseTimeStats.addSample(duration)
}

// RecordRetry records a retry attempt.
//
// Example:
//
//	collector.RecordRetry()
func (mc *MetricsCollector) RecordRetry() {
	atomic.AddInt64(&mc.totalRetries, 1)
}

// RecordThrottle records a throttling event where a request was delayed.
//
// Arguments:
//   - waitTime: The duration the request was delayed due to throttling.
//
// Example:
//
//	collector.RecordThrottle(100 * time.Millisecond)
func (mc *MetricsCollector) RecordThrottle(waitTime time.Duration) {
	atomic.AddInt64(&mc.throttleWaits, 1)
	atomic.AddInt64(&mc.throttleWaitTime, int64(waitTime))
}

// RecordRequestSimple records metrics for a completed request with basic information.
// This is a backward compatibility method for existing code.
//
// Arguments:
//   - operation: Description of the operation (e.g., "GET /pages").
//   - duration: How long the operation took.
//   - err: Error that occurred (nil if successful).
//
// Example:
//
//	start := time.Now()
//	resp, err := httpClient.Get(ctx, "/pages", nil)
//	collector.RecordRequestSimple("GET /pages", time.Since(start), err)
func (mc *MetricsCollector) RecordRequestSimple(operation string, duration time.Duration, err error) {
	// Extract method and path from operation string
	parts := strings.Fields(operation)
	method := "UNKNOWN"
	path := "/unknown"
	if len(parts) >= 2 {
		method = parts[0]
		path = parts[1]
	}
	
	mc.RecordRequest(operation, method, path, 0, duration, 0, 0, 0, err)
}

// GetMetrics returns a snapshot of current metrics.
//
// Returns:
//   - *Metrics: Current metrics snapshot.
//
// Example:
//
//	metrics := collector.GetMetrics()
//	fmt.Printf("Success rate: %.2f%%\n", metrics.SuccessRate*100)
func (mc *MetricsCollector) GetMetrics() *Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalRequests := atomic.LoadInt64(&mc.totalRequests)
	totalErrors := atomic.LoadInt64(&mc.totalErrors)
	totalRetries := atomic.LoadInt64(&mc.totalRetries)

	now := time.Now()
	uptime := now.Sub(mc.startTime)

	// Calculate derived metrics
	var errorRate, successRate, avgRetries, requestsPerSecond float64

	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests)
		successRate = 1.0 - errorRate
		avgRetries = float64(totalRetries) / float64(totalRequests)
		requestsPerSecond = float64(totalRequests) / uptime.Seconds()
	}

	// Copy operation stats
	operations := make(map[string]*OperationStats)
	for key, stats := range mc.operationStats {
		operations[key] = &OperationStats{
			Count:         stats.Count,
			Errors:        stats.Errors,
			TotalDuration: stats.TotalDuration,
			MinDuration:   stats.MinDuration,
			MaxDuration:   stats.MaxDuration,
			LastCall:      stats.LastCall,
		}
	}

	// Copy error breakdown
	errorBreakdown := make(map[string]int64)
	for errorType, count := range mc.errorStats {
		errorBreakdown[errorType] = count
	}

	// Calculate response time percentiles
	responseTimesCopy := &ResponseTimeStats{}
	if len(mc.responseTimeStats.samples) > 0 {
		responseTimesCopy = mc.responseTimeStats.calculatePercentiles()
	}

	// Copy status, path, and method breakdowns
	statusBreakdown := make(map[int]int64)
	for status, count := range mc.statusStats {
		statusBreakdown[status] = count
	}

	pathBreakdown := make(map[string]int64)
	for path, count := range mc.pathStats {
		pathBreakdown[path] = count
	}

	methodBreakdown := make(map[string]int64)
	for method, count := range mc.methodStats {
		methodBreakdown[method] = count
	}

	return &Metrics{
		TotalRequests:      totalRequests,
		TotalErrors:        totalErrors,
		TotalSuccesses:     atomic.LoadInt64(&mc.totalSuccesses),
		TotalRetries:       totalRetries,
		CircuitBreakerHits: atomic.LoadInt64(&mc.circuitBreakerHits),
		RateLimitHits:      atomic.LoadInt64(&mc.rateLimitHits),
		ThrottleWaits:      atomic.LoadInt64(&mc.throttleWaits),
		ThrottleWaitTime:   time.Duration(atomic.LoadInt64(&mc.throttleWaitTime)),
		ErrorRate:          errorRate,
		SuccessRate:        successRate,
		AverageRetries:     avgRetries,
		RequestsPerSecond:  requestsPerSecond,
		Operations:         operations,
		ErrorBreakdown:     errorBreakdown,
		StatusBreakdown:    statusBreakdown,
		PathBreakdown:      pathBreakdown,
		MethodBreakdown:    methodBreakdown,
		TotalBytesIn:       atomic.LoadInt64(&mc.totalBytesIn),
		TotalBytesOut:      atomic.LoadInt64(&mc.totalBytesOut),
		ResponseTimes:      responseTimesCopy,
		CollectionStart:    mc.startTime,
		CollectionTime:     now,
		Uptime:             uptime,
	}
}

// Reset clears all collected metrics.
//
// Example:
//
//	// After reporting metrics, reset for next collection period
//	collector.Reset()
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Reset atomic counters
	atomic.StoreInt64(&mc.totalRequests, 0)
	atomic.StoreInt64(&mc.totalErrors, 0)
	atomic.StoreInt64(&mc.totalSuccesses, 0)
	atomic.StoreInt64(&mc.totalRetries, 0)
	atomic.StoreInt64(&mc.circuitBreakerHits, 0)
	atomic.StoreInt64(&mc.rateLimitHits, 0)
	atomic.StoreInt64(&mc.throttleWaits, 0)
	atomic.StoreInt64(&mc.throttleWaitTime, 0)
	atomic.StoreInt64(&mc.totalBytesIn, 0)
	atomic.StoreInt64(&mc.totalBytesOut, 0)

	// Reset maps
	mc.operationStats = make(map[string]*OperationStats)
	mc.errorStats = make(map[string]int64)
	mc.statusStats = make(map[int]int64)
	mc.pathStats = make(map[string]int64)
	mc.methodStats = make(map[string]int64)

	// Reset response time stats
	mc.responseTimeStats = &ResponseTimeStats{samples: make([]time.Duration, 0, 1000)}

	// Reset start time
	mc.startTime = time.Now()
}

// GetOperationStats returns statistics for a specific operation.
//
// Arguments:
//   - operation: The operation name to get stats for.
//
// Returns:
//   - *OperationStats: Statistics for the operation, or nil if not found.
//
// Example:
//
//	stats := collector.GetOperationStats("GET /pages")
//	if stats != nil {
//	    fmt.Printf("Average duration: %v\n", stats.TotalDuration/time.Duration(stats.Count))
//	}
func (mc *MetricsCollector) GetOperationStats(operation string) *OperationStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats, exists := mc.operationStats[operation]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	return &OperationStats{
		Count:         stats.Count,
		Errors:        stats.Errors,
		TotalDuration: stats.TotalDuration,
		MinDuration:   stats.MinDuration,
		MaxDuration:   stats.MaxDuration,
		LastCall:      stats.LastCall,
	}
}

// addSample adds a response time sample for percentile calculation.
func (rts *ResponseTimeStats) addSample(duration time.Duration) {
	rts.samples = append(rts.samples, duration)
	rts.dirty = true

	// Keep only the last 1000 samples to prevent unbounded memory growth
	if len(rts.samples) > 1000 {
		copy(rts.samples, rts.samples[len(rts.samples)-1000:])
		rts.samples = rts.samples[:1000]
	}
}

// calculatePercentiles calculates response time percentiles from samples.
func (rts *ResponseTimeStats) calculatePercentiles() *ResponseTimeStats {
	if len(rts.samples) == 0 {
		return &ResponseTimeStats{}
	}

	// Sort samples for percentile calculation
	samples := make([]time.Duration, len(rts.samples))
	copy(samples, rts.samples)

	// Simple insertion sort (efficient for small arrays)
	for i := 1; i < len(samples); i++ {
		key := samples[i]
		j := i - 1
		for j >= 0 && samples[j] > key {
			samples[j+1] = samples[j]
			j--
		}
		samples[j+1] = key
	}

	result := &ResponseTimeStats{}
	length := len(samples)

	// Calculate percentiles
	result.P50 = samples[int(float64(length)*0.50)]
	result.P90 = samples[int(float64(length)*0.90)]
	result.P95 = samples[int(float64(length)*0.95)]
	result.P99 = samples[int(float64(length)*0.99)]

	return result
}

// getErrorType extracts a user-friendly error type from an error.
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	switch err.(type) {
	case *HTTPError:
		httpErr := err.(*HTTPError)
		if httpErr.IsRateLimited() {
			return "rate_limit"
		} else if httpErr.IsServerError() {
			return "server_error"
		} else if httpErr.IsBadRequest() {
			return "bad_request"
		} else if httpErr.IsUnauthorized() {
			return "unauthorized"
		} else if httpErr.IsForbidden() {
			return "forbidden"
		} else if httpErr.IsNotFound() {
			return "not_found"
		}
		return "http_error"
	case *RateLimitError:
		return "rate_limit"
	case *NetworkError:
		return "network_error"
	case *TimeoutError:
		return "timeout"
	case *AuthenticationError:
		return "authentication"
	case *AuthorizationError:
		return "authorization"
	case *ValidationError:
		return "validation"
	case *SerializationError:
		return "serialization"
	case *CircuitBreakerError:
		return "circuit_breaker"
	case *PaginationError:
		return "pagination"
	case *ConcurrencyError:
		return "concurrency"
	default:
		return "unknown"
	}
}

// AverageResponseTime calculates the average response time for an operation.
//
// Returns:
//   - time.Duration: Average response time, or 0 if no requests recorded.
func (os *OperationStats) AverageResponseTime() time.Duration {
	if os.Count == 0 {
		return 0
	}
	return os.TotalDuration / time.Duration(os.Count)
}

// ErrorRate calculates the error rate for an operation.
//
// Returns:
//   - float64: Error rate as a value between 0.0 and 1.0.
func (os *OperationStats) ErrorRate() float64 {
	if os.Count == 0 {
		return 0.0
	}
	return float64(os.Errors) / float64(os.Count)
}

// SuccessRate calculates the success rate for an operation.
//
// Returns:
//   - float64: Success rate as a value between 0.0 and 1.0.
func (os *OperationStats) SuccessRate() float64 {
	return 1.0 - os.ErrorRate()
}

// MetricsSnapshot provides a convenient way to take periodic snapshots of metrics.
type MetricsSnapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Metrics   *Metrics  `json:"metrics"`
}

// TakeSnapshot creates a snapshot of current metrics with a timestamp.
//
// Returns:
//   - *MetricsSnapshot: A timestamped snapshot of current metrics.
//
// Example:
//
//	snapshot := collector.TakeSnapshot()
//	fmt.Printf("Snapshot taken at %v\n", snapshot.Timestamp)
func (mc *MetricsCollector) TakeSnapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   mc.GetMetrics(),
	}
}

// MetricsReporter provides functionality for periodic metrics reporting.
type MetricsReporter struct {
	collector *MetricsCollector
	interval  time.Duration
	callback  func(*Metrics)
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewMetricsReporter creates a new metrics reporter that calls a callback function periodically.
//
// Arguments:
//   - collector: The metrics collector to report from.
//   - interval: How often to report metrics.
//   - callback: Function to call with metrics data.
//
// Returns:
//   - *MetricsReporter: A new metrics reporter instance.
//
// Example:
//
//	reporter := NewMetricsReporter(collector, 60*time.Second, func(metrics *Metrics) {
//	    log.Printf("Requests: %d, Errors: %d, Success Rate: %.2f%%",
//	        metrics.TotalRequests, metrics.TotalErrors, metrics.SuccessRate*100)
//	})
//	reporter.Start()
//	defer reporter.Stop()
func NewMetricsReporter(collector *MetricsCollector, interval time.Duration, callback func(*Metrics)) *MetricsReporter {
	return &MetricsReporter{
		collector: collector,
		interval:  interval,
		callback:  callback,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// Start begins periodic metrics reporting.
func (mr *MetricsReporter) Start() {
	go func() {
		defer close(mr.doneCh)

		ticker := time.NewTicker(mr.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				metrics := mr.collector.GetMetrics()
				mr.callback(metrics)
			case <-mr.stopCh:
				return
			}
		}
	}()
}

// Stop stops the metrics reporter.
func (mr *MetricsReporter) Stop() {
	close(mr.stopCh)
	<-mr.doneCh
}
