package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_BasicFlow(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.MaxFailures = 3
	config.Timeout = 100 * time.Millisecond
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Initially closed
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected circuit breaker to be closed initially")
	}
	
	// Successful requests
	for i := 0; i < 5; i++ {
		err := cb.Execute(ctx, func() error { return nil })
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
	
	// Failed requests to open circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error { return errors.New("test error") })
		if err == nil {
			t.Errorf("Expected error")
		}
	}
	
	// Circuit should be open now
	if cb.GetState() != CircuitBreakerOpen {
		t.Errorf("Expected circuit breaker to be open after failures")
	}
	
	// Requests should fail immediately
	err := cb.Execute(ctx, func() error { return nil })
	if !IsCircuitBreakerError(err) {
		t.Errorf("Expected circuit breaker error, got: %v", err)
	}
	
	// Wait for timeout
	time.Sleep(150 * time.Millisecond)
	
	// Should be half-open now
	if cb.GetState() != CircuitBreakerHalfOpen {
		t.Errorf("Expected circuit breaker to be half-open after timeout")
	}
	
	// Successful request should close circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error { return nil })
		if err != nil {
			t.Errorf("Unexpected error in half-open state: %v", err)
		}
	}
	
	// Should be closed again
	if cb.GetState() != CircuitBreakerClosed {
		t.Errorf("Expected circuit breaker to be closed after successful requests")
	}
}

func TestCircuitBreaker_HalfOpenConcurrency(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.MaxFailures = 1
	config.MaxConcurrentRequests = 2
	config.Timeout = 100 * time.Millisecond
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Open the circuit
	cb.Execute(ctx, func() error { return errors.New("fail") })
	
	// Wait for timeout
	time.Sleep(150 * time.Millisecond)
	
	// Should be half-open
	if cb.GetState() != CircuitBreakerHalfOpen {
		t.Errorf("Expected half-open state")
	}
	
	// Test concurrent requests
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(ctx, func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	
	wg.Wait()
	
	// Should have limited concurrent requests
	if successCount+errorCount != 5 {
		t.Errorf("Expected 5 total requests")
	}
	
	// Some requests should be rejected due to concurrency limit
	if errorCount == 0 {
		t.Errorf("Expected some requests to be rejected due to concurrency limit")
	}
}

func TestRetryExecutor_ExponentialBackoff(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 3
	policy.InitialInterval = 10 * time.Millisecond
	policy.Multiplier = 2.0
	policy.Jitter = false // Disable jitter for predictable testing
	
	executor := NewRetryExecutor(policy)
	ctx := context.Background()
	
	var attempts int64
	start := time.Now()
	
	err := executor.Execute(ctx, func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("test error")
	})
	
	duration := time.Since(start)
	
	// Should have attempted 4 times (initial + 3 retries)
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}
	
	// Should have waited approximately 10ms + 20ms + 40ms = 70ms
	expectedDuration := 70 * time.Millisecond
	if duration < expectedDuration {
		t.Errorf("Expected duration >= %v, got %v", expectedDuration, duration)
	}
	
	if err == nil {
		t.Errorf("Expected error after all retries")
	}
}

