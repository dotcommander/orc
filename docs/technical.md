# Systematic AI Novel Generation - Technical Reference

**AI Context**: Complete technical documentation for The Orchestrator's revolutionary systematic approach to AI novel generation with word budget engineering and contextual intelligence.

**Cross-references**: [`../PLAN.md`](../PLAN.md) for strategic overview, [`../SYSTEMATIC_ARCHITECTURE.md`](../SYSTEMATIC_ARCHITECTURE.md) for deep technical dive, [`../CLAUDE.md`](../CLAUDE.md) for development navigation, [`flow.md`](flow.md) for execution flows, [`patterns.md`](patterns.md) for implementation patterns, [`errors.md`](errors.md) for troubleshooting.

## Revolutionary Systematic Architecture

The Orchestrator implements **Word Budget Engineering** - the first mathematically reliable approach to AI novel generation. Instead of hoping AI produces the right length, we **engineer the exact structure** that guarantees predictable results.

### Core Innovation: Mathematical Precision

```
Traditional: "Write a 20k word novel" → Unpredictable (3k-15k words)
Systematic:  20 chapters × 1,000 words → Reliable (20,100 words = 100.5% accuracy)
```

**Breakthrough Result**: Proven 100.5% word count accuracy through systematic orchestration.

### Systematic Architecture Principles

#### 1. Word Budget Engineering
Mathematical approach to predictable creative output:
- **Structured Planning**: 20 chapters × 1,000 words = 20,000 words exactly
- **Scene-Level Precision**: 3 scenes × 333 words = 1,000 words per chapter  
- **Mathematical Certainty**: Engineering replaces hoping

#### 2. Contextual Intelligence  
Every component has complete novel awareness:
- **Full Story Context**: AI knows entire story before writing each scene
- **Editor Intelligence**: Reads complete novel before making improvements
- **Progressive Awareness**: Each phase builds on complete context

#### 3. Systematic Phase Pipeline
Revolutionary phase structure optimized for AI strengths:
```
SystematicPlanner → TargetedWriter → ContextualEditor → SystematicAssembler
```

#### 4. AI-Friendly Design
Works with AI's natural abilities rather than against them:
- **Conversational Development**: Natural dialogue for story creation
- **Manageable Chunks**: 333-word scenes optimal for AI composition
- **Context Provision**: Complete story awareness improves AI performance

## Core Interfaces & Contracts

### Phase Interface

The `Phase` interface defines the contract for all pipeline phases.

```go
// Package: internal/core
type Phase interface {
    // Name returns the human-readable phase name for logging and debugging
    Name() string
    
    // Execute runs the phase with the given input and returns structured output
    Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error)
    
    // ValidateInput performs pre-flight checks on input before execution
    ValidateInput(ctx context.Context, input PhaseInput) error
    
    // ValidateOutput verifies output meets phase requirements
    ValidateOutput(ctx context.Context, output PhaseOutput) error
    
    // EstimatedDuration returns expected execution time for timeout configuration
    EstimatedDuration() time.Duration
    
    // CanRetry determines if an error is retryable for this specific phase
    CanRetry(err error) bool
}
```

**Implementation Example**:
```go
type MyPhase struct {
    agent   Agent
    storage Storage
    config  MyPhaseConfig
}

func (p *MyPhase) Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error) {
    // Phase-specific implementation
    result, err := p.agent.Execute(ctx, p.buildPrompt(input), input.Data)
    if err != nil {
        return PhaseOutput{}, err
    }
    
    return PhaseOutput{
        Data: result,
        Metadata: map[string]interface{}{
            "phase": p.Name(),
            "timestamp": time.Now(),
        },
    }, nil
}
```

### Agent Interface

The `Agent` interface abstracts AI client interactions.

```go
type Agent interface {
    // Execute sends a prompt to the AI and returns the response
    Execute(ctx context.Context, prompt string, input any) (string, error)
    
    // ExecuteJSON requests structured JSON response from the AI
    ExecuteJSON(ctx context.Context, prompt string, input any) (string, error)
}
```

### Storage Interface

The `Storage` interface provides persistent data management.

```go
type Storage interface {
    // Save writes data to the specified path
    Save(ctx context.Context, path string, data []byte) error
    
    // Load reads data from the specified path
    Load(ctx context.Context, path string) ([]byte, error)
    
    // List returns file paths matching the pattern
    List(ctx context.Context, pattern string) ([]string, error)
    
    // Exists checks if a path exists
    Exists(ctx context.Context, path string) bool
    
    // Delete removes data at the specified path
    Delete(ctx context.Context, path string) error
}
```

