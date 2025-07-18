// Package plugin provides integration between the event bus and phase execution
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// EventAwarePhase wraps a regular phase with event bus integration
type EventAwarePhase struct {
	wrapped core.Phase
	bus     *EventBus
	logger  *slog.Logger
}

// NewEventAwarePhase creates a phase wrapper that publishes events
func NewEventAwarePhase(phase core.Phase, bus *EventBus, logger *slog.Logger) *EventAwarePhase {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &EventAwarePhase{
		wrapped: phase,
		bus:     bus,
		logger:  logger,
	}
}

// Name returns the wrapped phase name
func (eap *EventAwarePhase) Name() string {
	return eap.wrapped.Name()
}

// Execute wraps phase execution with event publishing
func (eap *EventAwarePhase) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	phaseName := eap.wrapped.Name()
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "unknown"
	}
	
	// Publish phase started event
	startEvent := NewPhaseStartedEvent(phaseName, sessionID, input)
	if err := eap.bus.Publish(ctx, startEvent); err != nil {
		eap.logger.Warn("Failed to publish phase started event",
			"phase", phaseName,
			"error", err,
		)
	}
	
	start := time.Now()
	output, err := eap.wrapped.Execute(ctx, input)
	duration := time.Since(start)
	
	if err != nil {
		// Publish phase failed event
		failedEvent := NewPhaseFailedEvent(phaseName, sessionID, err, 1, 1)
		if publishErr := eap.bus.Publish(ctx, failedEvent); publishErr != nil {
			eap.logger.Warn("Failed to publish phase failed event",
				"phase", phaseName,
				"error", publishErr,
			)
		}
		return output, err
	}
	
	// Publish phase completed event
	completedEvent := NewPhaseCompletedEvent(phaseName, sessionID, output, duration)
	if publishErr := eap.bus.Publish(ctx, completedEvent); publishErr != nil {
		eap.logger.Warn("Failed to publish phase completed event",
			"phase", phaseName,
			"error", publishErr,
		)
	}
	
	return output, nil
}

// ValidateInput delegates to wrapped phase
func (eap *EventAwarePhase) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	return eap.wrapped.ValidateInput(ctx, input)
}

// ValidateOutput delegates to wrapped phase
func (eap *EventAwarePhase) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return eap.wrapped.ValidateOutput(ctx, output)
}

// EstimatedDuration delegates to wrapped phase
func (eap *EventAwarePhase) EstimatedDuration() time.Duration {
	return eap.wrapped.EstimatedDuration()
}

// CanRetry delegates to wrapped phase
func (eap *EventAwarePhase) CanRetry(err error) bool {
	return eap.wrapped.CanRetry(err)
}

// PhaseOrchestrator manages phase execution with event integration
type PhaseOrchestrator struct {
	bus    *EventBus
	logger *slog.Logger
}

// NewPhaseOrchestrator creates a new orchestrator with event bus integration
func NewPhaseOrchestrator(bus *EventBus, logger *slog.Logger) *PhaseOrchestrator {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &PhaseOrchestrator{
		bus:    bus,
		logger: logger,
	}
}

// WrapPhase wraps a phase with event awareness
func (po *PhaseOrchestrator) WrapPhase(phase core.Phase) core.Phase {
	return NewEventAwarePhase(phase, po.bus, po.logger)
}

// ExecutePhaseChain executes a sequence of phases with comprehensive event publishing
func (po *PhaseOrchestrator) ExecutePhaseChain(ctx context.Context, phases []core.Phase, input core.PhaseInput) (core.PhaseOutput, error) {
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("chain_%d", time.Now().UnixNano())
		input.SessionID = sessionID
	}
	
	// Publish chain started event
	chainStartEvent := Event{
		Type:      "chain.started",
		Source:    "phase_orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"phase_count": len(phases),
			"phase_names": po.getPhaseNames(phases),
		},
	}
	
	if err := po.bus.Publish(ctx, chainStartEvent); err != nil {
		po.logger.Warn("Failed to publish chain started event", "error", err)
	}
	
	currentInput := input
	var lastOutput core.PhaseOutput
	startTime := time.Now()
	
	for i, phase := range phases {
		wrappedPhase := po.WrapPhase(phase)
		
		po.logger.Info("Executing phase in chain",
			"phase", phase.Name(),
			"position", i+1,
			"total", len(phases),
			"session_id", sessionID,
		)
		
		output, err := wrappedPhase.Execute(ctx, currentInput)
		if err != nil {
			// Publish chain failed event
			chainFailedEvent := Event{
				Type:      "chain.failed",
				Source:    "phase_orchestrator",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"session_id":    sessionID,
					"failed_phase":  phase.Name(),
					"phase_index":   i,
					"error":         err.Error(),
					"elapsed_time":  time.Since(startTime),
					"completed_phases": po.getPhaseNames(phases[:i]),
				},
			}
			
			if publishErr := po.bus.Publish(ctx, chainFailedEvent); publishErr != nil {
				po.logger.Warn("Failed to publish chain failed event", "error", publishErr)
			}
			
			return output, fmt.Errorf("phase %s failed in chain: %w", phase.Name(), err)
		}
		
		lastOutput = output
		
		// Prepare input for next phase
		if i < len(phases)-1 {
			currentInput = core.PhaseInput{
				Request:   input.Request, // Keep original request
				Data:      output.Data,   // Use previous output as input
				SessionID: sessionID,
				Metadata:  output.Metadata,
			}
		}
	}
	
	// Publish chain completed event
	chainCompletedEvent := Event{
		Type:      "chain.completed",
		Source:    "phase_orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session_id":      sessionID,
			"total_duration":  time.Since(startTime),
			"phase_count":     len(phases),
			"completed_phases": po.getPhaseNames(phases),
		},
	}
	
	if err := po.bus.Publish(ctx, chainCompletedEvent); err != nil {
		po.logger.Warn("Failed to publish chain completed event", "error", err)
	}
	
	return lastOutput, nil
}

