// Package plugin demonstrates a complete example of the event bus system
// integrated with phase execution and plugin communication
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// Example: Complete Integration Demonstration
// This example shows how to use the event bus system for inter-plugin communication
// in a real orchestrator scenario.

// MockPhase implements core.Phase for demonstration
type MockPhase struct {
	name     string
	duration time.Duration
	shouldFail bool
	failAfter  int
	attempts   int
}

func NewMockPhase(name string, duration time.Duration) *MockPhase {
	return &MockPhase{
		name:     name,
		duration: duration,
	}
}

func (mp *MockPhase) WithFailure(shouldFail bool, failAfter int) *MockPhase {
	mp.shouldFail = shouldFail
	mp.failAfter = failAfter
	return mp
}

func (mp *MockPhase) Name() string {
	return mp.name
}

func (mp *MockPhase) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	mp.attempts++
	
	// Simulate work
	time.Sleep(mp.duration)
	
	// Simulate failure
	if mp.shouldFail && mp.attempts >= mp.failAfter {
		return core.PhaseOutput{}, fmt.Errorf("mock failure in phase %s", mp.name)
	}
	
	return core.PhaseOutput{
		Data: fmt.Sprintf("Output from %s phase", mp.name),
		Metadata: map[string]interface{}{
			"execution_time": mp.duration,
			"attempt":        mp.attempts,
		},
	}, nil
}

func (mp *MockPhase) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	return nil
}

func (mp *MockPhase) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil
}

func (mp *MockPhase) EstimatedDuration() time.Duration {
	return mp.duration
}

func (mp *MockPhase) CanRetry(err error) bool {
	return true // Always retryable for demo
}

// ComprehensiveExample demonstrates the complete event bus system
func ComprehensiveExample() {
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	fmt.Println("🚀 Starting Comprehensive Event Bus Example")
	fmt.Println("===========================================")
	
	// 1. Create event bus
	bus := NewEventBus(logger)
	defer bus.Stop()
	
	// 2. Create and start monitoring plugins
	startMonitoringPlugins(bus, logger)
	
	// 3. Create phase orchestrator
	orchestrator := NewPhaseOrchestrator(bus, logger)
	subscriber := NewPhaseEventSubscriber(bus)
	
	// 4. Set up phase event monitoring
	setupPhaseEventMonitoring(subscriber, logger)
	
	// 5. Create mock phases
	phases := []core.Phase{
		NewMockPhase("analyzer", 50*time.Millisecond),
		NewMockPhase("planner", 100*time.Millisecond),
		NewMockPhase("writer", 200*time.Millisecond).WithFailure(true, 2), // Will fail on 2nd attempt
		NewMockPhase("editor", 75*time.Millisecond),
		NewMockPhase("validator", 25*time.Millisecond),
	}
	
	// 6. Execute phase chain with event integration
	ctx := context.Background()
	input := core.PhaseInput{
		Request:   "Create a comprehensive AI story about plugin communication",
		SessionID: "demo_session_001",
		Metadata: map[string]interface{}{
			"user_id":     "demo_user",
			"project_id":  "event_bus_demo",
			"timestamp":   time.Now(),
		},
	}
	
	fmt.Println("\n📋 Executing Phase Chain with Event Integration")
	fmt.Println("-----------------------------------------------")
	
	// Wrap phases with retry capability
	retryablePhases := make([]core.Phase, len(phases))
	for i, phase := range phases {
		retryablePhases[i] = NewRetryablePhaseWrapper(
			phase, 
			bus, 
			3,                        // max attempts
			100*time.Millisecond,     // backoff
			logger,
		)
	}
	
	startTime := time.Now()
	output, err := orchestrator.ExecutePhaseChain(ctx, retryablePhases, input)
	totalDuration := time.Since(startTime)
	
	if err != nil {
		fmt.Printf("❌ Chain execution failed: %v\n", err)
	} else {
		fmt.Printf("✅ Chain execution completed successfully\n")
		fmt.Printf("📊 Final output: %v\n", output.Data)
	}
	
	fmt.Printf("⏱️  Total execution time: %v\n", totalDuration)
	
	// 7. Wait for async events to be processed
	time.Sleep(200 * time.Millisecond)
	
	// 8. Display metrics and analytics
	displayMetricsAndAnalytics(bus, logger)
	
	// 9. Demonstrate custom plugin communication
	demonstrateCustomPluginCommunication(bus, logger)
	
	fmt.Println("\n🎉 Example completed successfully!")
}

