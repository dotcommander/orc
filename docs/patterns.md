# Orchestrator Code Patterns & Conventions

**Last Updated**: 2025-07-17  
**Purpose**: Code conventions, architectural patterns, and implementation guidelines  
**Cross-References**: See [flow.md](flow.md) for execution patterns and [paths.md](paths.md) for file locations

## 📋 Quick Reference

### Code Style Conventions
| Pattern | Usage | Example |
|---------|-------|---------|
| **Interface Naming** | `{Function}er` suffix | `Phase`, `Orchestrator`, `Verifier` |
| **Error Types** | `{Context}Error` struct | `PhaseError`, `ValidationError` |
| **Config Structs** | `{Component}Config` | `ResilienceConfig`, `AIConfig` |
| **Context Keys** | Typed constants | `type contextKey string` |
| **File Organization** | Feature-based directories | `internal/core/`, `internal/phase/` |

## 🏗️ Architectural Patterns

### 1. Interface-Driven Design

#### Core Pattern
```go
// Define interface in consuming package
type Phase interface {
    Name() string
    Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
    Validate(input PhaseInput) error
    EstimatedDuration() time.Duration
    CanRetry(err error) bool
}

// Implement in specific packages
type ConversationalExplorer struct {
    agent    Agent
    config   CodeConfig
    logger   *slog.Logger
}

func (ce *ConversationalExplorer) Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error) {
    // Implementation specific to this phase
}
```

#### Benefits
- **Testability**: Easy to mock interfaces
- **Flexibility**: Swap implementations without changing consumers
- **Clean Dependencies**: High-level modules don't depend on low-level details

### 2. Dependency Injection Pattern

#### Constructor Injection (Primary Pattern)
```go
// Constructor with explicit dependencies
func NewFluidOrchestrator(
    storage Storage,
    sessionID string,
    outputDir string,
    logger *slog.Logger,
    config *Config,
) *FluidOrchestrator {
    return &FluidOrchestrator{
        storage:   storage,
        sessionID: sessionID,
        outputDir: outputDir,
        logger:    logger,
        config:    config,
        verifier:  NewStageVerifier(logger),
    }
}

// Usage in main.go
orchestrator := NewFluidOrchestrator(storage, sessionID, outputDir, logger, cfg)
```

#### Registry Pattern (Plugin System)
```go
// Plugin registry for dynamic registration
type PluginRegistry struct {
    plugins map[string]DomainPlugin
    mu      sync.RWMutex
}

func (pr *PluginRegistry) Register(name string, plugin DomainPlugin) error {
    pr.mu.Lock()
    defer pr.mu.Unlock()
    
    if _, exists := pr.plugins[name]; exists {
        return &DomainPluginAlreadyRegisteredError{Name: name}
    }
    
    pr.plugins[name] = plugin
    return nil
}
```

### 3. Error Handling Patterns

#### Structured Error Types
```go
// Base error with rich context
type PhaseError struct {
    Phase        string      `json:"phase"`
    Attempt      int         `json:"attempt"`
    Cause        error       `json:"cause"`
    Partial      interface{} `json:"partial,omitempty"`
    Retryable    bool        `json:"retryable"`
    RecoveryHint string      `json:"recovery_hint,omitempty"`
    Timestamp    time.Time   `json:"timestamp"`
}

func (pe *PhaseError) Error() string {
    return fmt.Sprintf("phase %s failed (attempt %d): %v", pe.Phase, pe.Attempt, pe.Cause)
}

func (pe *PhaseError) Unwrap() error {
    return pe.Cause
}
```

#### Error Classification Pattern
```go
// Centralized error classification
func classifyError(err error) ErrorType {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return TimeoutError
    case errors.Is(err, ErrNetworkError):
        return NetworkError
    case strings.Contains(err.Error(), "rate limit"):
        return RateLimitError
    default:
        return UnknownError
    }
}

// Retry decision based on classification
func isRetryable(err error) bool {
    switch classifyError(err) {
    case TimeoutError, NetworkError, RateLimitError:
        return true
    case ValidationError, ConfigError:
        return false
    default:
        return true // Conservative approach
    }
}
```

### 4. Configuration Pattern

#### Hierarchical Configuration
```go
// Main configuration with nested structs
type Config struct {
    AI        AIConfig        `yaml:"ai" validate:"required"`
    Storage   StorageConfig   `yaml:"storage" validate:"required"`
    Limits    LimitsConfig    `yaml:"limits" validate:"required"`
    Quality   QualityConfig   `yaml:"quality"`
    Logging   LoggingConfig   `yaml:"logging"`
}

// Validation using struct tags
func (c *Config) Validate() error {
    validate := validator.New()
    return validate.Struct(c)
}
```

