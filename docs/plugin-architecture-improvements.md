# Plugin System Architecture Improvements

## Executive Summary

This document proposes concrete improvements to the Orchestrator's plugin system across eight key areas. Each improvement includes practical implementation examples that maintain backward compatibility while significantly enhancing capabilities.

## 1. Plugin Discovery & Loading

### Current State
- Manual imports and registration in main.go
- Static compile-time plugin inclusion
- No plugin metadata or versioning

### Proposed Improvements

#### A. Plugin Manifest System

```go
// internal/domain/plugin/manifest.go
package plugin

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "github.com/hashicorp/go-version"
)

// PluginManifest describes a plugin's metadata and requirements
type PluginManifest struct {
    // Basic metadata
    Name        string `json:"name" validate:"required,alphanum"`
    Version     string `json:"version" validate:"required,semver"`
    Description string `json:"description" validate:"required"`
    Author      string `json:"author"`
    License     string `json:"license"`
    
    // Technical requirements
    MinOrchestratorVersion string   `json:"min_orchestrator_version"`
    Dependencies          []string  `json:"dependencies"`
    Capabilities          []string  `json:"capabilities"`
    
    // Resource requirements
    Resources PluginResources `json:"resources"`
    
    // Entry point
    EntryPoint string `json:"entry_point" validate:"required"`
    
    // Configuration schema
    ConfigSchema json.RawMessage `json:"config_schema,omitempty"`
}

type PluginResources struct {
    MaxMemoryMB int `json:"max_memory_mb"`
    MaxCPU      int `json:"max_cpu_percent"`
    MaxDiskMB   int `json:"max_disk_mb"`
}

// PluginLoader handles dynamic plugin discovery and loading
type PluginLoader struct {
    pluginDirs []string
    registry   *DomainRegistry
    validator  ManifestValidator
}

func NewPluginLoader(registry *DomainRegistry) *PluginLoader {
    return &PluginLoader{
        pluginDirs: getPluginSearchPaths(),
        registry:   registry,
        validator:  NewManifestValidator(),
    }
}

// DiscoverPlugins finds all available plugins
func (pl *PluginLoader) DiscoverPlugins() ([]*PluginManifest, error) {
    var manifests []*PluginManifest
    
    for _, dir := range pl.pluginDirs {
        err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return nil // Skip inaccessible directories
            }
            
            if info.Name() == "plugin.json" {
                manifest, err := pl.loadManifest(path)
                if err != nil {
                    // Log error but continue discovery
                    continue
                }
                
                if err := pl.validator.Validate(manifest); err == nil {
                    manifests = append(manifests, manifest)
                }
            }
            return nil
        })
        if err != nil {
            return nil, fmt.Errorf("walking plugin directory %s: %w", dir, err)
        }
    }
    
    return manifests, nil
}

// LoadPlugin loads a specific plugin by manifest
func (pl *PluginLoader) LoadPlugin(manifest *PluginManifest) error {
    // Check version compatibility
    if err := pl.checkVersionCompatibility(manifest); err != nil {
        return fmt.Errorf("version incompatible: %w", err)
    }
    
    // Check dependencies
    if err := pl.checkDependencies(manifest); err != nil {
        return fmt.Errorf("dependency check failed: %w", err)
    }
    
    // Load the plugin based on entry point
    plugin, err := pl.loadPluginImplementation(manifest)
    if err != nil {
        return fmt.Errorf("loading implementation: %w", err)
    }
    
    // Apply resource limits
    if err := pl.applyResourceLimits(plugin, manifest.Resources); err != nil {
        return fmt.Errorf("applying resource limits: %w", err)
    }
    
    // Register with the system
    return pl.registry.Register(plugin)
}

// getPluginSearchPaths returns XDG-compliant plugin directories
func getPluginSearchPaths() []string {
    paths := []string{
        // User plugins
        filepath.Join(os.Getenv("XDG_DATA_HOME"), "orchestrator", "plugins"),
        filepath.Join(os.Getenv("HOME"), ".local", "share", "orchestrator", "plugins"),
        
        // System plugins
        "/usr/local/share/orchestrator/plugins",
        "/usr/share/orchestrator/plugins",
    }
    
    // Add custom paths from environment
    if customPaths := os.Getenv("ORCHESTRATOR_PLUGIN_PATH"); customPaths != "" {
        paths = append(paths, filepath.SplitList(customPaths)...)
    }
    
    return paths
}
```

#### B. Plugin Auto-Discovery with Factories

```go
// internal/domain/plugin/factory.go
package plugin

import (
    "fmt"
    "plugin"
    "reflect"
)

// PluginFactory creates plugin instances
type PluginFactory interface {
    Create(config DomainPluginConfig) (DomainPlugin, error)
    Metadata() PluginMetadata
}

// PluginMetadata provides runtime information about a plugin
type PluginMetadata struct {
    Name         string
    Version      string
    BuildTime    string
    Dependencies map[string]string
}

// DynamicPluginFactory loads plugins from shared libraries
type DynamicPluginFactory struct {
    soPath   string
    metadata PluginMetadata
}

func (f *DynamicPluginFactory) Create(config DomainPluginConfig) (DomainPlugin, error) {
    // Load the shared library
    p, err := plugin.Open(f.soPath)
    if err != nil {
        return nil, fmt.Errorf("opening plugin: %w", err)
    }
    
    // Look for the factory function
    factorySym, err := p.Lookup("NewPlugin")
    if err != nil {
        return nil, fmt.Errorf("plugin missing NewPlugin function: %w", err)
    }
    
    // Type assert to factory function
    factoryFunc, ok := factorySym.(func(DomainPluginConfig) DomainPlugin)
    if !ok {
        return nil, fmt.Errorf("NewPlugin has wrong signature")
    }
    
    return factoryFunc(config), nil
}

// BuiltinPluginFactory creates built-in plugins
type BuiltinPluginFactory struct {
    constructor func(DomainPluginConfig) DomainPlugin
    metadata    PluginMetadata
}

func (f *BuiltinPluginFactory) Create(config DomainPluginConfig) (DomainPlugin, error) {
    return f.constructor(config), nil
}

func (f *BuiltinPluginFactory) Metadata() PluginMetadata {
    return f.metadata
}

// PluginRegistrar manages plugin factories
type PluginRegistrar struct {
    factories map[string]PluginFactory
}

func NewPluginRegistrar() *PluginRegistrar {
    return &PluginRegistrar{
        factories: make(map[string]PluginFactory),
    }
}

// RegisterBuiltin registers a built-in plugin factory
func (r *PluginRegistrar) RegisterBuiltin(name string, factory func(DomainPluginConfig) DomainPlugin, metadata PluginMetadata) {
    r.factories[name] = &BuiltinPluginFactory{
        constructor: factory,
        metadata:    metadata,
    }
}

// RegisterDynamic registers a dynamic plugin factory
func (r *PluginRegistrar) RegisterDynamic(name, soPath string, metadata PluginMetadata) {
    r.factories[name] = &DynamicPluginFactory{
        soPath:   soPath,
        metadata: metadata,
    }
}
```

## 2. Lifecycle Management

### Current State
- Basic Init/Close pattern
- No startup/shutdown hooks
- No dependency resolution

### Proposed Improvements

