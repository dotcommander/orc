# File Path Reference
**Last Updated**: 2025-07-17  
**Purpose**: Complete file location map for the orchestrator project  
**Cross-References**: See [flow.md](flow.md) for execution flows and [patterns.md](patterns.md) for code conventions

> The answer to "where is X?" is always here.

## Quick Lookups
| Looking for | Location | Purpose |
|------------|----------|---------|
| Main entry | `/Users/vampire/go/src/orc/cmd/orc/main.go` | CLI application start with dependency injection |
| Iterator agents | `/Users/vampire/go/src/orc/internal/core/iterator.go` | Infinite quality improvement until criteria met |
| JSON parsing fixes | `/Users/vampire/go/src/orc/internal/phase/utils.go` | CleanJSONResponse and malformed JSON recovery |
| Code generation | `/Users/vampire/go/src/orc/internal/phase/code/` | ConversationalExplorer, IncrementalBuilder, IterativeRefiner |
| Fluid orchestrator | `/Users/vampire/go/src/orc/internal/core/fluid_orchestrator.go` | "Be Like Water" adaptive orchestration |
| Configuration | `/Users/vampire/.config/orchestrator/config.yaml` | Runtime config with timeouts and quality settings |
| Debug logs | `/Users/vampire/.local/state/orchestrator/debug.log` | Error and debug logging |

## By Feature

### Iterator Agent Architecture
- **Core Engine**: `/Users/vampire/go/src/orc/internal/core/iterator.go` - IteratorAgent with infinite quality convergence
- **Improvement Engine**: `/Users/vampire/go/src/orc/internal/core/iterative_improvement.go` - Combines inspectors and iterators
- **Inspector System**: `/Users/vampire/go/src/orc/internal/core/inspector.go` - Deep quality analysis agents
- **Documentation**: `/Users/vampire/go/src/orc/ITERATOR_ARCHITECTURE.md` - Complete architectural guide

### Code Generation Plugin
- **Conversational Explorer**: `/Users/vampire/go/src/orc/internal/phase/code/conversational_explorer.go` - Natural dialogue for requirements
- **Incremental Builder**: `/Users/vampire/go/src/orc/internal/phase/code/incremental_builder.go` - Systematic code building
- **Iterative Refiner**: `/Users/vampire/go/src/orc/internal/phase/code/iterative_refiner.go` - Quality-driven infinite improvement
- **JSON Utilities**: `/Users/vampire/go/src/orc/internal/phase/utils.go` - Robust AI response parsing

### Quality & Verification System
- **Stage Verifier**: `/Users/vampire/go/src/orc/internal/core/verification.go` - Quality checks with retry logic
- **Adaptive Errors**: `/Users/vampire/go/src/orc/internal/core/adaptive_errors.go` - Intelligent error handling
- **Issue Tracking**: Issues documented in `~/.local/share/orchestrator/output/sessions/*/issues/`
- **Quality Criteria**: Built into iterator agents with configurable thresholds

### Configuration & Timeouts
- **Main Config**: `/Users/vampire/go/src/orc/internal/config/config.go` - XDG-compliant configuration loader
- **Runtime Config**: `/Users/vampire/.config/orchestrator/config.yaml` - User timeout and quality settings
- **Example Config**: `/Users/vampire/go/src/orc/config.yaml.example` - Template with all options
- **Current Settings**: AI timeout 300s, Phase timeouts 8-20min, Rate limit 30 req/min

### Orchestration Modes
- **Standard Orchestrator**: `/Users/vampire/go/src/orc/internal/core/orchestrator.go` - Basic phase execution
- **Fluid Orchestrator**: `/Users/vampire/go/src/orc/internal/core/fluid_orchestrator.go` - Adaptive "Be Like Water" execution
- **Goal Orchestrator**: `/Users/vampire/go/src/orc/internal/core/goal_orchestrator.go` - Continue until objectives met
- **Phase Flow**: `/Users/vampire/go/src/orc/internal/core/phase_flow.go` - Dynamic phase discovery

### Fiction Generation (Legacy - Still Functional)
- **Systematic Planner**: `/Users/vampire/go/src/orc/internal/phase/fiction/systematic_planner.go` - Word budget engineering
- **Targeted Writer**: `/Users/vampire/go/src/orc/internal/phase/fiction/targeted_writer.go` - Context-aware scene writing
- **Contextual Editor**: `/Users/vampire/go/src/orc/internal/phase/fiction/contextual_editor.go` - Full-novel intelligence editing
- **Systematic Assembler**: `/Users/vampire/go/src/orc/internal/phase/fiction/systematic_assembler.go` - Final polished output

