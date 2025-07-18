package plugin

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vampirenirmal/orchestrator/internal/domain"
)

// Capability represents a permission that a plugin can request
type Capability string

const (
	// Core capabilities
	CapabilityAI        Capability = "ai"        // Access to AI agent
	CapabilityStorage   Capability = "storage"   // File storage access
	CapabilityNetwork   Capability = "network"   // Network access
	CapabilityExec      Capability = "exec"      // Execute commands
	CapabilityEnv       Capability = "env"       // Environment variables
	CapabilityFileRead  Capability = "file:read" // Read file system
	CapabilityFileWrite Capability = "file:write" // Write file system
	
	// Advanced capabilities
	CapabilityPluginComm Capability = "plugin:comm" // Inter-plugin communication
	CapabilityMetrics    Capability = "metrics"     // Metrics collection
	CapabilityLogs       Capability = "logs"        // Log access
	CapabilityConfig     Capability = "config"      // Configuration access
)

// SecurityPolicy defines security constraints for a plugin
type SecurityPolicy struct {
	// Capabilities granted to the plugin
	Capabilities map[Capability]bool
	
	// File system restrictions
	AllowedReadPaths  []string // Paths the plugin can read from
	AllowedWritePaths []string // Paths the plugin can write to
	
	// Network restrictions
	AllowedHosts []string // Hosts the plugin can connect to
	AllowedPorts []int    // Ports the plugin can use
	
	// Resource limits
	MaxMemory     int64 // Maximum memory in bytes
	MaxCPU        int   // Maximum CPU percentage
	MaxGoroutines int   // Maximum concurrent goroutines
	MaxOpenFiles  int   // Maximum open file descriptors
	
	// API restrictions
	MaxAPICallsPerMinute int
	MaxDataSize          int64 // Maximum data size per operation
}

// DefaultSecurityPolicy returns a restrictive default policy
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		Capabilities: map[Capability]bool{
			CapabilityAI:      true,
			CapabilityStorage: true,
			// Other capabilities must be explicitly granted
		},
		AllowedReadPaths:     []string{}, // No file access by default
		AllowedWritePaths:    []string{},
		AllowedHosts:         []string{}, // No network access by default
		AllowedPorts:         []int{},
		MaxMemory:            512 * 1024 * 1024, // 512MB
		MaxCPU:               50,                // 50% CPU
		MaxGoroutines:        100,
		MaxOpenFiles:         50,
		MaxAPICallsPerMinute: 60,
		MaxDataSize:          10 * 1024 * 1024, // 10MB
	}
}

// SecurityManager enforces security policies for plugins
type SecurityManager struct {
	policies map[string]SecurityPolicy
	monitors map[string]*ResourceMonitor
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(logger *slog.Logger) *SecurityManager {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &SecurityManager{
		policies: make(map[string]SecurityPolicy),
		monitors: make(map[string]*ResourceMonitor),
		logger:   logger,
	}
}

// SetPolicy sets the security policy for a plugin
func (sm *SecurityManager) SetPolicy(pluginName string, policy SecurityPolicy) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.policies[pluginName] = policy
	sm.logger.Info("security policy set",
		"plugin", pluginName,
		"capabilities", len(policy.Capabilities))
}

// CheckCapability verifies if a plugin has a specific capability
func (sm *SecurityManager) CheckCapability(pluginName string, capability Capability) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	policy, exists := sm.policies[pluginName]
	if !exists {
		return fmt.Errorf("no security policy for plugin %s", pluginName)
	}
	
	if granted, ok := policy.Capabilities[capability]; !ok || !granted {
		return fmt.Errorf("plugin %s lacks capability: %s", pluginName, capability)
	}
	
	return nil
}

// CheckFileAccess verifies if a plugin can access a file
func (sm *SecurityManager) CheckFileAccess(pluginName string, path string, write bool) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	policy, exists := sm.policies[pluginName]
	if !exists {
		return fmt.Errorf("no security policy for plugin %s", pluginName)
	}
	
	// Check capability first
	capability := CapabilityFileRead
	if write {
		capability = CapabilityFileWrite
	}
	
	if granted, ok := policy.Capabilities[capability]; !ok || !granted {
		return fmt.Errorf("plugin %s lacks capability: %s", pluginName, capability)
	}
	
	// Check path restrictions
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	allowedPaths := policy.AllowedReadPaths
	if write {
		allowedPaths = policy.AllowedWritePaths
	}
	
	for _, allowed := range allowedPaths {
		absAllowed, _ := filepath.Abs(allowed)
		if strings.HasPrefix(absPath, absAllowed) {
			return nil
		}
	}
	
	action := "read from"
	if write {
		action = "write to"
	}
	return fmt.Errorf("plugin %s not allowed to %s path: %s", pluginName, action, path)
}

