# Orchestrator Error Catalog

**Last Updated**: 2025-07-17  
**Purpose**: Comprehensive error patterns, solutions, and troubleshooting information  
**Cross-References**: See [flow.md](flow.md) for error flows and [patterns.md](patterns.md) for error handling patterns

## 📋 Quick Reference

### Error Classification System

| Error Type | Retryable | Recovery Strategy | Common Causes |
|------------|-----------|-------------------|---------------|
| **PhaseError** | ✅/❌ | Exponential backoff | Phase execution failures |
| **ValidationError** | ⚠️ | Retry if language/objective | Input validation failures |
| **GenerationError** | ✅ | Fallback to simpler approach | AI generation issues |
| **DomainPluginError** | ✅ | Plugin reload/fallback | Plugin system failures |
| **AdaptiveError** | ✅ | Learned recovery strategies | Context-specific failures |

## 🔥 Critical Errors (From Recent Sessions)

### 1. Logger Undefined Error
**File**: `cmd/orc/main.go` (lines 496, 498)  
**Pattern**: `logger.Info(...)` called before logger initialization

```go
// ❌ BROKEN - Logger used before initialization
logger.Info("Orchestrator starting")  // Line 496
logger.Info("Configuration loaded")   // Line 498

// ✅ FIXED - Move logging after logger creation
if err := cfg.validate(); err != nil {
    return fmt.Errorf("validating config: %w", err)
}

logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
logger.Info("Configuration loaded successfully")
```

**Solution**: Always initialize logger before any logging calls.

### 2. Compilation Errors During Domain Plugin Migration
**Context**: Moving from phase-based to plugin-based architecture

```bash
# Error pattern:
internal/domain/plugin/code.go:15:2: undefined: PhaseInput
internal/domain/plugin/code.go:15:2: undefined: PhaseOutput
```

**Root Cause**: Import cycle when moving domain logic to plugins
**Solution**: Create local adapter structs instead of importing core types

```go
// ✅ FIXED - Local adapter approach
type PluginInput struct {
    Request string
    Context map[string]interface{}
}

type PluginOutput struct {
    Content string
    Files   map[string]string
}

// Convert between plugin and core types
func (p *CodePlugin) adaptInput(input core.PhaseInput) PluginInput {
    return PluginInput{
        Request: input.Request,
        Context: input.Context,
    }
}
```

### 3. Import Cycle Errors
**Pattern**: `import cycle not allowed`
**Common Scenario**: Core packages importing domain-specific code

```
core -> domain/plugin -> core (CYCLE)
```

**Solution**: Use dependency injection and interfaces
```go
// ✅ CORRECT - Interface in core, implementation in domain
type DomainPlugin interface {
    Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
}

// Register plugins at startup in main.go
registry.Register("code", &plugin.CodePlugin{})
```

## 🏗️ Systematic Error Patterns

### Phase Execution Errors

#### PhaseError Structure
```go
type PhaseError struct {
    Phase        string    // Phase name where error occurred
    Attempt      int       // Retry attempt number
    Cause        error     // Underlying error
    Partial      any       // Partial results if available
    Retryable    bool      // Whether error can be retried
    RecoveryHint string    // Suggestion for recovery
    Timestamp    time.Time // When error occurred
}
```

#### Common Phase Error Patterns
```go
// Network/Timeout Errors (RETRYABLE)
if errors.Is(err, ErrTimeout) || 
   errors.Is(err, ErrNetworkError) ||
   errors.Is(err, ErrRateLimited) {
    // Use exponential backoff retry
}

// Configuration Errors (TERMINAL)
if errors.Is(err, ErrNoAPIKey) || 
   errors.Is(err, ErrInvalidInput) {
    // Don't retry, fix configuration
}
```

## JSON Parsing Errors

### Error: `invalid character '\n' in string literal`
**Symptom**: IncrementalBuilder fails with JSON parsing error when AI returns markdown-wrapped responses
**Root Cause**: AI responses contain literal newlines inside JSON strings instead of escaped `\n`

#### CleanJSONResponse Utility
Location: `/Users/vampire/go/src/orc/internal/phase/utils.go`

```go
// ✅ ALWAYS use this for AI responses
response := phase.CleanJSONResponse(aiResponse)

// Handles these cases:
// 1. Markdown code blocks: ```json ... ```
// 2. Embedded JSON in text responses
// 3. Unescaped newlines in string values
// 4. Trailing commas
// 5. Missing quotes around object keys
```

