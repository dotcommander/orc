package core

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// ResilienceConfig configures retry and fallback behavior
type ResilienceConfig struct {
	MaxRetries       int
	BaseDelay        time.Duration
	MaxDelay         time.Duration
	BackoffMultiplier float64
	EnableFallbacks   bool
}

// DefaultResilienceConfig provides sensible defaults
func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		MaxRetries:       3,
		BaseDelay:        1 * time.Second,
		MaxDelay:         30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableFallbacks:   true,
	}
}

// PhaseResilienceManager handles retries and fallbacks for phase execution
type PhaseResilienceManager struct {
	config ResilienceConfig
	logger ValidationLogger
}

func NewPhaseResilienceManager(config ResilienceConfig) *PhaseResilienceManager {
	return &PhaseResilienceManager{
		config: config,
		logger: *NewValidationLogger(),
	}
}

// IsRetryableCustom checks if an error should be retried (custom logic for resilience)
func IsRetryableCustom(err error) bool {
	switch e := err.(type) {
	case *RetryableError:
		return true
	case *ValidationError:
		// Validation errors for empty/missing data are retryable
		// Language detection failures are retryable
		return e.Field == "language" || e.Field == "main_objective"
	default:
		// Network, timeout, and JSON parsing errors are typically retryable
		errStr := err.Error()
		return contains(errStr, "timeout") || 
		       contains(errStr, "connection") || 
		       contains(errStr, "parse") ||
		       contains(errStr, "json")
	}
}

// Removed duplicate contains function - using the one from adaptive_errors.go

// ExecuteWithRetry executes a function with exponential backoff retry
func (rm *PhaseResilienceManager) ExecuteWithRetry(ctx context.Context, operation func() error, operationName string) error {
	var lastErr error
	
	for attempt := 0; attempt <= rm.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := rm.calculateDelay(attempt)
			
			rm.logger.LogValidation(operationName, "retry", false, lastErr, map[string]interface{}{
				"attempt": attempt,
				"delay":   delay.String(),
			})
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}
		
		err := operation()
		if err == nil {
			if attempt > 0 {
				rm.logger.LogValidation(operationName, "retry_success", true, nil, map[string]interface{}{
					"successful_attempt": attempt + 1,
				})
			}
			return nil
		}
		
		lastErr = err
		
		// Check if we should retry this error
		if !IsRetryableCustom(err) {
			rm.logger.LogValidation(operationName, "retry_abort", false, err, map[string]interface{}{
				"reason": "error_not_retryable",
			})
			return err
		}
		
		// Don't retry on the last attempt
		if attempt == rm.config.MaxRetries {
			break
		}
	}
	
	rm.logger.LogValidation(operationName, "retry_exhausted", false, lastErr, map[string]interface{}{
		"max_retries": rm.config.MaxRetries,
	})
	
	return fmt.Errorf("operation failed after %d retries: %w", rm.config.MaxRetries, lastErr)
}

func (rm *PhaseResilienceManager) calculateDelay(attempt int) time.Duration {
	delay := float64(rm.config.BaseDelay) * math.Pow(rm.config.BackoffMultiplier, float64(attempt-1))
	
	if delay > float64(rm.config.MaxDelay) {
		delay = float64(rm.config.MaxDelay)
	}
	
	return time.Duration(delay)
}

// FallbackOption represents a fallback strategy
type FallbackOption struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, input interface{}) (interface{}, error)
}

// FallbackManager handles fallback strategies when primary operations fail
type FallbackManager struct {
	fallbacks map[string][]FallbackOption
}

func NewFallbackManager() *FallbackManager {
	return &FallbackManager{
		fallbacks: make(map[string][]FallbackOption),
	}
}

func (fm *FallbackManager) RegisterFallback(operation string, fallback FallbackOption) {
	fm.fallbacks[operation] = append(fm.fallbacks[operation], fallback)
}