```go
// internal/domain/plugin/lifecycle.go
package plugin

import (
    "context"
    "fmt"
    "sort"
    "sync"
    "time"
)

// LifecyclePlugin extends DomainPlugin with lifecycle hooks
type LifecyclePlugin interface {
    DomainPlugin
    
    // Lifecycle hooks
    OnInit(ctx context.Context) error
    OnStart(ctx context.Context) error
    OnStop(ctx context.Context) error
    OnDestroy(ctx context.Context) error
    
    // Health check
    HealthCheck(ctx context.Context) error
    
    // Dependencies
    DependsOn() []string
}

// PluginLifecycleManager manages plugin lifecycles
type PluginLifecycleManager struct {
    plugins    map[string]LifecyclePlugin
    states     map[string]PluginState
    mu         sync.RWMutex
    startOrder []string
}

type PluginState int

const (
    StateUninitialized PluginState = iota
    StateInitialized
    StateStarting
    StateRunning
    StateStopping
    StateStopped
    StateError
)

func NewPluginLifecycleManager() *PluginLifecycleManager {
    return &PluginLifecycleManager{
        plugins: make(map[string]LifecyclePlugin),
        states:  make(map[string]PluginState),
    }
}

// AddPlugin adds a plugin to lifecycle management
func (m *PluginLifecycleManager) AddPlugin(plugin LifecyclePlugin) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    name := plugin.Name()
    if _, exists := m.plugins[name]; exists {
        return fmt.Errorf("plugin %s already registered", name)
    }
    
    m.plugins[name] = plugin
    m.states[name] = StateUninitialized
    
    // Rebuild startup order
    return m.buildStartupOrder()
}

// buildStartupOrder creates a dependency-ordered startup sequence
func (m *PluginLifecycleManager) buildStartupOrder() error {
    // Create dependency graph
    graph := make(map[string][]string)
    for name, plugin := range m.plugins {
        graph[name] = plugin.DependsOn()
    }
    
    // Topological sort
    order, err := topologicalSort(graph)
    if err != nil {
        return fmt.Errorf("dependency cycle detected: %w", err)
    }
    
    m.startOrder = order
    return nil
}

// InitializeAll initializes all plugins in dependency order
func (m *PluginLifecycleManager) InitializeAll(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for _, name := range m.startOrder {
        plugin := m.plugins[name]
        
        if err := plugin.OnInit(ctx); err != nil {
            m.states[name] = StateError
            return fmt.Errorf("initializing plugin %s: %w", name, err)
        }
        
        m.states[name] = StateInitialized
    }
    
    return nil
}

// StartAll starts all plugins with proper sequencing
func (m *PluginLifecycleManager) StartAll(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Start plugins in dependency order
    for _, name := range m.startOrder {
        if err := m.startPlugin(ctx, name); err != nil {
            // Rollback started plugins
            m.rollbackStartedPlugins(ctx)
            return err
        }
    }
    
    return nil
}

func (m *PluginLifecycleManager) startPlugin(ctx context.Context, name string) error {
    plugin := m.plugins[name]
    m.states[name] = StateStarting
    
    // Check dependencies are running
    for _, dep := range plugin.DependsOn() {
        if m.states[dep] != StateRunning {
            return fmt.Errorf("dependency %s not running for plugin %s", dep, name)
        }
    }
    
    // Start with timeout
    startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    if err := plugin.OnStart(startCtx); err != nil {
        m.states[name] = StateError
        return fmt.Errorf("starting plugin %s: %w", name, err)
    }
    
    m.states[name] = StateRunning
    
    // Verify health
    if err := plugin.HealthCheck(startCtx); err != nil {
        m.states[name] = StateError
        return fmt.Errorf("plugin %s health check failed: %w", name, err)
    }
    
    return nil
}

// StopAll gracefully stops all plugins in reverse order
func (m *PluginLifecycleManager) StopAll(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Stop in reverse dependency order
    for i := len(m.startOrder) - 1; i >= 0; i-- {
        name := m.startOrder[i]
        if m.states[name] == StateRunning {
            m.states[name] = StateStopping
            
            stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
            if err := m.plugins[name].OnStop(stopCtx); err != nil {
                // Log error but continue stopping others
                cancel()
                continue
            }
            cancel()
            
            m.states[name] = StateStopped
        }
    }
    
    return nil
}

// HealthCheckAll performs health checks on all running plugins
func (m *PluginLifecycleManager) HealthCheckAll(ctx context.Context) map[string]error {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    results := make(map[string]error)
    var wg sync.WaitGroup
    
    for name, plugin := range m.plugins {
        if m.states[name] == StateRunning {
            wg.Add(1)
            go func(n string, p LifecyclePlugin) {
                defer wg.Done()
                
                checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
                defer cancel()
                
                results[n] = p.HealthCheck(checkCtx)
            }(name, plugin)
        }
    }
    
    wg.Wait()
    return results
}

// topologicalSort performs topological sorting for dependency resolution
func topologicalSort(graph map[string][]string) ([]string, error) {
    var result []string
    visited := make(map[string]bool)
    tempMark := make(map[string]bool)
    
    var visit func(string) error
    visit = func(node string) error {
        if tempMark[node] {
            return fmt.Errorf("circular dependency detected")
        }
        if visited[node] {
            return nil
        }
        
        tempMark[node] = true
        for _, dep := range graph[node] {
            if err := visit(dep); err != nil {
                return err
            }
        }
        tempMark[node] = false
        visited[node] = true
        result = append(result, node)
        
        return nil
    }
    
    for node := range graph {
        if !visited[node] {
            if err := visit(node); err != nil {
                return nil, err
            }
        }
    }
    
    return result, nil
}
```

## 3. Configuration Management

### Current State
- Simple key-value interface
- No schema validation
- No environment overrides

### Proposed Improvements

