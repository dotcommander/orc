package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the current health state of a plugin
type HealthStatus string

const (
	// HealthStatusHealthy indicates the plugin is functioning normally
	HealthStatusHealthy HealthStatus = "healthy"
	
	// HealthStatusDegraded indicates the plugin is working but with reduced functionality
	HealthStatusDegraded HealthStatus = "degraded"
	
	// HealthStatusUnhealthy indicates the plugin is not functioning properly
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	
	// HealthStatusUnknown indicates health status cannot be determined
	HealthStatusUnknown HealthStatus = "unknown"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	// Name of the health check
	Name string `json:"name"`
	
	// Status of this specific check
	Status HealthStatus `json:"status"`
	
	// Message provides additional context
	Message string `json:"message,omitempty"`
	
	// Timestamp when the check was performed
	Timestamp time.Time `json:"timestamp"`
	
	// Duration how long the check took
	Duration time.Duration `json:"duration"`
	
	// Metadata for additional check-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthReport represents the overall health status of a plugin
type HealthReport struct {
	// Plugin name
	Plugin string `json:"plugin"`
	
	// Overall status (worst of all checks)
	Status HealthStatus `json:"status"`
	
	// Individual health checks
	Checks []HealthCheck `json:"checks"`
	
	// Timestamp of the report
	Timestamp time.Time `json:"timestamp"`
	
	// TotalDuration of all health checks
	TotalDuration time.Duration `json:"total_duration"`
	
	// Consecutive failures count
	ConsecutiveFailures int `json:"consecutive_failures,omitempty"`
	
	// Last successful check time
	LastSuccess *time.Time `json:"last_success,omitempty"`
}

// HealthCheckable interface for plugins that support health checks
type HealthCheckable interface {
	// HealthCheck performs health checks and returns a report
	HealthCheck(ctx context.Context) (*HealthReport, error)
	
	// GetHealthCheckInterval returns how often health checks should run
	GetHealthCheckInterval() time.Duration
	
	// IsHealthCheckCritical returns whether health check failures should block execution
	IsHealthCheckCritical() bool
}

// HealthChecker provides health check functionality
type HealthChecker interface {
	// CheckHealth performs a specific health check
	CheckHealth(ctx context.Context, name string) (*HealthCheck, error)
}

// HealthMonitor manages continuous health monitoring for plugins
type HealthMonitor struct {
	mu              sync.RWMutex
	plugins         map[string]HealthCheckable
	reports         map[string]*HealthReport
	stopChan        chan struct{}
	checkIntervals  map[string]time.Duration
	criticalPlugins map[string]bool
	callbacks       []HealthCallback
}

// HealthCallback is called when health status changes
type HealthCallback func(plugin string, oldStatus, newStatus HealthStatus, report *HealthReport)

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		plugins:         make(map[string]HealthCheckable),
		reports:         make(map[string]*HealthReport),
		stopChan:        make(chan struct{}),
		checkIntervals:  make(map[string]time.Duration),
		criticalPlugins: make(map[string]bool),
		callbacks:       make([]HealthCallback, 0),
	}
}

// RegisterPlugin registers a plugin for health monitoring
func (hm *HealthMonitor) RegisterPlugin(name string, plugin HealthCheckable) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if _, exists := hm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	hm.plugins[name] = plugin
	hm.checkIntervals[name] = plugin.GetHealthCheckInterval()
	hm.criticalPlugins[name] = plugin.IsHealthCheckCritical()
	
	return nil
}

// UnregisterPlugin removes a plugin from health monitoring
func (hm *HealthMonitor) UnregisterPlugin(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	delete(hm.plugins, name)
	delete(hm.reports, name)
	delete(hm.checkIntervals, name)
	delete(hm.criticalPlugins, name)
}

// Start begins continuous health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) {
	for name, plugin := range hm.plugins {
		go hm.monitorPlugin(ctx, name, plugin)
	}
}

// Stop halts health monitoring
func (hm *HealthMonitor) Stop() {
	close(hm.stopChan)
}

// monitorPlugin continuously monitors a single plugin
func (hm *HealthMonitor) monitorPlugin(ctx context.Context, name string, plugin HealthCheckable) {
	interval := hm.checkIntervals[name]
	if interval <= 0 {
		interval = 30 * time.Second // Default interval
	}
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Initial check
	hm.performHealthCheck(ctx, name, plugin)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopChan:
			return
		case <-ticker.C:
			hm.performHealthCheck(ctx, name, plugin)
		}
	}
}

// performHealthCheck executes a health check for a plugin
func (hm *HealthMonitor) performHealthCheck(ctx context.Context, name string, plugin HealthCheckable) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	report, err := plugin.HealthCheck(checkCtx)
	if err != nil {
		// Create error report
		report = &HealthReport{
			Plugin:    name,
			Status:    HealthStatusUnhealthy,
			Timestamp: time.Now(),
			Checks: []HealthCheck{
				{
					Name:      "health_check_error",
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("Health check failed: %v", err),
					Timestamp: time.Now(),
				},
			},
		}
	}
	
	hm.updateReport(name, report)
}

