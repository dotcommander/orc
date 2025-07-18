package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/domain"
)

// CircuitBreakerState represents the current state of a circuit breaker
type CircuitBreakerState int32

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig defines configuration for a circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures to trigger opening
	FailureThreshold int
	
	// SuccessThreshold is the number of successes needed to close from half-open
	SuccessThreshold int
	
	// Timeout is how long to wait before trying half-open
	Timeout time.Duration
	
	// MaxRequests is the maximum number of requests allowed in half-open
	MaxRequests int
	
	// OnStateChange callback when circuit breaker state changes
	OnStateChange func(name string, from, to CircuitBreakerState)
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          60 * time.Second,
		MaxRequests:      3,
		OnStateChange:    func(string, CircuitBreakerState, CircuitBreakerState) {},
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig
	logger *slog.Logger

	mu              sync.RWMutex
	state           CircuitBreakerState
	generation      uint64
	failures        int64
	successes       int64
	requests        int64
	expiry          time.Time
	lastFailureTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &CircuitBreaker{
		name:   name,
		config: config,
		logger: logger,
		state:  CircuitBreakerClosed,
	}
}

// Execute runs the function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.onFailure(generation)
			panic(r)
		}
	}()

	err = fn()
	if err != nil {
		cb.onFailure(generation)
		return err
	}

	cb.onSuccess(generation)
	return nil
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Metrics returns current circuit breaker metrics
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return CircuitBreakerMetrics{
		Name:            cb.name,
		State:           cb.state,
		Failures:        atomic.LoadInt64(&cb.failures),
		Successes:       atomic.LoadInt64(&cb.successes),
		Requests:        atomic.LoadInt64(&cb.requests),
		LastFailureTime: cb.lastFailureTime,
	}
}

type CircuitBreakerMetrics struct {
	Name            string
	State           CircuitBreakerState
	Failures        int64
	Successes       int64
	Requests        int64
	LastFailureTime time.Time
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)

	switch cb.state {
	case CircuitBreakerClosed:
		return cb.generation, nil
	case CircuitBreakerOpen:
		if time.Now().After(cb.expiry) {
			cb.toHalfOpen()
			return cb.generation, nil
		}
		return 0, fmt.Errorf("circuit breaker %s is open", cb.name)
	case CircuitBreakerHalfOpen:
		if cb.requests <= int64(cb.config.MaxRequests) {
			return cb.generation, nil
		}
		return 0, fmt.Errorf("circuit breaker %s half-open max requests exceeded", cb.name)
	default:
		return 0, fmt.Errorf("circuit breaker %s unknown state: %v", cb.name, cb.state)
	}
}

func (cb *CircuitBreaker) onSuccess(generation uint64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if generation != cb.generation {
		return
	}

	atomic.AddInt64(&cb.successes, 1)

	switch cb.state {
	case CircuitBreakerHalfOpen:
		if atomic.LoadInt64(&cb.successes) >= int64(cb.config.SuccessThreshold) {
			cb.toClosed()
		}
	}
}

func (cb *CircuitBreaker) onFailure(generation uint64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if generation != cb.generation {
		return
	}

	atomic.AddInt64(&cb.failures, 1)
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if atomic.LoadInt64(&cb.failures) >= int64(cb.config.FailureThreshold) {
			cb.toOpen()
		}
	case CircuitBreakerHalfOpen:
		cb.toOpen()
	}
}

func (cb *CircuitBreaker) toOpen() {
	cb.setState(CircuitBreakerOpen)
	cb.expiry = time.Now().Add(cb.config.Timeout)
	cb.generation++
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
}

func (cb *CircuitBreaker) toHalfOpen() {
	cb.setState(CircuitBreakerHalfOpen)
	cb.generation++
	atomic.StoreInt64(&cb.requests, 0)
	atomic.StoreInt64(&cb.successes, 0)
}

func (cb *CircuitBreaker) toClosed() {
	cb.setState(CircuitBreakerClosed)
	cb.generation++
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.logger.Info("circuit breaker state change",
		"name", cb.name,
		"from", prev.String(),
		"to", state.String())

	cb.config.OnStateChange(cb.name, prev, state)
}