```go
// internal/domain/plugin/config.go
package plugin

import (
    "encoding/json"
    "fmt"
    "os"
    "reflect"
    "strings"
    
    "github.com/go-playground/validator/v10"
    "github.com/mitchellh/mapstructure"
)

// ConfigurablePlugin extends plugins with advanced configuration
type ConfigurablePlugin interface {
    DomainPlugin
    
    // Configuration schema
    ConfigSchema() ConfigSchema
    
    // Configuration validation
    ValidateConfig(config interface{}) error
    
    // Hot reload support
    SupportsHotReload() bool
    OnConfigChange(oldConfig, newConfig interface{}) error
}

// ConfigSchema describes a plugin's configuration structure
type ConfigSchema struct {
    // JSON Schema for validation
    Schema json.RawMessage `json:"schema"`
    
    // Example configuration
    Example interface{} `json:"example"`
    
    // Environment variable mapping
    EnvMapping map[string]string `json:"env_mapping"`
    
    // Sensitive fields that should be masked
    SensitiveFields []string `json:"sensitive_fields"`
}

// PluginConfigManager handles plugin configuration
type PluginConfigManager struct {
    configs   map[string]interface{}
    schemas   map[string]ConfigSchema
    validator *validator.Validate
    watchers  map[string]*ConfigWatcher
    mu        sync.RWMutex
}

func NewPluginConfigManager() *PluginConfigManager {
    return &PluginConfigManager{
        configs:   make(map[string]interface{}),
        schemas:   make(map[string]ConfigSchema),
        validator: validator.New(),
        watchers:  make(map[string]*ConfigWatcher),
    }
}

// LoadConfig loads configuration for a plugin
func (m *PluginConfigManager) LoadConfig(pluginName string, configPath string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Read config file
    data, err := os.ReadFile(configPath)
    if err != nil {
        return fmt.Errorf("reading config: %w", err)
    }
    
    // Parse as generic map
    var rawConfig map[string]interface{}
    if err := json.Unmarshal(data, &rawConfig); err != nil {
        return fmt.Errorf("parsing config: %w", err)
    }
    
    // Apply environment overrides
    if schema, ok := m.schemas[pluginName]; ok {
        m.applyEnvironmentOverrides(rawConfig, schema.EnvMapping)
    }
    
    // Validate against schema
    if err := m.validateConfig(pluginName, rawConfig); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    m.configs[pluginName] = rawConfig
    return nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (m *PluginConfigManager) applyEnvironmentOverrides(config map[string]interface{}, envMapping map[string]string) {
    for configKey, envVar := range envMapping {
        if value := os.Getenv(envVar); value != "" {
            // Navigate nested keys (e.g., "database.host")
            keys := strings.Split(configKey, ".")
            current := config
            
            for i, key := range keys {
                if i == len(keys)-1 {
                    // Set the value
                    current[key] = m.parseEnvValue(value)
                } else {
                    // Navigate deeper
                    if next, ok := current[key].(map[string]interface{}); ok {
                        current = next
                    } else {
                        // Create missing intermediate maps
                        next := make(map[string]interface{})
                        current[key] = next
                        current = next
                    }
                }
            }
        }
    }
}

// parseEnvValue attempts to parse environment values to appropriate types
func (m *PluginConfigManager) parseEnvValue(value string) interface{} {
    // Try parsing as number
    if v, err := strconv.ParseInt(value, 10, 64); err == nil {
        return v
    }
    if v, err := strconv.ParseFloat(value, 64); err == nil {
        return v
    }
    
    // Try parsing as boolean
    if v, err := strconv.ParseBool(value); err == nil {
        return v
    }
    
    // Try parsing as JSON (for arrays/objects)
    var jsonValue interface{}
    if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
        return jsonValue
    }
    
    // Default to string
    return value
}

// WatchConfig enables hot reloading for a plugin's configuration
func (m *PluginConfigManager) WatchConfig(pluginName string, configPath string, plugin ConfigurablePlugin) error {
    if !plugin.SupportsHotReload() {
        return fmt.Errorf("plugin %s does not support hot reload", pluginName)
    }
    
    watcher := &ConfigWatcher{
        path:      configPath,
        plugin:    plugin,
        manager:   m,
        pluginName: pluginName,
    }
    
    if err := watcher.Start(); err != nil {
        return fmt.Errorf("starting config watcher: %w", err)
    }
    
    m.mu.Lock()
    m.watchers[pluginName] = watcher
    m.mu.Unlock()
    
    return nil
}

// ConfigWatcher watches for configuration changes
type ConfigWatcher struct {
    path       string
    plugin     ConfigurablePlugin
    manager    *PluginConfigManager
    pluginName string
    stop       chan struct{}
}

func (w *ConfigWatcher) Start() error {
    w.stop = make(chan struct{})
    
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        var lastMod time.Time
        
        for {
            select {
            case <-ticker.C:
                info, err := os.Stat(w.path)
                if err != nil {
                    continue
                }
                
                if info.ModTime().After(lastMod) {
                    lastMod = info.ModTime()
                    w.reloadConfig()
                }
                
            case <-w.stop:
                return
            }
        }
    }()
    
    return nil
}

func (w *ConfigWatcher) reloadConfig() {
    // Get current config
    w.manager.mu.RLock()
    oldConfig := w.manager.configs[w.pluginName]
    w.manager.mu.RUnlock()
    
    // Load new config
    if err := w.manager.LoadConfig(w.pluginName, w.path); err != nil {
        // Log error but don't crash
        return
    }
    
    // Get new config
    w.manager.mu.RLock()
    newConfig := w.manager.configs[w.pluginName]
    w.manager.mu.RUnlock()
    
    // Notify plugin of change
    if err := w.plugin.OnConfigChange(oldConfig, newConfig); err != nil {
        // Rollback to old config
        w.manager.mu.Lock()
        w.manager.configs[w.pluginName] = oldConfig
        w.manager.mu.Unlock()
    }
}

// StructuredConfig provides type-safe configuration access
type StructuredConfig[T any] struct {
    manager    *PluginConfigManager
    pluginName string
}

func NewStructuredConfig[T any](manager *PluginConfigManager, pluginName string) *StructuredConfig[T] {
    return &StructuredConfig[T]{
        manager:    manager,
        pluginName: pluginName,
    }
}

func (s *StructuredConfig[T]) Get() (*T, error) {
    s.manager.mu.RLock()
    defer s.manager.mu.RUnlock()
    
    rawConfig, ok := s.manager.configs[s.pluginName]
    if !ok {
        return nil, fmt.Errorf("no configuration found for plugin %s", s.pluginName)
    }
    
    var config T
    decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
        Result:           &config,
        WeaklyTypedInput: true,
        TagName:          "json",
    })
    if err != nil {
        return nil, fmt.Errorf("creating decoder: %w", err)
    }
    
    if err := decoder.Decode(rawConfig); err != nil {
        return nil, fmt.Errorf("decoding config: %w", err)
    }
    
    return &config, nil
}
```

## 4. Inter-Plugin Communication

### Current State
- No plugin-to-plugin communication
- No shared context

### Proposed Improvements

```go
// internal/domain/plugin/communication.go
package plugin

import (
    "context"
    "fmt"
    "reflect"
    "sync"
    "time"
)

// EventBus enables plugin communication via events
type EventBus interface {
    // Publishing
    Publish(ctx context.Context, event Event) error
    PublishAsync(event Event)
    
    // Subscribing
    Subscribe(eventType string, handler EventHandler) (Subscription, error)
    SubscribePattern(pattern string, handler EventHandler) (Subscription, error)
    
    // Request/Response
    Request(ctx context.Context, request Event, timeout time.Duration) (Event, error)
}

// Event represents a plugin event
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Source    string                 `json:"source"`
    Timestamp time.Time              `json:"timestamp"`
    Data      interface{}            `json:"data"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event Event) error

// Subscription represents an event subscription
type Subscription interface {
    Unsubscribe() error
    ID() string
}

// DefaultEventBus implements the EventBus interface
type DefaultEventBus struct {
    subscribers map[string][]subscriberInfo
    patterns    []patternSubscriber
    mu          sync.RWMutex
    
    // Request/response tracking
    pending map[string]chan Event
    
    // Metrics
    metrics EventBusMetrics
}

type subscriberInfo struct {
    id      string
    handler EventHandler
}

type patternSubscriber struct {
    pattern string
    info    subscriberInfo
}

func NewEventBus() *DefaultEventBus {
    return &DefaultEventBus{
        subscribers: make(map[string][]subscriberInfo),
        pending:     make(map[string]chan Event),
    }
}

