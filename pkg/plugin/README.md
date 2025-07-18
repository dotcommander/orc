# Plugin System

This package provides a comprehensive plugin discovery, loading, and management system for the orchestrator. It includes:

- **Plugin Discovery**: Automatic discovery of plugins from XDG-compliant locations
- **Manifest System**: Rich metadata and capability description for plugins
- **Plugin Loading**: Support for built-in and external plugins (binaries and Go plugins)
- **Context Sharing**: Thread-safe context sharing mechanism for phases
- **Session Management**: Isolated contexts per execution session

## Features

### Plugin Discovery & Loading
- **XDG-compliant search paths**: Searches standard locations for plugins
- **Manifest-based**: Rich plugin metadata and capability description
- **Multiple plugin types**: Support for built-in, Go plugins (.so), and binary plugins
- **Domain filtering**: Find plugins by supported domains (fiction, code, docs)
- **Hot reload**: Reload plugins without restarting
- **Dependency management**: Track plugin dependencies

### Context Sharing
- **Thread-safe context storage**: Safe concurrent access to shared data
- **Type-safe accessors**: Convenient methods for common data types
- **Session management**: Isolated contexts per execution session
- **TTL and cleanup**: Automatic cleanup of expired contexts
- **JSON serialization**: Easy persistence and debugging
- **Phase integration**: Seamless integration with domain phases

## Quick Start

### Plugin Discovery and Loading

```go
import (
    "log/slog"
    "github.com/vampirenirmal/orchestrator/pkg/plugin"
    domainPlugin "github.com/vampirenirmal/orchestrator/internal/domain/plugin"
)

// Create a logger
logger := slog.Default()

// Create a discoverer
discoverer := plugin.NewDiscoverer(logger)

// Discover all plugins
manifests, err := discoverer.Discover()
if err != nil {
    log.Fatal(err)
}

// Create a registry and loader
registry := domainPlugin.NewDomainRegistry()
loader := plugin.NewLoader(logger, discoverer, registry)

// Load all plugins
if err := loader.LoadAll(); err != nil {
    log.Fatal(err)
}

// Use a specific plugin
fictionPlugin, err := registry.Get("fiction")
if err == nil {
    phases := fictionPlugin.GetPhases()
    // Execute phases...
}
```

### Context Sharing

```go
import "github.com/vampirenirmal/orchestrator/pkg/plugin"

// Create a context manager
contextManager := plugin.NewContextManager()
defer contextManager.Stop()

// Create a session context
sessionID := "my-session-123"
pluginCtx := contextManager.CreateContext(sessionID)

// Add to execution context
ctx := plugin.WithPluginContext(context.Background(), pluginCtx)

// Use in your phase
helper, err := plugin.NewPhaseContextHelper(ctx)
if err == nil {
    // Store data
    helper.SetMetadata("key", "value")
    
    // Retrieve data from previous phases
    output, _ := helper.GetPreviousPhaseOutput("Analysis")
}
```

### Integration with Domain Plugins

1. **Wrap your phases with context awareness**:

```go
phases := []domain.Phase{
    &MyAnalysisPhase{},
    &MyPlanningPhase{},
    &MyImplementationPhase{},
}

// Wrap phases to enable context sharing
wrappedPhases := plugin.WrapPhasesWithContext(phases)
```

2. **Use the helper in your phase implementation**:

```go
func (p *MyPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
    helper, err := plugin.NewPhaseContextHelper(ctx)
    if err != nil {
        // Context not available, execute normally
        return p.executeWithoutContext(ctx, input)
    }
    
    // Access previous phase output
    analysisData, err := helper.GetPreviousPhaseOutput("Analysis")
    if err == nil {
        // Use the data from analysis phase
    }
    
    // Store metadata for subsequent phases
    helper.SetMetadata("language", "go")
    helper.SetMetadata("framework", "gin")
    
    // Your phase logic here
    result := processData(input, analysisData)
    
    return domain.PhaseOutput{Data: result}, nil
}
```

## API Reference

### PluginContext

The core interface for storing and retrieving shared data:

```go
type PluginContext interface {
    // Basic operations
    Set(key string, value interface{})
    Get(key string) (interface{}, bool)
    Delete(key string)
    Clear()
    Keys() []string
    
    // Type-safe accessors
    GetString(key string) (string, error)
    GetInt(key string) (int, error)
    GetBool(key string) (bool, error)
    GetMap(key string) (map[string]interface{}, error)
    GetSlice(key string) ([]interface{}, error)
    
    // Utilities
    Clone() PluginContext
    MarshalJSON() ([]byte, error)
    UnmarshalJSON(data []byte) error
}
```

### ContextManager

Manages contexts across multiple sessions:

```go
// Create with options
manager := plugin.NewContextManager(
    plugin.WithTTL(24 * time.Hour),
    plugin.WithCleanupInterval(1 * time.Hour),
)

// Session management
ctx := manager.CreateContext(sessionID)
ctx, exists := manager.GetContext(sessionID)
manager.DeleteContext(sessionID)
sessions := manager.ListSessions()
```