// getPhaseNames extracts phase names from a slice of phases
func (po *PhaseOrchestrator) getPhaseNames(phases []core.Phase) []string {
	names := make([]string, len(phases))
	for i, phase := range phases {
		names[i] = phase.Name()
	}
	return names
}

// RetryablePhaseWrapper wraps a phase with retry logic and event publishing
type RetryablePhaseWrapper struct {
	wrapped     core.Phase
	bus         *EventBus
	logger      *slog.Logger
	maxAttempts int
	backoff     time.Duration
}

// NewRetryablePhaseWrapper creates a phase wrapper with retry capabilities
func NewRetryablePhaseWrapper(phase core.Phase, bus *EventBus, maxAttempts int, backoff time.Duration, logger *slog.Logger) *RetryablePhaseWrapper {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &RetryablePhaseWrapper{
		wrapped:     phase,
		bus:         bus,
		logger:      logger,
		maxAttempts: maxAttempts,
		backoff:     backoff,
	}
}

// Execute implements phase execution with retry logic and event publishing
func (rpw *RetryablePhaseWrapper) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	phaseName := rpw.wrapped.Name()
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "unknown"
	}
	
	var lastErr error
	var lastOutput core.PhaseOutput
	
	for attempt := 1; attempt <= rpw.maxAttempts; attempt++ {
		// Publish retry event (if not first attempt)
		if attempt > 1 {
			retryEvent := Event{
				Type:      EventTypePhaseRetrying,
				Source:    fmt.Sprintf("phase.%s", phaseName),
				Timestamp: time.Now(),
				Data: PhaseEventData{
					PhaseName:   phaseName,
					SessionID:   sessionID,
					Attempt:     attempt,
					MaxAttempts: rpw.maxAttempts,
				},
			}
			
			if err := rpw.bus.Publish(ctx, retryEvent); err != nil {
				rpw.logger.Warn("Failed to publish retry event", "error", err)
			}
			
			// Wait before retry
			select {
			case <-ctx.Done():
				return core.PhaseOutput{}, ctx.Err()
			case <-time.After(rpw.backoff):
				// Continue with retry
			}
		}
		
		// Execute the wrapped phase
		eventAware := NewEventAwarePhase(rpw.wrapped, rpw.bus, rpw.logger)
		output, err := eventAware.Execute(ctx, input)
		
		if err == nil {
			if attempt > 1 {
				// Publish retry success event
				retrySuccessEvent := Event{
					Type:      "phase.retry_success",
					Source:    fmt.Sprintf("phase.%s", phaseName),
					Timestamp: time.Now(),
					Data: PhaseEventData{
						PhaseName:   phaseName,
						SessionID:   sessionID,
						Attempt:     attempt,
						MaxAttempts: rpw.maxAttempts,
						Output:      output,
					},
				}
				
				if publishErr := rpw.bus.Publish(ctx, retrySuccessEvent); publishErr != nil {
					rpw.logger.Warn("Failed to publish retry success event", "error", publishErr)
				}
			}
			
			return output, nil
		}
		
		lastErr = err
		lastOutput = output
		
		// Check if error is retryable
		if !rpw.wrapped.CanRetry(err) {
			rpw.logger.Info("Error is not retryable, stopping attempts",
				"phase", phaseName,
				"attempt", attempt,
				"error", err,
			)
			break
		}
		
		rpw.logger.Warn("Phase attempt failed, will retry",
			"phase", phaseName,
			"attempt", attempt,
			"max_attempts", rpw.maxAttempts,
			"error", err,
		)
	}
	
	// All attempts failed
	finalFailEvent := Event{
		Type:      "phase.final_failure",
		Source:    fmt.Sprintf("phase.%s", phaseName),
		Timestamp: time.Now(),
		Data: PhaseEventData{
			PhaseName:   phaseName,
			SessionID:   sessionID,
			Error:       lastErr.Error(),
			Attempt:     rpw.maxAttempts,
			MaxAttempts: rpw.maxAttempts,
		},
	}
	
	if err := rpw.bus.Publish(ctx, finalFailEvent); err != nil {
		rpw.logger.Warn("Failed to publish final failure event", "error", err)
	}
	
	return lastOutput, fmt.Errorf("phase %s failed after %d attempts: %w", phaseName, rpw.maxAttempts, lastErr)
}