// CheckNetworkAccess verifies if a plugin can make a network connection
func (sm *SecurityManager) CheckNetworkAccess(pluginName string, host string, port int) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	policy, exists := sm.policies[pluginName]
	if !exists {
		return fmt.Errorf("no security policy for plugin %s", pluginName)
	}
	
	// Check capability
	if granted, ok := policy.Capabilities[CapabilityNetwork]; !ok || !granted {
		return fmt.Errorf("plugin %s lacks network capability", pluginName)
	}
	
	// Check host restrictions
	hostAllowed := len(policy.AllowedHosts) == 0 // Empty means all hosts
	for _, allowed := range policy.AllowedHosts {
		if allowed == "*" || allowed == host {
			hostAllowed = true
			break
		}
		// Check wildcard domains
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(host, domain) {
				hostAllowed = true
				break
			}
		}
	}
	
	if !hostAllowed {
		return fmt.Errorf("plugin %s not allowed to connect to host: %s", pluginName, host)
	}
	
	// Check port restrictions
	if len(policy.AllowedPorts) > 0 {
		portAllowed := false
		for _, allowed := range policy.AllowedPorts {
			if allowed == port {
				portAllowed = true
				break
			}
		}
		if !portAllowed {
			return fmt.Errorf("plugin %s not allowed to use port: %d", pluginName, port)
		}
	}
	
	return nil
}

// SecurePlugin wraps a plugin with security enforcement
type SecurePlugin struct {
	plugin          domain.Plugin
	securityManager *SecurityManager
	logger          *slog.Logger
}

// NewSecurePlugin creates a security-enforcing plugin wrapper
func NewSecurePlugin(plugin domain.Plugin, sm *SecurityManager, logger *slog.Logger) *SecurePlugin {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &SecurePlugin{
		plugin:          plugin,
		securityManager: sm,
		logger:          logger,
	}
}

// Name implements domain.Plugin
func (sp *SecurePlugin) Name() string {
	return sp.plugin.Name()
}

// Domain implements domain.Plugin
func (sp *SecurePlugin) Domain() string {
	return sp.plugin.Domain()
}

// GetPhases implements domain.Plugin with security wrappers
func (sp *SecurePlugin) GetPhases() []domain.Phase {
	originalPhases := sp.plugin.GetPhases()
	securePhases := make([]domain.Phase, len(originalPhases))
	
	for i, phase := range originalPhases {
		securePhases[i] = &SecurePhase{
			phase:           phase,
			pluginName:      sp.plugin.Name(),
			securityManager: sp.securityManager,
			logger:          sp.logger,
		}
	}
	
	return securePhases
}

// SecurePhase wraps a phase with security checks
type SecurePhase struct {
	phase           domain.Phase
	pluginName      string
	securityManager *SecurityManager
	logger          *slog.Logger
}

// Name implements domain.Phase
func (sp *SecurePhase) Name() string {
	return sp.phase.Name()
}

// Execute implements domain.Phase with security enforcement
func (sp *SecurePhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Wrap the input with secure versions
	secureInput := sp.wrapInput(input)
	
	// Monitor resource usage during execution
	monitor := sp.securityManager.StartMonitoring(sp.pluginName)
	defer monitor.Stop()
	
	// Execute with monitoring
	output, err := sp.phase.Execute(ctx, secureInput)
	
	// Check if any limits were exceeded
	if violations := monitor.GetViolations(); len(violations) > 0 {
		return domain.PhaseOutput{}, fmt.Errorf("security violations: %v", violations)
	}
	
	return output, err
}

// Other domain.Phase methods...
func (sp *SecurePhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	return sp.phase.ValidateInput(ctx, input)
}

func (sp *SecurePhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return sp.phase.ValidateOutput(ctx, output)
}

func (sp *SecurePhase) EstimatedDuration() time.Duration {
	return sp.phase.EstimatedDuration()
}

func (sp *SecurePhase) CanRetry(err error) bool {
	return sp.phase.CanRetry(err)
}

// wrapInput creates secure wrappers for phase input components
func (sp *SecurePhase) wrapInput(input domain.PhaseInput) domain.PhaseInput {
	// Create a copy with secure wrappers
	secureInput := input
	
	// Wrap storage if present
	if input.Storage != nil {
		secureInput.Storage = &SecureStorage{
			storage:         input.Storage,
			pluginName:      sp.pluginName,
			securityManager: sp.securityManager,
		}
	}
	
	// Wrap AI agent if present
	if input.Agent != nil {
		secureInput.Agent = &SecureAgent{
			agent:           input.Agent,
			pluginName:      sp.pluginName,
			securityManager: sp.securityManager,
		}
	}
	
	return secureInput
}