// updateReport updates the health report for a plugin
func (hm *HealthMonitor) updateReport(name string, report *HealthReport) {
	hm.mu.Lock()
	
	oldReport := hm.reports[name]
	var oldStatus HealthStatus
	if oldReport != nil {
		oldStatus = oldReport.Status
		
		// Update consecutive failures
		if report.Status == HealthStatusUnhealthy {
			report.ConsecutiveFailures = oldReport.ConsecutiveFailures + 1
		} else {
			report.ConsecutiveFailures = 0
			now := time.Now()
			report.LastSuccess = &now
		}
		
		// Preserve last success if not updated
		if report.LastSuccess == nil && oldReport.LastSuccess != nil {
			report.LastSuccess = oldReport.LastSuccess
		}
	} else {
		oldStatus = HealthStatusUnknown
		if report.Status != HealthStatusUnhealthy {
			now := time.Now()
			report.LastSuccess = &now
		}
	}
	
	hm.reports[name] = report
	callbacks := hm.callbacks
	
	hm.mu.Unlock()
	
	// Notify callbacks if status changed
	if oldStatus != report.Status {
		for _, callback := range callbacks {
			callback(name, oldStatus, report.Status, report)
		}
	}
}

// GetReport returns the latest health report for a plugin
func (hm *HealthMonitor) GetReport(name string) (*HealthReport, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	report, exists := hm.reports[name]
	return report, exists
}

// GetAllReports returns health reports for all plugins
func (hm *HealthMonitor) GetAllReports() map[string]*HealthReport {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	reports := make(map[string]*HealthReport)
	for name, report := range hm.reports {
		reports[name] = report
	}
	return reports
}

// IsHealthy checks if a plugin is healthy
func (hm *HealthMonitor) IsHealthy(name string) bool {
	report, exists := hm.GetReport(name)
	if !exists {
		return false
	}
	return report.Status == HealthStatusHealthy
}

// IsCriticalPlugin checks if a plugin is marked as critical
func (hm *HealthMonitor) IsCriticalPlugin(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	return hm.criticalPlugins[name]
}

// RegisterCallback adds a health status change callback
func (hm *HealthMonitor) RegisterCallback(callback HealthCallback) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.callbacks = append(hm.callbacks, callback)
}

// CheckNow forces an immediate health check for a plugin
func (hm *HealthMonitor) CheckNow(ctx context.Context, name string) (*HealthReport, error) {
	hm.mu.RLock()
	plugin, exists := hm.plugins[name]
	hm.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	report, err := plugin.HealthCheck(ctx)
	if err != nil {
		return nil, err
	}
	
	hm.updateReport(name, report)
	return report, nil
}

// CompositeHealthChecker combines multiple health checkers
type CompositeHealthChecker struct {
	checkers map[string]HealthChecker
}

// NewCompositeHealthChecker creates a new composite health checker
func NewCompositeHealthChecker() *CompositeHealthChecker {
	return &CompositeHealthChecker{
		checkers: make(map[string]HealthChecker),
	}
}

// AddChecker adds a health checker
func (c *CompositeHealthChecker) AddChecker(name string, checker HealthChecker) {
	c.checkers[name] = checker
}

// CheckAll performs all health checks
func (c *CompositeHealthChecker) CheckAll(ctx context.Context) (*HealthReport, error) {
	report := &HealthReport{
		Status:    HealthStatusHealthy,
		Checks:    make([]HealthCheck, 0, len(c.checkers)),
		Timestamp: time.Now(),
	}
	
	start := time.Now()
	
	for name, checker := range c.checkers {
		check, err := checker.CheckHealth(ctx, name)
		if err != nil {
			check = &HealthCheck{
				Name:      name,
				Status:    HealthStatusUnhealthy,
				Message:   fmt.Sprintf("Check failed: %v", err),
				Timestamp: time.Now(),
			}
		}
		
		report.Checks = append(report.Checks, *check)
		
		// Update overall status to worst
		if check.Status == HealthStatusUnhealthy {
			report.Status = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && report.Status != HealthStatusUnhealthy {
			report.Status = HealthStatusDegraded
		}
	}
	
	report.TotalDuration = time.Since(start)
	return report, nil
}

// HealthCheckFunc is a function adapter for simple health checks
type HealthCheckFunc func(ctx context.Context) error

// CheckHealth implements HealthChecker
func (f HealthCheckFunc) CheckHealth(ctx context.Context, name string) (*HealthCheck, error) {
	start := time.Now()
	err := f(ctx)
	
	check := &HealthCheck{
		Name:      name,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
	
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = err.Error()
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Check passed"
	}
	
	return check, nil
}