func (eb *DefaultEventBus) Publish(ctx context.Context, event Event) error {
    eb.mu.RLock()
    defer eb.mu.RUnlock()
    
    // Notify exact matches
    if subs, ok := eb.subscribers[event.Type]; ok {
        for _, sub := range subs {
            if err := sub.handler(ctx, event); err != nil {
                // Log error but continue
                eb.metrics.HandlerErrors++
            }
        }
    }
    
    // Notify pattern matches
    for _, psub := range eb.patterns {
        if matched, _ := filepath.Match(psub.pattern, event.Type); matched {
            if err := psub.info.handler(ctx, event); err != nil {
                eb.metrics.HandlerErrors++
            }
        }
    }
    
    // Check for pending requests
    if respChan, ok := eb.pending[event.ID]; ok {
        select {
        case respChan <- event:
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    eb.metrics.PublishedEvents++
    return nil
}

// SharedContext enables plugins to share state
type SharedContext struct {
    data  map[string]interface{}
    mu    sync.RWMutex
    hooks map[string][]ContextHook
}

type ContextHook func(key string, oldValue, newValue interface{})

func NewSharedContext() *SharedContext {
    return &SharedContext{
        data:  make(map[string]interface{}),
        hooks: make(map[string][]ContextHook),
    }
}

func (sc *SharedContext) Set(key string, value interface{}) {
    sc.mu.Lock()
    oldValue := sc.data[key]
    sc.data[key] = value
    hooks := sc.hooks[key]
    sc.mu.Unlock()
    
    // Notify hooks outside the lock
    for _, hook := range hooks {
        hook(key, oldValue, value)
    }
}

func (sc *SharedContext) Get(key string) (interface{}, bool) {
    sc.mu.RLock()
    defer sc.mu.RUnlock()
    value, ok := sc.data[key]
    return value, ok
}

// TypedGet provides type-safe access
func TypedGet[T any](sc *SharedContext, key string) (T, bool) {
    value, ok := sc.Get(key)
    if !ok {
        var zero T
        return zero, false
    }
    
    typed, ok := value.(T)
    return typed, ok
}

// PluginMessenger provides direct plugin-to-plugin messaging
type PluginMessenger struct {
    registry *DomainRegistry
    bus      EventBus
}

func NewPluginMessenger(registry *DomainRegistry, bus EventBus) *PluginMessenger {
    return &PluginMessenger{
        registry: registry,
        bus:      bus,
    }
}

// SendMessage sends a message to a specific plugin
func (pm *PluginMessenger) SendMessage(ctx context.Context, from, to string, message interface{}) error {
    event := Event{
        ID:        generateID(),
        Type:      fmt.Sprintf("plugin.message.%s", to),
        Source:    from,
        Timestamp: time.Now(),
        Data:      message,
        Metadata: map[string]interface{}{
            "target": to,
        },
    }
    
    return pm.bus.Publish(ctx, event)
}

// RequestResponse sends a request and waits for response
func (pm *PluginMessenger) RequestResponse(ctx context.Context, from, to string, request interface{}, timeout time.Duration) (interface{}, error) {
    event := Event{
        ID:        generateID(),
        Type:      fmt.Sprintf("plugin.request.%s", to),
        Source:    from,
        Timestamp: time.Now(),
        Data:      request,
        Metadata: map[string]interface{}{
            "target":       to,
            "reply_needed": true,
        },
    }
    
    response, err := pm.bus.Request(ctx, event, timeout)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    
    return response.Data, nil
}

// PluginCapabilities allows plugins to advertise and discover capabilities
type PluginCapabilities struct {
    capabilities map[string]map[string]Capability
    mu           sync.RWMutex
}

type Capability struct {
    Name        string
    Version     string
    Description string
    Schema      json.RawMessage
}

func NewPluginCapabilities() *PluginCapabilities {
    return &PluginCapabilities{
        capabilities: make(map[string]map[string]Capability),
    }
}

func (pc *PluginCapabilities) Register(pluginName string, capability Capability) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    
    if pc.capabilities[pluginName] == nil {
        pc.capabilities[pluginName] = make(map[string]Capability)
    }
    
    pc.capabilities[pluginName][capability.Name] = capability
}

func (pc *PluginCapabilities) Find(capabilityName string) []PluginWithCapability {
    pc.mu.RLock()
    defer pc.mu.RUnlock()
    
    var results []PluginWithCapability
    
    for pluginName, caps := range pc.capabilities {
        if cap, ok := caps[capabilityName]; ok {
            results = append(results, PluginWithCapability{
                PluginName: pluginName,
                Capability: cap,
            })
        }
    }
    
    return results
}

type PluginWithCapability struct {
    PluginName string
    Capability Capability
}
```

## 5. Error Handling & Resilience

### Current State
- Basic error returns
- No circuit breakers
- Limited retry logic

### Proposed Improvements

```go
// internal/domain/plugin/resilience.go
package plugin

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/sony/gobreaker"
)

// ResilientPlugin wraps plugins with resilience patterns
type ResilientPlugin struct {
    DomainPlugin
    
    circuitBreaker *gobreaker.CircuitBreaker
    retryPolicy    RetryPolicy
    fallback       FallbackHandler
    limiter        RateLimiter
    
    metrics PluginMetrics
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
    MaxAttempts     int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
    RetryableErrors func(error) bool
}

// FallbackHandler provides fallback behavior
type FallbackHandler func(ctx context.Context, request string, err error) (interface{}, error)

// RateLimiter controls request rate
type RateLimiter interface {
    Allow() bool
    Wait(ctx context.Context) error
}

// NewResilientPlugin creates a resilient plugin wrapper
func NewResilientPlugin(plugin DomainPlugin, opts ...ResilienceOption) *ResilientPlugin {
    rp := &ResilientPlugin{
        DomainPlugin: plugin,
    }
    
    // Apply options
    cfg := defaultResilienceConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    
    // Initialize circuit breaker
    rp.circuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        plugin.Name(),
        MaxRequests: cfg.CircuitBreaker.MaxRequests,
        Interval:    cfg.CircuitBreaker.Interval,
        Timeout:     cfg.CircuitBreaker.Timeout,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 10 && failureRatio >= 0.6
        },
        OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
            rp.metrics.CircuitBreakerStateChanges++
        },
    })
    
    rp.retryPolicy = cfg.RetryPolicy
    rp.fallback = cfg.Fallback
    rp.limiter = cfg.RateLimiter
    
    return rp
}

// Execute wraps phase execution with resilience patterns
func (rp *ResilientPlugin) Execute(ctx context.Context, request string) (interface{}, error) {
    // Rate limiting
    if rp.limiter != nil {
        if err := rp.limiter.Wait(ctx); err != nil {
            rp.metrics.RateLimitExceeded++
            return nil, fmt.Errorf("rate limit exceeded: %w", err)
        }
    }
    
    // Circuit breaker
    result, err := rp.circuitBreaker.Execute(func() (interface{}, error) {
        return rp.executeWithRetry(ctx, request)
    })
    
    if err != nil {
        rp.metrics.TotalErrors++
        
        // Try fallback
        if rp.fallback != nil {
            fallbackResult, fallbackErr := rp.fallback(ctx, request, err)
            if fallbackErr == nil {
                rp.metrics.FallbackSuccesses++
                return fallbackResult, nil
            }
            rp.metrics.FallbackFailures++
        }
        
        return nil, err
    }
    
    rp.metrics.TotalSuccesses++
    return result, nil
}