func TestRetryExecutor_NonRetryableError(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 3
	policy.RetryableErrors = func(err error) bool {
		return err.Error() != "non-retryable"
	}
	
	executor := NewRetryExecutor(policy)
	ctx := context.Background()
	
	var attempts int64
	
	err := executor.Execute(ctx, func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("non-retryable")
	})
	
	// Should only attempt once
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
	
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestFallbackRegistry_BestFallback(t *testing.T) {
	registry := NewFallbackRegistry()
	
	// Register fallbacks with different qualities
	fallback1 := NewStaticFallbackHandler("fallback1", 0.3, func(op string, err error) bool {
		return true
	})
	fallback2 := NewStaticFallbackHandler("fallback2", 0.8, func(op string, err error) bool {
		return true
	})
	fallback3 := NewStaticFallbackHandler("fallback3", 0.5, func(op string, err error) bool {
		return true
	})
	
	registry.RegisterHandler("test_op", fallback1)
	registry.RegisterHandler("test_op", fallback2)
	registry.RegisterHandler("test_op", fallback3)
	
	// Should select the highest quality fallback
	best := registry.GetFallback("test_op", errors.New("test"))
	if best == nil {
		t.Fatal("Expected fallback handler")
	}
	
	result, err := best.Handle(context.Background(), "test_op", errors.New("test"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result != "fallback2" {
		t.Errorf("Expected fallback2 (highest quality), got %v", result)
	}
}

func TestFallbackRegistry_ConditionalFallback(t *testing.T) {
	registry := NewFallbackRegistry()
	
	// Register fallback that only handles specific errors
	fallback := NewStaticFallbackHandler("handled", 1.0, func(op string, err error) bool {
		return err.Error() == "specific-error"
	})
	
	registry.RegisterHandler("test_op", fallback)
	
	// Should handle specific error
	handler := registry.GetFallback("test_op", errors.New("specific-error"))
	if handler == nil {
		t.Errorf("Expected handler for specific error")
	}
	
	// Should not handle other errors
	handler = registry.GetFallback("test_op", errors.New("other-error"))
	if handler != nil {
		t.Errorf("Expected no handler for other error")
	}
}

func TestResilientWrapper_Integration(t *testing.T) {
	config := ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig("test"),
		Retry:          DefaultRetryPolicy(),
		EnableFallback: true,
	}
	config.CircuitBreaker.MaxFailures = 2
	config.Retry.MaxAttempts = 1
	
	wrapper := NewResilientWrapper(config)
	
	// Register a fallback
	fallback := NewStaticFallbackHandler("fallback-result", 0.8, func(op string, err error) bool {
		return true
	})
	wrapper.RegisterFallback("test_operation", fallback)
	
	ctx := context.Background()
	var attempts int64
	
	// This function will always fail
	failingFn := func() (interface{}, error) {
		atomic.AddInt64(&attempts, 1)
		return nil, errors.New("persistent failure")
	}
	
	// Execute with resilience
	result, err := wrapper.Execute(ctx, "test_operation", failingFn)
	
	// Should have attempted twice (initial + 1 retry)
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	
	// Should have fallen back
	if err != nil {
		t.Errorf("Expected fallback to succeed, got error: %v", err)
	}
	
	if result != "fallback-result" {
		t.Errorf("Expected fallback result, got %v", result)
	}
	
	// Circuit should still be closed (fallback succeeded)
	if wrapper.IsCircuitOpen() {
		t.Errorf("Expected circuit to remain closed after successful fallback")
	}
}

func TestResilientPluginWrapper_MethodExecution(t *testing.T) {
	config := ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig("test-plugin"),
		Retry:          DefaultRetryPolicy(),
		EnableFallback: true,
	}
	config.CircuitBreaker.MaxFailures = 1
	config.Retry.MaxAttempts = 1
	
	// Mock plugin
	plugin := &mockPlugin{}
	health := NewHealthMonitor()
	
	wrapper := NewResilientPluginWrapper("test-plugin", plugin, config, health)
	
	// Register fallback for a method
	fallback := NewStaticFallbackHandler("fallback-output", 0.7, func(op string, err error) bool {
		return true
	})
	wrapper.RegisterFallback("process", fallback)
	
	ctx := context.Background()
	
	// Test successful method execution
	result, err := wrapper.ExecutePluginMethod(ctx, "process", func() (interface{}, error) {
		return "success", nil
	})
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}
	
	// Test failing method with fallback
	result, err = wrapper.ExecutePluginMethod(ctx, "process", func() (interface{}, error) {
		return nil, errors.New("method failed")
	})
	
	if err != nil {
		t.Errorf("Expected fallback to succeed, got error: %v", err)
	}
	
	if result != "fallback-output" {
		t.Errorf("Expected fallback output, got %v", result)
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var stateChanges []string
	var mu sync.Mutex
	
	config := DefaultCircuitBreakerConfig("test")
	config.MaxFailures = 2
	config.Timeout = 50 * time.Millisecond
	config.OnStateChange = func(name string, from, to CircuitBreakerState) {
		mu.Lock()
		defer mu.Unlock()
		stateChanges = append(stateChanges, fmt.Sprintf("%s: %s -> %s", name, from, to))
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Cause failures to open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error { return errors.New("fail") })
	}
	
	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	
	// Trigger half-open
	cb.Execute(ctx, func() error { return nil })
	
	// Close circuit with successes
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func() error { return nil })
	}
	
	mu.Lock()
	defer mu.Unlock()
	
	if len(stateChanges) < 2 {
		t.Errorf("Expected at least 2 state changes, got %d: %v", len(stateChanges), stateChanges)
	}
	
	// Should have transitioned: closed -> open -> half-open -> closed
	expectedTransitions := []string{
		"test: closed -> open",
		"test: open -> half-open",
		"test: half-open -> closed",
	}
	
	for i, expected := range expectedTransitions {
		if i >= len(stateChanges) {
			t.Errorf("Missing expected transition: %s", expected)
			continue
		}
		if stateChanges[i] != expected {
			t.Errorf("Expected transition %s, got %s", expected, stateChanges[i])
		}
	}
}

