// Package plugin provides an event bus system for inter-plugin communication
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"
)

// Event represents a system event that can be published and subscribed to
type Event struct {
	// ID uniquely identifies this event instance
	ID string `json:"id"`
	
	// Type categorizes the event (e.g., "phase.started", "plugin.loaded")
	Type string `json:"type"`
	
	// Source identifies what generated the event
	Source string `json:"source"`
	
	// Timestamp when the event was created
	Timestamp time.Time `json:"timestamp"`
	
	// Data contains event-specific payload
	Data interface{} `json:"data"`
	
	// Metadata for additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EventHandler is a function that processes events
type EventHandler func(ctx context.Context, event Event) error

// EventSubscription represents an active subscription to events
type EventSubscription struct {
	ID       string
	Pattern  string
	Handler  EventHandler
	Options  SubscriptionOptions
	compiled *regexp.Regexp
}

// SubscriptionOptions configure how subscriptions behave
type SubscriptionOptions struct {
	// Async determines if the handler should be called asynchronously
	Async bool
	
	// Buffer size for async handlers (ignored if Async is false)
	BufferSize int
	
	// Timeout for handler execution
	Timeout time.Duration
	
	// MaxRetries for failed handler calls
	MaxRetries int
	
	// FilterFunc provides additional filtering beyond pattern matching
	FilterFunc func(Event) bool
	
	// Priority affects the order handlers are called (higher = earlier)
	Priority int
}

// DefaultSubscriptionOptions provides sensible defaults
var DefaultSubscriptionOptions = SubscriptionOptions{
	Async:      false,
	BufferSize: 100,
	Timeout:    30 * time.Second,
	MaxRetries: 3,
	Priority:   0,
}

// EventBus manages event publishing and subscription
type EventBus struct {
	mu           sync.RWMutex
	subscriptions map[string]*EventSubscription
	logger       *slog.Logger
	running      bool
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	
	// Metrics
	metrics *EventMetrics
}

// EventMetrics tracks bus performance
type EventMetrics struct {
	mu              sync.RWMutex
	TotalPublished  int64
	TotalDelivered  int64
	TotalFailed     int64
	HandlerDurations map[string]time.Duration
	LastActivity    time.Time
}

// NewEventBus creates a new event bus
func NewEventBus(logger *slog.Logger) *EventBus {
	if logger == nil {
		logger = slog.Default()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EventBus{
		subscriptions: make(map[string]*EventSubscription),
		logger:       logger,
		running:      true,
		ctx:          ctx,
		cancel:       cancel,
		metrics: &EventMetrics{
			HandlerDurations: make(map[string]time.Duration),
			LastActivity:     time.Now(),
		},
	}
}

// Subscribe registers a handler for events matching the given pattern
func (eb *EventBus) Subscribe(pattern string, handler EventHandler, options ...SubscriptionOptions) (*EventSubscription, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}
	
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}
	
	// Compile pattern as regex
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	
	// Use default options if none provided
	opts := DefaultSubscriptionOptions
	if len(options) > 0 {
		opts = options[0]
	}
	
	// Generate unique subscription ID
	subID := fmt.Sprintf("sub_%d_%s", time.Now().UnixNano(), generateShortID())
	
	subscription := &EventSubscription{
		ID:       subID,
		Pattern:  pattern,
		Handler:  handler,
		Options:  opts,
		compiled: compiled,
	}
	
	eb.mu.Lock()
	eb.subscriptions[subID] = subscription
	eb.mu.Unlock()
	
	eb.logger.Debug("Event subscription created",
		"subscription_id", subID,
		"pattern", pattern,
		"async", opts.Async,
		"priority", opts.Priority,
	)
	
	return subscription, nil
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	if _, exists := eb.subscriptions[subscriptionID]; !exists {
		return fmt.Errorf("subscription %q not found", subscriptionID)
	}
	
	delete(eb.subscriptions, subscriptionID)
	
	eb.logger.Debug("Event subscription removed", "subscription_id", subscriptionID)
	return nil
}