// RetryPolicy defines how retries should be handled
type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
	
	// ShouldRetry determines if an error is retryable
	ShouldRetry func(error) bool
}

// DefaultRetryPolicy returns sensible defaults
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		ShouldRetry: func(err error) bool {
			// Retry on temporary errors, not on validation errors
			return !errors.Is(err, domain.ErrInvalidInput)
		},
	}
}

// FallbackHandler provides fallback behavior when all retries fail
type FallbackHandler func(ctx context.Context, err error) error

// ResilientPlugin wraps a plugin with circuit breaker and retry logic
type ResilientPlugin struct {
	plugin          domain.Plugin
	circuitBreaker  *CircuitBreaker
	retryPolicy     RetryPolicy
	fallbackHandler FallbackHandler
	logger          *slog.Logger
}

// NewResilientPlugin creates a plugin with resilience patterns
func NewResilientPlugin(
	plugin domain.Plugin,
	circuitBreaker *CircuitBreaker,
	retryPolicy RetryPolicy,
	fallbackHandler FallbackHandler,
	logger *slog.Logger,
) *ResilientPlugin {
	if logger == nil {
		logger = slog.Default()
	}

	return &ResilientPlugin{
		plugin:          plugin,
		circuitBreaker:  circuitBreaker,
		retryPolicy:     retryPolicy,
		fallbackHandler: fallbackHandler,
		logger:          logger,
	}
}

// Name implements domain.Plugin
func (rp *ResilientPlugin) Name() string {
	return rp.plugin.Name()
}

// Domain implements domain.Plugin
func (rp *ResilientPlugin) Domain() string {
	return rp.plugin.Domain()
}

// GetPhases implements domain.Plugin
func (rp *ResilientPlugin) GetPhases() []domain.Phase {
	originalPhases := rp.plugin.GetPhases()
	resilientPhases := make([]domain.Phase, len(originalPhases))
	
	for i, phase := range originalPhases {
		resilientPhases[i] = &ResilientPhase{
			phase:           phase,
			circuitBreaker:  rp.circuitBreaker,
			retryPolicy:     rp.retryPolicy,
			fallbackHandler: rp.fallbackHandler,
			logger:          rp.logger,
		}
	}
	
	return resilientPhases
}

// ResilientPhase wraps a phase with resilience patterns
type ResilientPhase struct {
	phase           domain.Phase
	circuitBreaker  *CircuitBreaker
	retryPolicy     RetryPolicy
	fallbackHandler FallbackHandler
	logger          *slog.Logger
}

// Name implements domain.Phase
func (rp *ResilientPhase) Name() string {
	return rp.phase.Name()
}

// Execute implements domain.Phase with resilience
func (rp *ResilientPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	var lastErr error
	
	for attempt := 1; attempt <= rp.retryPolicy.MaxAttempts; attempt++ {
		err := rp.circuitBreaker.Execute(ctx, func() error {
			output, err := rp.phase.Execute(ctx, input)
			if err != nil {
				return err
			}
			// Store output for successful execution
			input.SetPreviousOutput(output)
			return nil
		})
		
		if err == nil {
			rp.logger.Debug("phase execution succeeded",
				"phase", rp.phase.Name(),
				"attempt", attempt)
			return input.GetPreviousOutput().(domain.PhaseOutput), nil
		}
		
		lastErr = err
		
		// Check if we should retry
		if attempt >= rp.retryPolicy.MaxAttempts || !rp.retryPolicy.ShouldRetry(err) {
			break
		}
		
		// Calculate delay with exponential backoff
		delay := rp.calculateDelay(attempt)
		
		rp.logger.Warn("phase execution failed, retrying",
			"phase", rp.phase.Name(),
			"attempt", attempt,
			"delay", delay,
			"error", err)
		
		// Wait before retry
		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return domain.PhaseOutput{}, ctx.Err()
		}
	}
	
	// All retries failed, try fallback if available
	if rp.fallbackHandler != nil {
		rp.logger.Info("executing fallback handler",
			"phase", rp.phase.Name(),
			"error", lastErr)
		
		if fallbackErr := rp.fallbackHandler(ctx, lastErr); fallbackErr != nil {
			return domain.PhaseOutput{}, fmt.Errorf("fallback failed: %w", fallbackErr)
		}
		
		// Return empty output to indicate fallback was used
		return domain.PhaseOutput{Success: false, Message: "fallback executed"}, nil
	}
	
	return domain.PhaseOutput{}, fmt.Errorf("phase %s failed after %d attempts: %w", 
		rp.phase.Name(), rp.retryPolicy.MaxAttempts, lastErr)
}