func TestRetryExecutor_ContextCancellation(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 10
	policy.InitialInterval = 100 * time.Millisecond
	
	executor := NewRetryExecutor(policy)
	
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	
	var attempts int64
	start := time.Now()
	
	err := executor.Execute(ctx, func() error {
		atomic.AddInt64(&attempts, 1)
		return errors.New("test error")
	})
	
	duration := time.Since(start)
	
	// Should be cancelled before completing all retries
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
	
	// Should not have completed all retries
	if attempts >= 10 {
		t.Errorf("Expected fewer than 10 attempts due to cancellation, got %d", attempts)
	}
	
	// Should have been cancelled around the timeout
	if duration > 200*time.Millisecond {
		t.Errorf("Expected duration around 150ms, got %v", duration)
	}
}

func TestResilientWrapper_CircuitBreakerIntegration(t *testing.T) {
	config := ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig("test"),
		Retry:          DefaultRetryPolicy(),
		EnableFallback: false,
	}
	config.CircuitBreaker.MaxFailures = 3
	config.Retry.MaxAttempts = 1
	
	wrapper := NewResilientWrapper(config)
	ctx := context.Background()
	
	// Cause failures to open circuit
	for i := 0; i < 3; i++ {
		_, err := wrapper.Execute(ctx, "test_op", func() (interface{}, error) {
			return nil, errors.New("fail")
		})
		if err == nil {
			t.Errorf("Expected error on attempt %d", i+1)
		}
	}
	
	// Circuit should be open
	if !wrapper.IsCircuitOpen() {
		t.Errorf("Expected circuit to be open")
	}
	
	// Next request should fail immediately with circuit breaker error
	start := time.Now()
	_, err := wrapper.Execute(ctx, "test_op", func() (interface{}, error) {
		return nil, errors.New("fail")
	})
	duration := time.Since(start)
	
	if !IsCircuitBreakerError(err) {
		t.Errorf("Expected circuit breaker error, got %v", err)
	}
	
	// Should fail fast (no retry delay)
	if duration > 10*time.Millisecond {
		t.Errorf("Expected fast failure, took %v", duration)
	}
}

// mockPlugin is a simple plugin implementation for testing
type mockPlugin struct {
	callCount int64
}

func (m *mockPlugin) Process(input string) (string, error) {
	atomic.AddInt64(&m.callCount, 1)
	if input == "fail" {
		return "", errors.New("processing failed")
	}
	return fmt.Sprintf("processed: %s", input), nil
}

func (m *mockPlugin) GetCallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

// Benchmark circuit breaker performance
func BenchmarkCircuitBreaker_SuccessfulRequests(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("benchmark"))
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func() error { return nil })
		}
	})
}

func BenchmarkCircuitBreaker_FailedRequests(b *testing.B) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig("benchmark"))
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func() error { return errors.New("fail") })
		}
	})
}

func BenchmarkRetryExecutor_NoRetries(b *testing.B) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 0
	executor := NewRetryExecutor(policy)
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			executor.Execute(ctx, func() error { return nil })
		}
	})
}

func BenchmarkResilientWrapper_Integration(b *testing.B) {
	config := ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig("benchmark"),
		Retry:          DefaultRetryPolicy(),
		EnableFallback: false,
	}
	config.Retry.MaxAttempts = 0 // No retries for benchmark
	
	wrapper := NewResilientWrapper(config)
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wrapper.Execute(ctx, "test", func() (interface{}, error) {
				return "result", nil
			})
		}
	})
}