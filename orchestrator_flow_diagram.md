# Orchestrator Execution Flow Visualization

## Overview
The Orchestrator system provides multiple execution modes for AI-powered content generation. This document visualizes the major execution flows and their relationships.

## System Architecture

```mermaid
graph TB
    subgraph "Entry Points (CLI)"
        CLI[cmd/orc/main.go]
        CREATE[orc create]
        RESUME[orc resume]
        LIST[orc list]
        CONFIG[orc config]
    end

    subgraph "Execution Modes"
        STANDARD[Standard Mode]
        OPTIMIZED[Optimized Mode]
        GOALAWARE[Goal-Aware Mode]
        FLUID[Fluid Mode]
    end

    subgraph "Core Components"
        ORCH[Orchestrator]
        ENGINE[ExecutionEngine]
        FLUIDORCH[FluidOrchestrator]
        GOALORCH[GoalAwareOrchestrator]
    end

    subgraph "Support Systems"
        VERIFY[StageVerifier]
        CHECKPOINT[CheckpointManager]
        CACHE[PhaseResultCache]
        ERRORHANDLER[AdaptiveErrorHandler]
        ISSUETRACKER[IssueTracker]
    end

    CLI --> CREATE
    CLI --> RESUME
    CREATE --> |--fluid| FLUID
    CREATE --> |--goal-aware| GOALAWARE
    CREATE --> |--optimized| OPTIMIZED
    CREATE --> |default| STANDARD

    STANDARD --> ORCH
    OPTIMIZED --> ENGINE
    GOALAWARE --> GOALORCH
    FLUID --> FLUIDORCH

    ORCH --> ENGINE
    GOALORCH --> ORCH
    FLUIDORCH --> VERIFY
    ENGINE --> CACHE
    ENGINE --> CHECKPOINT
    FLUIDORCH --> ERRORHANDLER
    VERIFY --> ISSUETRACKER
```

## 1. Standard Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Orchestrator
    participant ExecutionEngine
    participant Phase
    participant Storage
    participant CheckpointManager

    User->>CLI: orc create fiction "prompt"
    CLI->>Orchestrator: New(phases, storage)
    CLI->>Orchestrator: Run(ctx, request)
    
    loop For each phase
        Orchestrator->>ExecutionEngine: ExecutePhases()
        ExecutionEngine->>Phase: ValidateInput()
        ExecutionEngine->>Phase: Execute(ctx, input)
        ExecutionEngine->>Phase: ValidateOutput()
        
        alt Success
            ExecutionEngine->>Storage: Save output
            ExecutionEngine->>CheckpointManager: Save checkpoint
        else Failure
            ExecutionEngine->>ExecutionEngine: Retry (up to maxRetries)
            alt Retry Success
                ExecutionEngine->>Storage: Save output
            else All Retries Failed
                ExecutionEngine-->>User: PhaseError
            end
        end
    end
    
    Orchestrator-->>User: Success
```

## 2. Optimized Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant ExecutionEngine
    participant PhaseResultCache
    participant ParallelExecutor
    participant Phase
    participant Storage

    User->>CLI: orc create fiction "prompt" --optimized
    CLI->>ExecutionEngine: WithPerformanceOptimization(true)
    ExecutionEngine->>ExecutionEngine: Create ParallelExecutor & Cache
    
    CLI->>ExecutionEngine: ExecutePhases()
    
    alt Many Phases (>2)
        ExecutionEngine->>ExecutionEngine: runOptimizedParallel()
        note over ExecutionEngine: Currently falls back to sequential
    else Few Phases (≤2)
        ExecutionEngine->>ExecutionEngine: runOptimizedSequential()
        
        loop For each phase
            ExecutionEngine->>PhaseResultCache: Get(phase, input)
            alt Cache Hit
                PhaseResultCache-->>ExecutionEngine: Cached result
            else Cache Miss
                ExecutionEngine->>Phase: Execute(ctx, input)
                ExecutionEngine->>PhaseResultCache: Set(phase, input, output)
            end
            ExecutionEngine->>Storage: Save output
        end
    end
    
    ExecutionEngine-->>User: Success
```