### Goal System Interfaces

#### Goal Interface
```go
type Goal struct {
    Type        GoalType
    Target      interface{}
    Current     interface{}
    Priority    int
    Met         bool
    Validator   func(interface{}) bool
}

// Methods
func (g *Goal) Progress() float64    // Returns 0-100% progress
func (g *Goal) Gap() interface{}     // Returns deficit for numeric goals
```

#### Strategy Interface
```go
type Strategy interface {
    // Name returns the strategy identifier
    Name() string
    
    // CanHandle checks if this strategy can handle the given goals
    CanHandle(goals []*Goal) bool
    
    // Execute applies the strategy to achieve the goals
    Execute(ctx context.Context, input interface{}, goals []*Goal) (interface{}, error)
    
    // EstimateEffectiveness returns a score 0-1 for how well this strategy fits
    EstimateEffectiveness(goals []*Goal) float64
}
```

## Data Structures

### PhaseInput
```go
type PhaseInput struct {
    Request   string                 // User's original request
    Prompt    string                 // Phase-specific prompt template
    Data      interface{}            // Output from previous phase
    SessionID string                 // Session identifier for checkpointing
    Metadata  map[string]interface{} // Additional context
}
```

### PhaseOutput
```go
type PhaseOutput struct {
    Data     interface{}            // Primary phase output
    Error    error                  // Execution error if any
    Metadata map[string]interface{} // Additional context and metrics
}
```

## Architecture Layers

### 1. Domain Layer (`internal/domain/`)
- **Plugin interfaces**: Abstract plugin contracts
- **Business logic**: Core domain models and rules
- **Plugin implementations**: Fiction and Code plugins

### 2. Core Layer (`internal/core/`)
- **Orchestrator**: Main coordination logic
- **Goal System**: Goal tracking and strategies
- **Phase management**: Execution engine and validation
- **Error handling**: Structured error types and recovery

### 3. Infrastructure Layer (`internal/`)
- **Agent**: AI client implementation with caching
- **Storage**: File system storage with XDG compliance
- **Config**: Configuration management and validation
- **Adapter**: Clean architecture adapters

### 4. Application Layer (`cmd/orc/`)
- **CLI interface**: User-facing command-line tool
- **Dependency wiring**: Dependency injection setup
- **Option handling**: Configuration and flag processing

## Performance Architecture

### Execution Engine
The extracted `ExecutionEngine` provides:
- **Caching**: Response caching with TTL
- **Concurrency**: Parallel execution where possible  
- **Retry logic**: Exponential backoff with circuit breakers
- **Validation**: Input/output validation pipeline

### Optimization Features
- **Worker pools**: Parallel scene generation
- **Response cache**: 24-hour TTL with size limits
- **Checkpointing**: Resume from failures
- **Smart defaults**: Zero-configuration operation

## Error Handling

### PhaseError Structure
```go
type PhaseError struct {
    Phase   string      // Phase name where error occurred
    Attempt int         // Retry attempt number
    Cause   error       // Underlying error
    Partial interface{} // Partial results for recovery
}
```

### Error Categories
- **Validation errors**: Input/output contract violations
- **Retry errors**: Temporary failures (API timeouts, rate limits)
- **Terminal errors**: Permanent failures (invalid API keys, malformed prompts)
- **Partial errors**: Failures with recoverable state

## Configuration Architecture

### Consolidated Configuration
```go
type OrchestratorConfig struct {
    CheckpointingEnabled bool
    MaxRetries          int
    PerformanceEnabled  bool
    MaxConcurrency      int
}
```

### XDG Compliance
- **Config**: `~/.config/orchestrator/config.yaml`
- **Data**: `~/.local/share/orchestrator/`
- **Cache**: `~/.cache/orchestrator/`
- **Logs**: `~/.local/state/orchestrator/`

## Plugin Architecture

### Plugin Interface
```go
type Plugin interface {
    Name() string
    Description() string
    GetPhases() []Phase
    ValidateRequest(request string) error
    GetOutputSpec() OutputSpec
}
```

### Available Plugins
- **Fiction Plugin**: Novel and story generation
- **Code Plugin**: Code analysis and generation
- **Docs Plugin**: Documentation generation (planned)

This technical documentation provides both the architectural overview and detailed API contracts needed for development and AI assistant navigation.