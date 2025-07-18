# CLAUDE.md - AI Assistant Navigation Hub

**Project**: The Orchestrator - AI Novel Generation Orchestrator  
**Location**: `/Users/vampire/go/src/orchestrator`  
**Repository**: `github.com/vampirenirmal/orchestrator`  
**Go Module**: `github.com/vampirenirmal/orchestrator`  
**Architecture**: Clean Architecture with Phase-based Orchestration  

## Quick Context for AI Assistants

The Orchestrator is a **Go application that orchestrates multiple AI agents** to generate complete content through structured, quality-focused processes. The application uses clean architecture with interface-driven design, dependency injection, and revolutionary iterator agent technology for infinite quality improvement.

### Core Concept
- **Iterator Agent Architecture**: Infinite quality convergence until all criteria pass
- **Multi-Domain Generation**: Fiction (novels), Code (applications), Documentation
- **Quality-First Approach**: Extended timeouts and verification loops prioritize thoroughness
- **"Be Like Water" Philosophy**: Adaptive orchestration that flows naturally with AI capabilities
- **Proven Results**: Successfully generates PHP, JavaScript, Go code with quality verification

## Project Structure & Navigation

### Key Directories
```
orc/
├── cmd/orchestrator/main.go          # Entry point - dependency wiring
├── internal/                    # Private application code
│   ├── orchestrator/           # Core orchestration logic
│   ├── phases/                 # Individual phase implementations
│   ├── agent/                  # AI client abstraction
│   ├── config/                 # XDG-compliant configuration
│   └── storage/                # Storage abstraction
├── docs/                       # Comprehensive documentation
│   ├── architecture.md         # Detailed system design
│   ├── api.md                  # Interface documentation
│   ├── development.md          # Contribution guidelines
│   ├── configuration.md        # Complete config reference
│   └── examples/               # Code examples and patterns
├── prompts/                    # AI prompt templates
├── scripts/                    # Installation utilities
├── concept.md                  # Architecture assessment & migration plan
├── concept2.md                 # Additional architectural insights
└── config.yaml.example        # Configuration template
```

### Critical Files for AI Development

| File | Purpose | When to Edit |
|------|---------|--------------|
| `internal/core/iterator.go` | Iterator agents for infinite quality improvement | Adding quality criteria or convergence logic |
| `internal/phase/code/conversational_explorer.go` | Natural dialogue for project requirements | Improving requirement gathering |
| `internal/phase/code/incremental_builder.go` | Systematic incremental code building | Enhancing code generation logic |
| `internal/phase/code/iterative_refiner.go` | Quality-driven infinite improvement | Adding domain-specific inspectors |
| `internal/phase/utils.go` | JSON parsing and AI response utilities | Fixing AI response parsing issues |
| `internal/core/fluid_orchestrator.go` | "Be Like Water" adaptive orchestration | Modifying adaptive behavior |
| `internal/core/verification.go` | Stage verification with retry logic | Adding verification criteria |
| `internal/agent/agent.go` | AI client implementation | Modifying AI interactions |
| `internal/config/config.go` | XDG-compliant configuration | Adding config options |

## Development Context

### Current State (Post-Iterator Architecture + Plugin System)
- ✅ **Iterator Agent System**: Infinite quality improvement until all criteria pass
- ✅ **Multi-Domain Generation**: Fiction, Code (PHP/JS/Go), Documentation
- ✅ **Enhanced Plugin System**: Auto-discovery of external plugins with built-in domain plugins
- ✅ **Plugin Discovery**: XDG-compliant paths with configurable discovery locations
- ✅ **External Plugin Support**: Shared object (.so) and executable plugin loading
- ✅ **Plugin Configuration**: Per-plugin settings, timeouts, and enable/disable controls
- ✅ **Quality-First Configuration**: 5min AI timeouts, 8-20min phase timeouts, 30 req/min rate limiting
- ✅ **Robust JSON Parsing**: CleanJSONResponse handles malformed AI responses
- ✅ **Verification Loops**: Automatic retry with issue tracking to `/issues` directory
- ✅ **Fluid Orchestration**: "Be Like Water" adaptive execution with verification
- ✅ **Language Enforcement**: Explicit language constraints properly respected
- ✅ **Model Specification**: User model choices (gpt-4.1) never overridden
- ✅ **XDG Compliance**: Follows XDG Base Directory Specification
- ✅ **Proven Code Generation**: Successfully generates PHP, React, Go applications
- ✅ **Enhanced Prompts**: Professional-grade prompts following Anthropic 2025 best practices

