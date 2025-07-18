# Plugin Development Guide

This guide explains how to create plugins for Orc.

## Quick Start

1. **Copy a template plugin**:
   ```bash
   cp -r plugins/fiction plugins/my-plugin
   cd plugins/my-plugin
   ```

2. **Update plugin metadata**:
   Edit `manifest.yaml` with your plugin information

3. **Implement your plugin**:
   Edit `plugin.go` to implement the Plugin interface

4. **Build and test**:
   ```bash
   make build
   make test
   ```

5. **Install**:
   ```bash
   make install
   ```

## Plugin Structure

```
my-plugin/
├── go.mod              # Go module (independent versioning)
├── plugin.go           # Main plugin implementation
├── manifest.yaml       # Plugin metadata
├── Makefile           # Build configuration
├── phases/            # Phase implementations
│   ├── phase1.go
│   └── phase2.go
└── prompts/           # AI prompt templates
    ├── phase1.txt
    └── phase2.txt
```

## Implementing the Plugin Interface

Your plugin must implement the `orc.Plugin` interface:

```go
package myplugin

import (
    "github.com/dotcommander/orc/pkg/orc"
)

type Plugin struct {
    // Your fields
}

func (p *Plugin) GetInfo() orc.PluginInfo {
    return orc.PluginInfo{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "My awesome plugin",
        Author:      "Your Name",
        Domains:     []string{"mydomain"},
    }
}

func (p *Plugin) CreatePhases() ([]orc.Phase, error) {
    // Return your phase implementations
    return []orc.Phase{
        NewPhase1(),
        NewPhase2(),
    }, nil
}

// ... implement other required methods
```

## Implementing Phases

Each phase must implement the `orc.Phase` interface:

```go
type MyPhase struct {
    agent   orc.Agent
    storage orc.Storage
}

func (p *MyPhase) Name() string {
    return "MyPhase"
}

func (p *MyPhase) Execute(ctx context.Context, input orc.PhaseInput) (orc.PhaseOutput, error) {
    // Your phase logic here
    
    // Use the AI agent
    response, err := p.agent.Execute(ctx, "Your prompt", input.Data)
    if err != nil {
        return orc.PhaseOutput{Error: err}, err
    }
    
    // Save results
    err = p.storage.SaveOutput(input.SessionID, "output.txt", []byte(response))
    
    return orc.PhaseOutput{
        Data: response,
    }, nil
}

// ... implement other required methods
```

## Using the Plugin SDK

The plugin SDK provides utilities to simplify plugin development:

```go
import "github.com/dotcommander/orc/pkg/plugin-sdk"

type MyPlugin struct {
    sdk.BasePlugin
}

func NewPlugin() *MyPlugin {
    p := &MyPlugin{}
    p.BasePlugin = sdk.NewBasePlugin(
        "my-plugin",
        "1.0.0", 
        "Description",
        "Author",
        []string{"mydomain"},
    )
    return p
}
```

## Plugin Types

### 1. Go Plugins (.so files)
Built as shared libraries and loaded dynamically:
```makefile
build:
    go build -buildmode=plugin -o my-plugin.so .
```

### 2. Binary Plugins
Standalone executables that communicate via JSON-RPC:
```go
func main() {
    plugin := NewPlugin()
    sdk.ServeBinaryPlugin(plugin)
}
```

## Configuration

Plugins can define configuration in their manifest:

```yaml
config_schema:
  type: object
  properties:
    my_option:
      type: string
      default: "value"
    feature_enabled:
      type: boolean
      default: true
```

Users configure plugins in their Orc config:

```yaml
plugins:
  my-plugin:
    my_option: "custom value"
    feature_enabled: false
```

## Best Practices

1. **Error Handling**: Always return meaningful errors
2. **Timeouts**: Respect context cancellation
3. **Logging**: Use the provided logger interface
4. **Testing**: Write tests for your phases
5. **Documentation**: Include clear prompts and examples

## Testing Your Plugin

```go
func TestMyPhase(t *testing.T) {
    // Create mock dependencies
    mockAgent := &MockAgent{}
    mockStorage := &MockStorage{}
    
    // Create phase
    phase := NewMyPhase(mockAgent, mockStorage)
    
    // Test execution
    output, err := phase.Execute(context.Background(), orc.PhaseInput{
        Request: "test request",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, output.Data)
}
```

## Distribution

1. **GitHub Release**: Publish compiled plugins
2. **Plugin Registry**: Submit to Orc plugin registry
3. **Documentation**: Include usage examples

## Example Plugins

See the `plugins/` directory for examples:
- `fiction/` - Novel generation plugin
- `code/` - Code generation plugin

## Getting Help

- Check existing plugins for patterns
- Read the [API documentation](../pkg/orc/)
- Open an issue for questions