// Publish sends an event to all matching subscribers
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}
	
	// Set event ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), generateShortID())
	}
	
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	eb.updateMetrics(func(m *EventMetrics) {
		m.TotalPublished++
		m.LastActivity = time.Now()
	})
	
	eb.logger.Debug("Publishing event",
		"event_id", event.ID,
		"type", event.Type,
		"source", event.Source,
	)
	
	// Get matching subscriptions
	matching := eb.getMatchingSubscriptions(event)
	
	if len(matching) == 0 {
		eb.logger.Debug("No matching subscriptions for event", "event_type", event.Type)
		return nil
	}
	
	// Sort by priority (descending)
	eb.sortSubscriptionsByPriority(matching)
	
	// Deliver to each subscription
	for _, sub := range matching {
		if err := eb.deliverToSubscription(ctx, event, sub); err != nil {
			eb.logger.Error("Failed to deliver event to subscription",
				"event_id", event.ID,
				"subscription_id", sub.ID,
				"error", err,
			)
			
			eb.updateMetrics(func(m *EventMetrics) {
				m.TotalFailed++
			})
		} else {
			eb.updateMetrics(func(m *EventMetrics) {
				m.TotalDelivered++
			})
		}
	}
	
	return nil
}

// PublishPhaseEvent is a convenience method for publishing phase lifecycle events
func (eb *EventBus) PublishPhaseEvent(ctx context.Context, eventType, phaseName, sessionID string, data interface{}) error {
	event := Event{
		Type:      eventType,
		Source:    fmt.Sprintf("phase.%s", phaseName),
		Timestamp: time.Now(),
		Data:      data,
		Metadata: map[string]interface{}{
			"phase_name": phaseName,
			"session_id": sessionID,
		},
	}
	
	return eb.Publish(ctx, event)
}

// getMatchingSubscriptions finds all subscriptions that match the event
func (eb *EventBus) getMatchingSubscriptions(event Event) []*EventSubscription {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	var matching []*EventSubscription
	
	for _, sub := range eb.subscriptions {
		// Check pattern match
		if !sub.compiled.MatchString(event.Type) {
			continue
		}
		
		// Check additional filter
		if sub.Options.FilterFunc != nil && !sub.Options.FilterFunc(event) {
			continue
		}
		
		matching = append(matching, sub)
	}
	
	return matching
}

// sortSubscriptionsByPriority sorts subscriptions by priority (descending)
func (eb *EventBus) sortSubscriptionsByPriority(subs []*EventSubscription) {
	// Simple bubble sort by priority (descending)
	for i := 0; i < len(subs)-1; i++ {
		for j := 0; j < len(subs)-i-1; j++ {
			if subs[j].Options.Priority < subs[j+1].Options.Priority {
				subs[j], subs[j+1] = subs[j+1], subs[j]
			}
		}
	}
}

// deliverToSubscription delivers an event to a specific subscription
func (eb *EventBus) deliverToSubscription(ctx context.Context, event Event, sub *EventSubscription) error {
	if sub.Options.Async {
		// Async delivery
		eb.wg.Add(1)
		go func() {
			defer eb.wg.Done()
			eb.executeHandler(ctx, event, sub)
		}()
		return nil
	}
	
	// Sync delivery
	return eb.executeHandler(ctx, event, sub)
}

// executeHandler executes a subscription handler with timeout and retry logic
func (eb *EventBus) executeHandler(ctx context.Context, event Event, sub *EventSubscription) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		eb.updateMetrics(func(m *EventMetrics) {
			m.HandlerDurations[sub.ID] = duration
		})
	}()
	
	// Create context with timeout
	handlerCtx := ctx
	if sub.Options.Timeout > 0 {
		var cancel context.CancelFunc
		handlerCtx, cancel = context.WithTimeout(ctx, sub.Options.Timeout)
		defer cancel()
	}
	
	var lastErr error
	maxRetries := sub.Options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := eb.safeExecuteHandler(handlerCtx, event, sub)
		if err == nil {
			if attempt > 1 {
				eb.logger.Debug("Handler succeeded after retry",
					"subscription_id", sub.ID,
					"attempt", attempt,
				)
			}
			return nil
		}
		
		lastErr = err
		eb.logger.Warn("Handler execution failed",
			"subscription_id", sub.ID,
			"attempt", attempt,
			"max_retries", maxRetries,
			"error", err,
		)
		
		// Don't retry if context is cancelled
		if handlerCtx.Err() != nil {
			break
		}
		
		// Brief delay between retries
		if attempt < maxRetries {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return lastErr
}

