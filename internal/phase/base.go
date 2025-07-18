package phase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// BasePhase provides common functionality for all phases
type BasePhase struct {
	name              string
	estimatedDuration time.Duration
	logger            *slog.Logger
	retryConfig       RetryConfig
}

// RetryConfig defines retry behavior for phases
type RetryConfig struct {
	MaxAttempts    int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableErrors []error
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:   3,
	InitialDelay:  100 * time.Millisecond,
	MaxDelay:      5 * time.Second,
	BackoffFactor: 2.0,
	RetryableErrors: []error{
		core.ErrRateLimited,
		core.ErrTimeout,
	},
}

// BasePhaseOption allows customization of BasePhase
type BasePhaseOption func(*BasePhase)

// WithLogger configures a custom logger
func WithLogger(logger *slog.Logger) BasePhaseOption {
	return func(b *BasePhase) {
		b.logger = logger
	}
}

// WithRetryConfig configures retry behavior
func WithRetryConfig(config RetryConfig) BasePhaseOption {
	return func(b *BasePhase) {
		b.retryConfig = config
	}
}

// NewBasePhase creates a new base phase with optional configuration
func NewBasePhase(name string, duration time.Duration, options ...BasePhaseOption) BasePhase {
	base := BasePhase{
		name:              name,
		estimatedDuration: duration,
		logger:            slog.Default(),
		retryConfig:       DefaultRetryConfig,
	}

	for _, option := range options {
		option(&base)
	}

	return base
}

// Name returns the phase name
func (b BasePhase) Name() string {
	return b.name
}

// EstimatedDuration returns the expected phase duration
func (b BasePhase) EstimatedDuration() time.Duration {
	return b.estimatedDuration
}

// Validate performs comprehensive input validation
func (b BasePhase) Validate(input core.PhaseInput) error {
	// Basic validation from both implementations
	if input.Request == "" && input.Data == nil {
		return fmt.Errorf("phase %s: %w - request is empty and no data provided", b.name, core.ErrInvalidInput)
	}

	// Enhanced validation for prompt structure
	if input.Request != "" && len(input.Request) > 50000 {
		return fmt.Errorf("phase %s: %w - request exceeds maximum length", b.name, core.ErrPromptTooLarge)
	}

	// Validate session ID for resumeable operations
	if input.SessionID == "" {
		b.logger.Warn("No session ID provided for phase", "phase", b.name)
	}

	return nil
}

// CanRetry determines if an error is retryable based on sophisticated logic
func (b BasePhase) CanRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a terminal error first
	if core.IsTerminal(err) {
		return false
	}

	// Use core retryable logic as base
	if core.IsRetryable(err) {
		return true
	}

	// Check against configured retryable errors
	for _, retryableErr := range b.retryConfig.RetryableErrors {
		if err == retryableErr {
			return true
		}
	}

	return false
}

// ExecuteWithRetry provides a retry wrapper for phase execution
func (b BasePhase) ExecuteWithRetry(ctx context.Context, executor func(context.Context) (core.PhaseOutput, error)) (core.PhaseOutput, error) {
	var lastErr error
	delay := b.retryConfig.InitialDelay

	for attempt := 1; attempt <= b.retryConfig.MaxAttempts; attempt++ {
		b.logger.Debug("Executing phase attempt", 
			"phase", b.name, 
			"attempt", attempt,
			"max_attempts", b.retryConfig.MaxAttempts,
		)

		output, err := executor(ctx)
		if err == nil {
			if attempt > 1 {
				b.logger.Info("Phase succeeded after retries", 
					"phase", b.name, 
					"attempt", attempt,
				)
			}
			return output, nil
		}

		lastErr = err

		// Check if we should retry
		if !b.CanRetry(err) {
			b.logger.Error("Phase failed with non-retryable error", 
				"phase", b.name, 
				"attempt", attempt,
				"error", err,
			)
			return core.PhaseOutput{}, &core.PhaseError{
				Phase:   b.name,
				Attempt: attempt,
				Cause:   err,
			}
		}

		// Don't sleep on the last attempt
		if attempt < b.retryConfig.MaxAttempts {
			select {
			case <-ctx.Done():
				return core.PhaseOutput{}, ctx.Err()
			case <-time.After(delay):
				// Exponential backoff with jitter
				delay = time.Duration(float64(delay) * b.retryConfig.BackoffFactor)
				if delay > b.retryConfig.MaxDelay {
					delay = b.retryConfig.MaxDelay
				}
			}
		}

		b.logger.Warn("Phase attempt failed, retrying", 
			"phase", b.name, 
			"attempt", attempt,
			"error", err,
			"next_delay", delay,
		)
	}

	b.logger.Error("Phase failed after all retries", 
		"phase", b.name, 
		"attempts", b.retryConfig.MaxAttempts,
		"final_error", lastErr,
	)

	return core.PhaseOutput{}, &core.PhaseError{
		Phase:   b.name,
		Attempt: b.retryConfig.MaxAttempts,
		Cause:   lastErr,
	}
}

// LogStart logs the start of phase execution
func (b BasePhase) LogStart(ctx context.Context, input core.PhaseInput) {
	b.logger.Info("Starting phase execution", 
		"phase", b.name,
		"estimated_duration", b.estimatedDuration,
		"session_id", input.SessionID,
		"has_data", input.Data != nil,
		"request_length", len(input.Request),
	)
}

// LogComplete logs successful phase completion
func (b BasePhase) LogComplete(ctx context.Context, output core.PhaseOutput, duration time.Duration) {
	b.logger.Info("Phase completed successfully", 
		"phase", b.name,
		"actual_duration", duration,
		"estimated_duration", b.estimatedDuration,
		"has_output", output.Data != nil,
	)
}

// LogError logs phase execution errors
func (b BasePhase) LogError(ctx context.Context, err error, duration time.Duration) {
	b.logger.Error("Phase execution failed", 
		"phase", b.name,
		"error", err,
		"duration", duration,
	)
}

// CreatePhaseError creates a properly formatted phase error
func (b BasePhase) CreatePhaseError(attempt int, cause error, partial interface{}) *core.PhaseError {
	return &core.PhaseError{
		Phase:   b.name,
		Attempt: attempt,
		Cause:   cause,
		Partial: partial,
	}
}

// ValidateContext checks if the context is valid for execution
func (b BasePhase) ValidateContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("phase %s: context cannot be nil", b.name)
	}
	
	select {
	case <-ctx.Done():
		return fmt.Errorf("phase %s: context already cancelled: %w", b.name, ctx.Err())
	default:
		return nil
	}
}

// GetMetrics returns phase execution metrics
func (b BasePhase) GetMetrics() PhaseMetrics {
	return PhaseMetrics{
		Name:              b.name,
		EstimatedDuration: b.estimatedDuration,
		RetryConfig:       b.retryConfig,
	}
}

// PhaseMetrics contains metrics about phase configuration
type PhaseMetrics struct {
	Name              string
	EstimatedDuration time.Duration
	RetryConfig       RetryConfig
}