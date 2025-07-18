# Plugin Health Check System

The plugin health check system provides comprehensive health monitoring capabilities for orchestrator plugins. It allows plugins to report their health status and enables the system to monitor plugin health continuously.

## Core Components

### 1. Health Status Types

```go
- HealthStatusHealthy    // Plugin functioning normally
- HealthStatusDegraded   // Working with reduced functionality  
- HealthStatusUnhealthy  // Not functioning properly
- HealthStatusUnknown    // Status cannot be determined
```

### 2. Health Check Interfaces

**HealthCheckable** - For plugins that support health checks:
```go
type HealthCheckable interface {
    HealthCheck(ctx context.Context) (*HealthReport, error)
    GetHealthCheckInterval() time.Duration
    IsHealthCheckCritical() bool
}
```

**HealthChecker** - For individual health check implementations:
```go
type HealthChecker interface {
    CheckHealth(ctx context.Context, name string) (*HealthCheck, error)
}
```

### 3. Health Monitor

The `HealthMonitor` manages continuous health monitoring for all registered plugins:

```go
monitor := NewHealthMonitor()

// Register plugins
monitor.RegisterPlugin("fiction", fictionPlugin)
monitor.RegisterPlugin("code", codePlugin)

// Start monitoring
monitor.Start(ctx)

// Check health status
if monitor.IsHealthy("fiction") {
    // Plugin is healthy
}

// Get detailed report
report, _ := monitor.GetReport("fiction")
```

## Implementation Examples

### 1. Basic Plugin with Health Checks

```go
type MyPlugin struct {
    name string
    db   *sql.DB
    api  *APIClient
}

func (p *MyPlugin) HealthCheck(ctx context.Context) (*HealthReport, error) {
    checker := NewCompositeHealthChecker()
    
    // Add database check
    checker.AddChecker("database", HealthCheckFunc(func(ctx context.Context) error {
        return p.db.PingContext(ctx)
    }))
    
    // Add API check
    checker.AddChecker("api", HealthCheckFunc(func(ctx context.Context) error {
        return p.api.HealthCheck(ctx)
    }))
    
    report, err := checker.CheckAll(ctx)
    if err != nil {
        return nil, err
    }
    
    report.Plugin = p.name
    return report, nil
}

func (p *MyPlugin) GetHealthCheckInterval() time.Duration {
    return 30 * time.Second
}

func (p *MyPlugin) IsHealthCheckCritical() bool {
    return true // This plugin is critical
}
```

### 2. Using Specialized Health Checkers

```go
// HTTP endpoint health check
httpChecker := &HTTPHealthChecker{
    URL:     "https://api.example.com/health",
    Timeout: 5 * time.Second,
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
}

// Threshold-based health check
memoryChecker := &ThresholdHealthChecker{
    GetValue: func(ctx context.Context) (float64, error) {
        return getMemoryUsagePercent(), nil
    },
    HealthyMax:  70.0,  // Healthy if <= 70%
    DegradedMax: 85.0,  // Degraded if <= 85%
    Unit:        "percent",
}

// Dependency health check
depChecker := &DependencyHealthChecker{
    Dependencies: map[string]func(ctx context.Context) error{
        "redis":    checkRedis,
        "postgres": checkPostgres,
        "kafka":    checkKafka,
    },
}
```

### 3. Health-Aware Plugin Execution

```go
// Create health-aware runner
runner := NewHealthAwarePluginRunner(baseRunner, monitor)
runner.SetBlockOnUnhealthy(true) // Block execution if unhealthy

// Execute with health checks
err := runner.Execute(ctx, "fiction", "Write a story")
```

### 4. Health Middleware

```go
// Configure middleware
middleware := NewHealthMiddleware(monitor, HealthMiddlewareConfig{
    PreCheckEnabled:   true,
    PostCheckEnabled:  true,
    BlockOnFailure:    true,
    FailureThreshold:  3,    // Block after 3 failures
    RecoveryThreshold: 2,    // Unblock after 2 successes
    CheckTimeout:      5 * time.Second,
})

// Wrap execution function
wrappedFunc := middleware.Wrap("myPlugin", func(ctx context.Context) error {
    // Your plugin execution logic
    return executePlugin(ctx)
})

// Execute with health checks
err := wrappedFunc(ctx)
```

