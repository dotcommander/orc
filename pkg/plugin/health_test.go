package plugin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dotcommander/orc/pkg/plugin"
)

func TestHealthMonitor(t *testing.T) {
	// Create health monitor
	monitor := plugin.NewHealthMonitor()
	
	// Create example plugin
	examplePlugin := plugin.NewExampleHealthyPlugin("test-plugin")
	
	// Register plugin
	if err := monitor.RegisterPlugin("test-plugin", examplePlugin); err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}
	
	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	monitor.Start(ctx)
	
	// Wait for initial check
	time.Sleep(2 * time.Second)
	
	// Get health report
	report, exists := monitor.GetReport("test-plugin")
	if !exists {
		t.Fatal("No health report found")
	}
	
	t.Logf("Plugin status: %s", report.Status)
	t.Logf("Total checks: %d", len(report.Checks))
	
	for _, check := range report.Checks {
		t.Logf("  Check %s: %s - %s", check.Name, check.Status, check.Message)
	}
	
	// Force immediate check
	forcedReport, err := monitor.CheckNow(ctx, "test-plugin")
	if err != nil {
		t.Errorf("Failed to force check: %v", err)
	} else {
		t.Logf("Forced check status: %s", forcedReport.Status)
	}
}

func TestHealthMiddleware(t *testing.T) {
	// Create monitor
	monitor := plugin.NewHealthMonitor()
	
	// Register a test plugin
	testPlugin := &TestHealthPlugin{
		healthy: true,
	}
	monitor.RegisterPlugin("test", testPlugin)
	
	// Create middleware
	middleware := plugin.NewHealthMiddleware(monitor, plugin.HealthMiddlewareConfig{
		PreCheckEnabled:   true,
		PostCheckEnabled:  true,
		BlockOnFailure:    true,
		FailureThreshold:  3,
		RecoveryThreshold: 2,
		CheckTimeout:      2 * time.Second,
	})
	
	// Test execution counter
	execCount := 0
	
	// Wrap execution function
	wrappedFunc := middleware.Wrap("test", func(ctx context.Context) error {
		execCount++
		return nil
	})
	
	// Execute with healthy plugin
	ctx := context.Background()
	if err := wrappedFunc(ctx); err != nil {
		t.Errorf("Execution failed with healthy plugin: %v", err)
	}
	
	if execCount != 1 {
		t.Errorf("Expected execution count 1, got %d", execCount)
	}
	
	// Make plugin unhealthy
	testPlugin.healthy = false
	
	// Force multiple checks to hit failure threshold
	for i := 0; i < 3; i++ {
		monitor.CheckNow(ctx, "test")
		time.Sleep(100 * time.Millisecond)
	}
	
	// Try execution with unhealthy plugin
	if err := wrappedFunc(ctx); err == nil {
		t.Error("Expected execution to fail with unhealthy plugin after threshold")
	}
}

func TestHealthAwarePluginManager(t *testing.T) {
	// Create manager
	manager := plugin.NewHealthAwarePluginManager()
	
	// Register plugins
	healthyPlugin := plugin.NewExampleHealthyPlugin("healthy-plugin")
	if err := manager.RegisterPlugin("healthy-plugin", healthyPlugin); err != nil {
		t.Fatalf("Failed to register healthy plugin: %v", err)
	}
	
	unhealthyPlugin := &TestHealthPlugin{healthy: false}
	if err := manager.RegisterPlugin("unhealthy-plugin", unhealthyPlugin); err != nil {
		t.Fatalf("Failed to register unhealthy plugin: %v", err)
	}
	
	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop()
	
	// Wait for health checks
	time.Sleep(2 * time.Second)
	
	// Check plugin health
	if !manager.IsPluginHealthy("healthy-plugin") {
		t.Error("Expected healthy-plugin to be healthy")
	}
	
	if manager.IsPluginHealthy("unhealthy-plugin") {
		t.Error("Expected unhealthy-plugin to be unhealthy")
	}
	
	// Get all reports
	reports := manager.GetAllHealthReports()
	t.Logf("Total plugins with health reports: %d", len(reports))
	
	for name, report := range reports {
		t.Logf("Plugin %s: %s", name, report.Status)
	}
	
	// Test waiting for healthy with timeout
	waitCtx, waitCancel := context.WithTimeout(ctx, 3*time.Second)
	defer waitCancel()
	
	err := manager.WaitForHealthy(waitCtx, "unhealthy-plugin", 3*time.Second)
	if err == nil {
		t.Error("Expected timeout waiting for unhealthy plugin")
	}
}

