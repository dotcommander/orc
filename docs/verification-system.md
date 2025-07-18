# Verification and Issue Tracking System

## Overview

The Orchestrator now includes a robust verification system that ensures each stage completes sufficiently before proceeding. If a stage fails verification, it automatically retries up to 3 times. Persistent failures are documented in an `issues/` directory for later analysis.

## Features

### 1. Stage Verification

Each stage output is verified against specific criteria:

- **Planning Stage**: Checks for required elements (outline, characters, plot, theme) and minimum content length
- **Architecture Stage**: Verifies chapter structure is present
- **Writing Stage**: Ensures minimum word count is met
- **Implementation Stage**: Confirms code patterns are present

### 2. Automatic Retry Logic

When a stage fails verification:
1. The system automatically retries up to 3 times
2. Each retry includes exponential backoff
3. Verification issues are logged for each attempt

### 3. Issue Documentation

Failed stages are documented in the `issues/` directory with:
- Detailed JSON reports for each failure
- Human-readable summary markdown files
- Session-specific organization

### 4. Adaptive Recovery

The fluid orchestrator includes:
- Learning from failure patterns
- Adaptive recovery strategies
- Context-aware error handling

## Usage

Enable the verification system with the `--fluid` flag:

```bash
# Create content with verification and issue tracking
orc create fiction "Write a novel" --fluid

# Create code with adaptive phases
orc create code "Build an API" --fluid --verbose
```

## Issue Directory Structure

```
output/
└── sessions/
    └── <session-id>/
        └── issues/
            ├── <session>-<stage>-<timestamp>.json  # Detailed failure report
            └── <session>-summary.md               # Human-readable summary
```

## Example Issue Report

```json
{
  "session_id": "abc123de",
  "stage": "Writing",
  "timestamp": "2024-01-15T10:30:45Z",
  "attempts": 3,
  "duration": "45s",
  "issues": [
    {
      "type": "insufficient_content",
      "severity": "critical",
      "description": "Writing output too short",
      "details": {
        "word_count": 85,
        "minimum_expected": 100
      }
    }
  ]
}
```

## Verification Functions

You can register custom verifiers for your phases:

```go
verifier.RegisterVerifier("MyPhase", func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
    // Custom verification logic
    if outputIsValid(output) {
        return true, nil
    }
    
    return false, []VerificationIssue{{
        Type:        "custom_validation_failed",
        Severity:    "major",
        Description: "Output did not meet custom criteria",
    }}
})
```

## Benefits

1. **Quality Assurance**: Ensures each stage meets minimum quality standards
2. **Automatic Recovery**: Reduces manual intervention for transient failures
3. **Debugging Aid**: Issue reports help identify systematic problems
4. **Learning System**: Adapts to failure patterns over time
5. **Transparency**: All failures are documented for review

## Configuration

The fluid orchestrator supports various configuration options:

```go
fluidConfig := core.DefaultFluidConfig()
fluidConfig.EnableLearning = true        // Learn from patterns
fluidConfig.ErrorRecoveryLevel = 2       // Adaptive recovery
fluidConfig.EnablePromptFlow = true      // Dynamic prompts
```

## Future Enhancements

- Machine learning-based failure prediction
- Automatic issue resolution suggestions
- Cross-session pattern analysis
- Integration with monitoring systems