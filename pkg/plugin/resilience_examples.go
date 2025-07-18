package plugin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// Example: Basic Circuit Breaker Usage
func ExampleCircuitBreaker_basicUsage() {
	// Configure circuit breaker
	config := CircuitBreakerConfig{
		Name:                  "external-service",
		MaxFailures:          5,
		Timeout:              30 * time.Second,
		MaxConcurrentRequests: 3,
		SuccessThreshold:     2,
		OnStateChange: func(name string, from, to CircuitBreakerState) {
			log.Printf("Circuit breaker %s: %s -> %s", name, from, to)
		},
	}
	
	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	
	// Simulate service calls
	for i := 0; i < 10; i++ {
		err := cb.Execute(ctx, func() error {
			// Simulate external service call
			return callExternalService()
		})
		
		if err != nil {
			if IsCircuitBreakerError(err) {
				log.Printf("Request %d: Circuit breaker is open", i+1)
			} else {
				log.Printf("Request %d: Service error: %v", i+1, err)
			}
		} else {
			log.Printf("Request %d: Success", i+1)
		}
		
		time.Sleep(1 * time.Second)
	}
}

// Example: Retry Policy with Exponential Backoff
func ExampleRetryExecutor_exponentialBackoff() {
	// Configure retry policy
	policy := RetryPolicy{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
		RetryableErrors: func(err error) bool {
			// Only retry temporary errors
			if err.Error() == "temporary-error" {
				return true
			}
			return false
		},
	}
	
	executor := NewRetryExecutor(policy)
	ctx := context.Background()
	
	start := time.Now()
	err := executor.Execute(ctx, func() error {
		// Simulate unreliable operation
		return simulateUnreliableOperation()
	})
	
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("Operation failed after retries: %v (took %v)", err, duration)
	} else {
		log.Printf("Operation succeeded after retries (took %v)", duration)
	}
}

// Example: Fallback Handlers
func ExampleFallbackRegistry_fallbackHandlers() {
	registry := NewFallbackRegistry()
	
	// Register cache fallback for database operations
	cacheHandler := &CacheFallbackHandler{
		cache: make(map[string]interface{}),
		ttl:   5 * time.Minute,
	}
	registry.RegisterHandler("database.read", cacheHandler)
	
	// Register static fallback for configuration
	staticHandler := NewStaticFallbackHandler(
		map[string]string{"default": "value"},
		0.5, // Medium quality
		func(operation string, err error) bool {
			return operation == "config.load"
		},
	)
	registry.RegisterHandler("config.load", staticHandler)
	
	ctx := context.Background()
	
	// Try database operation with fallback
	result, err := registry.ExecuteWithFallback(ctx, "database.read", func() (interface{}, error) {
		return nil, errors.New("database connection failed")
	})
	
	if err != nil {
		log.Printf("Fallback also failed: %v", err)
	} else {
		log.Printf("Fallback result: %v", result)
	}
}

// Example: Complete Resilient Plugin Wrapper
func ExampleResilientPluginWrapper_complete() {
	// Configure resilience patterns
	config := ResilienceConfig{
		CircuitBreaker: CircuitBreakerConfig{
			Name:                  "ai-plugin",
			MaxFailures:          3,
			Timeout:              60 * time.Second,
			MaxConcurrentRequests: 5,
			SuccessThreshold:     2,
			OnStateChange: func(name string, from, to CircuitBreakerState) {
				log.Printf("Plugin %s circuit breaker: %s -> %s", name, from, to)
			},
		},
		Retry: RetryPolicy{
			MaxAttempts:     3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     10 * time.Second,
			Multiplier:      2.0,
			Jitter:          true,
			RetryableErrors: isRetryableError,
		},
		EnableFallback: true,
	}
	
	// Create plugin wrapper
	plugin := &AITextPlugin{}
	health := NewHealthMonitor()
	wrapper := NewResilientPluginWrapper("ai-text", plugin, config, health)
	
	// Register fallbacks
	simpleFallback := NewStaticFallbackHandler(
		"Sorry, I cannot process your request right now.",
		0.3, // Low quality but always available
		func(operation string, err error) bool { return true },
	)
	wrapper.RegisterFallback("generateText", simpleFallback)
	
	ctx := context.Background()
	
	// Execute plugin method with full resilience
	result, err := wrapper.ExecutePluginMethod(ctx, "generateText", func() (interface{}, error) {
		return plugin.GenerateText("Write a short story about robots")
	})
	
	if err != nil {
		log.Printf("All resilience patterns failed: %v", err)
	} else {
		log.Printf("Generated text: %s", result)
	}
	
	// Get resilience statistics
	stats := wrapper.GetStats()
	log.Printf("Resilience stats: %+v", stats)
}