// safeExecuteHandler executes a handler with panic recovery
func (eb *EventBus) safeExecuteHandler(ctx context.Context, event Event, sub *EventSubscription) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panicked: %v", r)
			eb.logger.Error("Event handler panicked",
				"subscription_id", sub.ID,
				"event_id", event.ID,
				"panic", r,
			)
		}
	}()
	
	return sub.Handler(ctx, event)
}

// Stop gracefully shuts down the event bus
func (eb *EventBus) Stop() {
	eb.mu.Lock()
	eb.running = false
	eb.mu.Unlock()
	
	eb.cancel()
	eb.wg.Wait()
	
	eb.logger.Info("Event bus stopped")
}

// GetMetrics returns current bus metrics
func (eb *EventBus) GetMetrics() EventMetrics {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *eb.metrics
	metrics.HandlerDurations = make(map[string]time.Duration)
	for k, v := range eb.metrics.HandlerDurations {
		metrics.HandlerDurations[k] = v
	}
	
	return metrics
}

// ListSubscriptions returns information about active subscriptions
func (eb *EventBus) ListSubscriptions() []SubscriptionInfo {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	var infos []SubscriptionInfo
	for _, sub := range eb.subscriptions {
		infos = append(infos, SubscriptionInfo{
			ID:       sub.ID,
			Pattern:  sub.Pattern,
			Priority: sub.Options.Priority,
			Async:    sub.Options.Async,
		})
	}
	
	return infos
}

// SubscriptionInfo provides public information about a subscription
type SubscriptionInfo struct {
	ID       string
	Pattern  string
	Priority int
	Async    bool
}

// updateMetrics safely updates metrics
func (eb *EventBus) updateMetrics(updater func(*EventMetrics)) {
	eb.metrics.mu.Lock()
	defer eb.metrics.mu.Unlock()
	updater(eb.metrics)
}

// generateShortID creates a short random identifier
func generateShortID() string {
	// Simple implementation - in production, consider using a proper UUID library
	return fmt.Sprintf("%x", time.Now().UnixNano()%0xFFFF)
}

// Predefined event types for phase lifecycle
const (
	// Phase Events
	EventTypePhaseStarted   = "phase.started"
	EventTypePhaseCompleted = "phase.completed"
	EventTypePhaseFailed    = "phase.failed"
	EventTypePhaseRetrying  = "phase.retrying"
	
	// Plugin Events
	EventTypePluginLoaded   = "plugin.loaded"
	EventTypePluginUnloaded = "plugin.unloaded"
	EventTypePluginError    = "plugin.error"
	
	// System Events
	EventTypeSystemStartup  = "system.startup"
	EventTypeSystemShutdown = "system.shutdown"
	EventTypeSystemError    = "system.error"
)

// Common event patterns for easy subscription
const (
	// Pattern to match all phase events
	PatternAllPhases = `^phase\..*`
	
	// Pattern to match all plugin events
	PatternAllPlugins = `^plugin\..*`
	
	// Pattern to match all system events
	PatternAllSystem = `^system\..*`
	
	// Pattern to match all events
	PatternAll = `.*`
	
	// Pattern for specific phase events
	PatternPhaseLifecycle = `^phase\.(started|completed|failed|retrying)$`
)

