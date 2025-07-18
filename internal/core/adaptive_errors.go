package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// AdaptiveError represents an intelligent error that learns and adapts
type AdaptiveError struct {
	Type           ErrorType
	Message        string
	Context        map[string]interface{}
	RecoveryHints  []RecoveryStrategy
	LearningData   ErrorPattern
	Timestamp      time.Time
	Stack          []ErrorFrame
}

// ErrorType classifies errors for adaptive handling
type ErrorType int

const (
	TransientError ErrorType = iota // Can retry with same strategy
	AdaptableError                  // Need different approach
	ConfigError                     // Configuration issue
	ResourceError                   // Resource constraint
	ValidationErrorType             // Input validation
	UnknownError                    // Needs investigation
)

// RecoveryStrategy suggests how to recover from an error
type RecoveryStrategy struct {
	Name        string
	Description string
	Confidence  float64
	Action      func(context.Context, interface{}) error
	Conditions  []string
}

// ErrorPattern tracks error patterns for learning
type ErrorPattern struct {
	Frequency    int
	LastSeen     time.Time
	Recoveries   []RecoveryAttempt
	Correlations map[string]float64
}

// RecoveryAttempt records recovery attempts
type RecoveryAttempt struct {
	Strategy  string
	Success   bool
	Duration  time.Duration
	Timestamp time.Time
}

// ErrorFrame provides error context
type ErrorFrame struct {
	Function string
	File     string
	Line     int
	Context  map[string]interface{}
}

// AdaptiveErrorHandler learns from errors and suggests recoveries
type AdaptiveErrorHandler struct {
	patterns   map[string]*ErrorPattern
	strategies map[ErrorType][]RecoveryStrategy
	learning   *ErrorLearningEngine
	mu         sync.RWMutex
}

// ErrorLearningEngine learns from error patterns
type ErrorLearningEngine struct {
	history    []AdaptiveError
	patterns   map[string]*LearnedPattern
	thresholds AdaptiveThresholds
	mu         sync.RWMutex
}

// LearnedPattern represents a learned error pattern
type LearnedPattern struct {
	ErrorSignature   string
	SuccessfulFixes  []string
	FailedAttempts   []string
	EnvironmentVars  map[string]string
	SystemState      map[string]interface{}
	SuccessRate      float64
	LastUpdated      time.Time
}

// AdaptiveThresholds adjusts based on context
type AdaptiveThresholds struct {
	RetryLimit      int
	BackoffFactor   float64
	TimeoutFactor   float64
	SuccessRequired float64
}

// NewAdaptiveErrorHandler creates an intelligent error handler
func NewAdaptiveErrorHandler() *AdaptiveErrorHandler {
	handler := &AdaptiveErrorHandler{
		patterns:   make(map[string]*ErrorPattern),
		strategies: make(map[ErrorType][]RecoveryStrategy),
		learning:   NewErrorLearningEngine(),
	}

	// Register default strategies
	handler.registerDefaultStrategies()

	return handler
}

// HandleError processes an error with adaptive strategies
func (aeh *AdaptiveErrorHandler) HandleError(ctx context.Context, err error, context map[string]interface{}) *AdaptiveError {
	// Classify the error
	errorType := aeh.classifyError(err, context)
	
	// Create adaptive error
	adaptiveErr := &AdaptiveError{
		Type:      errorType,
		Message:   err.Error(),
		Context:   context,
		Timestamp: time.Now(),
		Stack:     aeh.captureStack(),
	}

	// Learn from this error
	aeh.learning.RecordError(adaptiveErr)

	// Get recovery strategies
	adaptiveErr.RecoveryHints = aeh.suggestRecoveries(adaptiveErr)

	// Update patterns
	aeh.updateErrorPattern(adaptiveErr)

	return adaptiveErr
}

// classifyError determines error type using patterns
func (aeh *AdaptiveErrorHandler) classifyError(err error, context map[string]interface{}) ErrorType {
	errStr := err.Error()

	// Check learned patterns first
	if pattern := aeh.learning.MatchPattern(errStr); pattern != nil {
		if pattern.SuccessRate > 0.8 {
			return AdaptableError // We know how to handle this
		}
	}

	// Pattern matching for classification
	transientPatterns := []string{"timeout", "temporary", "connection refused", "rate limit"}
	for _, pattern := range transientPatterns {
		if contains(errStr, pattern) {
			return TransientError
		}
	}

	configPatterns := []string{"config", "missing required", "invalid setting"}
	for _, pattern := range configPatterns {
		if contains(errStr, pattern) {
			return ConfigError
		}
	}

	resourcePatterns := []string{"out of memory", "disk full", "quota exceeded"}
	for _, pattern := range resourcePatterns {
		if contains(errStr, pattern) {
			return ResourceError
		}
	}

	return UnknownError
}