func (fm *FallbackManager) ExecuteWithFallbacks(ctx context.Context, operation string, primaryFunc func() (interface{}, error), input interface{}) (interface{}, error) {
	// Try primary operation first
	result, err := primaryFunc()
	if err == nil {
		return result, nil
	}
	
	// Try fallbacks in order
	fallbacks, exists := fm.fallbacks[operation]
	if !exists {
		return nil, fmt.Errorf("primary operation failed and no fallbacks available: %w", err)
	}
	
	var lastErr error = err
	for i, fallback := range fallbacks {
		result, err := fallback.Execute(ctx, input)
		if err == nil {
			return result, nil
		}
		lastErr = err
		
		// Log fallback attempt
		fmt.Printf("Fallback %d (%s) failed: %v\n", i+1, fallback.Name, err)
	}
	
	return nil, fmt.Errorf("all fallbacks exhausted, last error: %w", lastErr)
}

// PhaseResilience provides phase-specific resilience patterns
type PhaseResilience struct {
	*PhaseResilienceManager
	*FallbackManager
}

func NewPhaseResilience() *PhaseResilience {
	pr := &PhaseResilience{
		PhaseResilienceManager: NewPhaseResilienceManager(DefaultResilienceConfig()),
		FallbackManager:        NewFallbackManager(),
	}
	
	// Register common fallbacks
	pr.registerCommonFallbacks()
	return pr
}

func (pr *PhaseResilience) registerCommonFallbacks() {
	// Analysis fallback: Use simplified analysis if AI fails
	pr.RegisterFallback("analysis", FallbackOption{
		Name:        "simple_language_detection",
		Description: "Extract language from keywords in request",
		Execute: func(ctx context.Context, input interface{}) (interface{}, error) {
			request, ok := input.(string)
			if !ok {
				return nil, fmt.Errorf("expected string input for analysis fallback")
			}
			
			return fallbackAnalysis(request), nil
		},
	})
	
	// Planning fallback: Use template-based plan if AI fails
	pr.RegisterFallback("planning", FallbackOption{
		Name:        "template_planning",
		Description: "Generate basic plan from language template",
		Execute: func(ctx context.Context, input interface{}) (interface{}, error) {
			// This would generate a basic plan based on the detected language
			return fallbackPlanning(input), nil
		},
	})
}

// fallbackAnalysis provides basic language detection as fallback
func fallbackAnalysis(request string) map[string]interface{} {
	request = strings.ToLower(request)
	
	language := "Other"
	if strings.Contains(request, "php") {
		language = "PHP"
	} else if strings.Contains(request, "python") {
		language = "Python"
	} else if strings.Contains(request, "javascript") || strings.Contains(request, "js") {
		language = "JavaScript"
	} else if strings.Contains(request, "go ") || strings.Contains(request, "golang") {
		language = "Go"
	} else if strings.Contains(request, "java") && !strings.Contains(request, "javascript") {
		language = "Java"
	}
	
	return map[string]interface{}{
		"language":         language,
		"complexity":       "Simple",
		"main_objective":   fmt.Sprintf("Generate %s code based on user request", language),
		"requirements":     []string{"Implement requested functionality"},
		"constraints":      []string{"Follow language best practices"},
		"potential_risks":  []string{"May need manual refinement"},
	}
}

// fallbackPlanning provides basic planning as fallback
func fallbackPlanning(input interface{}) map[string]interface{} {
	// Generate a basic implementation plan
	return map[string]interface{}{
		"overview": "Generate code based on detected language and requirements",
		"steps": []map[string]interface{}{
			{
				"order":         1,
				"description":   "Create main implementation file",
				"code_files":    []string{"main.ext"},
				"rationale":     "Implement core functionality",
				"time_estimate": "15 minutes",
			},
		},
		"testing": map[string]interface{}{
			"unit_tests":        []string{"Test main functionality"},
			"integration_tests": []string{"Test end-to-end workflow"},
			"edge_cases":        []string{"Handle error conditions"},
		},
	}
}