#### XDG Compliance Pattern
```go
// XDG directory resolution
func getConfigPath() string {
    if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
        return filepath.Join(xdgConfig, "orchestrator", "config.yaml")
    }
    
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".config", "orchestrator", "config.yaml")
}

// Ensure directories exist
func ensureDirectories(paths ...string) error {
    for _, path := range paths {
        if err := os.MkdirAll(path, 0755); err != nil {
            return fmt.Errorf("creating directory %s: %w", path, err)
        }
    }
    return nil
}
```

## 🔄 Concurrency Patterns

### 1. Context Propagation
```go
// Always accept and propagate context
func (phase *SomePhase) Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return PhaseOutput{}, ctx.Err()
    default:
    }
    
    // Propagate context to downstream calls
    response, err := phase.agent.Request(ctx, prompt)
    if err != nil {
        return PhaseOutput{}, fmt.Errorf("AI request failed: %w", err)
    }
    
    return PhaseOutput{Content: response}, nil
}
```

### 2. Worker Pool Pattern
```go
// Controlled concurrency for phase execution
type WorkerPool struct {
    workers    int
    taskQueue  chan Task
    resultChan chan Result
    wg         sync.WaitGroup
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.workers; i++ {
        wp.wg.Add(1)
        go wp.worker(ctx)
    }
}

func (wp *WorkerPool) worker(ctx context.Context) {
    defer wp.wg.Done()
    
    for {
        select {
        case task := <-wp.taskQueue:
            result := task.Execute(ctx)
            wp.resultChan <- result
        case <-ctx.Done():
            return
        }
    }
}
```

### 3. Graceful Shutdown Pattern
```go
// Graceful shutdown with timeout
func (orch *Orchestrator) Shutdown(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    // Signal shutdown
    close(orch.shutdownChan)
    
    // Wait for current operations to complete
    done := make(chan struct{})
    go func() {
        orch.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return fmt.Errorf("shutdown timeout exceeded")
    }
}
```

## 🎯 Quality Patterns

### 1. Iterator Agent Pattern
```go
// Infinite quality improvement until convergence
type IteratorAgent struct {
    maxIterations    int
    qualityThreshold float64
    inspector        Inspector
    improver         Improver
}

func (ia *IteratorAgent) IterateUntilQuality(ctx context.Context, content string) (string, error) {
    current := content
    
    for iteration := 0; iteration < ia.maxIterations; iteration++ {
        // Analyze current quality
        quality := ia.inspector.Analyze(current)
        
        if quality.Score >= ia.qualityThreshold && len(quality.Issues) == 0 {
            return current, nil // Converged
        }
        
        // Generate improvement
        improved, err := ia.improver.Improve(ctx, current, quality.Issues)
        if err != nil {
            return current, fmt.Errorf("improvement failed: %w", err)
        }
        
        current = improved
    }
    
    return current, nil // Best effort
}
```

### 2. Verification Pattern
```go
// Stage verification with retry logic
type StageVerifier struct {
    retryLimit   int
    issueTracker IssueTracker
    logger       *slog.Logger
}

func (sv *StageVerifier) VerifyStageWithRetry(ctx context.Context, stage string, executeFunc func() (interface{}, error)) StageResult {
    var lastOutput interface{}
    var issues []Issue
    
    for attempt := 1; attempt <= sv.retryLimit; attempt++ {
        output, err := executeFunc()
        if err != nil {
            sv.logger.Error("Stage execution failed", "stage", stage, "attempt", attempt, "error", err)
            continue
        }
        
        lastOutput = output
        issues = sv.verifyOutput(output)
        
        if len(issues) == 0 {
            return StageResult{Success: true, Output: output}
        }
        
        // Document issues for learning
        sv.issueTracker.Document(stage, attempt, issues)
        
        if attempt < sv.retryLimit {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    
    return StageResult{
        Success: false,
        Output:  lastOutput,
        Issues:  issues,
    }
}
```

## 🔧 Utility Patterns

### 1. JSON Response Cleaning
```go
// Robust JSON parsing for AI responses
func CleanJSONResponse(response string) string {
    // Remove markdown code blocks
    if strings.Contains(response, "```json") {
        re := regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")
        matches := re.FindStringSubmatch(response)
        if len(matches) > 1 {
            response = matches[1]
        }
    }
    
    // Extract JSON from mixed content
    if !strings.HasPrefix(strings.TrimSpace(response), "{") {
        re := regexp.MustCompile("(?s)\\{.*\\}")
        match := re.FindString(response)
        if match != "" {
            response = match
        }
    }
    
    // Fix common JSON issues
    response = fixUnescapedNewlines(response)
    response = removeTrailingCommas(response)
    response = quoteUnquotedKeys(response)
    
    return response
}
```

### 2. Exponential Backoff Pattern
```go
// Exponential backoff with jitter
type BackoffConfig struct {
    BaseDelay        time.Duration
    MaxDelay         time.Duration
    BackoffMultiplier float64
    MaxRetries       int
}

