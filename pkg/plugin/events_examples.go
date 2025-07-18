// Package plugin provides examples of inter-plugin communication using the event bus
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// Example 1: Quality Monitoring Plugin
// This plugin monitors phase completion and tracks quality metrics

type QualityMonitorPlugin struct {
	bus     *EventBus
	logger  *slog.Logger
	metrics map[string]*QualityMetrics
}

type QualityMetrics struct {
	PhaseName       string
	TotalExecutions int
	SuccessCount    int
	FailureCount    int
	AverageDuration time.Duration
	LastExecution   time.Time
}

func NewQualityMonitorPlugin(bus *EventBus, logger *slog.Logger) *QualityMonitorPlugin {
	return &QualityMonitorPlugin{
		bus:     bus,
		logger:  logger,
		metrics: make(map[string]*QualityMetrics),
	}
}

func (qmp *QualityMonitorPlugin) Start(ctx context.Context) error {
	// Subscribe to all phase lifecycle events
	_, err := qmp.bus.Subscribe(PatternPhaseLifecycle, qmp.handlePhaseEvent, SubscriptionOptions{
		Async:    true,
		Priority: 10, // High priority for monitoring
		Timeout:  5 * time.Second,
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to phase events: %w", err)
	}
	
	qmp.logger.Info("Quality monitor plugin started")
	return nil
}

func (qmp *QualityMonitorPlugin) handlePhaseEvent(ctx context.Context, event Event) error {
	data, ok := event.Data.(PhaseEventData)
	if !ok {
		// Try to parse from JSON if it's a map
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			jsonData, _ := json.Marshal(dataMap)
			json.Unmarshal(jsonData, &data)
		} else {
			return fmt.Errorf("unexpected event data type: %T", event.Data)
		}
	}
	
	qmp.updateMetrics(data.PhaseName, event.Type, data.Duration)
	
	// If phase fails repeatedly, publish a quality alert
	if event.Type == EventTypePhaseFailed {
		metrics := qmp.metrics[data.PhaseName]
		if metrics != nil && metrics.FailureCount > 3 {
			alertEvent := Event{
				Type:      "quality.alert",
				Source:    "quality_monitor",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"phase_name":    data.PhaseName,
					"failure_count": metrics.FailureCount,
					"success_rate":  float64(metrics.SuccessCount) / float64(metrics.TotalExecutions),
				},
			}
			
			qmp.bus.Publish(ctx, alertEvent)
		}
	}
	
	return nil
}

func (qmp *QualityMonitorPlugin) updateMetrics(phaseName, eventType string, duration time.Duration) {
	if qmp.metrics[phaseName] == nil {
		qmp.metrics[phaseName] = &QualityMetrics{
			PhaseName: phaseName,
		}
	}
	
	metrics := qmp.metrics[phaseName]
	metrics.LastExecution = time.Now()
	
	switch eventType {
	case EventTypePhaseCompleted:
		metrics.SuccessCount++
		metrics.TotalExecutions++
		// Update rolling average duration
		if metrics.AverageDuration == 0 {
			metrics.AverageDuration = duration
		} else {
			metrics.AverageDuration = (metrics.AverageDuration + duration) / 2
		}
	case EventTypePhaseFailed:
		metrics.FailureCount++
		metrics.TotalExecutions++
	}
}

// Example 2: Caching Plugin
// This plugin caches phase outputs and provides them for subsequent requests

type CachingPlugin struct {
	bus    *EventBus
	logger *slog.Logger
	cache  map[string]*CacheEntry
}

type CacheEntry struct {
	Output    interface{}
	Timestamp time.Time
	TTL       time.Duration
}

func NewCachingPlugin(bus *EventBus, logger *slog.Logger) *CachingPlugin {
	return &CachingPlugin{
		bus:    bus,
		logger: logger,
		cache:  make(map[string]*CacheEntry),
	}
}