func (rp *ResilientPlugin) executeWithRetry(ctx context.Context, request string) (interface{}, error) {
    var lastErr error
    delay := rp.retryPolicy.InitialDelay
    
    for attempt := 0; attempt <= rp.retryPolicy.MaxAttempts; attempt++ {
        if attempt > 0 {
            rp.metrics.RetryAttempts++
            
            // Wait with backoff
            select {
            case <-time.After(delay):
                delay = time.Duration(float64(delay) * rp.retryPolicy.BackoffFactor)
                if delay > rp.retryPolicy.MaxDelay {
                    delay = rp.retryPolicy.MaxDelay
                }
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
        
        // Execute phase
        phases := rp.DomainPlugin.GetPhases()
        input := domain.PhaseInput{Request: request}
        
        for _, phase := range phases {
            output, err := phase.Execute(ctx, input)
            if err != nil {
                lastErr = err
                
                // Check if error is retryable
                if rp.retryPolicy.RetryableErrors != nil && !rp.retryPolicy.RetryableErrors(err) {
                    return nil, err
                }
                
                break
            }
            
            input.Data = output.Data
        }
        
        if lastErr == nil {
            return input.Data, nil
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// PluginHealthMonitor monitors plugin health
type PluginHealthMonitor struct {
    plugins  map[string]HealthCheckable
    statuses map[string]*HealthStatus
    mu       sync.RWMutex
    
    checkInterval time.Duration
    stopChan      chan struct{}
}

type HealthCheckable interface {
    HealthCheck(ctx context.Context) error
}

type HealthStatus struct {
    Healthy          bool
    LastCheck        time.Time
    ConsecutiveFails int
    LastError        error
    Metrics          map[string]interface{}
}

func NewPluginHealthMonitor(checkInterval time.Duration) *PluginHealthMonitor {
    return &PluginHealthMonitor{
        plugins:       make(map[string]HealthCheckable),
        statuses:      make(map[string]*HealthStatus),
        checkInterval: checkInterval,
        stopChan:      make(chan struct{}),
    }
}

func (m *PluginHealthMonitor) RegisterPlugin(name string, plugin HealthCheckable) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.plugins[name] = plugin
    m.statuses[name] = &HealthStatus{
        Healthy:   true,
        LastCheck: time.Now(),
        Metrics:   make(map[string]interface{}),
    }
}

func (m *PluginHealthMonitor) Start() {
    go func() {
        ticker := time.NewTicker(m.checkInterval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                m.checkAllPlugins()
            case <-m.stopChan:
                return
            }
        }
    }()
}

func (m *PluginHealthMonitor) checkAllPlugins() {
    m.mu.RLock()
    plugins := make(map[string]HealthCheckable)
    for name, plugin := range m.plugins {
        plugins[name] = plugin
    }
    m.mu.RUnlock()
    
    var wg sync.WaitGroup
    
    for name, plugin := range plugins {
        wg.Add(1)
        go func(n string, p HealthCheckable) {
            defer wg.Done()
            
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            
            err := p.HealthCheck(ctx)
            
            m.mu.Lock()
            status := m.statuses[n]
            status.LastCheck = time.Now()
            
            if err != nil {
                status.Healthy = false
                status.LastError = err
                status.ConsecutiveFails++
            } else {
                status.Healthy = true
                status.LastError = nil
                status.ConsecutiveFails = 0
            }
            m.mu.Unlock()
        }(name, plugin)
    }
    
    wg.Wait()
}

// GetUnhealthyPlugins returns plugins that are currently unhealthy
func (m *PluginHealthMonitor) GetUnhealthyPlugins() map[string]*HealthStatus {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    unhealthy := make(map[string]*HealthStatus)
    
    for name, status := range m.statuses {
        if !status.Healthy {
            // Create a copy
            statusCopy := *status
            unhealthy[name] = &statusCopy
        }
    }
    
    return unhealthy
}
```

## 6. Development Experience

### Current State
- Basic interfaces
- Manual plugin creation
- Limited testing support

### Proposed Improvements

```go
// internal/domain/plugin/development.go
package plugin

import (
    "bytes"
    "fmt"
    "go/format"
    "os"
    "path/filepath"
    "text/template"
)

// PluginScaffolder generates plugin boilerplate
type PluginScaffolder struct {
    templatesDir string
}

func NewPluginScaffolder() *PluginScaffolder {
    return &PluginScaffolder{
        templatesDir: getTemplatesDir(),
    }
}

// ScaffoldPlugin creates a new plugin structure
func (s *PluginScaffolder) ScaffoldPlugin(name, outputDir string, opts ScaffoldOptions) error {
    // Create plugin directory
    pluginDir := filepath.Join(outputDir, name)
    if err := os.MkdirAll(pluginDir, 0755); err != nil {
        return fmt.Errorf("creating plugin directory: %w", err)
    }
    
    // Generate files from templates
    files := []struct {
        template string
        output   string
    }{
        {"plugin.go.tmpl", fmt.Sprintf("%s.go", name)},
        {"plugin_test.go.tmpl", fmt.Sprintf("%s_test.go", name)},
        {"config.go.tmpl", "config.go"},
        {"phases.go.tmpl", "phases.go"},
        {"README.md.tmpl", "README.md"},
        {"plugin.json.tmpl", "plugin.json"},
    }
    
    data := struct {
        Name        string
        Package     string
        Description string
        Author      string
        Version     string
        Options     ScaffoldOptions
    }{
        Name:        name,
        Package:     name,
        Description: opts.Description,
        Author:      opts.Author,
        Version:     "0.1.0",
        Options:     opts,
    }
    
    for _, file := range files {
        if err := s.generateFile(file.template, filepath.Join(pluginDir, file.output), data); err != nil {
            return fmt.Errorf("generating %s: %w", file.output, err)
        }
    }
    
    // Create subdirectories
    subdirs := []string{"phases", "prompts", "testdata"}
    for _, dir := range subdirs {
        if err := os.MkdirAll(filepath.Join(pluginDir, dir), 0755); err != nil {
            return fmt.Errorf("creating %s directory: %w", dir, err)
        }
    }
    
    return nil
}

func (s *PluginScaffolder) generateFile(templateName, outputPath string, data interface{}) error {
    tmplPath := filepath.Join(s.templatesDir, templateName)
    tmpl, err := template.ParseFiles(tmplPath)
    if err != nil {
        return fmt.Errorf("parsing template: %w", err)
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return fmt.Errorf("executing template: %w", err)
    }
    
    // Format Go code
    content := buf.Bytes()
    if filepath.Ext(outputPath) == ".go" {
        formatted, err := format.Source(content)
        if err == nil {
            content = formatted
        }
    }
    
    return os.WriteFile(outputPath, content, 0644)
}

// PluginTestHarness provides testing utilities
type PluginTestHarness struct {
    plugin   DomainPlugin
    recorder *EventRecorder
    mocks    map[string]interface{}
}

func NewPluginTestHarness(plugin DomainPlugin) *PluginTestHarness {
    return &PluginTestHarness{
        plugin:   plugin,
        recorder: NewEventRecorder(),
        mocks:    make(map[string]interface{}),
    }
}

// MockPhase creates a mock phase for testing
func (h *PluginTestHarness) MockPhase(name string) *MockPhase {
    mock := &MockPhase{
        name:      name,
        responses: make(map[string]domain.PhaseOutput),
    }
    h.mocks[name] = mock
    return mock
}

// ExecuteWithMocks runs the plugin with mocked phases
func (h *PluginTestHarness) ExecuteWithMocks(ctx context.Context, request string) error {
    // Replace real phases with mocks
    phases := h.plugin.GetPhases()
    for i, phase := range phases {
        if mock, ok := h.mocks[phase.Name()]; ok {
            phases[i] = mock.(*MockPhase)
        }
    }
    
    // Execute with recording
    return h.executeWithRecording(ctx, phases, request)
}

// GetEvents returns recorded events
func (h *PluginTestHarness) GetEvents() []RecordedEvent {
    return h.recorder.GetEvents()
}

// AssertPhaseExecuted checks if a phase was executed
func (h *PluginTestHarness) AssertPhaseExecuted(phaseName string) error {
    for _, event := range h.recorder.GetEvents() {
        if event.Type == "phase.executed" && event.Data["phase"] == phaseName {
            return nil
        }
    }
    return fmt.Errorf("phase %s was not executed", phaseName)
}

// MockPhase implements a mock phase for testing
type MockPhase struct {
    name      string
    responses map[string]domain.PhaseOutput
    calls     []PhaseCall
    mu        sync.Mutex
}

type PhaseCall struct {
    Input     domain.PhaseInput
    Timestamp time.Time
}

func (m *MockPhase) Name() string {
    return m.name
}

func (m *MockPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
    m.mu.Lock()
    m.calls = append(m.calls, PhaseCall{
        Input:     input,
        Timestamp: time.Now(),
    })
    m.mu.Unlock()
    
    // Return configured response
    if response, ok := m.responses[input.Request]; ok {
        return response, nil
    }
    
    return domain.PhaseOutput{}, fmt.Errorf("no mock response configured")
}

func (m *MockPhase) SetResponse(request string, output domain.PhaseOutput) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.responses[request] = output
}

func (m *MockPhase) GetCalls() []PhaseCall {
    m.mu.Lock()
    defer m.mu.Unlock()
    return append([]PhaseCall{}, m.calls...)
}

// PluginDebugger provides debugging utilities
type PluginDebugger struct {
    plugin       DomainPlugin
    breakpoints  map[string]Breakpoint
    traceEnabled bool
    stepMode     bool
}

type Breakpoint struct {
    Phase     string
    Condition func(input domain.PhaseInput) bool
    Handler   func(input domain.PhaseInput)
}

func NewPluginDebugger(plugin DomainPlugin) *PluginDebugger {
    return &PluginDebugger{
        plugin:      plugin,
        breakpoints: make(map[string]Breakpoint),
    }
}

func (d *PluginDebugger) SetBreakpoint(phase string, condition func(domain.PhaseInput) bool) {
    d.breakpoints[phase] = Breakpoint{
        Phase:     phase,
        Condition: condition,
        Handler: func(input domain.PhaseInput) {
            fmt.Printf("Breakpoint hit at phase %s with input: %+v\n", phase, input)
        },
    }
}

func (d *PluginDebugger) EnableTrace() {
    d.traceEnabled = true
}

func (d *PluginDebugger) EnableStepMode() {
    d.stepMode = true
}

// PluginProfiler provides performance profiling
type PluginProfiler struct {
    plugin   DomainPlugin
    profiles map[string]*PhaseProfile
    mu       sync.RWMutex
}

type PhaseProfile struct {
    Phase          string
    ExecutionTimes []time.Duration
    MemoryUsage    []uint64
    ErrorCount     int
}

func NewPluginProfiler(plugin DomainPlugin) *PluginProfiler {
    return &PluginProfiler{
        plugin:   plugin,
        profiles: make(map[string]*PhaseProfile),
    }
}

func (p *PluginProfiler) Profile(ctx context.Context, request string) error {
    phases := p.plugin.GetPhases()
    input := domain.PhaseInput{Request: request}
    
    for _, phase := range phases {
        profile := p.getOrCreateProfile(phase.Name())
        
        // Measure execution
        start := time.Now()
        var memBefore runtime.MemStats
        runtime.ReadMemStats(&memBefore)
        
        output, err := phase.Execute(ctx, input)
        
        duration := time.Since(start)
        var memAfter runtime.MemStats
        runtime.ReadMemStats(&memAfter)
        
        // Record metrics
        p.mu.Lock()
        profile.ExecutionTimes = append(profile.ExecutionTimes, duration)
        profile.MemoryUsage = append(profile.MemoryUsage, memAfter.Alloc-memBefore.Alloc)
        if err != nil {
            profile.ErrorCount++
        }
        p.mu.Unlock()
        
        if err != nil {
            return err
        }
        
        input.Data = output.Data
    }
    
    return nil
}

func (p *PluginProfiler) GetProfile(phaseName string) *PhaseProfile {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    if profile, ok := p.profiles[phaseName]; ok {
        // Return a copy
        profileCopy := *profile
        profileCopy.ExecutionTimes = append([]time.Duration{}, profile.ExecutionTimes...)
        profileCopy.MemoryUsage = append([]uint64{}, profile.MemoryUsage...)
        return &profileCopy
    }
    
    return nil
}

func (p *PluginProfiler) getOrCreateProfile(phaseName string) *PhaseProfile {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if profile, ok := p.profiles[phaseName]; ok {
        return profile
    }
    
    profile := &PhaseProfile{
        Phase: phaseName,
    }
    p.profiles[phaseName] = profile
    return profile
}

// GetAverageExecutionTime returns average execution time for a phase
func (profile *PhaseProfile) GetAverageExecutionTime() time.Duration {
    if len(profile.ExecutionTimes) == 0 {
        return 0
    }
    
    var total time.Duration
    for _, t := range profile.ExecutionTimes {
        total += t
    }
    
    return total / time.Duration(len(profile.ExecutionTimes))
}
```

## 7. Performance & Monitoring

### Current State
- Basic metrics interface
- No resource limits
- Limited observability

### Proposed Improvements

```go
// internal/domain/plugin/monitoring.go
package plugin

import (
    "context"
    "fmt"
    "runtime"
    "sync"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// PluginMetricsCollector collects and exposes plugin metrics
type PluginMetricsCollector struct {
    // Prometheus metrics
    phaseExecutions *prometheus.CounterVec
    phaseDurations  *prometheus.HistogramVec
    phaseErrors     *prometheus.CounterVec
    
    // Resource metrics
    memoryUsage *prometheus.GaugeVec
    cpuUsage    *prometheus.GaugeVec
    
    // Custom metrics
    customMetrics map[string]prometheus.Collector
    mu            sync.RWMutex
}

func NewPluginMetricsCollector() *PluginMetricsCollector {
    return &PluginMetricsCollector{
        phaseExecutions: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "orchestrator_plugin_phase_executions_total",
                Help: "Total number of phase executions",
            },
            []string{"plugin", "phase"},
        ),
        phaseDurations: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "orchestrator_plugin_phase_duration_seconds",
                Help:    "Phase execution duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"plugin", "phase"},
        ),
        phaseErrors: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "orchestrator_plugin_phase_errors_total",
                Help: "Total number of phase execution errors",
            },
            []string{"plugin", "phase", "error_type"},
        ),
        memoryUsage: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "orchestrator_plugin_memory_usage_bytes",
                Help: "Current memory usage in bytes",
            },
            []string{"plugin"},
        ),
        cpuUsage: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "orchestrator_plugin_cpu_usage_percent",
                Help: "Current CPU usage percentage",
            },
            []string{"plugin"},
        ),
        customMetrics: make(map[string]prometheus.Collector),
    }
}

