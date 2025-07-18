package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ExecutionEngine handles phase execution with performance optimizations
type ExecutionEngine struct {
	executor         *ParallelExecutor
	resultCache      *PhaseResultCache
	validationLogger *ValidationLogger
	enableCache      bool
	maxRetries       int
	logger           *slog.Logger
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(logger *slog.Logger, maxRetries int) *ExecutionEngine {
	return &ExecutionEngine{
		validationLogger: NewValidationLogger(),
		maxRetries:       maxRetries,
		logger:           logger,
	}
}

// WithPerformanceOptimization enables caching and parallel execution
func (e *ExecutionEngine) WithPerformanceOptimization(enabled bool) *ExecutionEngine {
	if enabled {
		ctx := context.Background()
		e.executor = NewParallelExecutor(ctx, 0) // Auto-detect optimal concurrency
		e.resultCache = NewPhaseResultCache(30*time.Minute, 1000)
		e.enableCache = true
	}
	return e
}

// WithCustomConcurrency sets specific concurrency levels
func (e *ExecutionEngine) WithCustomConcurrency(maxConcurrency int) *ExecutionEngine {
	ctx := context.Background()
	e.executor = NewParallelExecutor(ctx, maxConcurrency)
	if e.resultCache == nil {
		e.resultCache = NewPhaseResultCache(30*time.Minute, 1000)
	}
	e.enableCache = true
	return e
}

// ExecutePhases runs all phases with the appropriate execution strategy
func (e *ExecutionEngine) ExecutePhases(ctx context.Context, phases []Phase, request string, sessionID string, startPhase int, checkpoint *CheckpointManager) error {
	// Use optimized execution if available
	if e.executor != nil {
		return e.executeOptimized(ctx, phases, request, sessionID, startPhase, checkpoint)
	}
	
	// Standard execution
	return e.executeStandard(ctx, phases, request, sessionID, startPhase, checkpoint)
}

// executeOptimized handles execution with performance optimizations
func (e *ExecutionEngine) executeOptimized(ctx context.Context, phases []Phase, request string, sessionID string, startPhase int, checkpoint *CheckpointManager) error {
	e.logger.Info("starting optimized orchestration", 
		"request", request, 
		"session", sessionID,
		"cache_enabled", e.enableCache)
	
	// For sequential phases, use standard flow with caching
	if len(phases) <= 2 {
		return e.runOptimizedSequential(ctx, phases, request, sessionID)
	}
	
	// For many phases, use parallel execution where possible
	return e.runOptimizedParallel(ctx, phases, request, sessionID)
}

// runOptimizedSequential handles sequential execution with caching
func (e *ExecutionEngine) runOptimizedSequential(ctx context.Context, phases []Phase, request string, sessionID string) error {
	lastOutput := PhaseOutput{Data: nil}
	
	for i, phase := range phases {
		input := PhaseInput{
			Request:   request,
			Data:      lastOutput.Data,
			SessionID: sessionID,
			Metadata:  map[string]interface{}{"phase_index": i},
		}
		
		output, err := e.executePhaseOptimized(ctx, phase, input)
		if err != nil {
			return NewPhaseError(phase.Name(), 1, err, output.Data)
		}
		
		lastOutput = output
		e.logger.Info("phase completed", "name", phase.Name())
	}
	
	return nil
}

// runOptimizedParallel handles parallel execution for independent phases
func (e *ExecutionEngine) runOptimizedParallel(ctx context.Context, phases []Phase, request string, sessionID string) error {
	// For now, fall back to sequential since phases typically depend on each other
	// In future, could analyze phase dependencies for true parallel execution
	return e.runOptimizedSequential(ctx, phases, request, sessionID)
}

// executeStandard handles standard execution with retry logic
func (e *ExecutionEngine) executeStandard(ctx context.Context, phases []Phase, request string, sessionID string, startPhase int, checkpoint *CheckpointManager) error {
	e.logger.Info("starting orchestration", 
		"request", request, 
		"session", sessionID,
		"start_phase", startPhase)
	
	var lastOutput PhaseOutput
	
	if startPhase > 0 && checkpoint != nil {
		chkpt, err := checkpoint.Load(ctx, sessionID)
		if err == nil && chkpt.State != nil {
			if data, ok := chkpt.State["last_output"]; ok {
				lastOutput.Data = data
			}
		}
	}
	
	for i := startPhase; i < len(phases); i++ {
		phase := phases[i]
		
		if err := e.executePhaseWithRetry(ctx, phase, request, &lastOutput, sessionID); err != nil {
			return err
		}
		
		if checkpoint != nil {
			// Check if this is a resumeable writer phase with scene tracking
			if phase.Name() == "Writing" {
				// Try to get scene tracker from phase for enhanced checkpointing
				if err := checkpoint.Save(ctx, sessionID, i+1, phase.Name(), lastOutput.Data); err != nil {
					e.logger.Warn("failed to save checkpoint", "error", err)
				}
			} else {
				if err := checkpoint.Save(ctx, sessionID, i+1, phase.Name(), lastOutput.Data); err != nil {
					e.logger.Warn("failed to save checkpoint", "error", err)
				}
			}
		}
	}
	
	e.logger.Info("orchestration completed successfully", "session", sessionID)
	return nil
}

// executePhaseOptimized uses caching and performance optimizations
func (e *ExecutionEngine) executePhaseOptimized(ctx context.Context, phase Phase, input PhaseInput) (PhaseOutput, error) {
	// Check cache first if enabled
	if e.enableCache && e.resultCache != nil {
		if cached, found := e.resultCache.Get(ctx, phase.Name(), input); found {
			e.logger.Debug("cache hit", "phase", phase.Name())
			return cached, nil
		}
	}
	
	// Execute phase
	output, err := phase.Execute(ctx, input)
	
	// Cache successful results
	if err == nil && e.enableCache && e.resultCache != nil {
		e.resultCache.Set(ctx, phase.Name(), input, output)
	}
	
	return output, err
}

// executePhaseWithRetry executes a single phase with retry logic and validation
func (e *ExecutionEngine) executePhaseWithRetry(ctx context.Context, phase Phase, request string, lastOutput *PhaseOutput, sessionID string) error {
	input := PhaseInput{
		Request:   request,
		Data:      lastOutput.Data,
		SessionID: sessionID,
	}
	
	phaseCtx, cancel := context.WithTimeout(ctx, phase.EstimatedDuration())
	defer cancel()
	
	// Standardized validation using the Phase interface
	if err := phase.ValidateInput(phaseCtx, input); err != nil {
		e.logger.Error("Input validation failed", "phase", phase.Name(), "error", err)
		e.validationLogger.LogValidation(phase.Name(), "input", false, err, input)
		return NewPhaseError(phase.Name(), 0, fmt.Errorf("input validation failed: %w", err), nil)
	}
	e.validationLogger.LogValidation(phase.Name(), "input", true, nil, input)
	
	var lastErr error
	for attempt := 1; attempt <= e.maxRetries; attempt++ {
		e.logger.Info("executing phase", 
			"name", phase.Name(),
			"attempt", attempt,
			"timeout", phase.EstimatedDuration())
		
		output, err := phase.Execute(phaseCtx, input)
		
		// Standardized output validation using the Phase interface
		if err == nil {
			if validateErr := phase.ValidateOutput(phaseCtx, output); validateErr != nil {
				// Output validation failed - treat as execution error
				e.logger.Error("Output validation failed", "phase", phase.Name(), "error", validateErr)
				e.validationLogger.LogValidation(phase.Name(), "output", false, validateErr, output)
				err = NewPhaseError(phase.Name(), attempt, fmt.Errorf("output validation failed: %w", validateErr), output)
			} else {
				e.validationLogger.LogValidation(phase.Name(), "output", true, nil, output)
			}
		}
		
		if err == nil {
			*lastOutput = output
			e.logger.Info("phase completed", "name", phase.Name())
			return nil
		}
		
		lastErr = err
		
		if !phase.CanRetry(err) || attempt == e.maxRetries {
			return NewPhaseError(phase.Name(), attempt, err, output.Data)
		}
		
		retryDelay := time.Duration(attempt) * time.Second
		e.logger.Warn("phase failed, retrying", 
			"name", phase.Name(),
			"attempt", attempt,
			"error", err,
			"retry_after", retryDelay)
		
		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return NewPhaseError(phase.Name(), e.maxRetries, lastErr, nil)
}

// GetValidationReport returns the validation report for this execution
func (e *ExecutionEngine) GetValidationReport() string {
	if e.validationLogger == nil {
		return "No validation logger available"
	}
	return e.validationLogger.GetValidationReport()
}