### Key Implementation Patterns

#### 1. Phase Interface
```go
type Phase interface {
    Name() string
    Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
    Validate(input PhaseInput) error
    EstimatedDuration() time.Duration
    CanRetry(err error) bool
}
```

#### 2. Dependency Injection (main.go)
All dependencies wired in `cmd/orchestrator/main.go`:
- Storage abstraction
- AI client with retries
- Phase implementations
- Configuration management

#### 3. Error Handling Strategy
- **PhaseError**: Structured error type with retry information
- **Circuit Breaker**: Prevents cascade failures
- **Exponential Backoff**: Rate-limited retries
- **Terminal vs Retryable**: Error classification

## Configuration & Paths (XDG Compliant)

### File Locations
- **Config**: `~/.config/orchestrator/config.yaml`
- **Data**: `~/.local/share/orchestrator/`
- **Prompts**: `~/.local/share/orchestrator/prompts/`
- **Output**: `~/.local/share/orchestrator/output/`
- **Plugins**: `~/.local/share/orchestrator/plugins/` (built-in and external)
- **Binary**: `~/.local/bin/orchestrator`
- **Logs**: `~/.local/state/orchestrator/` (error/debug logs)

### Environment Variables
- `OPENAI_API_KEY` - OpenAI API key (required for gpt-4.1 model)
- `XDG_CONFIG_HOME` - Override config directory
- `XDG_DATA_HOME` - Override data directory
- `REFINER_CONFIG` - Override config file path

### **CRITICAL MODEL SPECIFICATION RULE**
- **NEVER question, correct, or override user-specified model names**
- User knows their available models better than you do
- Older/deprecated models may cost more but are user's choice
- Always trust user specifications even if you think it's a typo

## Common Development Tasks

### Adding a New Built-in Plugin
1. Create `internal/domain/plugin/newplugin.go`
2. Implement `DomainPlugin` interface with phases
3. Register in `internal/plugin/integration.go` 
4. Add default configuration in `config.yaml.example`
5. Update plugin documentation

### Adding a New Phase to Existing Plugin
1. Create phase in `internal/phase/domain/newphase.go`
2. Implement `Phase` interface
3. Add to plugin's `GetPhases()` method
4. Create prompt template in prompts directory
5. Update plugin tests

### Creating an External Plugin
1. Choose plugin type: shared object (.so) or executable
2. Implement plugin interface (see plugin development guide)
3. Install to `~/.local/share/orchestrator/plugins/external/`
4. Configure in `config.yaml` if needed
5. Test with `orc list plugins`

### Modifying AI Interactions
- **Client Logic**: `internal/agent/agent.go`
- **Retry Strategy**: `internal/agent/client.go`
- **Prompt Templates**: `prompts/*.txt`
- **Caching**: `internal/agent/prompt_cache.go`

### Testing Strategy
- **Unit Tests**: Test individual phases with mock AI client
- **Integration Tests**: Full pipeline with testcontainers
- **Benchmarks**: Worker pool performance validation

## Working Memory for AI Sessions

### Current Session Context (2025-07-17)
```
Session: Documentation Structure Organization
Task: Organize and interconnect all orchestrator documentation
Agent: Structure Organizer (5 specialist agents completed)
Discovered:
- Logger initialization fixes needed in main.go
- Current working directory is /Users/vampire/go/src/orc
- Domain plugin migration in progress
- Comprehensive error catalog completed
- File location map updated
- Execution flows documented
- Code patterns identified

Modified Files:
- CLAUDE.md (this file) - Updated navigation
- docs/paths.md - Complete file location reference
- docs/errors.md - Comprehensive error catalog
- docs/flow.md - Execution flow documentation
- docs/patterns.md - Code conventions and patterns

Next Steps:
- Cross-reference all documentation
- Validate navigation links
- Update troubleshooting guides
```

### Session Tracking Template
```
Current Session: [SESSION_ID]
Task: [BRIEF_DESCRIPTION]
Modified Files: [LIST]
Next Steps: [PLANNED_ACTIONS]
Blocked On: [ISSUES]
```