// ObservePhaseExecution records phase execution metrics
func (c *PluginMetricsCollector) ObservePhaseExecution(plugin, phase string, duration time.Duration, err error) {
    c.phaseExecutions.WithLabelValues(plugin, phase).Inc()
    c.phaseDurations.WithLabelValues(plugin, phase).Observe(duration.Seconds())
    
    if err != nil {
        errorType := "unknown"
        if phaseErr, ok := err.(*domain.PhaseError); ok {
            errorType = phaseErr.Type()
        }
        c.phaseErrors.WithLabelValues(plugin, phase, errorType).Inc()
    }
}

// RegisterCustomMetric allows plugins to register custom metrics
func (c *PluginMetricsCollector) RegisterCustomMetric(name string, metric prometheus.Collector) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if _, exists := c.customMetrics[name]; exists {
        return fmt.Errorf("metric %s already registered", name)
    }
    
    c.customMetrics[name] = metric
    prometheus.MustRegister(metric)
    
    return nil
}

// ResourceLimiter enforces resource constraints on plugins
type ResourceLimiter struct {
    limits   map[string]ResourceLimits
    monitors map[string]*ResourceMonitor
    mu       sync.RWMutex
}

type ResourceLimits struct {
    MaxMemoryMB int
    MaxCPU      int // percentage
    MaxGoroutines int
    MaxOpenFiles  int
}

type ResourceMonitor struct {
    plugin    string
    limits    ResourceLimits
    stopChan  chan struct{}
    violations chan ResourceViolation
}