// Example: Health-Aware Resilient Plugin
func ExampleHealthAwareResilientPlugin() {
	// Create health monitor
	health := NewHealthMonitor()
	
	// Configure resilience with health integration
	config := ResilienceConfig{
		CircuitBreaker: CircuitBreakerConfig{
			Name:        "database-plugin",
			MaxFailures: 3,
			Timeout:     30 * time.Second,
			OnStateChange: func(name string, from, to CircuitBreakerState) {
				log.Printf("Circuit breaker state change: %s -> %s", from, to)
				
				// Trigger health check when circuit opens
				if to == CircuitBreakerOpen {
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						
						if report, err := health.CheckNow(ctx, name); err == nil {
							log.Printf("Health check report: %+v", report)
						}
					}()
				}
			},
		},
		Retry:          DefaultRetryPolicy(),
		EnableFallback: true,
	}
	
	plugin := &DatabasePlugin{}
	wrapper := NewResilientPluginWrapper("database", plugin, config, health)
	
	// Register health-aware fallback
	healthFallback := &HealthAwareFallbackHandler{
		health:    health,
		pluginName: "database",
		fallbackData: "cached-result",
	}
	wrapper.resilience.RegisterFallback("database.query", healthFallback)
	
	ctx := context.Background()
	
	// Execute with health-aware resilience
	result, err := wrapper.ExecutePluginMethod(ctx, "query", func() (interface{}, error) {
		return plugin.Query("SELECT * FROM users")
	})
	
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		log.Printf("Query result: %v", result)
	}
}

// Example: Custom Fallback Handler with Quality Assessment
func ExampleCustomFallbackHandler() {
	// Create intelligent fallback that adapts based on error type
	smartFallback := &SmartFallbackHandler{
		primaryCache:   make(map[string]interface{}),
		secondaryCache: make(map[string]interface{}),
		lastUpdated:    time.Now(),
	}
	
	registry := NewFallbackRegistry()
	registry.RegisterHandler("data.fetch", smartFallback)
	
	ctx := context.Background()
	
	// Simulate different types of failures
	scenarios := []struct {
		name string
		err  error
	}{
		{"network timeout", errors.New("network timeout")},
		{"rate limited", errors.New("rate limit exceeded")},
		{"server error", errors.New("internal server error")},
		{"data not found", errors.New("not found")},
	}
	
	for _, scenario := range scenarios {
		result, err := registry.ExecuteWithFallback(ctx, "data.fetch", func() (interface{}, error) {
			return nil, scenario.err
		})
		
		if err != nil {
			log.Printf("Scenario %s: Fallback failed: %v", scenario.name, err)
		} else {
			log.Printf("Scenario %s: Fallback result: %v", scenario.name, result)
		}
	}
}