// startMonitoringPlugins initializes all monitoring plugins
func startMonitoringPlugins(bus *EventBus, logger *slog.Logger) {
	ctx := context.Background()
	
	fmt.Println("\n🔧 Starting Monitoring Plugins")
	fmt.Println("-----------------------------")
	
	// Quality Monitor Plugin
	qualityMonitor := NewQualityMonitorPlugin(bus, logger)
	if err := qualityMonitor.Start(ctx); err != nil {
		logger.Error("Failed to start quality monitor", "error", err)
	} else {
		fmt.Println("✅ Quality Monitor Plugin started")
	}
	
	// Caching Plugin
	cachingPlugin := NewCachingPlugin(bus, logger)
	if err := cachingPlugin.Start(ctx); err != nil {
		logger.Error("Failed to start caching plugin", "error", err)
	} else {
		fmt.Println("✅ Caching Plugin started")
	}
	
	// Notification Plugin
	notificationPlugin := NewNotificationPlugin(bus, logger)
	if err := notificationPlugin.Start(ctx); err != nil {
		logger.Error("Failed to start notification plugin", "error", err)
	} else {
		fmt.Println("✅ Notification Plugin started")
	}
	
	// Performance Analytics Plugin
	analyticsPlugin := NewPerformanceAnalyticsPlugin(bus, logger)
	if err := analyticsPlugin.Start(ctx); err != nil {
		logger.Error("Failed to start analytics plugin", "error", err)
	} else {
		fmt.Println("✅ Performance Analytics Plugin started")
	}
}

// setupPhaseEventMonitoring sets up detailed phase event monitoring
func setupPhaseEventMonitoring(subscriber *PhaseEventSubscriber, logger *slog.Logger) {
	fmt.Println("\n🔍 Setting up Phase Event Monitoring")
	fmt.Println("-----------------------------------")
	
	// Monitor phase starts
	subscriber.OnPhaseStarted(func(ctx context.Context, phaseName, sessionID string, input interface{}) error {
		fmt.Printf("🟢 Phase STARTED: %s (session: %s)\n", phaseName, sessionID)
		return nil
	}, SubscriptionOptions{
		Priority: 100,
		Async:    false,
	})
	
	// Monitor phase completions
	subscriber.OnPhaseCompleted(func(ctx context.Context, phaseName, sessionID string, output interface{}, duration time.Duration) error {
		fmt.Printf("✅ Phase COMPLETED: %s in %v (session: %s)\n", phaseName, duration, sessionID)
		return nil
	}, SubscriptionOptions{
		Priority: 100,
		Async:    false,
	})
	
	// Monitor phase failures
	subscriber.OnPhaseFailed(func(ctx context.Context, phaseName, sessionID, errorMsg string, attempt, maxAttempts int) error {
		fmt.Printf("❌ Phase FAILED: %s (attempt %d/%d) - %s (session: %s)\n", 
			phaseName, attempt, maxAttempts, errorMsg, sessionID)
		return nil
	}, SubscriptionOptions{
		Priority: 100,
		Async:    false,
	})
	
	// Monitor retries
	_, err := subscriber.bus.Subscribe(EventTypePhaseRetrying, func(ctx context.Context, event Event) error {
		data := event.Data.(PhaseEventData)
		fmt.Printf("🔄 Phase RETRYING: %s (attempt %d/%d) (session: %s)\n", 
			data.PhaseName, data.Attempt, data.MaxAttempts, data.SessionID)
		return nil
	}, SubscriptionOptions{
		Priority: 100,
		Async:    false,
	})
	
	if err != nil {
		logger.Error("Failed to subscribe to retry events", "error", err)
	}
	
	// Monitor chain events
	subscriber.bus.Subscribe("chain.*", func(ctx context.Context, event Event) error {
		fmt.Printf("🔗 Chain Event: %s\n", event.Type)
		return nil
	}, SubscriptionOptions{
		Priority: 90,
		Async:    false,
	})
	
	fmt.Println("✅ Phase event monitoring configured")
}

// displayMetricsAndAnalytics shows comprehensive system metrics
func displayMetricsAndAnalytics(bus *EventBus, logger *slog.Logger) {
	fmt.Println("\n📊 Event Bus Metrics & Analytics")
	fmt.Println("-------------------------------")
	
	metrics := bus.GetMetrics()
	
	fmt.Printf("📈 Total Events Published: %d\n", metrics.TotalPublished)
	fmt.Printf("📨 Total Events Delivered: %d\n", metrics.TotalDelivered)
	fmt.Printf("❌ Total Events Failed: %d\n", metrics.TotalFailed)
	fmt.Printf("🕐 Last Activity: %v\n", metrics.LastActivity.Format(time.RFC3339))
	
	if metrics.TotalPublished > 0 {
		successRate := float64(metrics.TotalDelivered) / float64(metrics.TotalPublished) * 100
		fmt.Printf("📊 Success Rate: %.2f%%\n", successRate)
	}
	
	fmt.Println("\n🔧 Active Subscriptions:")
	subscriptions := bus.ListSubscriptions()
	for i, sub := range subscriptions {
		fmt.Printf("  %d. ID: %s, Pattern: %s, Priority: %d, Async: %t\n",
			i+1, sub.ID[:8]+"...", sub.Pattern, sub.Priority, sub.Async)
	}
	
	fmt.Println("\n⏱️  Handler Performance:")
	for subID, duration := range metrics.HandlerDurations {
		fmt.Printf("  %s: %v\n", subID[:8]+"...", duration)
	}
}