## 3. Goal-Aware Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant GoalAwareOrchestrator
    participant GoalTracker
    participant StrategyManager
    participant Orchestrator
    participant Storage

    User->>CLI: orc create fiction "Write 50,000 word novel" --goal-aware
    CLI->>GoalAwareOrchestrator: New(orchestrator, agent)
    CLI->>GoalAwareOrchestrator: RunUntilGoalsMet(ctx, request)
    
    GoalAwareOrchestrator->>GoalAwareOrchestrator: ParseGoals(request)
    note over GoalTracker: Goals: WordCount(50000), ChapterCount, Quality, Completeness
    
    GoalAwareOrchestrator->>Orchestrator: Run(ctx, request)
    note over Orchestrator: Initial execution
    
    GoalAwareOrchestrator->>Storage: Load manuscript
    GoalAwareOrchestrator->>GoalTracker: UpdateGoals(metrics)
    
    loop While goals not met (max 5 attempts)
        GoalAwareOrchestrator->>GoalTracker: GetUnmetGoals()
        GoalAwareOrchestrator->>StrategyManager: SelectOptimal(unmetGoals)
        
        alt Strategy Available
            GoalAwareOrchestrator->>StrategyManager: Execute(strategy)
            GoalAwareOrchestrator->>Storage: Save improved content
            GoalAwareOrchestrator->>GoalTracker: UpdateGoals()
            
            alt All Goals Met
                GoalAwareOrchestrator-->>User: Success
            else Progress Made
                note over GoalAwareOrchestrator: Continue iteration
            else No Progress
                GoalAwareOrchestrator-->>User: Partial Success
            end
        else No Strategy
            GoalAwareOrchestrator-->>User: Best Effort Result
        end
    end
```

## 4. Fluid Mode Execution Flow (with Verification)

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant FluidOrchestrator
    participant StageVerifier
    participant PhaseFlow
    participant AdaptiveErrorHandler
    participant IssueTracker
    participant Phase
    participant Storage

    User->>CLI: orc create code "Build REST API" --fluid
    CLI->>FluidOrchestrator: New(storage, sessionID, outputDir, logger, config)
    CLI->>FluidOrchestrator: Run(ctx, request)
    
    FluidOrchestrator->>FluidOrchestrator: discoverAndRegisterPhases(request)
    note over PhaseFlow: Dynamic phase discovery based on request patterns
    
    FluidOrchestrator->>FluidOrchestrator: executeWithRecovery(ctx, request)
    
    loop For each phase
        FluidOrchestrator->>StageVerifier: VerifyStageWithRetry(ctx, stage, executeFunc)
        
        loop Retry up to 3 times
            StageVerifier->>Phase: Execute()
            
            alt Execution Success
                StageVerifier->>StageVerifier: Verify output
                alt Verification Pass
                    StageVerifier->>Storage: Save output
                    StageVerifier-->>FluidOrchestrator: StageResult{Success: true}
                else Verification Fail
                    note over StageVerifier: Issues: missing_output, insufficient_content, etc.
                    alt Can Retry
                        StageVerifier->>StageVerifier: Wait & Retry
                    else Max Retries
                        StageVerifier->>IssueTracker: DocumentFailure()
                        StageVerifier-->>FluidOrchestrator: StageResult{Success: false}
                    end
                end
            else Execution Error
                StageVerifier->>AdaptiveErrorHandler: HandleError()
                AdaptiveErrorHandler->>AdaptiveErrorHandler: Analyze error patterns
                alt Recovery Strategy Found
                    AdaptiveErrorHandler->>AdaptiveErrorHandler: RecoverWithLearning()
                    note over FluidOrchestrator: Mark as recovered, continue
                else No Recovery
                    StageVerifier->>IssueTracker: DocumentFailure()
                    StageVerifier-->>FluidOrchestrator: Error
                end
            end
        end
    end
    
    FluidOrchestrator->>FluidOrchestrator: learnFromExecution()
    note over FluidOrchestrator: Update patterns, success rates, execution times
    
    FluidOrchestrator-->>User: Success/Failure with detailed tracking
```

## 5. Plugin Architecture Flow

```mermaid
graph TB
    subgraph "Plugin System"
        REGISTRY[PluginRegistry]
        FICTION[FictionPlugin]
        CODE[CodePlugin]
        DOCS[DocsPlugin<br/>future]
    end

    subgraph "Fiction Plugin Phases"
        PLANNING[Planning Phase]
        ARCHITECTURE[Architecture Phase]
        WRITING[Writing Phase]
        ASSEMBLY[Assembly Phase]
        CRITIQUE[Critique Phase]
    end

    subgraph "Code Plugin Phases"
        ANALYSIS[Analysis Phase]
        CODEPLAN[Planning Phase]
        IMPLEMENTATION[Implementation Phase]
        REVIEW[Review Phase]
    end

    REGISTRY --> FICTION
    REGISTRY --> CODE
    REGISTRY --> DOCS

    FICTION --> PLANNING
    PLANNING --> ARCHITECTURE
    ARCHITECTURE --> WRITING
    WRITING --> ASSEMBLY
    ASSEMBLY --> CRITIQUE

    CODE --> ANALYSIS
    ANALYSIS --> CODEPLAN
    CODEPLAN --> IMPLEMENTATION
    IMPLEMENTATION --> REVIEW
```

## 6. Error Handling and Recovery Flow

