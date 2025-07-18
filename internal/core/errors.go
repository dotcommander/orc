package core

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// Core Error Types
// =============================================================================

// PhaseError represents an error during phase execution with comprehensive recovery information
type PhaseError struct {
	Phase        string
	Attempt      int
	Cause        error
	Partial      any    // Partial results from failed execution
	Retryable    bool   // Whether this error can be retried
	RecoveryHint string // Hint for recovery actions
	Timestamp    time.Time
}

func (e *PhaseError) Error() string {
	return fmt.Sprintf("phase %s failed (attempt %d): %v", e.Phase, e.Attempt, e.Cause)
}

func (e *PhaseError) Unwrap() error {
	return e.Cause
}

// IsRetryable indicates if the error can be retried
func (e *PhaseError) IsRetryable() bool {
	return e.Retryable
}

// GetRecoveryHint returns recovery suggestions
func (e *PhaseError) GetRecoveryHint() string {
	return e.RecoveryHint
}

// RetryableError wraps errors that can be retried with timing information
type RetryableError struct {
	Err        error
	RetryAfter time.Duration
	MaxRetries int
	Attempts   int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d/%d, retry after %v): %v", 
		e.Attempts, e.MaxRetries, e.RetryAfter, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// CanRetry checks if more retries are allowed
func (e *RetryableError) CanRetry() bool {
	return e.Attempts < e.MaxRetries
}

// ValidationError represents validation failures with comprehensive context
type ValidationError struct {
	Phase      string      // Phase where validation failed
	Type       string      // "input", "output", or "internal"
	Field      string      // Field that failed validation
	Message    string      // Human-readable error message
	Data       interface{} // The data that failed validation
	Value      interface{} // Specific value that failed (for backward compatibility)
	Suggestion string      // Suggested fix (for backward compatibility)
	Timestamp  time.Time   // When the validation failed
}

func (e *ValidationError) Error() string {
	if e.Phase != "" {
		return fmt.Sprintf("validation failed in %s %s.%s: %s", e.Phase, e.Type, e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed for %s: %s (value: %v)", e.Field, e.Message, e.Value)
}

// GenerationError represents code generation failures
type GenerationError struct {
	Step     string      // Generation step that failed
	Details  string      // Detailed error description
	Partial  interface{} // Partial results if any
	Language string      // Programming language context
	Context  string      // Additional context information
}

func (e *GenerationError) Error() string {
	return fmt.Sprintf("generation failed at %s: %s", e.Step, e.Details)
}

// =============================================================================
// Predefined Error Values
// =============================================================================

var (
	ErrRateLimited    = errors.New("rate limited")
	ErrPromptTooLarge = errors.New("prompt exceeds limit")
	ErrTimeout        = errors.New("operation timed out")
	ErrNoAPIKey       = errors.New("API key not configured")
	ErrInvalidInput   = errors.New("invalid input")
	ErrPartialFailure = errors.New("partial failure")
	ErrNetworkError   = errors.New("network error")
	ErrServerError    = errors.New("server error")
	ErrContextCanceled = errors.New("context canceled")
	ErrDeadlineExceeded = errors.New("deadline exceeded")
)

// =============================================================================
// Error Classification Functions
// =============================================================================

// IsRetryable determines if an error can be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for RetryableError
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return retryable.CanRetry()
	}
	
	// Check for PhaseError with retry capability
	var phaseErr *PhaseError
	if errors.As(err, &phaseErr) {
		return phaseErr.IsRetryable()
	}
	
	// Check for known retryable errors
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrServerError) ||
		errors.Is(err, ErrDeadlineExceeded)
}

// IsTerminal determines if an error is terminal (cannot be retried)
func IsTerminal(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for PhaseError with explicit non-retryable flag
	var phaseErr *PhaseError
	if errors.As(err, &phaseErr) {
		return !phaseErr.IsRetryable()
	}
	
	// Check for known terminal errors
	return errors.Is(err, ErrPromptTooLarge) ||
		errors.Is(err, ErrNoAPIKey) ||
		errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrContextCanceled)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsGenerationError checks if an error is a generation error
func IsGenerationError(err error) bool {
	if err == nil {
		return false
	}
	var generationErr *GenerationError
	return errors.As(err, &generationErr)
}

// =============================================================================
// Error Creation Helpers
// =============================================================================

// NewPhaseError creates a new PhaseError with timestamp
func NewPhaseError(phase string, attempt int, cause error, partial any) *PhaseError {
	return &PhaseError{
		Phase:     phase,
		Attempt:   attempt,
		Cause:     cause,
		Partial:   partial,
		Retryable: IsRetryable(cause),
		Timestamp: time.Now(),
	}
}

// NewRetryableError creates a new RetryableError
func NewRetryableError(err error, retryAfter time.Duration, maxRetries, attempts int) *RetryableError {
	return &RetryableError{
		Err:        err,
		RetryAfter: retryAfter,
		MaxRetries: maxRetries,
		Attempts:   attempts,
	}
}

// NewValidationError creates a new ValidationError with timestamp
func NewValidationError(phase, validationType, field, message string, data interface{}) *ValidationError {
	return &ValidationError{
		Phase:     phase,
		Type:      validationType,
		Field:     field,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewGenerationError creates a new GenerationError
func NewGenerationError(step, details string, partial interface{}) *GenerationError {
	return &GenerationError{
		Step:    step,
		Details: details,
		Partial: partial,
	}
}

// =============================================================================
// Error Recovery System
// =============================================================================

// RecoveryManager handles error recovery and rollback
type RecoveryManager struct {
	checkpoints map[string]interface{}
	rollbacks   []func() error
	mu          sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager() *RecoveryManager {
	return &RecoveryManager{
		checkpoints: make(map[string]interface{}),
		rollbacks:   make([]func() error, 0),
	}
}

// SaveCheckpoint saves a recovery point
func (r *RecoveryManager) SaveCheckpoint(name string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkpoints[name] = data
}

// AddRollback adds a rollback function to be called on failure
func (r *RecoveryManager) AddRollback(fn func() error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rollbacks = append(r.rollbacks, fn)
}

// Rollback executes all rollback functions in reverse order
func (r *RecoveryManager) Rollback() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var lastErr error
	// Execute in reverse order (LIFO)
	for i := len(r.rollbacks) - 1; i >= 0; i-- {
		if err := r.rollbacks[i](); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// GetCheckpoint retrieves a saved checkpoint
func (r *RecoveryManager) GetCheckpoint(name string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.checkpoints[name]
	return data, exists
}

// ClearCheckpoints removes all saved checkpoints
func (r *RecoveryManager) ClearCheckpoints() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkpoints = make(map[string]interface{})
}

// ClearRollbacks removes all rollback functions
func (r *RecoveryManager) ClearRollbacks() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rollbacks = make([]func() error, 0)
}