# Plugin Event Bus System

A thread-safe, production-ready event bus system for inter-plugin communication in the Orchestrator project.

## Features

- 🔄 **Pattern-based Subscriptions**: Use regex patterns to subscribe to multiple event types
- 🚀 **Async/Sync Handlers**: Support for both synchronous and asynchronous event handling
- 🎯 **Priority System**: Control handler execution order with priority levels
- 🔄 **Retry Logic**: Automatic retry for failed handlers with exponential backoff
- 🛡️ **Panic Recovery**: Safe handler execution with panic recovery
- ⏱️ **Timeouts**: Configurable timeouts for handler execution
- 📊 **Metrics**: Comprehensive metrics and performance tracking
- 🎭 **Event Filtering**: Advanced filtering with custom filter functions
- 🔌 **Phase Integration**: Seamless integration with orchestrator phases

## Quick Start

### Basic Usage

```go
import "github.com/dotcommander/orc/pkg/plugin"

// Create event bus
bus := plugin.NewEventBus(logger)
defer bus.Stop()

// Subscribe to events
sub, err := bus.Subscribe("phase.*", func(ctx context.Context, event plugin.Event) error {
    fmt.Printf("Received event: %s\n", event.Type)
    return nil
})

// Publish events
event := plugin.Event{
    Type:   "phase.started",
    Source: "my_plugin",
    Data:   "some data",
}

err = bus.Publish(ctx, event)
```

### Phase Integration

```go
// Create phase orchestrator with event integration
orchestrator := plugin.NewPhaseOrchestrator(bus, logger)

// Wrap phases with event awareness
eventAwarePhase := orchestrator.WrapPhase(myPhase)

// Execute with automatic event publishing
output, err := eventAwarePhase.Execute(ctx, input)
```

### Plugin-to-Plugin Communication

```go
// Quality Monitor Plugin
qualityMonitor := plugin.NewQualityMonitorPlugin(bus, logger)
qualityMonitor.Start(ctx)

// Caching Plugin  
cachingPlugin := plugin.NewCachingPlugin(bus, logger)
cachingPlugin.Start(ctx)

// They automatically communicate via events!
```

## Event Types

### Predefined Event Types

| Event Type | Description | Data Type |
|------------|-------------|-----------|
| `phase.started` | Phase execution begins | `PhaseEventData` |
| `phase.completed` | Phase execution succeeds | `PhaseEventData` |
| `phase.failed` | Phase execution fails | `PhaseEventData` |
| `phase.retrying` | Phase is retrying after failure | `PhaseEventData` |
| `plugin.loaded` | Plugin has been loaded | `PluginEventData` |
| `plugin.unloaded` | Plugin has been unloaded | `PluginEventData` |
| `system.startup` | System is starting up | Custom |
| `system.shutdown` | System is shutting down | Custom |

### Common Patterns

| Pattern | Description |
|---------|-------------|
| `phase.*` | All phase events |
| `plugin.*` | All plugin events |
| `system.*` | All system events |
| `.*` | All events |
| `^phase\.(started\|completed)$` | Only phase start/complete |

## Subscription Options

```go
options := plugin.SubscriptionOptions{
    Async:      true,                    // Handle asynchronously
    BufferSize: 100,                     // Buffer size for async handlers
    Timeout:    30 * time.Second,        // Handler timeout
    MaxRetries: 3,                       // Retry attempts
    Priority:   10,                      // Handler priority (higher = earlier)
    FilterFunc: func(event Event) bool { // Custom filtering
        return event.Source == "my_plugin"
    },
}

sub, err := bus.Subscribe("pattern.*", handler, options)
```

## Phase Integration Features

### Event-Aware Phases

Automatically publish events for phase lifecycle:

```go
// Wrap any phase with event awareness
eventAware := plugin.NewEventAwarePhase(originalPhase, bus, logger)

// Events are published automatically:
// - phase.started when execution begins
// - phase.completed when execution succeeds  
// - phase.failed when execution fails
```

### Retryable Phases

Add retry logic with event publishing:

```go
retryable := plugin.NewRetryablePhaseWrapper(
    originalPhase,
    bus,
    3,                        // max attempts
    100*time.Millisecond,     // backoff duration
    logger,
)

// Additional events published:
// - phase.retrying on retry attempts
// - phase.retry_success on eventual success
// - phase.final_failure after all retries fail
```

### Phase Chain Orchestration

Execute multiple phases with comprehensive event tracking:

```go
orchestrator := plugin.NewPhaseOrchestrator(bus, logger)

output, err := orchestrator.ExecutePhaseChain(ctx, phases, input)

// Events published:
// - chain.started when chain begins
// - phase.* events for each phase
// - chain.completed when chain succeeds
// - chain.failed if any phase fails
```

## Plugin Examples

### Quality Monitor Plugin

Tracks phase performance and generates alerts:

```go
monitor := plugin.NewQualityMonitorPlugin(bus, logger)
monitor.Start(ctx)

// Automatically:
// - Tracks success/failure rates
// - Monitors execution times
// - Generates quality.alert events for issues
```