// SecureStorage wraps storage with security checks
type SecureStorage struct {
	storage         domain.Storage
	pluginName      string
	securityManager *SecurityManager
}

func (ss *SecureStorage) SaveOutput(sessionID, filename string, data []byte) error {
	// Check capability
	if err := ss.securityManager.CheckCapability(ss.pluginName, CapabilityStorage); err != nil {
		return err
	}
	
	// Check file write permission
	path := filepath.Join(sessionID, filename)
	if err := ss.securityManager.CheckFileAccess(ss.pluginName, path, true); err != nil {
		return err
	}
	
	// Check data size limit
	policy, _ := ss.securityManager.GetPolicy(ss.pluginName)
	if int64(len(data)) > policy.MaxDataSize {
		return fmt.Errorf("data size %d exceeds limit %d", len(data), policy.MaxDataSize)
	}
	
	return ss.storage.SaveOutput(sessionID, filename, data)
}

func (ss *SecureStorage) LoadOutput(sessionID, filename string) ([]byte, error) {
	// Check capability
	if err := ss.securityManager.CheckCapability(ss.pluginName, CapabilityStorage); err != nil {
		return nil, err
	}
	
	// Check file read permission
	path := filepath.Join(sessionID, filename)
	if err := ss.securityManager.CheckFileAccess(ss.pluginName, path, false); err != nil {
		return nil, err
	}
	
	return ss.storage.LoadOutput(sessionID, filename)
}

// Other storage methods...

// SecureAgent wraps AI agent with security checks
type SecureAgent struct {
	agent           domain.Agent
	pluginName      string
	securityManager *SecurityManager
}

func (sa *SecureAgent) Complete(ctx context.Context, prompt string) (string, error) {
	// Check capability
	if err := sa.securityManager.CheckCapability(sa.pluginName, CapabilityAI); err != nil {
		return "", err
	}
	
	// Check API rate limits
	if err := sa.securityManager.CheckAPIRateLimit(sa.pluginName); err != nil {
		return "", err
	}
	
	return sa.agent.Complete(ctx, prompt)
}

func (sa *SecureAgent) CompleteWithPersona(ctx context.Context, persona, prompt string) (string, error) {
	// Check capability
	if err := sa.securityManager.CheckCapability(sa.pluginName, CapabilityAI); err != nil {
		return "", err
	}
	
	// Check API rate limits
	if err := sa.securityManager.CheckAPIRateLimit(sa.pluginName); err != nil {
		return "", err
	}
	
	return sa.agent.CompleteWithPersona(ctx, persona, prompt)
}

// ResourceMonitor tracks resource usage for security enforcement
type ResourceMonitor struct {
	pluginName string
	startTime  time.Time
	violations []string
	mu         sync.Mutex
}

func (rm *ResourceMonitor) Stop() {
	// Collect final metrics
}

func (rm *ResourceMonitor) GetViolations() []string {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.violations
}

// Additional security manager methods
func (sm *SecurityManager) GetPolicy(pluginName string) (SecurityPolicy, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	policy, exists := sm.policies[pluginName]
	return policy, exists
}

func (sm *SecurityManager) StartMonitoring(pluginName string) *ResourceMonitor {
	monitor := &ResourceMonitor{
		pluginName: pluginName,
		startTime:  time.Now(),
	}
	
	sm.mu.Lock()
	sm.monitors[pluginName] = monitor
	sm.mu.Unlock()
	
	return monitor
}

func (sm *SecurityManager) CheckAPIRateLimit(pluginName string) error {
	// TODO: Implement rate limiting logic
	return nil
}

// CapabilitySet provides convenient capability management
type CapabilitySet struct {
	capabilities map[Capability]bool
}

// NewCapabilitySet creates a new capability set
func NewCapabilitySet(caps ...Capability) *CapabilitySet {
	cs := &CapabilitySet{
		capabilities: make(map[Capability]bool),
	}
	for _, cap := range caps {
		cs.capabilities[cap] = true
	}
	return cs
}

// Add adds a capability to the set
func (cs *CapabilitySet) Add(cap Capability) {
	cs.capabilities[cap] = true
}

// Remove removes a capability from the set
func (cs *CapabilitySet) Remove(cap Capability) {
	delete(cs.capabilities, cap)
}

// Has checks if a capability is in the set
func (cs *CapabilitySet) Has(cap Capability) bool {
	return cs.capabilities[cap]
}

// ToMap returns the capability set as a map
func (cs *CapabilitySet) ToMap() map[Capability]bool {
	result := make(map[Capability]bool)
	for k, v := range cs.capabilities {
		result[k] = v
	}
	return result
}