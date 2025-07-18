# Orchestrator Execution Flow Documentation

**Last Updated**: 2025-07-17  
**Purpose**: Comprehensive execution flow documentation from CLI to output  
**Cross-References**: See [orchestrator_flow_diagram.md](../orchestrator_flow_diagram.md) for visual diagrams

## 📋 Quick Reference

### Execution Modes
| Mode | Entry Point | Best For | Key Features |
|------|-------------|----------|--------------|
| **Standard** | `./orc create TYPE "prompt"` | Simple tasks | Sequential execution, basic retry |
| **Fluid** | `./orc create TYPE "prompt" --fluid` | Quality-critical tasks | Adaptive execution, verification loops |
| **Goal-Aware** | `./orc create TYPE "prompt" --goal-aware` | Complex objectives | Goal tracking, iterative improvement |
| **Optimized** | `./orc create TYPE "prompt" --optimized` | Performance-critical | Caching, parallel execution |

## 🔄 Core Execution Flows

### 1. CLI Entry Point Flow
**File**: `/Users/vampire/go/src/orc/cmd/orc/main.go`

```
User Command → CLI Parser → Mode Detection → Orchestrator Selection → Execution
```

#### CLI Command Processing
```go
// CLI command patterns
orc create fiction "Write a sci-fi novel"      // Standard mode
orc create code "Build REST API" --fluid       // Fluid mode with verification
orc create docs "API documentation" --verbose  // Verbose logging
orc resume SESSION_ID                          // Resume interrupted session
orc config set ai.model "gpt-4.1"            // Configuration management
```

#### Mode Selection Logic
```go
// In main.go
switch {
case fluidFlag:
    orchestrator = createFluidOrchestrator(cfg, storage, logger)
case goalAwareFlag:
    orchestrator = createGoalAwareOrchestrator(cfg, storage, logger)
case optimizedFlag:
    orchestrator = createOptimizedOrchestrator(cfg, storage, logger)
default:
    orchestrator = createStandardOrchestrator(cfg, storage, logger)
}
```

### 2. Standard Execution Flow
**Primary Files**: 
- `/Users/vampire/go/src/orc/internal/core/orchestrator.go`
- `/Users/vampire/go/src/orc/internal/core/execution_engine.go`

```
Request → Phase Discovery → Sequential Execution → Output Generation
```

#### Phase Execution Sequence
```go
for phaseIndex, phase := range phases {
    // 1. Input Validation
    if err := phase.Validate(input); err != nil {
        return ValidationError{Phase: phase.Name(), Field: "input"}
    }
    
    // 2. Phase Execution with Retry
    output, err := executeWithRetry(ctx, phase, input, maxRetries)
    if err != nil {
        return PhaseError{Phase: phase.Name(), Attempt: maxRetries, Cause: err}
    }
    
    // 3. Output Validation & Storage
    if err := validateOutput(output); err != nil {
        return ValidationError{Phase: phase.Name(), Field: "output"}
    }
    
    storage.Save(sessionID, phaseIndex, output)
    checkpoint.Save(sessionID, phaseIndex, output)
    
    // 4. Chain to next phase
    input = PhaseInput{Request: output.Content, Context: mergeContext(input.Context, output.Context)}
}
```

### 3. Fluid Mode Execution Flow
**Primary Files**:
- `/Users/vampire/go/src/orc/internal/core/fluid_orchestrator.go`
- `/Users/vampire/go/src/orc/internal/core/verification.go`

```
Request → Dynamic Phase Discovery → Adaptive Execution → Verification Loops → Quality Assurance
```

#### Dynamic Phase Discovery
```go
func (fo *FluidOrchestrator) discoverAndRegisterPhases(request string) {
    // Analyze request patterns
    requestLower := strings.ToLower(request)
    
    // Register phases based on detected patterns
    if strings.Contains(requestLower, "code") || strings.Contains(requestLower, "api") {
        fo.phases = append(fo.phases, 
            &code.ConversationalExplorer{},
            &code.IncrementalBuilder{},
            &code.IterativeRefiner{})
    }
    
    if strings.Contains(requestLower, "novel") || strings.Contains(requestLower, "story") {
        fo.phases = append(fo.phases,
            &fiction.SystematicPlanner{},
            &fiction.TargetedWriter{},
            &fiction.ContextualEditor{})
    }
}
```

#### Verification Loop Implementation
```go
func (sv *StageVerifier) VerifyStageWithRetry(ctx context.Context, stage string, executeFunc func() (interface{}, error)) StageResult {
    for attempt := 1; attempt <= sv.retryLimit; attempt++ {
        // Execute stage
        output, err := executeFunc()
        if err != nil {
            sv.logger.Error("Stage execution failed", "stage", stage, "attempt", attempt, "error", err)
            continue
        }
        
        // Verify output quality
        issues := sv.verifyOutput(output)
        if len(issues) == 0 {
            return StageResult{Success: true, Output: output}
        }
        
        // Document issues for learning
        sv.issueTracker.Document(stage, attempt, issues)
        
        if attempt < sv.retryLimit {
            backoff := time.Duration(attempt) * time.Second
            time.Sleep(backoff)
        }
    }
    
    return StageResult{Success: false, Issues: issues}
}
```

