# Plugin Context Sharing Implementation

## Overview

This document describes the plugin context sharing system that was implemented as one of the quick wins from the orchestrator improvements plan. The system allows phases within a plugin to share data and communicate during execution.

## Implementation Files

### Core Components

1. **`pkg/plugin/context.go`** - Core context interface and implementation
   - Thread-safe `PluginContext` interface
   - Type-safe accessors for common data types
   - JSON serialization support
   - Context injection into `context.Context`

2. **`pkg/plugin/manager.go`** - Session and lifecycle management
   - `ContextManager` for managing multiple session contexts
   - TTL-based cleanup of expired contexts
   - `SharedData` structure for phase outputs and metadata
   - Performance metrics tracking

3. **`pkg/plugin/integration.go`** - Integration with existing domain system
   - `ContextAwarePhase` wrapper for automatic context sharing
   - `PhaseContextHelper` for easy access to shared data
   - Helper functions for wrapping existing phases

4. **`pkg/plugin/runner_integration.go`** - Enhanced plugin runner
   - `ContextAwarePluginRunner` extending the domain plugin runner
   - Automatic context creation and management
   - Intermediate result persistence

5. **`pkg/plugin/example_test.go`** - Comprehensive examples
   - Demonstrates context sharing between phases
   - Shows session isolation
   - Provides testing patterns

6. **`pkg/plugin/README.md`** - Complete documentation
   - API reference
   - Usage examples
   - Best practices

## Key Features

### 1. Thread-Safe Context Storage
```go
ctx := plugin.NewPluginContext()
ctx.Set("key", "value")
value, exists := ctx.Get("key")
```

### 2. Type-Safe Accessors
```go
ctx.Set("count", 42)
count, err := ctx.GetInt("count")

ctx.Set("enabled", true)
enabled, err := ctx.GetBool("enabled")
```

### 3. Phase Data Sharing
```go
// In Analysis phase
helper.SetMetadata("language", "go")

// In Planning phase
language, _ := helper.GetMetadata("language")
```

### 4. Session Management
```go
manager := plugin.NewContextManager()
ctx := manager.CreateContext(sessionID)
// Context isolated per session
```

### 5. Automatic Cleanup
```go
manager := plugin.NewContextManager(
    plugin.WithTTL(24 * time.Hour),
    plugin.WithCleanupInterval(1 * time.Hour),
)
```

## Integration with Existing System

The context sharing system integrates seamlessly with the existing domain plugin architecture:

1. **Minimal Changes Required**: Existing phases can be wrapped with `WrapPhasesWithContext()`
2. **Backward Compatible**: Phases work with or without context
3. **Storage Integration**: Automatic persistence of context and intermediate results
4. **Performance Tracking**: Built-in metrics for phase execution times

## Usage Example

```go
// 1. Create context manager
contextManager := plugin.NewContextManager()
defer contextManager.Stop()

// 2. Wrap your phases
wrappedPhases := plugin.WrapPhasesWithContext(phases)

// 3. Create session context
sessionID := "user-session-123"
pluginCtx := contextManager.CreateContext(sessionID)

// 4. Execute with context
ctx := plugin.WithPluginContext(context.Background(), pluginCtx)
```

## Benefits

1. **Improved Phase Communication**: Phases can easily share data without complex parameter passing
2. **Better Debugging**: All phase outputs and metadata are tracked
3. **Session Persistence**: Context can be saved and restored across executions
4. **Performance Insights**: Automatic tracking of phase execution times
5. **Type Safety**: Compile-time type checking with generic accessors

## Next Steps

To fully integrate this system:

1. Update the main orchestrator to use `ContextAwarePluginRunner`
2. Modify existing plugins to leverage context sharing
3. Add context persistence to storage layer
4. Implement context visualization for debugging
5. Add metrics export for monitoring

This implementation provides a solid foundation for inter-phase communication and sets the stage for more advanced features like parallel phase execution and dynamic phase orchestration.