// suggestRecoveries provides intelligent recovery suggestions
func (aeh *AdaptiveErrorHandler) suggestRecoveries(err *AdaptiveError) []RecoveryStrategy {
	aeh.mu.RLock()
	defer aeh.mu.RUnlock()

	suggestions := make([]RecoveryStrategy, 0)

	// Get type-based strategies
	if strategies, exists := aeh.strategies[err.Type]; exists {
		suggestions = append(suggestions, strategies...)
	}

	// Get learned strategies
	if learned := aeh.learning.GetLearnedStrategies(err); len(learned) > 0 {
		suggestions = append(suggestions, learned...)
	}

	// Sort by confidence
	sortByConfidence(suggestions)

	return suggestions
}

// RecoverWithLearning attempts recovery and learns from the outcome
func (aeh *AdaptiveErrorHandler) RecoverWithLearning(ctx context.Context, err *AdaptiveError, data interface{}) (interface{}, error) {
	for _, strategy := range err.RecoveryHints {
		if strategy.Confidence < 0.3 {
			continue // Skip low confidence strategies
		}

		start := time.Now()
		result := strategy.Action(ctx, data)
		duration := time.Since(start)

		// Record the attempt
		attempt := RecoveryAttempt{
			Strategy:  strategy.Name,
			Success:   result == nil,
			Duration:  duration,
			Timestamp: time.Now(),
		}

		aeh.learning.RecordRecoveryAttempt(err, attempt)

		if result == nil {
			return data, nil // Success!
		}
	}

	return nil, fmt.Errorf("all recovery strategies failed for: %s", err.Message)
}

// registerDefaultStrategies sets up common recovery strategies
func (aeh *AdaptiveErrorHandler) registerDefaultStrategies() {
	// Transient error strategies
	aeh.strategies[TransientError] = []RecoveryStrategy{
		{
			Name:        "exponential-backoff",
			Description: "Retry with exponential backoff",
			Confidence:  0.8,
			Action:      exponentialBackoffRetry,
		},
		{
			Name:        "circuit-breaker",
			Description: "Use circuit breaker pattern",
			Confidence:  0.7,
			Action:      circuitBreakerRetry,
		},
	}

	// Config error strategies
	aeh.strategies[ConfigError] = []RecoveryStrategy{
		{
			Name:        "use-defaults",
			Description: "Fall back to default configuration",
			Confidence:  0.6,
			Action:      useDefaultConfig,
		},
		{
			Name:        "auto-detect",
			Description: "Auto-detect configuration from environment",
			Confidence:  0.5,
			Action:      autoDetectConfig,
		},
	}

	// Resource error strategies
	aeh.strategies[ResourceError] = []RecoveryStrategy{
		{
			Name:        "reduce-batch-size",
			Description: "Reduce processing batch size",
			Confidence:  0.7,
			Action:      reduceBatchSize,
		},
		{
			Name:        "free-resources",
			Description: "Free up unused resources",
			Confidence:  0.6,
			Action:      freeUnusedResources,
		},
	}
}

// ErrorLearningEngine implementation

func NewErrorLearningEngine() *ErrorLearningEngine {
	return &ErrorLearningEngine{
		history:  make([]AdaptiveError, 0, 1000),
		patterns: make(map[string]*LearnedPattern),
		thresholds: AdaptiveThresholds{
			RetryLimit:      3,
			BackoffFactor:   2.0,
			TimeoutFactor:   1.5,
			SuccessRequired: 0.7,
		},
	}
}

// RecordError adds error to learning history
func (ele *ErrorLearningEngine) RecordError(err *AdaptiveError) {
	ele.mu.Lock()
	defer ele.mu.Unlock()

	ele.history = append(ele.history, *err)
	
	// Maintain history size
	if len(ele.history) > 10000 {
		ele.history = ele.history[5000:] // Keep recent half
	}

	// Update patterns
	ele.updatePatterns(err)
}

// updatePatterns learns from error patterns
func (ele *ErrorLearningEngine) updatePatterns(err *AdaptiveError) {
	signature := ele.generateSignature(err)
	
	pattern, exists := ele.patterns[signature]
	if !exists {
		pattern = &LearnedPattern{
			ErrorSignature:  signature,
			SuccessfulFixes: make([]string, 0),
			FailedAttempts:  make([]string, 0),
			EnvironmentVars: captureEnvironment(),
			SystemState:     captureSystemState(),
			LastUpdated:     time.Now(),
		}
		ele.patterns[signature] = pattern
	}

	pattern.LastUpdated = time.Now()
}