// demonstrateCustomPluginCommunication shows advanced plugin-to-plugin communication
func demonstrateCustomPluginCommunication(bus *EventBus, logger *slog.Logger) {
	fmt.Println("\n🔄 Demonstrating Custom Plugin Communication")
	fmt.Println("-------------------------------------------")
	
	ctx := context.Background()
	
	// Create communication chain: Publisher -> Middleware -> Consumer
	
	// 1. Create Publisher Plugin
	publisher := &PublisherPlugin{bus: bus}
	
	// 2. Create and start Middleware Plugin
	middleware := &MiddlewarePlugin{bus: bus}
	middleware.Start()
	
	// 3. Create Consumer Plugin
	consumer := &ConsumerPlugin{
		bus: bus,
		processor: func(data interface{}) error {
			fmt.Printf("🎯 Consumer received data: %v\n", data)
			return nil
		},
	}
	consumer.Subscribe("transformed.*")
	
	// 4. Create Event Router
	router := &EventRouterPlugin{
		bus:    bus,
		routes: make(map[string]string),
	}
	router.AddRoute("custom.*", "routed.event")
	router.Start()
	
	// Subscribe to routed events
	bus.Subscribe("routed.*", func(ctx context.Context, event Event) error {
		fmt.Printf("📮 Routed event received: %s\n", event.Type)
		return nil
	})
	
	// 5. Publish custom events to demonstrate the communication chain
	customEvents := []struct {
		eventType string
		data      interface{}
	}{
		{"input.user_request", "Create a new feature"},
		{"input.system_update", map[string]string{"version": "1.2.3"}},
		{"custom.notification", "System maintenance scheduled"},
	}
	
	for _, customEvent := range customEvents {
		fmt.Printf("📤 Publishing: %s\n", customEvent.eventType)
		err := publisher.PublishCustomEvent(ctx, customEvent.eventType, customEvent.data)
		if err != nil {
			logger.Error("Failed to publish custom event", "error", err)
		}
		time.Sleep(50 * time.Millisecond) // Allow processing
	}
	
	// Wait for all async processing
	time.Sleep(200 * time.Millisecond)
	
	fmt.Println("✅ Custom plugin communication demonstration completed")
}

// PerformanceAnalyticsPlugin demonstrates advanced event analytics
type PerformanceAnalyticsPlugin struct {
	bus     *EventBus
	logger  *slog.Logger
	metrics map[string]*PerformanceMetric
}

type PerformanceMetric struct {
	EventCount      int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	LastSeen        time.Time
}

func NewPerformanceAnalyticsPlugin(bus *EventBus, logger *slog.Logger) *PerformanceAnalyticsPlugin {
	return &PerformanceAnalyticsPlugin{
		bus:     bus,
		logger:  logger,
		metrics: make(map[string]*PerformanceMetric),
	}
}

func (pap *PerformanceAnalyticsPlugin) Start(ctx context.Context) error {
	// Subscribe to all phase events for analytics
	_, err := pap.bus.Subscribe(PatternAllPhases, pap.handlePhaseEvent, SubscriptionOptions{
		Async:    true,
		Priority: 1, // Low priority to not interfere with business logic
	})
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to phase events: %w", err)
	}
	
	return nil
}

func (pap *PerformanceAnalyticsPlugin) handlePhaseEvent(ctx context.Context, event Event) error {
	if event.Type == EventTypePhaseCompleted {
		data, ok := event.Data.(PhaseEventData)
		if !ok {
			return nil
		}
		
		pap.updateMetrics(data.PhaseName, data.Duration)
		
		// Generate performance alerts if needed
		if metric := pap.metrics[data.PhaseName]; metric != nil {
			if data.Duration > metric.AverageDuration*2 {
				alertEvent := Event{
					Type:      "performance.slow_phase",
					Source:    "performance_analytics",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"phase_name":       data.PhaseName,
						"actual_duration":  data.Duration,
						"average_duration": metric.AverageDuration,
						"slowness_factor":  float64(data.Duration) / float64(metric.AverageDuration),
					},
				}
				
				pap.bus.Publish(ctx, alertEvent)
			}
		}
	}
	
	return nil
}

func (pap *PerformanceAnalyticsPlugin) updateMetrics(phaseName string, duration time.Duration) {
	if pap.metrics[phaseName] == nil {
		pap.metrics[phaseName] = &PerformanceMetric{
			MinDuration: duration,
			MaxDuration: duration,
		}
	}
	
	metric := pap.metrics[phaseName]
	metric.EventCount++
	metric.TotalDuration += duration
	metric.AverageDuration = time.Duration(int64(metric.TotalDuration) / metric.EventCount)
	metric.LastSeen = time.Now()
	
	if duration < metric.MinDuration {
		metric.MinDuration = duration
	}
	if duration > metric.MaxDuration {
		metric.MaxDuration = duration
	}
}

// RunExample is the main entry point for the comprehensive example
func RunExample() {
	ComprehensiveExample()
}