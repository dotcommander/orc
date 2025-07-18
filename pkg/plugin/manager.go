package plugin

import (
	"fmt"
	"sync"
	"time"
)

// ContextManager manages plugin contexts across sessions
type ContextManager struct {
	mu       sync.RWMutex
	contexts map[string]PluginContext
	ttl      time.Duration
	cleanupInterval time.Duration
	stopCleanup chan struct{}
	wg       sync.WaitGroup
}

// ContextManagerOption configures the ContextManager
type ContextManagerOption func(*ContextManager)

// WithTTL sets the time-to-live for contexts
func WithTTL(ttl time.Duration) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.ttl = ttl
	}
}

// WithCleanupInterval sets how often to clean up expired contexts
func WithCleanupInterval(interval time.Duration) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.cleanupInterval = interval
	}
}

// NewContextManager creates a new context manager
func NewContextManager(opts ...ContextManagerOption) *ContextManager {
	cm := &ContextManager{
		contexts:        make(map[string]PluginContext),
		ttl:             24 * time.Hour, // Default TTL
		cleanupInterval: 1 * time.Hour,  // Default cleanup interval
		stopCleanup:     make(chan struct{}),
	}
	
	for _, opt := range opts {
		opt(cm)
	}
	
	// Start cleanup goroutine
	cm.wg.Add(1)
	go cm.cleanupRoutine()
	
	return cm
}

// CreateContext creates a new plugin context for a session
func (cm *ContextManager) CreateContext(sessionID string) PluginContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	ctx := NewPluginContext()
	cm.contexts[sessionID] = ctx
	
	// Store creation time for TTL management
	ctx.Set("__created_at", time.Now())
	ctx.Set("__session_id", sessionID)
	
	return ctx
}

// GetContext retrieves a context by session ID
func (cm *ContextManager) GetContext(sessionID string) (PluginContext, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	ctx, exists := cm.contexts[sessionID]
	if exists {
		// Update last access time
		ctx.Set("__last_access", time.Now())
	}
	return ctx, exists
}

// DeleteContext removes a context
func (cm *ContextManager) DeleteContext(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	delete(cm.contexts, sessionID)
}

// ListSessions returns all active session IDs
func (cm *ContextManager) ListSessions() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	sessions := make([]string, 0, len(cm.contexts))
	for sessionID := range cm.contexts {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// cleanupRoutine periodically removes expired contexts
func (cm *ContextManager) cleanupRoutine() {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.cleanup()
		case <-cm.stopCleanup:
			return
		}
	}
}

// cleanup removes expired contexts
func (cm *ContextManager) cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	now := time.Now()
	expiredSessions := []string{}
	
	for sessionID, ctx := range cm.contexts {
		// Check creation time
		if createdAt, err := getTimeFromContext(ctx, "__created_at"); err == nil {
			if now.Sub(createdAt) > cm.ttl {
				expiredSessions = append(expiredSessions, sessionID)
			}
		}
	}
	
	// Remove expired sessions
	for _, sessionID := range expiredSessions {
		delete(cm.contexts, sessionID)
	}
}

// Stop gracefully shuts down the context manager
func (cm *ContextManager) Stop() {
	close(cm.stopCleanup)
	cm.wg.Wait()
}

// getTimeFromContext retrieves a time value from the context
func getTimeFromContext(ctx PluginContext, key string) (time.Time, error) {
	value, exists := ctx.Get(key)
	if !exists {
		return time.Time{}, fmt.Errorf("key %s not found", key)
	}
	
	t, ok := value.(time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("value for key %s is not a time.Time", key)
	}
	
	return t, nil
}

// SharedData represents data shared between phases
type SharedData struct {
	// Phase-specific outputs indexed by phase name
	PhaseOutputs map[string]interface{} `json:"phase_outputs"`
	
	// Global metadata available to all phases
	Metadata map[string]interface{} `json:"metadata"`
	
	// Accumulated errors for debugging
	Errors []PhaseError `json:"errors,omitempty"`
	
	// Performance metrics
	Metrics PhaseMetrics `json:"metrics"`
}

// PhaseError represents an error that occurred during phase execution
type PhaseError struct {
	Phase     string    `json:"phase"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

// PhaseMetrics tracks performance metrics across phases
type PhaseMetrics struct {
	PhaseDurations map[string]time.Duration `json:"phase_durations"`
	TotalDuration  time.Duration            `json:"total_duration"`
	StartTime      time.Time                `json:"start_time"`
	EndTime        time.Time                `json:"end_time"`
}

// NewSharedData creates a new SharedData instance
func NewSharedData() *SharedData {
	return &SharedData{
		PhaseOutputs: make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		Errors:       []PhaseError{},
		Metrics: PhaseMetrics{
			PhaseDurations: make(map[string]time.Duration),
			StartTime:      time.Now(),
		},
	}
}

// SetPhaseOutput stores the output from a phase
func (sd *SharedData) SetPhaseOutput(phaseName string, output interface{}) {
	sd.PhaseOutputs[phaseName] = output
}

// GetPhaseOutput retrieves the output from a specific phase
func (sd *SharedData) GetPhaseOutput(phaseName string) (interface{}, bool) {
	output, exists := sd.PhaseOutputs[phaseName]
	return output, exists
}

// AddError records an error that occurred during phase execution
func (sd *SharedData) AddError(phaseName string, err error, retryable bool) {
	sd.Errors = append(sd.Errors, PhaseError{
		Phase:     phaseName,
		Error:     err.Error(),
		Timestamp: time.Now(),
		Retryable: retryable,
	})
}

// RecordPhaseDuration records how long a phase took to execute
func (sd *SharedData) RecordPhaseDuration(phaseName string, duration time.Duration) {
	sd.Metrics.PhaseDurations[phaseName] = duration
}

// Finalize marks the shared data as complete
func (sd *SharedData) Finalize() {
	sd.Metrics.EndTime = time.Now()
	sd.Metrics.TotalDuration = sd.Metrics.EndTime.Sub(sd.Metrics.StartTime)
}