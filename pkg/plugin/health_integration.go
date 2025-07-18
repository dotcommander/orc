package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthAwarePluginRunner extends plugin execution with health monitoring
type HealthAwarePluginRunner struct {
	monitor          *HealthMonitor
	baseRunner       PluginRunner
	preCheckEnabled  bool
	blockOnUnhealthy bool
	checkTimeout     time.Duration
}

// PluginRunner interface for plugin execution (to be implemented by domain plugins)
type PluginRunner interface {
	Execute(ctx context.Context, pluginName, request string) error
}

// NewHealthAwarePluginRunner creates a runner with health monitoring
func NewHealthAwarePluginRunner(baseRunner PluginRunner, monitor *HealthMonitor) *HealthAwarePluginRunner {
	return &HealthAwarePluginRunner{
		monitor:          monitor,
		baseRunner:       baseRunner,
		preCheckEnabled:  true,
		blockOnUnhealthy: true,
		checkTimeout:     5 * time.Second,
	}
}

// SetPreCheckEnabled enables/disables pre-execution health checks
func (r *HealthAwarePluginRunner) SetPreCheckEnabled(enabled bool) {
	r.preCheckEnabled = enabled
}

// SetBlockOnUnhealthy sets whether to block execution on unhealthy plugins
func (r *HealthAwarePluginRunner) SetBlockOnUnhealthy(block bool) {
	r.blockOnUnhealthy = block
}

// Execute runs a plugin with health checks
func (r *HealthAwarePluginRunner) Execute(ctx context.Context, pluginName, request string) error {
	// Pre-execution health check
	if r.preCheckEnabled {
		if err := r.checkHealth(ctx, pluginName); err != nil {
			if r.blockOnUnhealthy && r.monitor.IsCriticalPlugin(pluginName) {
				return fmt.Errorf("plugin %s health check failed: %w", pluginName, err)
			}
			// Log warning but continue for non-critical plugins
			// TODO: Add proper logging
		}
	}
	
	// Execute the plugin
	executionErr := r.baseRunner.Execute(ctx, pluginName, request)
	
	// Post-execution health check (async)
	go func() {
		checkCtx, cancel := context.WithTimeout(context.Background(), r.checkTimeout)
		defer cancel()
		r.monitor.CheckNow(checkCtx, pluginName)
	}()
	
	return executionErr
}

// checkHealth performs a health check for a plugin
func (r *HealthAwarePluginRunner) checkHealth(ctx context.Context, pluginName string) error {
	report, err := r.monitor.CheckNow(ctx, pluginName)
	if err != nil {
		return fmt.Errorf("health check error: %w", err)
	}
	
	if report.Status == HealthStatusUnhealthy {
		return fmt.Errorf("plugin unhealthy: %s", report.Status)
	}
	
	if report.Status == HealthStatusDegraded && r.blockOnUnhealthy {
		// For degraded state, check if it's critical
		if r.monitor.IsCriticalPlugin(pluginName) {
			return fmt.Errorf("critical plugin degraded: %s", report.Status)
		}
	}
	
	return nil
}

// HealthMiddleware provides health check middleware for plugin execution
type HealthMiddleware struct {
	monitor *HealthMonitor
	config  HealthMiddlewareConfig
}

// HealthMiddlewareConfig configures health middleware behavior
type HealthMiddlewareConfig struct {
	// PreCheckEnabled enables pre-execution health checks
	PreCheckEnabled bool
	
	// PostCheckEnabled enables post-execution health checks
	PostCheckEnabled bool
	
	// BlockOnFailure blocks execution if health check fails
	BlockOnFailure bool
	
	// FailureThreshold number of consecutive failures before blocking
	FailureThreshold int
	
	// RecoveryThreshold number of successes needed to unblock
	RecoveryThreshold int
	
	// CheckTimeout timeout for health checks
	CheckTimeout time.Duration
}

// NewHealthMiddleware creates health check middleware
func NewHealthMiddleware(monitor *HealthMonitor, config HealthMiddlewareConfig) *HealthMiddleware {
	if config.CheckTimeout <= 0 {
		config.CheckTimeout = 5 * time.Second
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 3
	}
	if config.RecoveryThreshold <= 0 {
		config.RecoveryThreshold = 2
	}
	
	return &HealthMiddleware{
		monitor: monitor,
		config:  config,
	}
}

// Wrap wraps a plugin execution function with health checks
func (m *HealthMiddleware) Wrap(pluginName string, execFunc func(ctx context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		// Pre-execution check
		if m.config.PreCheckEnabled {
			if err := m.preCheck(ctx, pluginName); err != nil {
				return err
			}
		}
		
		// Execute the function
		execErr := execFunc(ctx)
		
		// Post-execution check (async if successful)
		if m.config.PostCheckEnabled {
			if execErr == nil {
				go m.postCheck(pluginName)
			} else {
				// Synchronous check on failure
				m.postCheck(pluginName)
			}
		}
		
		return execErr
	}
}