### AI Agent System
- **Agent Interface**: `/Users/vampire/go/src/orc/internal/agent/agent.go` - AI client abstraction
- **HTTP Client**: `/Users/vampire/go/src/orc/internal/agent/client.go` - Retry logic and rate limiting
- **Response Cache**: `/Users/vampire/go/src/orc/internal/agent/cache.go` - Performance caching
- **Prompt Cache**: `/Users/vampire/go/src/orc/internal/agent/prompt_cache.go` - Template-based caching

### Storage & Sessions
- **Filesystem Storage**: `/Users/vampire/go/src/orc/internal/storage/filesystem.go` - File-based storage
- **Session Management**: `/Users/vampire/go/src/orc/internal/storage/session.go` - Metadata and checkpointing
- **Output Directory**: `~/.local/share/orchestrator/output/` - Generated content and sessions
- **Checkpoints**: Session state saved for resumption capabilities

## XDG Compliant Paths

### Configuration
- **Config Home**: `~/.config/orchestrator/`
- **Main Config**: `~/.config/orchestrator/config.yaml`
- **Prompts**: `~/.config/orchestrator/prompts/` (optional override)

### Data
- **Data Home**: `~/.local/share/orchestrator/`
- **Output**: `~/.local/share/orchestrator/output/`
- **Prompts**: `~/.local/share/orchestrator/prompts/`
- **Cache**: `~/.local/share/orchestrator/cache/`

### State & Logs  
- **State Home**: `~/.local/state/orchestrator/`
- **Debug Log**: `~/.local/state/orchestrator/debug.log`
- **Error Log**: `~/.local/state/orchestrator/error.log`

### Binary
- **Executable**: `~/.local/bin/orc` (symlinked to build output)

## How to Find Things

### Search Commands
```bash
# Find Go files containing specific functionality
find /Users/vampire/go/src/orc -name "*.go" | xargs grep -l "IteratorAgent"

# Locate configuration files
fd config.yaml ~/.config/orchestrator/

# Find documentation
fd "*.md" /Users/vampire/go/src/orc/docs/

# Search debug logs for errors  
tail -f ~/.local/state/orchestrator/debug.log | grep ERROR

# Find session output
ls -la ~/.local/share/orchestrator/output/sessions/
```

### Code Pattern Searches
```bash
# Find all phase implementations
grep -r "func.*Execute.*context.Context" /Users/vampire/go/src/orc/internal/phase/

# Find interface definitions
grep -r "type.*interface" /Users/vampire/go/src/orc/internal/core/

# Find timeout configurations
grep -r "time\\..*" /Users/vampire/go/src/orc/internal/config/
```

## Recent Updates (Quality-Focused Architecture)

### New Files Added (2025-07-17)
- `/Users/vampire/go/src/orc/internal/core/iterator.go` - Infinite quality improvement engine
- `/Users/vampire/go/src/orc/internal/core/verification.go` - Stage verification with retry logic  
- `/Users/vampire/go/src/orc/internal/phase/utils.go` - Robust JSON parsing utilities
- `/Users/vampire/go/src/orc/internal/core/fluid_orchestrator.go` - Adaptive orchestration
- `/Users/vampire/go/src/orc/docs/flow.md` - Comprehensive execution flow documentation
- `/Users/vampire/go/src/orc/docs/patterns.md` - Code conventions and architectural patterns

### Modified Files (Current Session)
- `/Users/vampire/.config/orchestrator/config.yaml` - Extended timeouts (AI: 300s, Phases: 8-20min)
- `/Users/vampire/go/src/orc/internal/phase/code/` - All files updated with improved JSON parsing
- `/Users/vampire/.claude/CLAUDE.md` - Updated with model specification rules
- `/Users/vampire/go/src/orc/CLAUDE.md` - Enhanced navigation and working memory
- `/Users/vampire/go/src/orc/docs/paths.md` - This file, comprehensive updates
- `/Users/vampire/go/src/orc/docs/errors.md` - Complete error catalog with solutions

### Known Issues to Fix
- **Logger initialization**: Lines 496, 498 in `cmd/orc/main.go` use logger before initialization
- **Current directory**: CLI shows `/Users/vampire/go/src/orc` as working directory
- **Domain plugin migration**: Import cycle issues during plugin architecture transition

### Key Improvements
- **JSON Parsing**: CleanJSONResponse handles markdown-wrapped and malformed JSON responses
- **Quality Focus**: Timeouts increased 3-5x to prioritize quality over speed
- **Iterator Agents**: Infinite improvement until all quality criteria pass
- **Verification Loops**: Automatic retry with issue documentation
- **Language Constraints**: Explicit language specification now properly enforced
- **Documentation Structure**: Comprehensive cross-referenced documentation system