func (bc *BackoffConfig) CalculateDelay(attempt int) time.Duration {
    delay := time.Duration(float64(bc.BaseDelay) * math.Pow(bc.BackoffMultiplier, float64(attempt-1)))
    
    if delay > bc.MaxDelay {
        delay = bc.MaxDelay
    }
    
    // Add jitter (±25%)
    jitter := time.Duration(rand.Float64() * 0.5 * float64(delay))
    if rand.Float64() < 0.5 {
        delay -= jitter
    } else {
        delay += jitter
    }
    
    return delay
}
```

### 3. Resource Cleanup Pattern
```go
// Proper resource cleanup with defer
func (orch *Orchestrator) executePhase(ctx context.Context, phase Phase, input PhaseInput) (PhaseOutput, error) {
    // Start timing
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        orch.logger.Info("Phase completed", "phase", phase.Name(), "duration", duration)
    }()
    
    // Create timeout context
    phaseCtx, cancel := context.WithTimeout(ctx, phase.EstimatedDuration())
    defer cancel()
    
    // Execute with proper cleanup
    output, err := phase.Execute(phaseCtx, input)
    if err != nil {
        return PhaseOutput{}, fmt.Errorf("phase %s: %w", phase.Name(), err)
    }
    
    return output, nil
}
```

## 📁 File Organization Patterns

### 1. Package Structure
```
internal/
├── core/                    # Core orchestration logic
│   ├── orchestrator.go      # Standard orchestrator
│   ├── fluid_orchestrator.go # Adaptive orchestrator
│   ├── iterator.go          # Iterator agents
│   └── verification.go      # Quality verification
├── phase/                   # Phase implementations
│   ├── code/               # Code generation phases
│   ├── fiction/            # Fiction generation phases
│   └── utils.go            # Shared phase utilities
├── agent/                   # AI client abstraction
│   ├── agent.go            # Main interface
│   ├── client.go           # HTTP client implementation
│   └── cache.go            # Response caching
├── config/                  # Configuration management
│   └── config.go           # Configuration loading and validation
└── storage/                 # Storage abstraction
    ├── filesystem.go        # File-based storage
    └── session.go           # Session management
```

### 2. Naming Conventions

#### File Naming
- **Interfaces**: `{function}.go` (e.g., `orchestrator.go`)
- **Implementations**: `{type}_{function}.go` (e.g., `fluid_orchestrator.go`)
- **Tests**: `{source}_test.go`
- **Examples**: `example_{feature}.go`

#### Variable Naming
- **Interfaces**: Short names (`Agent`, `Storage`)
- **Structs**: Descriptive (`FluidOrchestrator`, `ConversationalExplorer`)
- **Functions**: Verb-based (`Execute`, `Validate`, `Transform`)
- **Constants**: UPPER_SNAKE_CASE (`MAX_RETRIES`, `DEFAULT_TIMEOUT`)

### 3. Import Organization
```go
import (
    // Standard library first
    "context"
    "fmt"
    "time"
    
    // Third-party packages
    "github.com/go-playground/validator/v10"
    "gopkg.in/yaml.v3"
    
    // Local packages (relative to module root)
    "github.com/vampirenirmal/orchestrator/internal/agent"
    "github.com/vampirenirmal/orchestrator/internal/config"
    "github.com/vampirenirmal/orchestrator/internal/storage"
)
```

## 🧪 Testing Patterns

### 1. Interface Mocking
```go
// Mock implementation for testing
type MockAgent struct {
    responses map[string]string
    calls     []string
}

func (ma *MockAgent) Request(ctx context.Context, prompt string) (string, error) {
    ma.calls = append(ma.calls, prompt)
    
    if response, exists := ma.responses[prompt]; exists {
        return response, nil
    }
    
    return "", fmt.Errorf("unexpected prompt: %s", prompt)
}