type ResourceViolation struct {
    Plugin       string
    Resource     string
    Current      int64
    Limit        int64
    Timestamp    time.Time
}

func NewResourceLimiter() *ResourceLimiter {
    return &ResourceLimiter{
        limits:   make(map[string]ResourceLimits),
        monitors: make(map[string]*ResourceMonitor),
    }
}

func (rl *ResourceLimiter) SetLimits(plugin string, limits ResourceLimits) {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    rl.limits[plugin] = limits
    
    // Create or update monitor
    if monitor, exists := rl.monitors[plugin]; exists {
        close(monitor.stopChan)
    }
    
    monitor := &ResourceMonitor{
        plugin:     plugin,
        limits:     limits,
        stopChan:   make(chan struct{}),
        violations: make(chan ResourceViolation, 10),
    }
    
    rl.monitors[plugin] = monitor
    go monitor.Start()
}

func (m *ResourceMonitor) Start() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.checkResources()
        case <-m.stopChan:
            return
        }
    }
}

func (m *ResourceMonitor) checkResources() {
    // Check memory usage
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    
    currentMemMB := int(memStats.Alloc / 1024 / 1024)
    if m.limits.MaxMemoryMB > 0 && currentMemMB > m.limits.MaxMemoryMB {
        violation := ResourceViolation{
            Plugin:    m.plugin,
            Resource:  "memory",
            Current:   int64(currentMemMB),
            Limit:     int64(m.limits.MaxMemoryMB),
            Timestamp: time.Now(),
        }
        
        select {
        case m.violations <- violation:
        default:
            // Channel full, drop oldest
        }
    }
    
    // Check goroutine count
    if m.limits.MaxGoroutines > 0 {
        currentGoroutines := runtime.NumGoroutine()
        if currentGoroutines > m.limits.MaxGoroutines {
            violation := ResourceViolation{
                Plugin:    m.plugin,
                Resource:  "goroutines",
                Current:   int64(currentGoroutines),
                Limit:     int64(m.limits.MaxGoroutines),
                Timestamp: time.Now(),
            }
            
            select {
            case m.violations <- violation:
            default:
            }
        }
    }
}

// PerformanceTracer provides detailed performance tracing
type PerformanceTracer struct {
    traces map[string]*Trace
    mu     sync.RWMutex
}

type Trace struct {
    ID        string
    Plugin    string
    StartTime time.Time
    EndTime   time.Time
    Spans     []*Span
}

type Span struct {
    Name      string
    StartTime time.Time
    EndTime   time.Time
    Tags      map[string]string
    Children  []*Span
}

func NewPerformanceTracer() *PerformanceTracer {
    return &PerformanceTracer{
        traces: make(map[string]*Trace),
    }
}

func (t *PerformanceTracer) StartTrace(plugin string) *Trace {
    trace := &Trace{
        ID:        generateID(),
        Plugin:    plugin,
        StartTime: time.Now(),
    }
    
    t.mu.Lock()
    t.traces[trace.ID] = trace
    t.mu.Unlock()
    
    return trace
}

func (t *Trace) StartSpan(name string) *Span {
    span := &Span{
        Name:      name,
        StartTime: time.Now(),
        Tags:      make(map[string]string),
    }
    
    t.Spans = append(t.Spans, span)
    return span
}

func (s *Span) End() {
    s.EndTime = time.Now()
}

func (s *Span) SetTag(key, value string) {
    s.Tags[key] = value
}

func (s *Span) StartChildSpan(name string) *Span {
    child := &Span{
        Name:      name,
        StartTime: time.Now(),
        Tags:      make(map[string]string),
    }
    
    s.Children = append(s.Children, child)
    return child
}

// ObservabilityPlugin provides plugin observability
type ObservabilityPlugin struct {
    DomainPlugin
    
    metrics *PluginMetricsCollector
    tracer  *PerformanceTracer
    limiter *ResourceLimiter
}

func NewObservabilityPlugin(plugin DomainPlugin, metrics *PluginMetricsCollector, tracer *PerformanceTracer, limiter *ResourceLimiter) *ObservabilityPlugin {
    return &ObservabilityPlugin{
        DomainPlugin: plugin,
        metrics:      metrics,
        tracer:       tracer,
        limiter:      limiter,
    }
}

func (op *ObservabilityPlugin) GetPhases() []domain.Phase {
    phases := op.DomainPlugin.GetPhases()
    
    // Wrap each phase with observability
    wrappedPhases := make([]domain.Phase, len(phases))
    for i, phase := range phases {
        wrappedPhases[i] = &ObservablePhase{
            Phase:   phase,
            plugin:  op.Name(),
            metrics: op.metrics,
            tracer:  op.tracer,
        }
    }
    
    return wrappedPhases
}

// ObservablePhase wraps a phase with observability
type ObservablePhase struct {
    domain.Phase
    
    plugin  string
    metrics *PluginMetricsCollector
    tracer  *PerformanceTracer
}

func (op *ObservablePhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
    // Start trace span
    trace := op.tracer.StartTrace(op.plugin)
    span := trace.StartSpan(op.Name())
    defer span.End()
    
    // Record start time
    start := time.Now()
    
    // Execute phase
    output, err := op.Phase.Execute(ctx, input)
    
    // Record metrics
    duration := time.Since(start)
    op.metrics.ObservePhaseExecution(op.plugin, op.Name(), duration, err)
    
    // Add trace tags
    span.SetTag("success", fmt.Sprintf("%v", err == nil))
    if err != nil {
        span.SetTag("error", err.Error())
    }
    
    return output, err
}
```

## 8. Security

### Current State
- No plugin isolation
- No capability restrictions
- Limited validation

### Proposed Improvements

```go
// internal/domain/plugin/security.go
package plugin

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// SecurePlugin wraps plugins with security controls
type SecurePlugin struct {
    DomainPlugin
    
    capabilities []Capability
    validator    SecurityValidator
    auditor      SecurityAuditor
}

// Capability represents a plugin permission
type Capability string

const (
    CapabilityFileRead    Capability = "file:read"
    CapabilityFileWrite   Capability = "file:write"
    CapabilityNetworkAccess Capability = "network:access"
    CapabilityProcessSpawn  Capability = "process:spawn"
    CapabilitySystemInfo    Capability = "system:info"
)

// SecurityValidator validates plugin operations
type SecurityValidator interface {
    ValidateOperation(ctx context.Context, op Operation) error
    ValidateCapability(plugin string, cap Capability) error
}

// Operation represents a plugin operation
type Operation struct {
    Plugin     string
    Type       string
    Resource   string
    Action     string
    Context    map[string]interface{}
}

// SecurityAuditor logs security events
type SecurityAuditor interface {
    LogOperation(op Operation, allowed bool, reason string)
    LogViolation(plugin string, violation SecurityViolation)
}

// SecurityViolation represents a security policy violation
type SecurityViolation struct {
    Plugin      string
    Capability  Capability
    Resource    string
    Timestamp   time.Time
    Description string
}

// PluginSandbox provides isolated execution environment
type PluginSandbox struct {
    plugin       DomainPlugin
    capabilities map[Capability]bool
    filesystem   SandboxedFilesystem
    network      SandboxedNetwork
}

// SandboxedFilesystem restricts file access
type SandboxedFilesystem struct {
    allowedPaths []string
    tempDir      string
}

func NewSandboxedFilesystem(allowedPaths []string) (*SandboxedFilesystem, error) {
    tempDir, err := os.MkdirTemp("", "plugin-sandbox-*")
    if err != nil {
        return nil, fmt.Errorf("creating temp dir: %w", err)
    }
    
    return &SandboxedFilesystem{
        allowedPaths: allowedPaths,
        tempDir:      tempDir,
    }, nil
}