### Common Commands
```bash
# Build and test
make build && make test

# Install locally (XDG compliant)
make install

# Content generation with enhanced prompts
./orc create fiction "Write a sci-fi thriller about AI consciousness"
./orc create code "Create a REST API in Go with authentication"

# Quality-focused generation (recommended)
./orc create code "ONLY USE PHP. Create hello.php that echoes Hello World" --fluid --verbose

# Plugin management
./orc list plugins              # List all available plugins
./orc list sessions            # List previous sessions

# Resume interrupted sessions
./orc resume SESSION_ID

# Check configuration and logs
./orc config get ai.model
./orc config get plugins.settings.auto_discovery
tail -f ~/.local/state/orchestrator/debug.log
```

## Plugin System Architecture

### Built-in Domain Plugins
- **Fiction Plugin**: Novel and story generation with quality iterations
- **Code Plugin**: Code analysis, generation, and quality refinement
- **Future**: Documentation, API, Testing, and other specialized plugins

### External Plugin Support
- **Shared Object Plugins**: Go-compiled .so files with plugin interface
- **Executable Plugins**: Standalone binaries following plugin protocol
- **Auto-Discovery**: Automatic scanning of configured plugin directories
- **Configuration**: Per-plugin settings, timeouts, and enable/disable controls

### Plugin Discovery Paths (searched in order)
1. `~/.local/share/orchestrator/plugins/builtin` (built-in plugins)
2. `~/.local/share/orchestrator/plugins/external` (user-installed plugins)
3. `/usr/local/lib/orchestrator/plugins` (system-wide plugins)
4. `/usr/lib/orchestrator/plugins` (distribution plugins)
5. Additional paths from system PATH (for executable plugins)

### Plugin Configuration Structure
```yaml
plugins:
  discovery_paths: [...]           # Plugin search paths
  settings:
    auto_discovery: true           # Enable automatic plugin discovery
    max_external_plugins: 10       # Limit external plugin loading
  configurations:
    plugin_name:
      enabled: true               # Enable/disable specific plugins
      settings: {...}             # Plugin-specific configuration
      timeouts: {...}             # Custom timeout overrides
```

## Architecture Decision Records

### Design Principles
1. **Interface Segregation**: Each component has minimal, focused interfaces
2. **Dependency Inversion**: High-level modules don't depend on low-level modules
3. **Single Responsibility**: Each phase handles one aspect of novel generation
4. **Open/Closed**: Easy to add new phases without modifying existing code

### Technology Choices
- **Go 1.21+**: Performance, concurrency, static typing
- **Structured Logging**: `log/slog` for observability
- **Validation**: `validator.v10` for configuration
- **Rate Limiting**: `golang.org/x/time/rate` for API management
- **Concurrency**: `golang.org/x/sync/errgroup` for worker pools

## Integration Points

### With README.md
- README provides user-facing documentation
- CLAUDE.md provides AI assistant context
- Both reference same configuration and structure

### With Existing User Preferences
- **XDG Compliance**: Follows user's global preferences
- **Go Binary Linking**: Uses `go/bin` symlink pattern
- **Script Organization**: Utilities in `scripts/` subdirectory
- **Debug Logging**: XDG-compliant error logging

## Documentation Navigation

### For AI Assistants (Recommended Reading Order)

1. **[CLAUDE.md](CLAUDE.md)** (this file) - Primary navigation hub
2. **[docs/paths.md](docs/paths.md)** - Complete file location reference
3. **[docs/errors.md](docs/errors.md)** - Comprehensive error catalog and solutions
4. **[docs/flow.md](docs/flow.md)** - Execution flows and system interactions
5. **[docs/patterns.md](docs/patterns.md)** - Code conventions and architectural patterns
6. **[orchestrator_flow_diagram.md](orchestrator_flow_diagram.md)** - Visual flow diagrams
7. **[docs/technical.md](docs/technical.md)** - Technical implementation details
8. **[docs/examples/](docs/examples/)** - Code examples and usage patterns

### For Specific Tasks