### PhaseContextHelper

Convenience methods for phases:

```go
helper, _ := plugin.NewPhaseContextHelper(ctx)

// Access previous phase outputs
output, _ := helper.GetPreviousPhaseOutput("Analysis")

// Manage shared metadata
helper.SetMetadata("key", "value")
value, _ := helper.GetMetadata("key")
```

### SharedData

Structured data shared between phases:

```go
type SharedData struct {
    PhaseOutputs map[string]interface{}  // Outputs by phase name
    Metadata     map[string]interface{}  // Global metadata
    Errors       []PhaseError            // Accumulated errors
    Metrics      PhaseMetrics            // Performance metrics
}
```

## Best Practices

1. **Always check for context availability**: Phases should work with or without context
2. **Use meaningful keys**: Use descriptive names for stored values
3. **Clean up large data**: Remove large intermediate results when no longer needed
4. **Handle type assertions carefully**: Use the type-safe accessors when possible
5. **Document shared data**: Clearly document what data your phase expects and produces

## Example: Code Generation Plugin

```go
// Analysis phase stores project requirements
func (a *AnalysisPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
    helper, _ := plugin.NewPhaseContextHelper(ctx)
    
    // Analyze the request
    analysis := analyzeRequest(input.Request)
    
    // Store for other phases
    helper.SetMetadata("language", analysis.Language)
    helper.SetMetadata("framework", analysis.Framework)
    helper.SetMetadata("requirements", analysis.Requirements)
    
    return domain.PhaseOutput{Data: analysis}, nil
}

// Planning phase uses analysis results
func (p *PlanningPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
    helper, _ := plugin.NewPhaseContextHelper(ctx)
    
    // Get analysis results
    language, _ := helper.GetMetadata("language")
    framework, _ := helper.GetMetadata("framework")
    
    // Create plan based on analysis
    plan := createPlan(language.(string), framework.(string))
    
    return domain.PhaseOutput{Data: plan}, nil
}
```

## Thread Safety

All operations on PluginContext are thread-safe. Multiple phases can safely read and write to the same context concurrently.

## Performance Considerations

- Context operations are protected by RWMutex for optimal read performance
- Clone operations use JSON marshaling for deep copying
- Cleanup runs periodically to remove expired contexts
- Consider data size when storing in context

## Plugin Manifest Format

Plugins are described by manifest files (`plugin.yaml` or `plugin.json`):

```yaml
# Metadata
name: "my-plugin"
version: "1.0.0"
description: "A custom plugin for advanced processing"
author: "Plugin Author"
license: "MIT"

# Type and compatibility
type: "external"  # or "builtin"
min_version: "1.0.0"  # Min orchestrator version

# Capabilities
domains:
  - fiction
  - docs

phases:
  - name: "analyze"
    description: "Analyze input"
    order: 1
    required: true
    timeout: 10m
    retryable: true
    max_retries: 3

# Prompts and configuration
prompts:
  analyze: "prompts/analyze.txt"

output_spec:
  primary_output: "output.md"
  descriptions:
    output.md: "Main output file"

# Entry point for external plugins
entry_point: "my-plugin"  # Binary name or .so file
binary: true  # false for .so plugins
```

## Plugin Search Paths

Plugins are discovered from these locations (in order):

1. **Built-in**: `<binary_dir>/../share/orchestrator/plugins/`
2. **User data**: `~/.local/share/orchestrator/plugins/`
3. **System**: `/usr/local/share/orchestrator/plugins/`, `/usr/share/orchestrator/plugins/`
4. **User config**: `~/.config/orchestrator/plugins/`
5. **Development**: `./plugins/` (current directory)

## Creating a Plugin

### Built-in Plugin

1. Implement the `DomainPlugin` interface
2. Register in the main binary at compile time
3. Create a manifest file in the built-in plugins directory

### External Go Plugin

1. Create a Go module implementing `DomainPlugin`
2. Build as a plugin: `go build -buildmode=plugin`
3. Create a manifest with `binary: false`
4. Place .so file and manifest in a plugin directory

### External Binary Plugin

1. Create an executable that handles these commands:
   - `execute <phase_name>`: Execute a phase (input via stdin, output to stdout)
   - `validate`: Validate the plugin
   - `info`: Return plugin information
2. Create a manifest with `binary: true`
3. Place executable and manifest in a plugin directory

## Testing

See `example_test.go` for comprehensive examples of:
- Plugin discovery and loading
- Manifest creation and validation
- Context sharing between phases
- Session isolation
- Error handling

## Examples

See the `examples/` directory for:
- `plugin-manifest.yaml`: Full-featured external plugin manifest
- `builtin-plugin-manifest.yaml`: Simple built-in plugin manifest