func (fs *SandboxedFilesystem) ValidatePath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("resolving path: %w", err)
    }
    
    // Check if path is within allowed directories
    for _, allowed := range fs.allowedPaths {
        if strings.HasPrefix(absPath, allowed) {
            return nil
        }
    }
    
    // Check if within sandbox temp directory
    if strings.HasPrefix(absPath, fs.tempDir) {
        return nil
    }
    
    return fmt.Errorf("access denied: path %s not in allowed directories", path)
}

// SandboxedNetwork restricts network access
type SandboxedNetwork struct {
    allowedHosts []string
    allowedPorts []int
}

func (n *SandboxedNetwork) ValidateConnection(host string, port int) error {
    // Check allowed hosts
    hostAllowed := false
    for _, allowed := range n.allowedHosts {
        if allowed == "*" || strings.HasSuffix(host, allowed) {
            hostAllowed = true
            break
        }
    }
    
    if !hostAllowed {
        return fmt.Errorf("host %s not allowed", host)
    }
    
    // Check allowed ports
    if len(n.allowedPorts) > 0 {
        portAllowed := false
        for _, allowed := range n.allowedPorts {
            if port == allowed {
                portAllowed = true
                break
            }
        }
        
        if !portAllowed {
            return fmt.Errorf("port %d not allowed", port)
        }
    }
    
    return nil
}

// PluginVerifier verifies plugin integrity
type PluginVerifier struct {
    trustedKeys map[string]string // plugin -> public key
    signatures  map[string]string // plugin -> signature
}

func NewPluginVerifier() *PluginVerifier {
    return &PluginVerifier{
        trustedKeys: make(map[string]string),
        signatures:  make(map[string]string),
    }
}

func (v *PluginVerifier) VerifyPlugin(pluginPath string) error {
    // Calculate plugin hash
    hash, err := v.calculateFileHash(pluginPath)
    if err != nil {
        return fmt.Errorf("calculating hash: %w", err)
    }
    
    // Get plugin name from path
    pluginName := filepath.Base(pluginPath)
    
    // Check signature
    expectedSig, ok := v.signatures[pluginName]
    if !ok {
        return fmt.Errorf("no signature found for plugin %s", pluginName)
    }
    
    if hash != expectedSig {
        return fmt.Errorf("signature mismatch for plugin %s", pluginName)
    }
    
    return nil
}

func (v *PluginVerifier) calculateFileHash(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return "", err
    }
    
    return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CapabilityManager manages plugin capabilities
type CapabilityManager struct {
    grants map[string]map[Capability]bool
    mu     sync.RWMutex
}

func NewCapabilityManager() *CapabilityManager {
    return &CapabilityManager{
        grants: make(map[string]map[Capability]bool),
    }
}

func (cm *CapabilityManager) GrantCapability(plugin string, cap Capability) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if cm.grants[plugin] == nil {
        cm.grants[plugin] = make(map[Capability]bool)
    }
    
    cm.grants[plugin][cap] = true
}

func (cm *CapabilityManager) RevokeCapability(plugin string, cap Capability) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if cm.grants[plugin] != nil {
        delete(cm.grants[plugin], cap)
    }
}

func (cm *CapabilityManager) HasCapability(plugin string, cap Capability) bool {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    if grants, ok := cm.grants[plugin]; ok {
        return grants[cap]
    }
    
    return false
}

func (cm *CapabilityManager) GetCapabilities(plugin string) []Capability {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    var caps []Capability
    if grants, ok := cm.grants[plugin]; ok {
        for cap, granted := range grants {
            if granted {
                caps = append(caps, cap)
            }
        }
    }
    
    return caps
}

// SecurePluginLoader adds security checks to plugin loading
type SecurePluginLoader struct {
    *PluginLoader
    verifier    *PluginVerifier
    capManager  *CapabilityManager
    auditor     SecurityAuditor
}

func NewSecurePluginLoader(loader *PluginLoader, verifier *PluginVerifier, capManager *CapabilityManager, auditor SecurityAuditor) *SecurePluginLoader {
    return &SecurePluginLoader{
        PluginLoader: loader,
        verifier:     verifier,
        capManager:   capManager,
        auditor:      auditor,
    }
}

func (sl *SecurePluginLoader) LoadPlugin(manifest *PluginManifest) error {
    // Verify plugin integrity
    if err := sl.verifier.VerifyPlugin(manifest.EntryPoint); err != nil {
        sl.auditor.LogViolation(manifest.Name, SecurityViolation{
            Plugin:      manifest.Name,
            Description: fmt.Sprintf("integrity check failed: %v", err),
            Timestamp:   time.Now(),
        })
        return fmt.Errorf("plugin verification failed: %w", err)
    }
    
    // Grant requested capabilities
    for _, cap := range manifest.Capabilities {
        capability := Capability(cap)
        
        // Check if capability is allowed
        if !sl.isCapabilityAllowed(capability) {
            return fmt.Errorf("capability %s not allowed", cap)
        }
        
        sl.capManager.GrantCapability(manifest.Name, capability)
    }
    
    // Load with base loader
    return sl.PluginLoader.LoadPlugin(manifest)
}

func (sl *SecurePluginLoader) isCapabilityAllowed(cap Capability) bool {
    // Define allowed capabilities based on security policy
    allowedCaps := map[Capability]bool{
        CapabilityFileRead:  true,
        CapabilityFileWrite: true,
        CapabilitySystemInfo: true,
        // Network and process spawn require explicit approval
        CapabilityNetworkAccess: false,
        CapabilityProcessSpawn:  false,
    }
    
    return allowedCaps[cap]
}
```

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)
1. Implement plugin manifest system
2. Add basic lifecycle management
3. Create plugin loader with discovery

### Phase 2: Core Features (Weeks 3-4)
1. Implement configuration management with hot reload
2. Add event bus for inter-plugin communication
3. Create resilience patterns (circuit breaker, retry)

### Phase 3: Developer Experience (Weeks 5-6)
1. Build plugin scaffolding tool
2. Create test harness and debugging utilities
3. Add comprehensive documentation

### Phase 4: Production Features (Weeks 7-8)
1. Implement monitoring and metrics
2. Add resource limiting
3. Create security sandboxing

### Phase 5: Polish & Testing (Weeks 9-10)
1. Performance optimization
2. Integration testing
3. Migration guide for existing plugins

## Migration Guide

### For Existing Plugins
1. Add `plugin.json` manifest
2. Implement lifecycle hooks (can be no-ops initially)
3. Update configuration to use new schema
4. Add capability declarations
5. Implement health checks

### Backward Compatibility
- Existing plugins work without modification
- New features are opt-in
- Gradual migration path available
- Legacy plugin wrapper provided

## Conclusion

These improvements transform the plugin system from a basic implementation to a production-ready, enterprise-grade architecture. The changes maintain backward compatibility while adding powerful new capabilities for plugin developers and system administrators.

Key benefits:
- **Dynamic Loading**: Plugins can be added without recompilation
- **Lifecycle Management**: Proper startup/shutdown with dependencies
- **Advanced Configuration**: Schema validation, hot reload, environment overrides
- **Communication**: Event-driven architecture with request/response patterns
- **Resilience**: Circuit breakers, retries, and fallbacks
- **Developer Experience**: Scaffolding, testing, debugging tools
- **Observability**: Comprehensive metrics and tracing
- **Security**: Capability-based permissions and sandboxing

The modular design allows teams to adopt improvements incrementally based on their needs.