| Task | Primary Documentation | Supporting Files |
|------|----------------------|------------------|
| **Finding Files** | `docs/paths.md` | `CLAUDE.md` (this file) |
| **Troubleshooting Errors** | `docs/errors.md` | `~/.local/state/orchestrator/debug.log` |
| **Understanding Architecture** | `docs/flow.md`, `orchestrator_flow_diagram.md` | `concept.md`, `concept2.md` |
| **Code Patterns & Conventions** | `docs/patterns.md` | `internal/*/` directories |
| **Implementing Features** | `docs/patterns.md` → `docs/examples/` | `internal/core/`, `internal/phase/` |
| **Modifying Configuration** | `docs/configuration.md` | `config.yaml.example`, `docs/paths.md` |
| **Adding New Phases** | `docs/patterns.md` → `docs/examples/` | `internal/phases/*/` |
| **Testing and Debugging** | `docs/errors.md` | `docs/examples/`, debug logs |
| **System Flows** | `docs/flow.md`, `orchestrator_flow_diagram.md` | Phase execution traces |

## Quick Start for AI Assistants

1. **Read this file first** - Understand project context
2. **Check `docs/architecture.md`** - Detailed system design
3. **Review `docs/api.md`** - Interface contracts
4. **Study `docs/examples/`** - Implementation patterns
5. **Test locally** - `make test` before modifications

## Troubleshooting Guide

### Common Issues
- **JSON Parsing Errors**: `invalid character '\n'` → Use CleanJSONResponse utility
- **Language Recognition**: AI generates wrong language → Use explicit constraints
- **Model Override**: Shows gpt-4 not gpt-4.1 → Check model specification rules
- **Quality Issues**: Code too fast/shallow → Use --fluid flag with extended timeouts
- **Build Failures**: Check Go version (1.21+) and dependencies
- **Config Errors**: Verify XDG paths and API key setup

### Debug Process
1. **Check errors first**: See [docs/errors.md](docs/errors.md) for known solutions
2. **Locate files**: Use [docs/paths.md](docs/paths.md) for file locations
3. **Understand flow**: Check [docs/flow.md](docs/flow.md) for execution patterns
4. **Enable verbose logging**: `./orc create code "..." --fluid --verbose`
5. **Check debug log**: `tail -f ~/.local/state/orchestrator/debug.log`
6. **Validate configuration**: Check `~/.config/orchestrator/config.yaml`
7. **Test AI connectivity**: Verify API key and model access
8. **Review patterns**: Use [docs/patterns.md](docs/patterns.md) for code conventions

### Quality Best Practices
- **Use explicit language constraints**: "ONLY USE PHP. No JavaScript, no React"
- **Prioritize quality over speed**: Use --fluid flag for better results
- **Always parse AI responses safely**: Use CleanJSONResponse for all JSON
- **Respect user model choices**: Never override model specifications

---

## 📖 Documentation Organization Summary

The orchestrator documentation is now fully organized and cross-referenced:

### Navigation Hub
- **[CLAUDE.md](CLAUDE.md)** (this file) - Primary navigation for AI assistants

### Core Reference Documents
- **[docs/paths.md](docs/paths.md)** - Complete file location map
- **[docs/errors.md](docs/errors.md)** - Comprehensive error catalog with solutions  
- **[docs/flow.md](docs/flow.md)** - Execution flows and system interactions
- **[docs/patterns.md](docs/patterns.md)** - Code conventions and architectural patterns

### Supporting Documentation  
- **[orchestrator_flow_diagram.md](orchestrator_flow_diagram.md)** - Visual flow diagrams
- **[docs/technical.md](docs/technical.md)** - Technical implementation details
- **[docs/configuration.md](docs/configuration.md)** - Configuration and setup
- **[docs/examples/](docs/examples/)** - Code examples and usage patterns

### Documentation Features
- ✅ **Cross-Referenced**: All files link to related documentation
- ✅ **Current Session Context**: Working memory updated with recent discoveries
- ✅ **Error Solutions**: Comprehensive troubleshooting with file locations
- ✅ **Quick Navigation**: Task-specific documentation guides
- ✅ **Implementation Maps**: Clear paths from problems to solutions

---

**For Humans**: See `README.md` for installation and usage instructions.  
**For AI Assistants**: Use this file as your primary navigation hub.

Last Updated: 2025-07-17 (Documentation Structure Organization + Iterator Agent Architecture)