// Example: Monitoring and Metrics Collection
func ExampleResilienceMetrics() {
	// Create multiple resilient wrappers with monitoring
	plugins := []string{"ai-service", "database", "cache", "external-api"}
	wrappers := make(map[string]*ResilientWrapper)
	
	for _, name := range plugins {
		config := ResilienceConfig{
			CircuitBreaker: DefaultCircuitBreakerConfig(name),
			Retry:          DefaultRetryPolicy(),
			EnableFallback: true,
		}
		
		// Add state change monitoring
		config.CircuitBreaker.OnStateChange = func(pluginName string, from, to CircuitBreakerState) {
			log.Printf("METRIC: circuit_breaker_state_change{plugin=%s,from=%s,to=%s}", 
				pluginName, from, to)
		}
		
		wrappers[name] = NewResilientWrapper(config)
	}
	
	// Collect metrics periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	go func() {
		for range ticker.C {
			collectResilienceMetrics(wrappers)
		}
	}()
	
	// Simulate plugin operations
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		for name, wrapper := range wrappers {
			go func(pluginName string, w *ResilientWrapper) {
				_, err := w.Execute(ctx, "operation", func() (interface{}, error) {
					return simulatePluginOperation(pluginName)
				})
				if err != nil {
					log.Printf("Plugin %s operation failed: %v", pluginName, err)
				}
			}(name, wrapper)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Helper functions for examples

func callExternalService() error {
	// Simulate intermittent failures
	if rand.Float32() < 0.3 {
		return errors.New("service unavailable")
	}
	return nil
}

func simulateUnreliableOperation() error {
	// Simulate operation that fails a few times then succeeds
	if rand.Float32() < 0.7 {
		return errors.New("temporary-error")
	}
	return nil
}

func isRetryableError(err error) bool {
	retryableErrors := []string{
		"temporary-error",
		"timeout",
		"rate limit exceeded",
		"service unavailable",
	}
	
	for _, retryable := range retryableErrors {
		if err.Error() == retryable {
			return true
		}
	}
	return false
}

func simulatePluginOperation(pluginName string) (interface{}, error) {
	// Simulate different failure rates for different plugins
	failureRates := map[string]float32{
		"ai-service":   0.1,
		"database":     0.05,
		"cache":        0.02,
		"external-api": 0.15,
	}
	
	rate := failureRates[pluginName]
	if rand.Float32() < rate {
		return nil, fmt.Errorf("%s operation failed", pluginName)
	}
	
	return fmt.Sprintf("%s result", pluginName), nil
}

func collectResilienceMetrics(wrappers map[string]*ResilientWrapper) {
	for name, wrapper := range wrappers {
		stats := wrapper.GetCircuitBreakerStats()
		
		log.Printf("METRIC: circuit_breaker_state{plugin=%s} %s", name, stats.State)
		log.Printf("METRIC: circuit_breaker_failures{plugin=%s} %d", name, stats.Failures)
		log.Printf("METRIC: circuit_breaker_requests{plugin=%s} %d", name, stats.Requests)
		
		if wrapper.IsCircuitOpen() {
			log.Printf("ALERT: Circuit breaker for %s is OPEN", name)
		}
	}
}

// Example plugin implementations

type AITextPlugin struct{}

func (p *AITextPlugin) GenerateText(prompt string) (string, error) {
	// Simulate AI processing time and occasional failures
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
	
	if rand.Float32() < 0.2 {
		return "", errors.New("AI service timeout")
	}
	
	return fmt.Sprintf("Generated text for: %s", prompt), nil
}

type DatabasePlugin struct{}

func (p *DatabasePlugin) Query(sql string) (interface{}, error) {
	// Simulate database query
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	
	if rand.Float32() < 0.1 {
		return nil, errors.New("database connection failed")
	}
	
	return []map[string]interface{}{
		{"id": 1, "name": "John"},
		{"id": 2, "name": "Jane"},
	}, nil
}

// Example fallback handlers

type CacheFallbackHandler struct {
	cache map[string]interface{}
	ttl   time.Duration
}

func (c *CacheFallbackHandler) CanHandle(operation string, err error) bool {
	return operation == "database.read"
}

func (c *CacheFallbackHandler) Handle(ctx context.Context, operation string, originalErr error) (interface{}, error) {
	// Try to serve from cache
	key := "cached_data"
	if data, exists := c.cache[key]; exists {
		return data, nil
	}
	
	// Cache miss
	return nil, errors.New("no cached data available")
}

func (c *CacheFallbackHandler) GetQuality() float64 {
	return 0.8 // High quality fallback
}

type HealthAwareFallbackHandler struct {
	health       *HealthMonitor
	pluginName   string
	fallbackData interface{}
}

func (h *HealthAwareFallbackHandler) CanHandle(operation string, err error) bool {
	return true
}

func (h *HealthAwareFallbackHandler) Handle(ctx context.Context, operation string, originalErr error) (interface{}, error) {
	// Check if plugin is healthy before providing fallback
	if h.health.IsHealthy(h.pluginName) {
		// Plugin is healthy, this might be a temporary issue
		return nil, fmt.Errorf("temporary failure, plugin is healthy: %w", originalErr)
	}
	
	// Plugin is unhealthy, provide fallback
	return h.fallbackData, nil
}

func (h *HealthAwareFallbackHandler) GetQuality() float64 {
	if h.health.IsHealthy(h.pluginName) {
		return 0.2 // Low quality if plugin should be working
	}
	return 0.7 // Higher quality if plugin is known to be unhealthy
}

type SmartFallbackHandler struct {
	primaryCache   map[string]interface{}
	secondaryCache map[string]interface{}
	lastUpdated    time.Time
}

func (s *SmartFallbackHandler) CanHandle(operation string, err error) bool {
	return operation == "data.fetch"
}

func (s *SmartFallbackHandler) Handle(ctx context.Context, operation string, originalErr error) (interface{}, error) {
	// Choose fallback strategy based on error type
	switch originalErr.Error() {
	case "network timeout":
		// Use primary cache for network issues
		if data, exists := s.primaryCache["primary"]; exists {
			return data, nil
		}
		
	case "rate limit exceeded":
		// Use secondary cache for rate limiting
		if data, exists := s.secondaryCache["secondary"]; exists {
			return data, nil
		}
		
	case "not found":
		// Return empty result for not found
		return map[string]interface{}{}, nil
		
	default:
		// Generic fallback
		return "fallback-data", nil
	}
	
	return nil, errors.New("no suitable fallback available")
}

func (s *SmartFallbackHandler) GetQuality() float64 {
	// Quality degrades over time since last update
	age := time.Since(s.lastUpdated)
	if age < 5*time.Minute {
		return 0.9
	} else if age < 30*time.Minute {
		return 0.7
	} else if age < 2*time.Hour {
		return 0.5
	}
	return 0.2
}