package core

import (
	"context"
	"time"
)

// ErrorFactory provides methods for creating domain errors
type ErrorFactory interface {
	// NewValidationError creates a validation error
	NewValidationError(phase, context, field, message string, value interface{}) error
	
	// NewRetryableError creates a retryable error
	NewRetryableError(phase, operation, message string, cause error) error
	
	// NewPhaseError creates a phase error
	NewPhaseError(phase string, attempt int, cause error, partial interface{}) error
}

// DefaultErrorFactory is the default implementation of ErrorFactory
type DefaultErrorFactory struct{}

// NewDefaultErrorFactory creates a new default error factory
func NewDefaultErrorFactory() *DefaultErrorFactory {
	return &DefaultErrorFactory{}
}

// NewValidationError creates a validation error
func (f *DefaultErrorFactory) NewValidationError(phase, context, field, message string, value interface{}) error {
	return &ValidationError{
		Phase:      phase,
		Type:       context,
		Field:      field,
		Message:    message,
		Value:      value,
		Timestamp:  time.Now(),
		Suggestion: "", // Can be enhanced later
	}
}

// NewRetryableError creates a retryable error
func (f *DefaultErrorFactory) NewRetryableError(phase, operation, message string, cause error) error {
	return &RetryableError{
		Err:        cause,
		RetryAfter: 2 * time.Second,
		MaxRetries: 3,
		Attempts:   0,
	}
}

// NewPhaseError creates a phase error
func (f *DefaultErrorFactory) NewPhaseError(phase string, attempt int, cause error, partial interface{}) error {
	return &PhaseError{
		Phase:     phase,
		Attempt:   attempt,
		Cause:     cause,
		Partial:   partial,
		Timestamp: time.Now(),
	}
}

// ValidatorFactory provides methods for creating validators
type ValidatorFactory interface {
	// NewBaseValidator creates a base validator
	NewBaseValidator(phaseName string) Validator
	
	// NewStandardPhaseValidator creates a standard phase validator
	NewStandardPhaseValidator(phaseName string, rules ValidationRules) PhaseValidator
}

// DefaultValidatorFactory is the default implementation of ValidatorFactory
type DefaultValidatorFactory struct{}

// NewDefaultValidatorFactory creates a new default validator factory
func NewDefaultValidatorFactory() *DefaultValidatorFactory {
	return &DefaultValidatorFactory{}
}

// NewBaseValidator creates a base validator
func (f *DefaultValidatorFactory) NewBaseValidator(phaseName string) Validator {
	return NewBaseValidator(phaseName)
}

// NewStandardPhaseValidator creates a standard phase validator
func (f *DefaultValidatorFactory) NewStandardPhaseValidator(phaseName string, rules ValidationRules) PhaseValidator {
	return NewStandardPhaseValidator(phaseName, rules)
}

// Validator provides basic validation functionality
type Validator interface {
	// ValidateRequired validates that a field is not empty
	ValidateRequired(field string, value string, context string) error
	
	// ValidateJSON validates JSON structure
	ValidateJSON(field string, data interface{}, context string) error
	
	// ValidateLanguage validates programming language
	ValidateLanguage(language string, context string) error
	
	// ValidateFileExtension validates file extension matches language
	ValidateFileExtension(filename string, expectedLanguage string, context string) error
}

// PhaseValidator provides phase-specific validation
type PhaseValidator interface {
	Validator
	
	// ValidateInput validates phase input
	ValidateInput(ctx context.Context, input PhaseInput) error
	
	// ValidateOutput validates phase output
	ValidateOutput(ctx context.Context, output PhaseOutput) error
}

// ResilienceManager provides resilience capabilities for phases
type ResilienceManager interface {
	// WithRetry wraps a function with retry logic
	WithRetry(operation string, fn func() error) error
	
	// WithCircuitBreaker wraps a function with circuit breaker
	WithCircuitBreaker(operation string, fn func() error) error
	
	// WithTimeout wraps a function with timeout
	WithTimeout(operation string, timeout time.Duration, fn func() error) error
	
	// IsRetryable determines if an error should be retried
	IsRetryable(err error) bool
}

// DefaultResilienceManager is the default implementation of ResilienceManager
type DefaultResilienceManager struct {
	maxRetries int
	baseDelay  time.Duration
}

// NewDefaultResilienceManager creates a new default resilience manager
func NewDefaultResilienceManager(maxRetries int, baseDelay time.Duration) *DefaultResilienceManager {
	return &DefaultResilienceManager{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// WithRetry wraps a function with retry logic
func (r *DefaultResilienceManager) WithRetry(operation string, fn func() error) error {
	// Implementation would go here - simplified for now
	return fn()
}

// WithCircuitBreaker wraps a function with circuit breaker
func (r *DefaultResilienceManager) WithCircuitBreaker(operation string, fn func() error) error {
	// Implementation would go here - simplified for now
	return fn()
}

// WithTimeout wraps a function with timeout
func (r *DefaultResilienceManager) WithTimeout(operation string, timeout time.Duration, fn func() error) error {
	// Implementation would go here - simplified for now
	return fn()
}

// IsRetryable determines if an error should be retried
func (r *DefaultResilienceManager) IsRetryable(err error) bool {
	return IsRetryable(err)
}