## Health Check Best Practices

### 1. Comprehensive Checks
Include checks for all critical dependencies:
- Database connections
- API endpoints
- Cache systems
- Message queues
- File system access
- Resource usage (CPU, memory, disk)

### 2. Meaningful Status Levels
- **Healthy**: All systems operational
- **Degraded**: Non-critical features unavailable
- **Unhealthy**: Critical features failing

### 3. Fast Health Checks
- Keep individual checks under 5 seconds
- Use timeouts for all external calls
- Run expensive checks less frequently

### 4. Informative Messages
```go
check := &HealthCheck{
    Name:    "database",
    Status:  HealthStatusDegraded,
    Message: "Connection pool exhausted: 95/100 connections in use",
    Metadata: map[string]interface{}{
        "connections_used": 95,
        "connections_max":  100,
        "avg_query_time":   "250ms",
    },
}
```

### 5. Circuit Breaker Integration
```go
circuitChecker := &CircuitBreakerHealthChecker{
    GetState:      func() string { return breaker.State() },
    GetErrorRate:  func() float64 { return breaker.ErrorRate() },
    GetTotalCalls: func() int64 { return breaker.TotalCalls() },
}
```

## Health Monitoring Integration

### 1. With Plugin Manager

```go
manager := NewHealthAwarePluginManager()

// Register plugins
manager.RegisterPlugin("fiction", fictionPlugin)
manager.RegisterPlugin("code", codePlugin)

// Start health monitoring
manager.Start(ctx)

// Wait for plugin to be healthy
err := manager.WaitForHealthy(ctx, "fiction", 30*time.Second)

// Get health reports
reports := manager.GetAllHealthReports()
```

### 2. Health Status Callbacks

```go
monitor.RegisterCallback(func(plugin string, oldStatus, newStatus HealthStatus, report *HealthReport) {
    if newStatus == HealthStatusUnhealthy {
        // Send alert
        alerting.SendAlert(fmt.Sprintf("Plugin %s is unhealthy", plugin))
    }
    
    if oldStatus == HealthStatusUnhealthy && newStatus == HealthStatusHealthy {
        // Plugin recovered
        alerting.ClearAlert(plugin)
    }
})
```

### 3. Metrics Export

```go
// Export health metrics for monitoring systems
for name, report := range monitor.GetAllReports() {
    metrics.SetGauge("plugin_health_status", statusToFloat(report.Status), 
        map[string]string{"plugin": name})
    
    metrics.SetGauge("plugin_health_consecutive_failures", 
        float64(report.ConsecutiveFailures),
        map[string]string{"plugin": name})
}
```

## Testing Health Checks

```go
func TestPluginHealth(t *testing.T) {
    plugin := NewMyPlugin()
    
    // Test healthy state
    report, err := plugin.HealthCheck(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, HealthStatusHealthy, report.Status)
    
    // Simulate failure
    plugin.db.Close()
    
    report, err = plugin.HealthCheck(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, HealthStatusUnhealthy, report.Status)
}
```

## Configuration Example

```yaml
health:
  enabled: true
  default_interval: 30s
  default_timeout: 5s
  
  plugins:
    fiction:
      interval: 1m
      critical: true
      checks:
        - name: api
          timeout: 3s
        - name: database
          timeout: 2s
    
    code:
      interval: 30s
      critical: false
      checks:
        - name: compiler
          timeout: 10s
```

## Summary

The plugin health check system provides:

1. **Continuous Monitoring** - Automatic health checks at configurable intervals
2. **Flexible Checkers** - Built-in checkers for common scenarios
3. **Integration Points** - Middleware and runners for health-aware execution
4. **Status Tracking** - Consecutive failures, recovery detection
5. **Extensibility** - Easy to add custom health checks
6. **Production Ready** - Timeouts, circuit breakers, alerting support

This enables robust plugin lifecycle management with automatic failure detection and recovery capabilities.