// Usage in tests
func TestPhaseExecution(t *testing.T) {
    mockAgent := &MockAgent{
        responses: map[string]string{
            "test prompt": "test response",
        },
    }
    
    phase := &ConversationalExplorer{agent: mockAgent}
    output, err := phase.Execute(context.Background(), PhaseInput{Request: "test prompt"})
    
    assert.NoError(t, err)
    assert.Equal(t, "test response", output.Content)
    assert.Contains(t, mockAgent.calls, "test prompt")
}
```

### 2. Table-Driven Tests
```go
// Comprehensive test coverage with table tests
func TestErrorClassification(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        expected ErrorType
        retryable bool
    }{
        {
            name:      "timeout error",
            err:       context.DeadlineExceeded,
            expected:  TimeoutError,
            retryable: true,
        },
        {
            name:      "validation error",
            err:       &ValidationError{Field: "test"},
            expected:  ValidationErrorType,
            retryable: false,
        },
        {
            name:      "network error",
            err:       ErrNetworkError,
            expected:  NetworkError,
            retryable: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errorType := classifyError(tt.err)
            assert.Equal(t, tt.expected, errorType)
            assert.Equal(t, tt.retryable, isRetryable(tt.err))
        })
    }
}
```

## 🔍 Logging Patterns

### 1. Structured Logging
```go
// Consistent structured logging
func (orch *Orchestrator) logPhaseStart(phase Phase, input PhaseInput) {
    orch.logger.Info("Phase starting",
        "phase", phase.Name(),
        "estimated_duration", phase.EstimatedDuration(),
        "input_length", len(input.Request),
        "session_id", orch.sessionID,
    )
}

func (orch *Orchestrator) logPhaseComplete(phase Phase, duration time.Duration, err error) {
    if err != nil {
        orch.logger.Error("Phase failed",
            "phase", phase.Name(),
            "duration", duration,
            "error", err,
            "session_id", orch.sessionID,
        )
    } else {
        orch.logger.Info("Phase completed",
            "phase", phase.Name(),
            "duration", duration,
            "session_id", orch.sessionID,
        )
    }
}
```

### 2. Debug Logging Pattern
```go
// Debug logging with conditional verbosity
func (ce *ConversationalExplorer) debugLog(message string, args ...interface{}) {
    if ce.config.Verbose {
        ce.logger.Debug(message, args...)
    }
}

// Usage
ce.debugLog("Processing request",
    "request_length", len(input.Request),
    "language_detected", detectedLanguage,
    "processing_time", time.Since(start),
)
```

## 🎯 Performance Patterns

### 1. Caching Pattern
```go
// LRU cache with TTL
type CacheEntry struct {
    Value     interface{}
    ExpiresAt time.Time
}

type LRUCache struct {
    cache    map[string]*list.Element
    lru      *list.List
    capacity int
    mu       sync.RWMutex
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if elem, exists := c.cache[key]; exists {
        entry := elem.Value.(*CacheEntry)
        if time.Now().Before(entry.ExpiresAt) {
            c.lru.MoveToFront(elem)
            return entry.Value, true
        }
        // Expired, remove
        c.removeElement(elem)
    }
    
    return nil, false
}
```

### 2. Rate Limiting Pattern
```go
// Token bucket rate limiter
type RateLimiter struct {
    limiter *rate.Limiter
    burst   int
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
    limit := rate.Limit(float64(requestsPerMinute) / 60.0) // per second
    return &RateLimiter{
        limiter: rate.NewLimiter(limit, burst),
        burst:   burst,
    }
}

func (rl *RateLimiter) Allow(ctx context.Context) error {
    if err := rl.limiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limit exceeded: %w", err)
    }
    return nil
}
```

## 🔄 Cross-References

### Related Documentation
- **Execution Flows**: [flow.md](flow.md) - How these patterns work together
- **File Locations**: [paths.md](paths.md) - Where to find pattern implementations
- **Error Handling**: [errors.md](errors.md) - Error pattern examples and solutions
- **Configuration**: [configuration.md](configuration.md) - Configuration pattern usage

### Pattern Implementation Files
| Pattern Category | Primary Files | Examples |
|------------------|---------------|----------|
| **Interface Design** | `internal/core/*.go` | Phase, Orchestrator, Agent interfaces |
| **Error Handling** | `internal/core/adaptive_errors.go` | PhaseError, ValidationError structs |
| **Configuration** | `internal/config/config.go` | Hierarchical config with validation |
| **Concurrency** | `internal/core/execution_engine.go` | Worker pools, context propagation |
| **Quality** | `internal/core/iterator.go` | Iterator agents, verification loops |
| **Utilities** | `internal/phase/utils.go` | JSON cleaning, backoff algorithms |

---

**Remember**: These patterns ensure consistency, maintainability, and reliability across the orchestrator codebase. When implementing new features, follow these established patterns for seamless integration.