### 4. Iterator Agent Flow (New Architecture)
**Primary Files**:
- `/Users/vampire/go/src/orc/internal/core/iterator.go`
- `/Users/vampire/go/src/orc/internal/core/iterative_improvement.go`

```
Initial Generation → Quality Analysis → Iterative Improvement → Convergence Check → Final Output
```

#### Infinite Quality Improvement Loop
```go
func (ia *IteratorAgent) IterateUntilQualityMet(ctx context.Context, input IteratorInput) (IteratorOutput, error) {
    current := input.InitialContent
    iteration := 0
    
    for iteration < ia.maxIterations {
        // Quality analysis
        qualityScore, issues := ia.analyzeQuality(current)
        
        // Check convergence criteria
        if qualityScore >= ia.qualityThreshold && len(issues) == 0 {
            return IteratorOutput{
                Content: current,
                Iterations: iteration,
                QualityScore: qualityScore,
                Converged: true,
            }, nil
        }
        
        // Generate improvement
        improved, err := ia.improveContent(ctx, current, issues)
        if err != nil {
            return IteratorOutput{}, fmt.Errorf("improvement failed at iteration %d: %w", iteration, err)
        }
        
        current = improved
        iteration++
    }
    
    return IteratorOutput{
        Content: current,
        Iterations: iteration,
        QualityScore: ia.analyzeQuality(current),
        Converged: false,
    }, nil
}
```

## 🏗️ Phase-Specific Flows

### Code Generation Flow
**Files**: `/Users/vampire/go/src/orc/internal/phase/code/`

#### 1. Conversational Explorer
```
User Request → Language Detection → Requirement Clarification → Technical Specification → Context Building
```

```go
// Language detection patterns
func (ce *ConversationalExplorer) detectLanguage(request string) string {
    request = strings.ToLower(request)
    
    // Explicit language indicators
    if strings.Contains(request, "only use php") || strings.Contains(request, "php language") {
        return "PHP"
    }
    if strings.Contains(request, "golang") || strings.Contains(request, "go ") {
        return "Go"
    }
    
    // File extension hints
    if strings.Contains(request, ".php") {
        return "PHP"
    }
    if strings.Contains(request, ".go") {
        return "Go"
    }
    
    return "Other" // Requires AI analysis
}
```

#### 2. Incremental Builder
```
Specification → Component Planning → Incremental Implementation → Integration → Testing
```

```go
// Incremental building pattern
func (ib *IncrementalBuilder) buildIncremental(ctx context.Context, spec TechnicalSpec) (BuildResult, error) {
    var components []Component
    
    // Build components incrementally
    for _, componentSpec := range spec.Components {
        component, err := ib.buildComponent(ctx, componentSpec)
        if err != nil {
            return BuildResult{}, fmt.Errorf("building component %s: %w", componentSpec.Name, err)
        }
        
        // Validate component before adding
        if err := ib.validateComponent(component); err != nil {
            return BuildResult{}, fmt.Errorf("validating component %s: %w", component.Name, err)
        }
        
        components = append(components, component)
    }
    
    // Integrate all components
    return ib.integrateComponents(components)
}
```

#### 3. Iterative Refiner
```
Initial Code → Quality Analysis → Improvement Generation → Integration Testing → Quality Verification
```

```go
// Quality-driven refinement
func (ir *IterativeRefiner) refineUntilQuality(ctx context.Context, code string) (string, error) {
    current := code
    
    for iteration := 0; iteration < ir.maxIterations; iteration++ {
        // Analyze current quality
        analysis := ir.analyzeCodeQuality(current)
        
        // Check if quality threshold met
        if analysis.Score >= ir.qualityThreshold {
            return current, nil
        }
        
        // Generate improvements
        improved, err := ir.generateImprovement(ctx, current, analysis.Issues)
        if err != nil {
            return current, fmt.Errorf("improvement generation failed: %w", err)
        }
        
        current = improved
    }
    
    return current, nil // Return best effort
}
```

### Fiction Generation Flow  
**Files**: `/Users/vampire/go/src/orc/internal/phase/fiction/`

#### 1. Systematic Planner
```
Novel Concept → Structure Planning → Chapter Breakdown → Word Budget → Writing Schedule
```

#### 2. Targeted Writer
```
Chapter Specifications → Scene Planning → Content Generation → Character Development → Narrative Flow
```