### Caching Plugin

Caches phase outputs for reuse:

```go
cache := plugin.NewCachingPlugin(bus, logger)
cache.Start(ctx)

// Automatically:
// - Caches phase outputs on completion
// - Checks cache on phase start
// - Publishes cache.hit events
```

### Notification Plugin

Sends notifications for important events:

```go
notifier := plugin.NewNotificationPlugin(bus, logger)
notifier.Start(ctx)

// Automatically:
// - Sends alerts for quality issues
// - Notifies on system errors
// - Handles critical events
```

## Advanced Features

### Custom Event Creation

```go
// Helper functions for common events
startEvent := plugin.NewPhaseStartedEvent("planner", "session123", input)
completeEvent := plugin.NewPhaseCompletedEvent("planner", "session123", output, duration)
failEvent := plugin.NewPhaseFailedEvent("planner", "session123", err, 1, 3)

// Or create custom events
customEvent := plugin.Event{
    Type:      "custom.business_event",
    Source:    "my_plugin",
    Timestamp: time.Now(),
    Data:      myCustomData,
    Metadata: map[string]interface{}{
        "user_id": "user123",
        "version": "1.0",
    },
}
```

### Event Filtering

```go
// Filter by data content
bus.Subscribe("phase.*", handler, plugin.SubscriptionOptions{
    FilterFunc: func(event plugin.Event) bool {
        data, ok := event.Data.(plugin.PhaseEventData)
        return ok && data.PhaseName == "writer"
    },
})

// Filter by source
bus.Subscribe(".*", handler, plugin.SubscriptionOptions{
    FilterFunc: func(event plugin.Event) bool {
        return event.Source == "quality_monitor"
    },
})
```

### Metrics and Monitoring

```go
// Get comprehensive metrics
metrics := bus.GetMetrics()
fmt.Printf("Published: %d, Delivered: %d, Failed: %d\n", 
    metrics.TotalPublished, metrics.TotalDelivered, metrics.TotalFailed)

// List active subscriptions
for _, sub := range bus.ListSubscriptions() {
    fmt.Printf("Subscription %s: %s (Priority: %d)\n", 
        sub.ID, sub.Pattern, sub.Priority)
}
```

## Best Practices

### 1. Use Appropriate Patterns

```go
// ✅ Good: Specific patterns
bus.Subscribe("phase\.(started|completed)", handler)

// ❌ Avoid: Overly broad patterns
bus.Subscribe(".*", handler) // Receives ALL events
```

### 2. Set Proper Priorities

```go
// High priority for critical handlers
bus.Subscribe("system.error", criticalHandler, plugin.SubscriptionOptions{
    Priority: 100,
})

// Low priority for analytics
bus.Subscribe(".*", analyticsHandler, plugin.SubscriptionOptions{
    Priority: 1,
})
```

### 3. Use Async for Non-Critical Handlers

```go
// ✅ Good: Async for monitoring/analytics
bus.Subscribe("phase.*", monitoringHandler, plugin.SubscriptionOptions{
    Async: true,
})

// ✅ Good: Sync for critical business logic
bus.Subscribe("phase.completed", businessHandler, plugin.SubscriptionOptions{
    Async: false,
})
```

### 4. Implement Proper Error Handling

```go
handler := func(ctx context.Context, event plugin.Event) error {
    // Always handle errors gracefully
    if err := processEvent(event); err != nil {
        logger.Error("Failed to process event", "error", err)
        return err // Will trigger retry if configured
    }
    return nil
}
```

### 5. Clean Up Subscriptions

```go
// Store subscription for cleanup
sub, err := bus.Subscribe("pattern.*", handler)
if err != nil {
    return err
}

// Clean up when done
defer func() {
    if err := bus.Unsubscribe(sub.ID); err != nil {
        logger.Error("Failed to unsubscribe", "error", err)
    }
}()
```

## Performance Considerations

- **Async Handlers**: Use for non-critical operations to avoid blocking
- **Pattern Specificity**: More specific patterns perform better than broad ones
- **Handler Efficiency**: Keep handlers lightweight for better throughput
- **Buffer Sizes**: Adjust buffer sizes based on event volume
- **Metrics**: Monitor metrics to identify performance bottlenecks

## Testing

The event bus includes comprehensive tests covering:

- Basic publish/subscribe functionality
- Pattern matching
- Async/sync handler execution
- Priority ordering
- Timeout handling
- Retry logic
- Panic recovery
- Concurrent publishing
- Metrics accuracy

Run tests with:
```bash
go test ./pkg/plugin -v
```

## Example Integration

See `example_integration.go` for a complete working example that demonstrates:

- Multiple plugins communicating via events
- Phase integration with automatic event publishing
- Real-time monitoring and analytics
- Custom event types and patterns
- Error handling and recovery

Run the example:
```go
plugin.RunExample()
```

This will output a detailed demonstration of the event bus system in action.