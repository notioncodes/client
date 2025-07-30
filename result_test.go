package client

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"testing"
// 	"time"
// )

// // TestResult_IsSuccess tests success state checking.
// func TestResult_IsSuccess(t *testing.T) {
// 	// Test successful result
// 	result := Success("test data")

// 	if !result.IsSuccess() {
// 		t.Error("Expected result to be successful")
// 	}

// 	if result.IsError() {
// 		t.Error("Expected result not to be error")
// 	}

// 	if result.Data != "test data" {
// 		t.Errorf("Expected data 'test data', got %v", result.Data)
// 	}

// 	if result.Error != nil {
// 		t.Errorf("Expected no error, got %v", result.Error)
// 	}
// }

// // TestResult_IsError tests error state checking.
// func TestResult_IsError(t *testing.T) {
// 	// Test error result
// 	testErr := errors.New("test error")
// 	result := Error[string](testErr)

// 	if result.IsSuccess() {
// 		t.Error("Expected result not to be successful")
// 	}

// 	if !result.IsError() {
// 		t.Error("Expected result to be error")
// 	}

// 	if result.Data != "" {
// 		t.Errorf("Expected empty data, got %v", result.Data)
// 	}

// 	if result.Error != testErr {
// 		t.Errorf("Expected error %v, got %v", testErr, result.Error)
// 	}
// }

// // TestResult_WithMetadata tests results with metadata.
// func TestResult_WithMetadata(t *testing.T) {
// 	metadata := &ResultMetadata{
// 		Duration:        time.Second,
// 		RequestDuration: time.Second,
// 		RequestID:       "test-request-123",
// 		FromStream:      true,
// 		StreamPosition:  5,
// 	}

// 	// Test successful result with metadata
// 	result := SuccessWithMetadata("test data", metadata)

// 	if !result.IsSuccess() {
// 		t.Error("Expected result to be successful")
// 	}

// 	if result.Metadata == nil {
// 		t.Fatal("Expected metadata to be present")
// 	}

// 	if result.Metadata.Duration != time.Second {
// 		t.Errorf("Expected duration %v, got %v", time.Second, result.Metadata.Duration)
// 	}

// 	if result.Metadata.RequestID != "test-request-123" {
// 		t.Errorf("Expected request ID 'test-request-123', got %s", result.Metadata.RequestID)
// 	}

// 	// Test error result with metadata
// 	testErr := errors.New("test error")
// 	errorResult := ErrorWithMetadata[string](testErr, metadata)

// 	if !errorResult.IsError() {
// 		t.Error("Expected result to be error")
// 	}

// 	if errorResult.Metadata == nil {
// 		t.Fatal("Expected metadata to be present")
// 	}

// 	if errorResult.Metadata.RequestID != "test-request-123" {
// 		t.Errorf("Expected request ID 'test-request-123', got %s", errorResult.Metadata.RequestID)
// 	}
// }

// // TestStreamProcessing_Map tests the Map function for stream processing.
// func TestStreamProcessing_Map(t *testing.T) {
// 	ctx := context.Background()

// 	// Create input channel
// 	input := make(chan Result[int], 3)
// 	input <- Success(1)
// 	input <- Success(2)
// 	input <- Success(3)
// 	close(input)

// 	// Map function to double values
// 	output := MapResults(ctx, input, func(x int) string {
// 		return fmt.Sprintf("value-%d", x*2)
// 	})

// 	// Collect results
// 	var results []string
// 	for result := range output {
// 		if result.IsError() {
// 			t.Fatalf("Unexpected error: %v", result.Error)
// 		}
// 		results = append(results, result.Data)
// 	}

// 	// Verify results
// 	expected := []string{"value-2", "value-4", "value-6"}
// 	if len(results) != len(expected) {
// 		t.Errorf("Expected %d results, got %d", len(expected), len(results))
// 	}

// 	for i, result := range results {
// 		if i < len(expected) && result != expected[i] {
// 			t.Errorf("Expected result %d to be %s, got %s", i, expected[i], result)
// 		}
// 	}
// }

// // TestStreamProcessing_Filter tests the Filter function for stream processing.
// func TestStreamProcessing_Filter(t *testing.T) {
// 	ctx := context.Background()

// 	// Create input channel
// 	input := make(chan Result[int], 5)
// 	input <- Success(1)
// 	input <- Success(2)
// 	input <- Success(3)
// 	input <- Success(4)
// 	input <- Success(5)
// 	close(input)

// 	// Filter function for even numbers
// 	output := FilterResults(ctx, input, func(x int) bool {
// 		return x%2 == 0
// 	})

// 	// Collect results
// 	var results []int
// 	for result := range output {
// 		if result.IsError() {
// 			t.Fatalf("Unexpected error: %v", result.Error)
// 		}
// 		results = append(results, result.Data)
// 	}

// 	// Verify results (should only have even numbers)
// 	expected := []int{2, 4}
// 	if len(results) != len(expected) {
// 		t.Errorf("Expected %d results, got %d", len(expected), len(results))
// 	}

// 	for i, result := range results {
// 		if i < len(expected) && result != expected[i] {
// 			t.Errorf("Expected result %d to be %d, got %d", i, expected[i], result)
// 		}
// 	}
// }