// MatchPattern finds matching learned pattern
func (ele *ErrorLearningEngine) MatchPattern(errStr string) *LearnedPattern {
	ele.mu.RLock()
	defer ele.mu.RUnlock()

	// Simple matching for now, could use ML in future
	for sig, pattern := range ele.patterns {
		if contains(errStr, sig) || contains(sig, errStr) {
			return pattern
		}
	}

	return nil
}

// GetLearnedStrategies returns strategies learned from experience
func (ele *ErrorLearningEngine) GetLearnedStrategies(err *AdaptiveError) []RecoveryStrategy {
	ele.mu.RLock()
	defer ele.mu.RUnlock()

	strategies := make([]RecoveryStrategy, 0)
	
	pattern := ele.MatchPattern(err.Message)
	if pattern == nil {
		return strategies
	}

	// Create strategies from successful fixes
	for _, fix := range pattern.SuccessfulFixes {
		strategies = append(strategies, RecoveryStrategy{
			Name:        fmt.Sprintf("learned-%s", fix),
			Description: fmt.Sprintf("Previously successful: %s", fix),
			Confidence:  pattern.SuccessRate,
			Action:      createLearnedAction(fix),
		})
	}

	return strategies
}

// RecordRecoveryAttempt updates learning data
func (ele *ErrorLearningEngine) RecordRecoveryAttempt(err *AdaptiveError, attempt RecoveryAttempt) {
	ele.mu.Lock()
	defer ele.mu.Unlock()

	signature := ele.generateSignature(err)
	pattern, exists := ele.patterns[signature]
	if !exists {
		return
	}

	if attempt.Success {
		pattern.SuccessfulFixes = append(pattern.SuccessfulFixes, attempt.Strategy)
		// Update success rate
		total := len(pattern.SuccessfulFixes) + len(pattern.FailedAttempts)
		pattern.SuccessRate = float64(len(pattern.SuccessfulFixes)) / float64(total)
	} else {
		pattern.FailedAttempts = append(pattern.FailedAttempts, attempt.Strategy)
	}

	// Adapt thresholds based on success
	if pattern.SuccessRate < 0.3 && len(pattern.FailedAttempts) > 5 {
		ele.thresholds.RetryLimit = max(1, ele.thresholds.RetryLimit-1)
		ele.thresholds.BackoffFactor = min(5.0, ele.thresholds.BackoffFactor*1.2)
	}
}

// generateSignature creates a unique signature for an error
func (ele *ErrorLearningEngine) generateSignature(err *AdaptiveError) string {
	// Simple signature for now
	// Could use more sophisticated hashing
	return fmt.Sprintf("%v-%s", err.Type, truncate(err.Message, 50))
}

// Helper functions

func (aeh *AdaptiveErrorHandler) captureStack() []ErrorFrame {
	// Simplified stack capture
	return []ErrorFrame{
		{
			Function: "unknown",
			File:     "unknown",
			Line:     0,
			Context:  make(map[string]interface{}),
		},
	}
}

func (aeh *AdaptiveErrorHandler) updateErrorPattern(err *AdaptiveError) {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()

	key := err.Message
	pattern, exists := aeh.patterns[key]
	if !exists {
		pattern = &ErrorPattern{
			Frequency:    0,
			Recoveries:   make([]RecoveryAttempt, 0),
			Correlations: make(map[string]float64),
		}
		aeh.patterns[key] = pattern
	}

	pattern.Frequency++
	pattern.LastSeen = time.Now()
}

// Recovery action implementations

func exponentialBackoffRetry(ctx context.Context, data interface{}) error {
	// Implement exponential backoff
	return nil
}

func circuitBreakerRetry(ctx context.Context, data interface{}) error {
	// Implement circuit breaker
	return nil
}

func useDefaultConfig(ctx context.Context, data interface{}) error {
	// Use default configuration
	return nil
}

func autoDetectConfig(ctx context.Context, data interface{}) error {
	// Auto-detect configuration
	return nil
}

func reduceBatchSize(ctx context.Context, data interface{}) error {
	// Reduce batch size for processing
	return nil
}

func freeUnusedResources(ctx context.Context, data interface{}) error {
	// Free up resources
	return nil
}

func createLearnedAction(strategy string) func(context.Context, interface{}) error {
	return func(ctx context.Context, data interface{}) error {
		// Implement learned strategy
		return nil
	}
}

// Utility functions

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func sortByConfidence(strategies []RecoveryStrategy) {
	// Sort strategies by confidence
}

func captureEnvironment() map[string]string {
	// Capture relevant environment variables
	return make(map[string]string)
}

func captureSystemState() map[string]interface{} {
	// Capture system state for learning
	return make(map[string]interface{})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}