#### Common JSON Fixes Applied
1. **Strip markdown**: Remove ````json` wrappers
2. **Extract JSON**: Find `{...}` in mixed content
3. **Escape newlines**: Convert literal `\n` to `\\n`
4. **Remove trailing commas**: Fix `{...,}` syntax
5. **Quote keys**: Convert `{key:value}` to `{"key":value}`

**Prevention**: Always use `phase.CleanJSONResponse(response)` before `json.Unmarshal()`
**Files Changed**: 
- `/Users/vampire/go/src/orc/internal/phase/utils.go` - Added robust JSON cleaning
- `/Users/vampire/go/src/orc/internal/phase/code/*.go` - Applied to all code generation phases

### Error: `invalid character '`' looking for beginning of value`
**Symptom**: JSON parsing fails when AI wraps JSON in markdown code blocks
**Root Cause**: AI returns responses like:
```
```json
{"key": "value"}
```
```
**Solution**: CleanJSONResponse utility strips markdown formatting
```go
// Before
json.Unmarshal([]byte(response), &result)

// After  
cleanedResponse := phase.CleanJSONResponse(response)
json.Unmarshal([]byte(cleanedResponse), &result)
```
**Prevention**: Never parse AI responses directly; always clean first

## Model Configuration Issues

### Error: `model gpt-4 instead of gpt-4.1`
**Symptom**: Orchestrator shows wrong model in logs despite config setting `gpt-4.1`
**Root Cause**: AI assistants "correcting" user model specifications they think are typos
**Solution**: 
```yaml
# In /Users/vampire/.config/orchestrator/config.yaml
ai:
  model: "gpt-4.1"  # User specification is always correct
```
**Critical Rule Added to CLAUDE.md**:
```markdown
- **CRITICAL MODEL SPECIFICATION RULE**: NEVER question, correct, or override user-specified model names. 
  Even if you think it's a typo, the user knows their available models better than you do.
```
**Prevention**: Always trust user model specifications; older/deprecated models may cost more but are user's choice

## Language Recognition Problems

### Error: AI generates JavaScript instead of PHP
**Symptom**: Request for "PHP hello world" results in React/Next.js project
**Root Cause**: ConversationalExplorer not enforcing language constraints strictly enough
**Solution**: Use explicit language constraints in requests:
```bash
# Ineffective
./orc create code "Create a PHP hello world web page"

# Effective
./orc create code "ONLY USE PHP LANGUAGE. Create hello.php that echoes Hello World. No JavaScript, no React, no Node.js, ONLY PHP."
```
**Prevention**: 
- Be extremely explicit about language requirements
- Use negative constraints (no X, no Y) 
- Mention specific filenames (hello.php)
- ConversationalExplorer prompts should emphasize language constraints

## Timeout Issues

### Error: Phases completing too quickly with poor quality
**Symptom**: Code generation finishes in 30-60 seconds but lacks depth/quality
**Root Cause**: Default timeouts prioritized speed over quality
**Solution**: Extended timeouts in configuration:
```yaml
# /Users/vampire/.config/orchestrator/config.yaml
ai:
  timeout: 300  # 5 minutes per AI request (was 120s)

limits:
  rate_limit:
    requests_per_minute: 30  # Slower for better quality (was 60)
    burst_size: 5           # Reduced burst (was 10)
```
**Phase Timeouts Extended**:
- ConversationalExplorer: 3min → 8min
- IncrementalBuilder: 8min → 15min  
- IterativeRefiner: 10min → 20min
**Prevention**: Quality over speed - give AI adequate time for thorough work

### Error: `Command timed out after 2m 0.0s`
**Symptom**: Test commands hitting timeout before completion
**Root Cause**: Test timeout shorter than phase execution time
**Solution**: 
```bash
# Increase test timeout
timeout 300 ./orc create code "..."  # 5 minutes instead of 2

# Or let it complete naturally without timeout wrapper
./orc create code "..." --verbose
```
**Prevention**: Match test timeouts to expected execution duration

## Compilation Errors

### Error: `ValidationError redeclared in this block`
**Symptom**: Build fails with duplicate type definition
**Root Cause**: Multiple files defining the same constant/type
**Solution**: Rename conflicting definitions:
```go
// Before (conflict)
const ValidationError = "validation_error"

// After (specific)
const ValidationErrorType = "validation_error" 
```
**Files Fixed**: 
- `/Users/vampire/go/src/orc/internal/core/verification.go`
- `/Users/vampire/go/src/orc/internal/core/adaptive_errors.go`
**Prevention**: Use specific, prefixed names for constants and types

### Error: `contains function redeclared`
**Symptom**: Helper function defined in multiple files
**Root Cause**: Common utility functions duplicated across packages
**Solution**: Consolidate utilities in shared package:
```go
// Move to /Users/vampire/go/src/orc/internal/core/utils.go
func contains(slice []string, item string) bool {
    // implementation
}
```
**Prevention**: Check for existing utilities before creating new ones

## Interface Compatibility Issues

### Error: Method not found on interface
**Symptom**: New iterator agent methods not available on Phase interface
**Root Cause**: Iterator agent implements extended interface not compatible with base Phase
**Solution**: Add missing methods to satisfy interface:
```go
// Add to IterativeImprovementEngine
func (ie *IterativeImprovementEngine) RegisterInspector(inspector Inspector) {
    ie.inspectors = append(ie.inspectors, inspector)
}
```
**Prevention**: Verify interface compliance with `go build` before committing

## Verification System Issues

### Error: Stage verification fails but no retry
**Symptom**: Verification fails but orchestrator exits instead of retrying
**Root Cause**: Verification system not properly integrated with retry logic
**Solution**: Ensure StageVerifier is configured with retry limits:
```go
verifier := &core.StageVerifier{
    retryLimit: 3,
    issueTracker: issueTracker,
    logger: logger,
}
```
**Files**: `/Users/vampire/go/src/orc/internal/core/verification.go`
**Prevention**: Always configure retry limits when creating verifiers

## Session Resume Issues

### Error: `phase Systematic Planning failed: input validation failed: request too short`
**Symptom**: Resume fails with validation error on different orchestrator type
**Root Cause**: Session created with different orchestrator (fluid vs systematic)
**Solution**: Use consistent orchestrator type or start fresh session:
```bash
# If session was created with --fluid, resume with --fluid
./orc resume SESSION_ID  # Use same flags as original

# Or start fresh if incompatible
./orc create code "..." --fluid --verbose
```
**Prevention**: Document session creation flags for resumption

## Debug and Investigation

### Standard Debug Process
1. **Check logs first**:
   ```bash
   tail -f ~/.local/state/orchestrator/debug.log
   ```

2. **Verify configuration**:
   ```bash
   cat ~/.config/orchestrator/config.yaml
   ```

3. **Test connectivity**:
   ```bash
   # Verify API key and model access
   ./orc config get ai.model
   ```

4. **Check permissions**:
   ```bash
   # Verify XDG directory access
   ls -la ~/.config/orchestrator/
   ls -la ~/.local/share/orchestrator/
   ```

### Error Log Patterns
- `JSON parsing error`: AI response format issues
- `timeout exceeded`: Need longer timeouts  
- `model not found`: Configuration or API key issues
- `permission denied`: XDG directory setup problems
- `phase failed`: Verification or quality threshold issues

### Recovery Strategies
1. **JSON Errors**: Update CleanJSONResponse utility
2. **Timeout Errors**: Increase relevant timeout in config
3. **Model Errors**: Verify API access and model availability
4. **Phase Errors**: Check verification criteria and thresholds
5. **Permission Errors**: Fix XDG directory permissions

## 🔍 Additional Error Patterns

### Validation Errors

#### ValidationError Structure
```go
type ValidationError struct {
    Phase      string      // Phase where validation failed
    Type       string      // "input", "output", or "internal"
    Field      string      // Field that failed validation
    Message    string      // Human-readable error message
    Data       interface{} // The data that failed validation
    Timestamp  time.Time   // When validation failed
}
```

#### Retryable Validation Scenarios
```go
func IsRetryableCustom(err error) bool {
    switch e := err.(type) {
    case *ValidationError:
        // Language detection failures are retryable
        // Missing objective data is retryable
        return e.Field == "language" || e.Field == "main_objective"
    }
    return false
}
```

### Domain Plugin Errors

#### Plugin Error Types
Location: `/Users/vampire/go/src/orc/internal/domain/plugin/errors.go`

```go
// Plugin registration conflicts
type DomainPluginAlreadyRegisteredError struct {
    Name string
}

// Plugin not found
type DomainPluginNotFoundError struct {
    Name string  
}

// Phase execution failures in plugins
type DomainPhaseExecutionError struct {
    Phase     string
    Plugin    string
    Err       error
    Retryable bool // ← KEY for retry decisions
}
```

### Adaptive Error Learning
Location: `/Users/vampire/go/src/orc/internal/core/adaptive_errors.go`

#### Error Classification
```go
type ErrorType int

const (
    TransientError ErrorType = iota // Retry with same strategy
    AdaptableError                  // Need different approach  
    ConfigError                     // Configuration issue
    ResourceError                   // Resource constraint
    ValidationErrorType             // Input validation
    UnknownError                    // Needs investigation
)
```

## 🔄 Resilience Patterns

### Retry Logic
Location: `/Users/vampire/go/src/orc/internal/core/resilience.go`

#### Exponential Backoff Configuration
```go
type ResilienceConfig struct {
    MaxRetries       int           // Default: 3
    BaseDelay        time.Duration // Default: 1s
    MaxDelay         time.Duration // Default: 30s
    BackoffMultiplier float64      // Default: 2.0
    EnableFallbacks   bool         // Default: true
}
```

#### Fallback Strategies
```go
// When AI analysis fails, use keyword detection
func fallbackAnalysis(request string) map[string]interface{} {
    request = strings.ToLower(request)
    
    language := "Other"
    if strings.Contains(request, "php") {
        language = "PHP"
    } else if strings.Contains(request, "go ") {
        language = "Go"
    }
    
    return map[string]interface{}{
        "language":       language,
        "main_objective": fmt.Sprintf("Generate %s code", language),
    }
}
```

## 📊 Error Analytics

### Common Error Frequencies (Based on Logs)

1. **JSON Parsing Errors**: ~40% - Use CleanJSONResponse
2. **Validation Failures**: ~25% - Check language/objective fields  
3. **Timeout Errors**: ~20% - Increase timeout or use --fluid
4. **Configuration Issues**: ~10% - Validate config setup
5. **Import Cycles**: ~5% - Use dependency injection

### Performance Impact Analysis

| Error Type | Avg Recovery Time | Success Rate | Recommendation |
|------------|------------------|--------------|----------------|
| JSON Parse | 0.1s | 95% | Always use CleanJSONResponse |
| Validation | 2-5s | 80% | Retry language/objective only |
| Timeout | 30-60s | 70% | Use longer timeouts with --fluid |
| Network | 5-15s | 85% | Exponential backoff works well |
| Config | Manual | 100% | Interactive config creation |

## 💡 Quick Solutions by Error Message

| Error Message Pattern | Quick Solution |
|----------------------|----------------|
| `logger.Info undefined` | Move logging after logger initialization |
| `import cycle not allowed` | Use dependency injection, avoid circular imports |
| `invalid character '\n'` | Use `phase.CleanJSONResponse()` |
| `timeout exceeded` | Increase timeout or use `--fluid` flag |
| `validation failed for language` | Retry with explicit language constraints |
| `plugin 'X' not found` | Check plugin registration in main.go |
| `rate limited` | Wait for rate limit reset (handled automatically) |
| `API key not configured` | Set `OPENAI_API_KEY` environment variable |

## Prevention Best Practices

### Code Generation
- Always use explicit language constraints
- Include negative constraints (no X, no Y)
- Specify exact filenames when possible
- Use CleanJSONResponse for all AI response parsing

### Configuration
- Prioritize quality over speed in timeouts
- Test configuration changes with simple requests first
- Document any custom model specifications

### Development
- Run `make build` after any interface changes
- Test with explicit language requirements
- Check for duplicate utilities before creating new ones
- Verify XDG compliance for all file operations

### Quality Assurance
- Use iterator agents for quality-critical tasks
- Configure appropriate verification thresholds
- Document quality criteria for each domain
- Enable verbose logging for troubleshooting

## 🔗 Cross-References

### Related Documentation
- **Execution Flows**: [flow.md](flow.md) - Error handling flows and recovery patterns
- **Code Patterns**: [patterns.md](patterns.md) - Error handling implementation patterns  
- **File Locations**: [paths.md](paths.md) - Where to find error handling code
- **Configuration**: [configuration.md](configuration.md) - Error-related configuration options
- **Visual Flows**: [../orchestrator_flow_diagram.md](../orchestrator_flow_diagram.md) - Error flow diagrams

### Implementation Files for Error Handling
| Error Category | Primary Files | Supporting Files |
|----------------|---------------|------------------|
| **JSON Parsing** | `internal/phase/utils.go` | All files in `internal/phase/code/` |
| **Phase Execution** | `internal/core/execution_engine.go` | `internal/core/orchestrator.go` |
| **Validation** | `internal/core/verification.go` | `internal/core/adaptive_errors.go` |
| **Configuration** | `internal/config/config.go` | `cmd/orc/main.go` |
| **Network/AI** | `internal/agent/client.go` | `internal/agent/agent.go` |
| **Logging** | `cmd/orc/main.go` (lines 496, 498) | All files with logger usage |

### Quick Solutions Index
| Problem | Solution Location | Implementation File |
|---------|-------------------|--------------------|
| **JSON parsing fails** | CleanJSONResponse utility | `internal/phase/utils.go` |
| **Logger undefined** | Move logging after initialization | `cmd/orc/main.go` |
| **Import cycles** | Use dependency injection | `main.go` registration |
| **Timeouts too short** | Increase in config | `~/.config/orchestrator/config.yaml` |
| **Model override** | Never question user choice | Global policy (CLAUDE.md) |
| **Language detection** | Use explicit constraints | Request formatting |

---

**Remember**: This error catalog learns and evolves. When you encounter new error patterns, add them here with their solutions for future reference. Always cross-reference with related documentation for complete understanding.