func TestCompositeHealthChecker(t *testing.T) {
	checker := plugin.NewCompositeHealthChecker()
	
	// Add various health checkers
	checker.AddChecker("api", plugin.HealthCheckFunc(func(ctx context.Context) error {
		// Simulate successful API check
		return nil
	}))
	
	checker.AddChecker("database", plugin.HealthCheckFunc(func(ctx context.Context) error {
		// Simulate database error
		return fmt.Errorf("connection refused")
	}))
	
	checker.AddChecker("cache", plugin.HealthCheckFunc(func(ctx context.Context) error {
		// Simulate successful cache check
		return nil
	}))
	
	// Perform all checks
	ctx := context.Background()
	report, err := checker.CheckAll(ctx)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}
	
	// Verify report
	if report.Status != plugin.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %s", report.Status)
	}
	
	if len(report.Checks) != 3 {
		t.Errorf("Expected 3 checks, got %d", len(report.Checks))
	}
	
	// Verify individual checks
	for _, check := range report.Checks {
		t.Logf("Check %s: %s - %s", check.Name, check.Status, check.Message)
		
		switch check.Name {
		case "api", "cache":
			if check.Status != plugin.HealthStatusHealthy {
				t.Errorf("Expected %s to be healthy", check.Name)
			}
		case "database":
			if check.Status != plugin.HealthStatusUnhealthy {
				t.Errorf("Expected database to be unhealthy")
			}
		}
	}
}

func TestSpecializedHealthCheckers(t *testing.T) {
	ctx := context.Background()
	
	t.Run("HTTPHealthChecker", func(t *testing.T) {
		checker := &plugin.HTTPHealthChecker{
			URL:     "https://httpbin.org/status/200",
			Timeout: 5 * time.Second,
			Headers: map[string]string{
				"User-Agent": "HealthCheck/1.0",
			},
		}
		
		check, err := checker.CheckHealth(ctx, "http_test")
		if err != nil {
			t.Errorf("HTTP health check failed: %v", err)
		}
		
		if check.Status != plugin.HealthStatusHealthy {
			t.Errorf("Expected healthy status, got %s", check.Status)
		}
		
		t.Logf("HTTP check: %s - %s (took %v)", check.Status, check.Message, check.Duration)
	})
	
	t.Run("ThresholdHealthChecker", func(t *testing.T) {
		// Test with value getter
		currentValue := 50.0
		checker := &plugin.ThresholdHealthChecker{
			GetValue: func(ctx context.Context) (float64, error) {
				return currentValue, nil
			},
			HealthyMax:  75.0,
			DegradedMax: 90.0,
			Unit:        "percent",
		}
		
		// Test healthy state
		check, _ := checker.CheckHealth(ctx, "threshold_test")
		if check.Status != plugin.HealthStatusHealthy {
			t.Errorf("Expected healthy at 50%%, got %s", check.Status)
		}
		
		// Test degraded state
		currentValue = 80.0
		check, _ = checker.CheckHealth(ctx, "threshold_test")
		if check.Status != plugin.HealthStatusDegraded {
			t.Errorf("Expected degraded at 80%%, got %s", check.Status)
		}
		
		// Test unhealthy state
		currentValue = 95.0
		check, _ = checker.CheckHealth(ctx, "threshold_test")
		if check.Status != plugin.HealthStatusUnhealthy {
			t.Errorf("Expected unhealthy at 95%%, got %s", check.Status)
		}
	})
	
	t.Run("DependencyHealthChecker", func(t *testing.T) {
		checker := &plugin.DependencyHealthChecker{
			Dependencies: map[string]func(ctx context.Context) error{
				"service1": func(ctx context.Context) error { return nil },
				"service2": func(ctx context.Context) error { return fmt.Errorf("timeout") },
				"service3": func(ctx context.Context) error { return nil },
			},
		}
		
		check, _ := checker.CheckHealth(ctx, "dependency_test")
		if check.Status != plugin.HealthStatusDegraded {
			t.Errorf("Expected degraded with one failed dependency, got %s", check.Status)
		}
		
		t.Logf("Dependency check: %s - %s", check.Status, check.Message)
		for dep, status := range check.Metadata {
			t.Logf("  %s: %v", dep, status)
		}
	})
}

// TestHealthPlugin is a simple plugin for testing
type TestHealthPlugin struct {
	healthy bool
}

func (p *TestHealthPlugin) HealthCheck(ctx context.Context) (*plugin.HealthReport, error) {
	status := plugin.HealthStatusHealthy
	if !p.healthy {
		status = plugin.HealthStatusUnhealthy
	}
	
	return &plugin.HealthReport{
		Plugin: "test",
		Status: status,
		Checks: []plugin.HealthCheck{
			{
				Name:      "test_check",
				Status:    status,
				Message:   fmt.Sprintf("Test plugin is %s", status),
				Timestamp: time.Now(),
			},
		},
		Timestamp: time.Now(),
	}, nil
}

func (p *TestHealthPlugin) GetHealthCheckInterval() time.Duration {
	return 10 * time.Second
}

func (p *TestHealthPlugin) IsHealthCheckCritical() bool {
	return false
}