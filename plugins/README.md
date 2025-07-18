# Orc Plugins

This directory contains the official Orc plugins for content generation.

## Available Plugins

### Fiction Plugin
Generate novels, short stories, and other fiction content.

```bash
cd fiction
make build
make install
```

### Code Plugin
Generate code, applications, and technical documentation.

```bash
cd code
make build
make install
```

## Plugin Development

To create a new plugin:

1. Copy one of the existing plugins as a template
2. Update the `plugin.go` file with your domain logic
3. Create phase implementations in the `phases/` directory
4. Add prompt templates to the `prompts/` directory
5. Update the `manifest.yaml` with your plugin metadata
6. Build and test your plugin

### Plugin Structure

```
my-plugin/
├── go.mod                 # Go module definition
├── plugin.go              # Main plugin implementation
├── manifest.yaml          # Plugin metadata
├── Makefile              # Build configuration
├── phases/               # Phase implementations
│   ├── planner.go
│   ├── generator.go
│   └── refiner.go
└── prompts/              # AI prompt templates
    ├── planner.txt
    ├── generator.txt
    └── refiner.txt
```

### Building Plugins

Plugins can be built in two modes:

1. **Shared Library** (.so file) - For Go plugins
   ```bash
   make build
   ```

2. **Standalone Binary** - For testing or cross-language plugins
   ```bash
   make build-binary
   ```

### Installing Plugins

Install to the user plugin directory:
```bash
make install
```

This copies the plugin and manifest to `~/.local/share/orchestrator/plugins/`

### Testing Plugins

Run the plugin test suite:
```bash
make test
```

## Plugin API

Plugins must implement the `orc.Plugin` interface:

```go
type Plugin interface {
    GetInfo() PluginInfo
    CreatePhases() ([]Phase, error)
    ValidateRequest(request string) error
    GetOutputSpec() OutputSpec
    GetPhaseTimeouts() map[string]time.Duration
    GetRequiredConfig() []string
    GetDefaultConfig() map[string]interface{}
}
```

See the [Plugin SDK documentation](../pkg/plugin-sdk/README.md) for more details.

## Plugin Discovery

Orc automatically discovers plugins in these locations:

1. Built-in plugins directory: `plugins/`
2. User plugins: `~/.local/share/orchestrator/plugins/`
3. System plugins: `/usr/local/share/orchestrator/plugins/`
4. Custom paths via configuration

## Configuration

Plugins can be configured in the main Orc config:

```yaml
plugins:
  fiction:
    chapter_word_target: 3000
    quality_threshold: 0.8
  code:
    language_detection: true
    test_generation: true
```