func (cp *CachingPlugin) Start(ctx context.Context) error {
	// Subscribe to phase completed events to cache outputs
	_, err := cp.bus.Subscribe(EventTypePhaseCompleted, cp.handlePhaseCompleted, SubscriptionOptions{
		Async:    true,
		Priority: 5, // Medium priority
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to phase completed events: %w", err)
	}
	
	// Subscribe to phase started events to check cache
	_, err = cp.bus.Subscribe(EventTypePhaseStarted, cp.handlePhaseStarted, SubscriptionOptions{
		Async:    false, // Synchronous to potentially modify execution
		Priority: 20,    // Very high priority to run before phase execution
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to phase started events: %w", err)
	}
	
	cp.logger.Info("Caching plugin started")
	return nil
}

func (cp *CachingPlugin) handlePhaseCompleted(ctx context.Context, event Event) error {
	data, ok := event.Data.(PhaseEventData)
	if !ok {
		return fmt.Errorf("unexpected event data type: %T", event.Data)
	}
	
	// Create cache key from phase name and input
	cacheKey := cp.createCacheKey(data.PhaseName, data.Input)
	
	// Store in cache with 1 hour TTL
	cp.cache[cacheKey] = &CacheEntry{
		Output:    data.Output,
		Timestamp: time.Now(),
		TTL:       1 * time.Hour,
	}
	
	cp.logger.Debug("Cached phase output", "phase", data.PhaseName, "cache_key", cacheKey)
	return nil
}

func (cp *CachingPlugin) handlePhaseStarted(ctx context.Context, event Event) error {
	data, ok := event.Data.(PhaseEventData)
	if !ok {
		return fmt.Errorf("unexpected event data type: %T", event.Data)
	}
	
	// Check if we have cached output
	cacheKey := cp.createCacheKey(data.PhaseName, data.Input)
	entry, exists := cp.cache[cacheKey]
	
	if exists && time.Since(entry.Timestamp) < entry.TTL {
		// Publish cache hit event
		cacheHitEvent := Event{
			Type:      "cache.hit",
			Source:    "caching_plugin",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"phase_name": data.PhaseName,
				"cache_key":  cacheKey,
				"output":     entry.Output,
			},
		}
		
		cp.bus.Publish(ctx, cacheHitEvent)
		cp.logger.Debug("Cache hit", "phase", data.PhaseName, "cache_key", cacheKey)
	}
	
	return nil
}

func (cp *CachingPlugin) createCacheKey(phaseName string, input interface{}) string {
	// Simple cache key generation - in production, use proper hashing
	inputJSON, _ := json.Marshal(input)
	return fmt.Sprintf("%s:%x", phaseName, inputJSON)
}

// Example 3: Notification Plugin
// This plugin sends notifications based on various events

type NotificationPlugin struct {
	bus    *EventBus
	logger *slog.Logger
}

func NewNotificationPlugin(bus *EventBus, logger *slog.Logger) *NotificationPlugin {
	return &NotificationPlugin{
		bus:    bus,
		logger: logger,
	}
}

func (np *NotificationPlugin) Start(ctx context.Context) error {
	// Subscribe to quality alerts
	_, err := np.bus.Subscribe("quality.alert", np.handleQualityAlert, SubscriptionOptions{
		Async: true,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to quality alerts: %w", err)
	}
	
	// Subscribe to system errors
	_, err = np.bus.Subscribe(EventTypeSystemError, np.handleSystemError, SubscriptionOptions{
		Async:    true,
		Priority: 15, // High priority for error notifications
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to system errors: %w", err)
	}
	
	np.logger.Info("Notification plugin started")
	return nil
}

func (np *NotificationPlugin) handleQualityAlert(ctx context.Context, event Event) error {
	np.logger.Warn("Quality alert received", "event", event)
	
	// In a real implementation, this would send emails, Slack messages, etc.
	fmt.Printf("🚨 QUALITY ALERT: %s\n", event.Data)
	
	return nil
}

func (np *NotificationPlugin) handleSystemError(ctx context.Context, event Event) error {
	np.logger.Error("System error notification", "event", event)
	
	// In a real implementation, this would send critical alerts
	fmt.Printf("💥 SYSTEM ERROR: %s\n", event.Data)
	
	return nil
}

// Example 4: Cross-Plugin Communication Demo
// This example shows how plugins can communicate with each other

func DemonstrateCrossPluginCommunication() {
	logger := slog.Default()
	
	// Create event bus
	bus := NewEventBus(logger)
	defer bus.Stop()
	
	// Create and start plugins
	qualityMonitor := NewQualityMonitorPlugin(bus, logger)
	cachingPlugin := NewCachingPlugin(bus, logger)
	notificationPlugin := NewNotificationPlugin(bus, logger)
	
	ctx := context.Background()
	
	// Start all plugins
	qualityMonitor.Start(ctx)
	cachingPlugin.Start(ctx)
	notificationPlugin.Start(ctx)
	
	// Simulate phase execution events
	simulatePhaseExecution(ctx, bus)
	
	// Wait a moment for async handlers
	time.Sleep(100 * time.Millisecond)
	
	// Print metrics
	fmt.Println("\n=== Event Bus Metrics ===")
	metrics := bus.GetMetrics()
	fmt.Printf("Total Published: %d\n", metrics.TotalPublished)
	fmt.Printf("Total Delivered: %d\n", metrics.TotalDelivered)
	fmt.Printf("Total Failed: %d\n", metrics.TotalFailed)
	
	fmt.Println("\n=== Active Subscriptions ===")
	for _, sub := range bus.ListSubscriptions() {
		fmt.Printf("ID: %s, Pattern: %s, Priority: %d, Async: %t\n",
			sub.ID, sub.Pattern, sub.Priority, sub.Async)
	}
}

func simulatePhaseExecution(ctx context.Context, bus *EventBus) {
	phases := []string{"planner", "writer", "editor", "validator"}
	
	for _, phase := range phases {
		// Simulate phase started
		startEvent := NewPhaseStartedEvent(phase, "demo_session", map[string]string{
			"request": "Write a story about AI",
		})
		bus.Publish(ctx, startEvent)
		
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		// Simulate phase completion (most of the time)
		if phase != "validator" { // Let validator fail for demo
			completedEvent := NewPhaseCompletedEvent(phase, "demo_session", 
				map[string]string{"result": fmt.Sprintf("%s output", phase)},
				50*time.Millisecond)
			bus.Publish(ctx, completedEvent)
		} else {
			// Simulate multiple failures to trigger quality alert
			for i := 0; i < 5; i++ {
				failedEvent := NewPhaseFailedEvent(phase, "demo_session", 
					fmt.Errorf("validation failed: attempt %d", i+1), i+1, 5)
				bus.Publish(ctx, failedEvent)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
}

// Example 5: Plugin Communication Patterns

// Publisher Plugin: Generates custom events
type PublisherPlugin struct {
	bus *EventBus
}

func (p *PublisherPlugin) PublishCustomEvent(ctx context.Context, eventType string, data interface{}) error {
	event := Event{
		Type:      eventType,
		Source:    "publisher_plugin",
		Timestamp: time.Now(),
		Data:      data,
	}
	return p.bus.Publish(ctx, event)
}

// Consumer Plugin: Reacts to custom events
type ConsumerPlugin struct {
	bus       *EventBus
	processor func(interface{}) error
}

func (c *ConsumerPlugin) Subscribe(pattern string) error {
	_, err := c.bus.Subscribe(pattern, func(ctx context.Context, event Event) error {
		return c.processor(event.Data)
	}, SubscriptionOptions{
		Async: true,
	})
	return err
}

// Middleware Plugin: Transforms events
type MiddlewarePlugin struct {
	bus *EventBus
}

func (m *MiddlewarePlugin) Start() error {
	// Subscribe to input events and publish transformed events
	_, err := m.bus.Subscribe("input..*", func(ctx context.Context, event Event) error {
		// Transform the event
		transformedEvent := Event{
			Type:      "transformed." + event.Type,
			Source:    "middleware_plugin",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"original": event,
				"transformed_at": time.Now(),
			},
		}
		
		return m.bus.Publish(ctx, transformedEvent)
	}, SubscriptionOptions{
		Async:    false, // Synchronous to ensure ordering
		Priority: 100,   // Very high priority
	})
	
	return err
}

// Event Router Plugin: Routes events based on content
type EventRouterPlugin struct {
	bus   *EventBus
	routes map[string]string // event type -> target pattern
}

func (er *EventRouterPlugin) AddRoute(sourcePattern, targetType string) {
	er.routes[sourcePattern] = targetType
}

func (er *EventRouterPlugin) Start() error {
	_, err := er.bus.Subscribe(PatternAll, func(ctx context.Context, event Event) error {
		for pattern, targetType := range er.routes {
			if matched, _ := regexp.MatchString(pattern, event.Type); matched {
				routedEvent := Event{
					Type:      targetType,
					Source:    "event_router",
					Timestamp: time.Now(),
					Data:      event.Data,
					Metadata: map[string]interface{}{
						"original_event": event,
						"route_pattern": pattern,
					},
				}
				
				er.bus.Publish(ctx, routedEvent)
			}
		}
		return nil
	}, SubscriptionOptions{
		Async:    true,
		Priority: 50, // Medium priority
	})
	
	return err
}