// Name returns the wrapped phase name
func (rpw *RetryablePhaseWrapper) Name() string {
	return rpw.wrapped.Name()
}

// ValidateInput delegates to wrapped phase
func (rpw *RetryablePhaseWrapper) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	return rpw.wrapped.ValidateInput(ctx, input)
}

// ValidateOutput delegates to wrapped phase
func (rpw *RetryablePhaseWrapper) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return rpw.wrapped.ValidateOutput(ctx, output)
}

// EstimatedDuration returns wrapped phase duration multiplied by max attempts
func (rpw *RetryablePhaseWrapper) EstimatedDuration() time.Duration {
	return rpw.wrapped.EstimatedDuration() * time.Duration(rpw.maxAttempts)
}

// CanRetry delegates to wrapped phase
func (rpw *RetryablePhaseWrapper) CanRetry(err error) bool {
	return rpw.wrapped.CanRetry(err)
}

// PhaseEventSubscriber provides convenience methods for subscribing to phase events
type PhaseEventSubscriber struct {
	bus *EventBus
}

// NewPhaseEventSubscriber creates a new phase event subscriber
func NewPhaseEventSubscriber(bus *EventBus) *PhaseEventSubscriber {
	return &PhaseEventSubscriber{bus: bus}
}

// OnPhaseStarted subscribes to phase started events
func (pes *PhaseEventSubscriber) OnPhaseStarted(handler func(ctx context.Context, phaseName, sessionID string, input interface{}) error, options ...SubscriptionOptions) (*EventSubscription, error) {
	return pes.bus.Subscribe(EventTypePhaseStarted, func(ctx context.Context, event Event) error {
		data, ok := event.Data.(PhaseEventData)
		if !ok {
			return fmt.Errorf("unexpected event data type: %T", event.Data)
		}
		return handler(ctx, data.PhaseName, data.SessionID, data.Input)
	}, options...)
}

// OnPhaseCompleted subscribes to phase completed events
func (pes *PhaseEventSubscriber) OnPhaseCompleted(handler func(ctx context.Context, phaseName, sessionID string, output interface{}, duration time.Duration) error, options ...SubscriptionOptions) (*EventSubscription, error) {
	return pes.bus.Subscribe(EventTypePhaseCompleted, func(ctx context.Context, event Event) error {
		data, ok := event.Data.(PhaseEventData)
		if !ok {
			return fmt.Errorf("unexpected event data type: %T", event.Data)
		}
		return handler(ctx, data.PhaseName, data.SessionID, data.Output, data.Duration)
	}, options...)
}

// OnPhaseFailed subscribes to phase failed events
func (pes *PhaseEventSubscriber) OnPhaseFailed(handler func(ctx context.Context, phaseName, sessionID, errorMsg string, attempt, maxAttempts int) error, options ...SubscriptionOptions) (*EventSubscription, error) {
	return pes.bus.Subscribe(EventTypePhaseFailed, func(ctx context.Context, event Event) error {
		data, ok := event.Data.(PhaseEventData)
		if !ok {
			return fmt.Errorf("unexpected event data type: %T", event.Data)
		}
		return handler(ctx, data.PhaseName, data.SessionID, data.Error, data.Attempt, data.MaxAttempts)
	}, options...)
}

// OnAnyPhaseEvent subscribes to all phase lifecycle events
func (pes *PhaseEventSubscriber) OnAnyPhaseEvent(handler EventHandler, options ...SubscriptionOptions) (*EventSubscription, error) {
	return pes.bus.Subscribe(PatternPhaseLifecycle, handler, options...)
}

// OnSpecificPhase subscribes to events for a specific phase
func (pes *PhaseEventSubscriber) OnSpecificPhase(phaseName string, handler EventHandler, options ...SubscriptionOptions) (*EventSubscription, error) {
	pattern := fmt.Sprintf("^phase\\.(started|completed|failed|retrying)$")
	
	// Add filter for specific phase
	opts := DefaultSubscriptionOptions
	if len(options) > 0 {
		opts = options[0]
	}
	
	originalFilter := opts.FilterFunc
	opts.FilterFunc = func(event Event) bool {
		// Check original filter first
		if originalFilter != nil && !originalFilter(event) {
			return false
		}
		
		// Check if this event is for the specific phase
		if data, ok := event.Data.(PhaseEventData); ok {
			return data.PhaseName == phaseName
		}
		
		return false
	}
	
	return pes.bus.Subscribe(pattern, handler, opts)
}