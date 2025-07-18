package errors

import (
	"errors"
	"fmt"
)

// Common error types for plugins
var (
	// ErrPhaseTimeout indicates a phase exceeded its time limit
	ErrPhaseTimeout = errors.New("phase timeout exceeded")
	
	// ErrInvalidInput indicates the input to a phase is invalid
	ErrInvalidInput = errors.New("invalid phase input")
	
	// ErrInvalidOutput indicates the output from a phase is invalid
	ErrInvalidOutput = errors.New("invalid phase output")
	
	// ErrAPILimit indicates an API rate limit was hit
	ErrAPILimit = errors.New("API rate limit exceeded")
	
	// ErrNoRetry indicates this error should not be retried
	ErrNoRetry = errors.New("operation cannot be retried")
)

// PhaseError represents an error that occurred during phase execution
type PhaseError struct {
	Phase   string
	Err     error
	Retry   bool
	Details map[string]interface{}
}

func (e *PhaseError) Error() string {
	return fmt.Sprintf("phase %s failed: %v", e.Phase, e.Err)
}

func (e *PhaseError) Unwrap() error {
	return e.Err
}

func (e *PhaseError) CanRetry() bool {
	return e.Retry
}

// NewPhaseError creates a new phase error
func NewPhaseError(phase string, err error, canRetry bool) *PhaseError {
	return &PhaseError{
		Phase:   phase,
		Err:     err,
		Retry:   canRetry,
		Details: make(map[string]interface{}),
	}
}

// IsRetryable checks if an error can be retried
func IsRetryable(err error) bool {
	if errors.Is(err, ErrNoRetry) {
		return false
	}
	
	var phaseErr *PhaseError
	if errors.As(err, &phaseErr) {
		return phaseErr.CanRetry()
	}
	
	// Default to retryable for unknown errors
	return true
}

// IsTimeout checks if an error is a timeout
func IsTimeout(err error) bool {
	return errors.Is(err, ErrPhaseTimeout)
}

// IsAPILimit checks if an error is due to rate limiting
func IsAPILimit(err error) bool {
	return errors.Is(err, ErrAPILimit)
}