// // TestStreamProcessing_Batch tests the Batch function for stream processing.
// func TestStreamProcessing_Batch(t *testing.T) {
// 	ctx := context.Background()

// 	// Create input channel
// 	input := make(chan Result[int], 7)
// 	for i := 1; i <= 7; i++ {
// 		input <- Success(i)
// 	}
// 	close(input)

// 	// Batch with size 3
// 	output := BatchResults(ctx, input, 3)

// 	// Collect batches
// 	var batches [][]int
// 	for result := range output {
// 		if result.IsError() {
// 			t.Fatalf("Unexpected error: %v", result.Error)
// 		}
// 		batches = append(batches, result.Data)
// 	}

// 	// Should have 3 batches: [1,2,3], [4,5,6], [7]
// 	expectedBatches := [][]int{
// 		{1, 2, 3},
// 		{4, 5, 6},
// 		{7},
// 	}

// 	if len(batches) != len(expectedBatches) {
// 		t.Errorf("Expected %d batches, got %d", len(expectedBatches), len(batches))
// 	}

// 	for i, batch := range batches {
// 		if i >= len(expectedBatches) {
// 			continue
// 		}
// 		expected := expectedBatches[i]
// 		if len(batch) != len(expected) {
// 			t.Errorf("Batch %d: expected length %d, got %d", i, len(expected), len(batch))
// 			continue
// 		}
// 		for j, item := range batch {
// 			if item != expected[j] {
// 				t.Errorf("Batch %d, item %d: expected %d, got %d", i, j, expected[j], item)
// 			}
// 		}
// 	}
// }

// // TestCollectResults tests collecting stream results into a slice.
// func TestCollectResults(t *testing.T) {
// 	ctx := context.Background()

// 	// Create input channel
// 	input := make(chan Result[string], 3)
// 	input <- Success("first")
// 	input <- Success("second")
// 	input <- Success("third")
// 	close(input)

// 	// Collect results
// 	results, err := CollectResults(ctx, input)
// 	if err != nil {
// 		t.Fatalf("Expected no error, got: %v", err)
// 	}

// 	// Verify results
// 	expected := []string{"first", "second", "third"}
// 	if len(results) != len(expected) {
// 		t.Errorf("Expected %d results, got %d", len(expected), len(results))
// 	}

// 	for i, result := range results {
// 		if i < len(expected) && result != expected[i] {
// 			t.Errorf("Expected result %d to be %s, got %s", i, expected[i], result)
// 		}
// 	}
// }

// // TestCollectResults_WithError tests collecting results when there's an error.
// func TestCollectResults_WithError(t *testing.T) {
// 	ctx := context.Background()

// 	// Create input channel with error
// 	input := make(chan Result[string], 3)
// 	input <- Success("first")
// 	input <- Error[string](errors.New("test error"))
// 	input <- Success("third")
// 	close(input)

// 	// Collect results (should fail on error)
// 	results, err := CollectResults(ctx, input)
// 	if err == nil {
// 		t.Fatal("Expected error, got nil")
// 	}

// 	if err.Error() != "test error" {
// 		t.Errorf("Expected error 'test error', got %v", err)
// 	}

// 	// Should have collected results up to the error
// 	if len(results) != 1 {
// 		t.Errorf("Expected 1 result before error, got %d", len(results))
// 	}

// 	if len(results) > 0 && results[0] != "first" {
// 		t.Errorf("Expected first result to be 'first', got %s", results[0])
// 	}
// }

// // TestCollectResults_ContextCancellation tests context cancellation during collection.
// func TestCollectResults_ContextCancellation(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())

// 	// Create input channel that will block
// 	input := make(chan Result[string])

// 	// Cancel context after a short delay
// 	go func() {
// 		time.Sleep(50 * time.Millisecond)
// 		cancel()
// 	}()

// 	// Try to collect results (should be cancelled)
// 	results, err := CollectResults(ctx, input)

// 	if err == nil {
// 		t.Fatal("Expected context cancellation error, got nil")
// 	}

// 	if err != context.Canceled {
// 		t.Errorf("Expected context.Canceled, got %v", err)
// 	}

// 	if len(results) != 0 {
// 		t.Errorf("Expected no results due to cancellation, got %d", len(results))
// 	}
// }

// // BenchmarkResult_Success benchmarks successful result creation.
// func BenchmarkResult_Success(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		result := Success("test data")
// 		if !result.IsSuccess() {
// 			b.Fatal("Expected successful result")
// 		}
// 	}
// }

// // BenchmarkResult_Error benchmarks error result creation.
// func BenchmarkResult_Error(b *testing.B) {
// 	testErr := errors.New("test error")

// 	for i := 0; i < b.N; i++ {
// 		result := Error[string](testErr)
// 		if !result.IsError() {
// 			b.Fatal("Expected error result")
// 		}
// 	}
// }

// // BenchmarkStreamProcessing_Map benchmarks the Map function.
// func BenchmarkStreamProcessing_Map(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		ctx := context.Background()

// 		// Create input channel
// 		input := make(chan Result[int], 100)
// 		for j := 0; j < 100; j++ {
// 			input <- Success(j)
// 		}
// 		close(input)

// 		// Map function
// 		output := MapResults(ctx, input, func(x int) int {
// 			return x * 2
// 		})

// 		// Consume all results
// 		for range output {
// 			// Just consume
// 		}
// 	}
// }
