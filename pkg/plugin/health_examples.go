package plugin

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// ExampleHealthyPlugin demonstrates a plugin with comprehensive health checks
type ExampleHealthyPlugin struct {
	name        string
	apiEndpoint string
	dbConnStr   string
	lastAPICall time.Time
	callCount   int
}

// NewExampleHealthyPlugin creates a new example plugin
func NewExampleHealthyPlugin(name string) *ExampleHealthyPlugin {
	return &ExampleHealthyPlugin{
		name:        name,
		apiEndpoint: "https://api.example.com/health",
		dbConnStr:   "postgresql://localhost/example",
	}
}

// HealthCheck implements HealthCheckable
func (p *ExampleHealthyPlugin) HealthCheck(ctx context.Context) (*HealthReport, error) {
	checker := NewCompositeHealthChecker()
	
	// Add API connectivity check
	checker.AddChecker("api_connectivity", HealthCheckFunc(func(ctx context.Context) error {
		// Simulate API check
		if rand.Float32() < 0.95 { // 95% success rate
			p.lastAPICall = time.Now()
			return nil
		}
		return fmt.Errorf("API endpoint unreachable")
	}))
	
	// Add database connectivity check
	checker.AddChecker("database_connectivity", HealthCheckFunc(func(ctx context.Context) error {
		// Simulate DB check
		if rand.Float32() < 0.98 { // 98% success rate
			return nil
		}
		return fmt.Errorf("database connection failed")
	}))
	
	// Add resource usage check
	checker.AddChecker("resource_usage", &ResourceHealthChecker{
		MaxMemoryMB: 500,
		MaxCPU:      80.0,
	})
	
	// Add rate limit check
	checker.AddChecker("rate_limits", HealthCheckFunc(func(ctx context.Context) error {
		if p.callCount > 1000 {
			return fmt.Errorf("approaching rate limit: %d calls", p.callCount)
		}
		return nil
	}))
	
	report, err := checker.CheckAll(ctx)
	if err != nil {
		return nil, err
	}
	
	report.Plugin = p.name
	return report, nil
}

// GetHealthCheckInterval implements HealthCheckable
func (p *ExampleHealthyPlugin) GetHealthCheckInterval() time.Duration {
	return 30 * time.Second
}

// IsHealthCheckCritical implements HealthCheckable
func (p *ExampleHealthyPlugin) IsHealthCheckCritical() bool {
	return true // This plugin is critical for system operation
}

// ResourceHealthChecker checks system resource usage
type ResourceHealthChecker struct {
	MaxMemoryMB int
	MaxCPU      float64
}

// CheckHealth implements HealthChecker
func (r *ResourceHealthChecker) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	
	// Simulate resource checks
	memoryUsage := rand.Intn(600) // Random memory usage in MB
	cpuUsage := rand.Float64() * 100 // Random CPU usage percentage
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Metadata: map[string]interface{}{
			"memory_mb": memoryUsage,
			"cpu_percent": cpuUsage,
		},
	}
	
	if memoryUsage > r.MaxMemoryMB {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Memory usage too high: %d MB", memoryUsage)
	} else if cpuUsage > r.MaxCPU {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("CPU usage high: %.1f%%", cpuUsage)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Resource usage within limits"
	}
	
	return check, nil
}

// HTTPHealthChecker performs HTTP endpoint health checks
type HTTPHealthChecker struct {
	URL     string
	Timeout time.Duration
	Headers map[string]string
}

// CheckHealth implements HealthChecker
func (h *HTTPHealthChecker) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
	}
	
	client := &http.Client{
		Timeout: h.Timeout,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", h.URL, nil)
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Failed to create request: %v", err)
		check.Duration = time.Since(start)
		return check, nil
	}
	
	// Add headers if provided
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}
	
	resp, err := client.Do(req)
	check.Duration = time.Since(start)
	
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Request failed: %v", err)
		return check, nil
	}
	defer resp.Body.Close()
	
	check.Metadata = map[string]interface{}{
		"status_code": resp.StatusCode,
		"response_time_ms": check.Duration.Milliseconds(),
	}
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Endpoint healthy (status: %d)", resp.StatusCode)
	} else if resp.StatusCode >= 500 {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Endpoint error (status: %d)", resp.StatusCode)
	} else {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Endpoint degraded (status: %d)", resp.StatusCode)
	}
	
	return check, nil
}

// ThresholdHealthChecker checks values against thresholds
type ThresholdHealthChecker struct {
	GetValue      func(ctx context.Context) (float64, error)
	HealthyMax    float64
	DegradedMax   float64
	Unit          string
}

// CheckHealth implements HealthChecker
func (t *ThresholdHealthChecker) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	
	value, err := t.GetValue(ctx)
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Metadata: map[string]interface{}{
			"value": value,
			"unit":  t.Unit,
		},
	}
	
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Failed to get value: %v", err)
		return check, nil
	}
	
	if value <= t.HealthyMax {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Value within healthy range: %.2f %s", value, t.Unit)
	} else if value <= t.DegradedMax {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Value in degraded range: %.2f %s", value, t.Unit)
	} else {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Value exceeds limits: %.2f %s", value, t.Unit)
	}
	
	return check, nil
}

// DependencyHealthChecker checks health of dependencies
type DependencyHealthChecker struct {
	Dependencies map[string]func(ctx context.Context) error
}

// CheckHealth implements HealthChecker
func (d *DependencyHealthChecker) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	failedDeps := []string{}
	
	for depName, checkFunc := range d.Dependencies {
		if err := checkFunc(ctx); err != nil {
			failedDeps = append(failedDeps, depName)
			check.Metadata[depName] = fmt.Sprintf("failed: %v", err)
		} else {
			check.Metadata[depName] = "healthy"
		}
	}
	
	check.Duration = time.Since(start)
	
	if len(failedDeps) == 0 {
		check.Status = HealthStatusHealthy
		check.Message = "All dependencies healthy"
	} else if len(failedDeps) < len(d.Dependencies) {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Some dependencies unhealthy: %v", failedDeps)
	} else {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("All dependencies failed: %v", failedDeps)
	}
	
	return check, nil
}

// CircuitBreakerHealthChecker monitors circuit breaker state
type CircuitBreakerHealthChecker struct {
	GetState      func() string
	GetErrorRate  func() float64
	GetTotalCalls func() int64
}

// CheckHealth implements HealthChecker
func (c *CircuitBreakerHealthChecker) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	
	state := c.GetState()
	errorRate := c.GetErrorRate()
	totalCalls := c.GetTotalCalls()
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Metadata: map[string]interface{}{
			"state":       state,
			"error_rate":  errorRate,
			"total_calls": totalCalls,
		},
	}
	
	switch state {
	case "closed":
		if errorRate < 0.1 { // Less than 10% error rate
			check.Status = HealthStatusHealthy
			check.Message = fmt.Sprintf("Circuit closed, error rate: %.1f%%", errorRate*100)
		} else {
			check.Status = HealthStatusDegraded
			check.Message = fmt.Sprintf("Circuit closed but high error rate: %.1f%%", errorRate*100)
		}
	case "open":
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Circuit open, error rate: %.1f%%", errorRate*100)
	case "half-open":
		check.Status = HealthStatusDegraded
		check.Message = "Circuit half-open, testing recovery"
	default:
		check.Status = HealthStatusUnknown
		check.Message = fmt.Sprintf("Unknown circuit state: %s", state)
	}
	
	return check, nil
}