// PhaseEventData represents data for phase lifecycle events
type PhaseEventData struct {
	PhaseName    string                 `json:"phase_name"`
	SessionID    string                 `json:"session_id"`
	Input        interface{}            `json:"input,omitempty"`
	Output       interface{}            `json:"output,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Attempt      int                    `json:"attempt,omitempty"`
	MaxAttempts  int                    `json:"max_attempts,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PluginEventData represents data for plugin lifecycle events
type PluginEventData struct {
	PluginName    string                 `json:"plugin_name"`
	PluginVersion string                 `json:"plugin_version"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Helper functions for creating common events

// NewPhaseStartedEvent creates a phase started event
func NewPhaseStartedEvent(phaseName, sessionID string, input interface{}) Event {
	return Event{
		Type:      EventTypePhaseStarted,
		Source:    fmt.Sprintf("phase.%s", phaseName),
		Timestamp: time.Now(),
		Data: PhaseEventData{
			PhaseName: phaseName,
			SessionID: sessionID,
			Input:     input,
		},
		Metadata: map[string]interface{}{
			"phase_name": phaseName,
			"session_id": sessionID,
		},
	}
}

// NewPhaseCompletedEvent creates a phase completed event
func NewPhaseCompletedEvent(phaseName, sessionID string, output interface{}, duration time.Duration) Event {
	return Event{
		Type:      EventTypePhaseCompleted,
		Source:    fmt.Sprintf("phase.%s", phaseName),
		Timestamp: time.Now(),
		Data: PhaseEventData{
			PhaseName: phaseName,
			SessionID: sessionID,
			Output:    output,
			Duration:  duration,
		},
		Metadata: map[string]interface{}{
			"phase_name": phaseName,
			"session_id": sessionID,
		},
	}
}

// NewPhaseFailedEvent creates a phase failed event
func NewPhaseFailedEvent(phaseName, sessionID string, err error, attempt, maxAttempts int) Event {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	
	return Event{
		Type:      EventTypePhaseFailed,
		Source:    fmt.Sprintf("phase.%s", phaseName),
		Timestamp: time.Now(),
		Data: PhaseEventData{
			PhaseName:   phaseName,
			SessionID:   sessionID,
			Error:       errorMsg,
			Attempt:     attempt,
			MaxAttempts: maxAttempts,
		},
		Metadata: map[string]interface{}{
			"phase_name": phaseName,
			"session_id": sessionID,
		},
	}
}

// EventBusMiddleware provides integration with existing phase execution
type EventBusMiddleware struct {
	bus *EventBus
}

// NewEventBusMiddleware creates middleware for phase integration
func NewEventBusMiddleware(bus *EventBus) *EventBusMiddleware {
	return &EventBusMiddleware{bus: bus}
}

// WrapPhaseExecution wraps phase execution with event publishing
func (m *EventBusMiddleware) WrapPhaseExecution(phaseName string, executor func(ctx context.Context) (interface{}, error)) func(ctx context.Context) (interface{}, error) {
	return func(ctx context.Context) (interface{}, error) {
		sessionID := "unknown" // TODO: Extract from context
		
		// Publish phase started event
		startEvent := NewPhaseStartedEvent(phaseName, sessionID, nil)
		if err := m.bus.Publish(ctx, startEvent); err != nil {
			// Log but don't fail execution
			slog.Warn("Failed to publish phase started event", "error", err)
		}
		
		start := time.Now()
		output, err := executor(ctx)
		duration := time.Since(start)
		
		if err != nil {
			// Publish phase failed event
			failedEvent := NewPhaseFailedEvent(phaseName, sessionID, err, 1, 1)
			if publishErr := m.bus.Publish(ctx, failedEvent); publishErr != nil {
				slog.Warn("Failed to publish phase failed event", "error", publishErr)
			}
			return output, err
		}
		
		// Publish phase completed event
		completedEvent := NewPhaseCompletedEvent(phaseName, sessionID, output, duration)
		if publishErr := m.bus.Publish(ctx, completedEvent); publishErr != nil {
			slog.Warn("Failed to publish phase completed event", "error", publishErr)
		}
		
		return output, nil
	}
}