// ValidateInput implements domain.Phase
func (rp *ResilientPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	return rp.phase.ValidateInput(ctx, input)
}

// ValidateOutput implements domain.Phase
func (rp *ResilientPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return rp.phase.ValidateOutput(ctx, output)
}

// EstimatedDuration implements domain.Phase
func (rp *ResilientPhase) EstimatedDuration() time.Duration {
	// Factor in potential retries
	baseDuration := rp.phase.EstimatedDuration()
	maxRetryDelay := rp.calculateDelay(rp.retryPolicy.MaxAttempts)
	return baseDuration + maxRetryDelay
}

// CanRetry implements domain.Phase
func (rp *ResilientPhase) CanRetry(err error) bool {
	return rp.retryPolicy.ShouldRetry(err)
}

func (rp *ResilientPhase) calculateDelay(attempt int) time.Duration {
	delay := float64(rp.retryPolicy.InitialDelay)
	
	// Apply exponential backoff
	for i := 1; i < attempt; i++ {
		delay *= rp.retryPolicy.Multiplier
	}
	
	// Apply jitter if enabled
	if rp.retryPolicy.Jitter {
		jitter := 0.1 * delay * (2*rand.Float64() - 1) // ±10% jitter
		delay += jitter
	}
	
	// Ensure we don't exceed max delay
	if delay > float64(rp.retryPolicy.MaxDelay) {
		delay = float64(rp.retryPolicy.MaxDelay)
	}
	
	return time.Duration(delay)
}

// ResilienceManager manages circuit breakers for multiple plugins
type ResilienceManager struct {
	circuitBreakers map[string]*CircuitBreaker
	mu              sync.RWMutex
	logger          *slog.Logger
}

// NewResilienceManager creates a new resilience manager
func NewResilienceManager(logger *slog.Logger) *ResilienceManager {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &ResilienceManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		logger:          logger,
	}
}

// GetCircuitBreaker gets or creates a circuit breaker for a plugin
func (rm *ResilienceManager) GetCircuitBreaker(pluginName string, config CircuitBreakerConfig) *CircuitBreaker {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if cb, exists := rm.circuitBreakers[pluginName]; exists {
		return cb
	}
	
	cb := NewCircuitBreaker(pluginName, config, rm.logger)
	rm.circuitBreakers[pluginName] = cb
	return cb
}

// GetMetrics returns metrics for all circuit breakers
func (rm *ResilienceManager) GetMetrics() map[string]CircuitBreakerMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	metrics := make(map[string]CircuitBreakerMetrics)
	for name, cb := range rm.circuitBreakers {
		metrics[name] = cb.Metrics()
	}
	
	return metrics
}

// HealthStatus returns health status based on circuit breaker states
func (rm *ResilienceManager) HealthStatus() map[string]string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	status := make(map[string]string)
	for name, cb := range rm.circuitBreakers {
		switch cb.State() {
		case CircuitBreakerClosed:
			status[name] = "healthy"
		case CircuitBreakerHalfOpen:
			status[name] = "degraded"
		case CircuitBreakerOpen:
			status[name] = "unhealthy"
		}
	}
	
	return status
}

// WrapPlugin wraps a plugin with standard resilience patterns
func (rm *ResilienceManager) WrapPlugin(
	plugin domain.Plugin,
	circuitBreakerConfig CircuitBreakerConfig,
	retryPolicy RetryPolicy,
	fallbackHandler FallbackHandler,
) *ResilientPlugin {
	cb := rm.GetCircuitBreaker(plugin.Name(), circuitBreakerConfig)
	
	return NewResilientPlugin(
		plugin,
		cb,
		retryPolicy,
		fallbackHandler,
		rm.logger,
	)
}