// preCheck performs pre-execution health check
func (m *HealthMiddleware) preCheck(ctx context.Context, pluginName string) error {
	if !m.config.BlockOnFailure {
		// Just check, don't block
		go func() {
			checkCtx, cancel := context.WithTimeout(context.Background(), m.config.CheckTimeout)
			defer cancel()
			m.monitor.CheckNow(checkCtx, pluginName)
		}()
		return nil
	}
	
	// Blocking check
	report, exists := m.monitor.GetReport(pluginName)
	if !exists {
		// No previous report, do a check
		checkCtx, cancel := context.WithTimeout(ctx, m.config.CheckTimeout)
		defer cancel()
		
		var err error
		report, err = m.monitor.CheckNow(checkCtx, pluginName)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
	}
	
	// Check consecutive failures
	if report.ConsecutiveFailures >= m.config.FailureThreshold {
		return fmt.Errorf("plugin %s blocked: %d consecutive failures", pluginName, report.ConsecutiveFailures)
	}
	
	return nil
}

// postCheck performs post-execution health check
func (m *HealthMiddleware) postCheck(pluginName string) {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.CheckTimeout)
	defer cancel()
	
	m.monitor.CheckNow(ctx, pluginName)
}

// HealthAwarePluginManager manages plugins with integrated health monitoring
type HealthAwarePluginManager struct {
	mu       sync.RWMutex
	plugins  map[string]interface{} // Can be any plugin type
	monitor  *HealthMonitor
	started  bool
}

// NewHealthAwarePluginManager creates a new health-aware plugin manager
func NewHealthAwarePluginManager() *HealthAwarePluginManager {
	return &HealthAwarePluginManager{
		plugins: make(map[string]interface{}),
		monitor: NewHealthMonitor(),
	}
}

// RegisterPlugin registers a plugin with optional health check support
func (m *HealthAwarePluginManager) RegisterPlugin(name string, plugin interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	m.plugins[name] = plugin
	
	// If plugin supports health checks, register with monitor
	if healthCheckable, ok := plugin.(HealthCheckable); ok {
		if err := m.monitor.RegisterPlugin(name, healthCheckable); err != nil {
			return fmt.Errorf("failed to register health monitor: %w", err)
		}
	}
	
	return nil
}

// Start begins health monitoring for all registered plugins
func (m *HealthAwarePluginManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.started {
		return fmt.Errorf("manager already started")
	}
	
	// Register health status callbacks
	m.monitor.RegisterCallback(m.onHealthStatusChange)
	
	// Start monitoring
	m.monitor.Start(ctx)
	m.started = true
	
	return nil
}

// Stop halts health monitoring
func (m *HealthAwarePluginManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.started {
		return
	}
	
	m.monitor.Stop()
	m.started = false
}

// GetPlugin retrieves a plugin by name
func (m *HealthAwarePluginManager) GetPlugin(name string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin, nil
}

// GetHealthReport gets the health report for a plugin
func (m *HealthAwarePluginManager) GetHealthReport(name string) (*HealthReport, error) {
	report, exists := m.monitor.GetReport(name)
	if !exists {
		return nil, fmt.Errorf("no health report for plugin %s", name)
	}
	return report, nil
}

// GetAllHealthReports returns health reports for all plugins
func (m *HealthAwarePluginManager) GetAllHealthReports() map[string]*HealthReport {
	return m.monitor.GetAllReports()
}

// IsPluginHealthy checks if a plugin is healthy
func (m *HealthAwarePluginManager) IsPluginHealthy(name string) bool {
	return m.monitor.IsHealthy(name)
}

// onHealthStatusChange handles health status changes
func (m *HealthAwarePluginManager) onHealthStatusChange(plugin string, oldStatus, newStatus HealthStatus, report *HealthReport) {
	// Log status change
	// TODO: Add proper logging
	
	// Handle critical plugin failures
	if newStatus == HealthStatusUnhealthy && m.monitor.IsCriticalPlugin(plugin) {
		// Could trigger alerts, notifications, or recovery procedures
		// TODO: Implement alerting mechanism
	}
	
	// Handle recovery
	if oldStatus == HealthStatusUnhealthy && newStatus == HealthStatusHealthy {
		// Plugin recovered, could clear alerts
		// TODO: Implement recovery notifications
	}
}

// WaitForHealthy waits for a plugin to become healthy
func (m *HealthAwarePluginManager) WaitForHealthy(ctx context.Context, pluginName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if m.IsPluginHealthy(pluginName) {
				return nil
			}
			if time.Now().After(deadline) {
				report, _ := m.GetHealthReport(pluginName)
				if report != nil {
					return fmt.Errorf("plugin %s did not become healthy within %v, status: %s", 
						pluginName, timeout, report.Status)
				}
				return fmt.Errorf("plugin %s did not become healthy within %v", pluginName, timeout)
			}
		}
	}
}