```mermaid
sequenceDiagram
    participant Phase
    participant ExecutionEngine
    participant AdaptiveErrorHandler
    participant IssueTracker
    participant CheckpointManager

    Phase->>ExecutionEngine: Execute fails
    ExecutionEngine->>Phase: CanRetry(error)?
    
    alt Can Retry
        loop Retry with exponential backoff
            ExecutionEngine->>Phase: Execute()
            alt Success
                ExecutionEngine->>CheckpointManager: Save checkpoint
                ExecutionEngine-->>ExecutionEngine: Continue
            else Failed
                alt Max Retries Reached
                    ExecutionEngine->>ExecutionEngine: Create PhaseError
                    ExecutionEngine-->>ExecutionEngine: Return error
                else Continue Retry
                    ExecutionEngine->>ExecutionEngine: Wait(backoff)
                end
            end
        end
    else Cannot Retry
        ExecutionEngine->>ExecutionEngine: Create PhaseError
        ExecutionEngine-->>ExecutionEngine: Return immediately
    end

    note over AdaptiveErrorHandler: In Fluid Mode Only:
    AdaptiveErrorHandler->>AdaptiveErrorHandler: Analyze error patterns
    AdaptiveErrorHandler->>AdaptiveErrorHandler: Generate recovery hints
    AdaptiveErrorHandler->>IssueTracker: Document for learning
```

## 7. Verification and Issue Tracking Flow

```mermaid
graph LR
    subgraph "Verification Process"
        EXECUTE[Execute Stage] --> VERIFY{Verify Output}
        VERIFY -->|Pass| SUCCESS[Continue]
        VERIFY -->|Fail| ISSUES[Generate Issues]
        ISSUES --> RETRY{Can Retry?}
        RETRY -->|Yes| EXECUTE
        RETRY -->|No| DOCUMENT[Document Failure]
    end

    subgraph "Issue Types"
        MISSING[missing_output]
        INCOMPLETE[incomplete_planning]
        INSUFFICIENT[insufficient_detail]
        NOCODE[no_code_detected]
        EXECERROR[execution_error]
    end

    subgraph "Issue Tracking"
        DOCUMENT --> JSON[Issue JSON File]
        DOCUMENT --> SUMMARY[Summary MD File]
        JSON --> PATTERNS[Pattern Analysis]
        SUMMARY --> PATTERNS
    end

    ISSUES --> MISSING
    ISSUES --> INCOMPLETE
    ISSUES --> INSUFFICIENT
    ISSUES --> NOCODE
    ISSUES --> EXECERROR
```

## 8. Resume and Checkpoint Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Orchestrator
    participant CheckpointManager
    participant Storage
    participant ExecutionEngine

    User->>CLI: orc resume <session-id>
    CLI->>CLI: Find session directory
    CLI->>Orchestrator: RunWithResume(ctx, request, startPhase)
    
    Orchestrator->>CheckpointManager: Load(ctx, sessionID)
    CheckpointManager->>Storage: Load checkpoint data
    
    alt Checkpoint Found
        CheckpointManager-->>Orchestrator: Checkpoint{PhaseIndex, State}
        Orchestrator->>ExecutionEngine: ExecutePhases(startPhase=checkpoint.PhaseIndex)
        note over ExecutionEngine: Resume from saved phase
    else No Checkpoint
        Orchestrator->>ExecutionEngine: ExecutePhases(startPhase=0)
        note over ExecutionEngine: Start from beginning
    end
    
    loop For each remaining phase
        ExecutionEngine->>ExecutionEngine: Execute phase
        ExecutionEngine->>CheckpointManager: Save(sessionID, phaseIndex, output)
    end
    
    ExecutionEngine-->>User: Completed
```

## Key Features by Mode

### Standard Mode
- Sequential phase execution
- Basic retry logic (3 attempts)
- Checkpoint support
- Simple error handling

### Optimized Mode
- Phase result caching
- Parallel execution support (future)
- Performance monitoring
- Auto-detected concurrency

### Goal-Aware Mode
- Goal parsing and tracking
- Strategy-based improvements
- Iterative refinement (up to 5 attempts)
- Progress monitoring
- Quality metrics

### Fluid Mode
- Dynamic phase discovery
- Adaptive error recovery
- Stage verification with retry
- Issue tracking and documentation
- Learning from execution patterns
- Hot configuration reload
- Flexible prompt templates

## Configuration and Paths

```yaml
XDG Compliant Paths:
  Config: ~/.config/orchestrator/
  Data: ~/.local/share/orchestrator/
  Output: ~/.local/share/orchestrator/output/
  Logs: ~/.local/state/orchestrator/
  Issues: <output_dir>/issues/
```

## Performance Considerations

1. **Caching**: Optimized mode caches phase results for 30 minutes
2. **Concurrency**: Auto-detects optimal concurrency or uses custom value
3. **Retries**: Exponential backoff between retries
4. **Timeouts**: Phase-specific timeouts based on EstimatedDuration()
5. **Verification**: Fluid mode adds verification overhead but ensures quality

## Error Recovery Strategies

1. **Standard Retry**: Simple exponential backoff
2. **Adaptive Recovery**: Analyzes error patterns and suggests fixes
3. **Checkpoint Resume**: Can resume from last successful phase
4. **Issue Documentation**: Tracks all failures for pattern analysis
5. **Goal-Based Recovery**: Continues until goals are met or max attempts