#### 3. Contextual Editor
```
Draft Content → Full-Novel Context → Style Consistency → Plot Coherence → Final Polish
```

## 🔄 Error Handling Flows

### Standard Error Flow
```
Error Detected → Error Classification → Retry Decision → Recovery Strategy → Documentation
```

```go
func (ee *ExecutionEngine) handleError(err error, phase Phase, attempt int) error {
    // Classify error
    if isRetryable(err) && attempt < maxRetries {
        backoff := calculateBackoff(attempt)
        time.Sleep(backoff)
        return nil // Continue retry loop
    }
    
    // Create structured error
    return &PhaseError{
        Phase:        phase.Name(),
        Attempt:      attempt,
        Cause:        err,
        Retryable:    false,
        RecoveryHint: generateRecoveryHint(err),
        Timestamp:    time.Now(),
    }
}
```

### Adaptive Error Flow (Fluid Mode)
```
Error Detection → Pattern Analysis → Learning Integration → Recovery Strategy → Success Tracking
```

```go
func (aeh *AdaptiveErrorHandler) HandleError(ctx context.Context, err error, context map[string]interface{}) error {
    // Analyze error patterns
    pattern := aeh.analyzeErrorPattern(err, context)
    
    // Check for learned recovery strategies
    if strategy, exists := aeh.recoveryStrategies[pattern.Type]; exists {
        if recovery := strategy.Attempt(ctx, err, context); recovery.Success {
            aeh.learnFromSuccess(pattern, recovery)
            return nil
        }
    }
    
    // Generate new recovery strategy
    newStrategy := aeh.generateRecoveryStrategy(pattern)
    aeh.recoveryStrategies[pattern.Type] = newStrategy
    
    return err // Propagate if no recovery possible
}
```

## 🔍 Data Flow Patterns

### Input Processing Flow
```
CLI Args → Request Parsing → Context Building → Phase Input Generation → Validation
```

### Output Processing Flow  
```
Phase Output → Validation → Context Extraction → Storage → Next Phase Input → Final Assembly
```

### Session Management Flow
```
Session Creation → Checkpoint Initialization → Phase Tracking → Intermediate Saves → Final Persistence
```

## 🎯 Performance Optimization Flows

### Caching Flow
```
Input Hash → Cache Lookup → Cache Hit/Miss → Execution/Retrieval → Cache Update → Result Return
```

### Parallel Execution Flow (Future)
```
Phase Dependency Analysis → Parallel Groups → Concurrent Execution → Result Synchronization → Next Phase
```

## 🔧 Configuration Flow

### Startup Configuration Flow
```
CLI Flags → Environment Variables → Config File → Defaults → Validation → Runtime Configuration
```

### Runtime Configuration Flow
```
Hot Reload Detection → Config Validation → Component Updates → Phase Reconfiguration → Execution Continuation
```

## 📊 Monitoring & Observability Flows

### Logging Flow
```
Event Generation → Log Level Check → Structured Logging → File Rotation → Debug Access
```

### Metrics Flow
```
Performance Measurement → Metric Collection → Aggregation → Storage → Analysis
```

## 🔄 Resume & Recovery Flows

### Session Resume Flow
```
Session ID → Metadata Loading → Checkpoint Discovery → Phase Index → Execution Continuation
```

### Checkpoint Flow
```
Phase Completion → Output Validation → Metadata Extraction → Persistent Storage → Recovery Preparation
```

## 🎯 Cross-References

### Related Documentation
- **Visual Flows**: [orchestrator_flow_diagram.md](../orchestrator_flow_diagram.md) - Comprehensive visual diagrams
- **File Locations**: [paths.md](paths.md) - Where to find implementation files  
- **Error Handling**: [errors.md](errors.md) - Error patterns and solutions
- **Code Patterns**: [patterns.md](patterns.md) - Implementation conventions
- **Configuration**: [configuration.md](configuration.md) - Setup and tuning

### Implementation Files by Flow
| Flow Type | Primary Files | Supporting Files |
|-----------|--------------|------------------|
| **CLI Entry** | `cmd/orc/main.go` | `internal/config/config.go` |
| **Standard Execution** | `internal/core/orchestrator.go` | `internal/core/execution_engine.go` |
| **Fluid Execution** | `internal/core/fluid_orchestrator.go` | `internal/core/verification.go` |
| **Iterator Agents** | `internal/core/iterator.go` | `internal/core/iterative_improvement.go` |
| **Code Generation** | `internal/phase/code/*.go` | `internal/phase/utils.go` |
| **Fiction Generation** | `internal/phase/fiction/*.go` | `prompts/fiction/*.txt` |
| **Error Handling** | `internal/core/adaptive_errors.go` | `internal/core/resilience.go` |

---

**Remember**: This flow documentation is designed to help AI assistants understand the complete execution patterns and make informed decisions about modifications and troubleshooting.