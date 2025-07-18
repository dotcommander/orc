package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// Analyzer phase analyzes the code request
type Analyzer struct {
	BasePhase
	agent      core.Agent
	storage    core.Storage
	promptPath string
	logger       *slog.Logger
	validator    core.Validator
	errorFactory core.ErrorFactory
	resilience   *core.PhaseResilience
}

// NewAnalyzer creates a new analyzer phase
func NewAnalyzer(agent core.Agent, storage core.Storage, promptPath string, logger *slog.Logger) *Analyzer {
	return &Analyzer{
		BasePhase:    NewBasePhase("Analysis", 5*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		logger:       logger,
		validator:    core.NewBaseValidator("Analysis"),
		errorFactory: core.NewDefaultErrorFactory(),
		resilience:   core.NewPhaseResilience(),
	}
}

// Execute performs code task analysis
func (a *Analyzer) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	// First run consolidated validation
	if err := a.validator.ValidateRequired("request", input.Request, "input"); err != nil {
		return err
	}
	
	// Additional validation for analysis requirements
	
	// Validate request has minimum length for meaningful analysis
	if len(strings.TrimSpace(input.Request)) < 10 {
		return a.errorFactory.NewValidationError(a.Name(), "input", "request", 
			"request too short for meaningful code analysis", input.Request)
	}
	
	return nil
}

func (a *Analyzer) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	// First run consolidated validation
	if output.Data == nil {
		return a.errorFactory.NewValidationError(a.Name(), "output", "data", "output data cannot be nil", nil)
	}
	if err := a.validator.ValidateJSON("data", output.Data, "output"); err != nil {
		return err
	}
	
	analysis, ok := output.Data.(CodeAnalysis)
	if !ok {
		return a.errorFactory.NewValidationError(a.Name(), "output", "data", 
			"output data must be a CodeAnalysis", fmt.Sprintf("%T", output.Data))
	}
	
	// Validate analysis has required fields
	if err := a.validator.ValidateRequired("language", analysis.Language, "output"); err != nil {
		return err
	}
	
	if err := a.validator.ValidateRequired("main_objective", analysis.MainObjective, "output"); err != nil {
		return err
	}
	
	if len(analysis.Requirements) == 0 {
		return a.errorFactory.NewValidationError(a.Name(), "output", "requirements", 
			"no requirements extracted", analysis.Requirements)
	}
	
	return nil
}

func (a *Analyzer) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	
	// Create recovery manager
	recovery := NewRecoveryManager()
	
	// Use resilient AI call with retry and fallback mechanisms
	var analysis CodeAnalysis
	
	result, err := a.resilience.ExecuteWithFallbacks(ctx, "analysis", func() (interface{}, error) {
		// Primary operation: AI-powered analysis
		return a.executeAIAnalysis(ctx, input.Request)
	}, input.Request)
	
	if err != nil {
		return core.PhaseOutput{}, &PhaseError{
			Phase:        a.Name(),
			Attempt:      1,
			Cause:        fmt.Errorf("analysis failed after retries and fallbacks: %w", err),
			Retryable:    false,
			RecoveryHint: "All analysis methods failed - check request format and API connectivity",
			Timestamp:    time.Now(),
		}
	}
	
	// Convert result to CodeAnalysis
	switch v := result.(type) {
	case CodeAnalysis:
		analysis = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &analysis); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to convert analysis result: %w", err)
		}
	default:
		return core.PhaseOutput{}, fmt.Errorf("unexpected analysis result type: %T", result)
	}
	
	// Validate analysis results using validation system
	if err := a.validator.ValidateLanguage(analysis.Language, "output"); err != nil {
		a.logger.Error("Analysis output validation failed", "error", err, "language", analysis.Language)
		return core.PhaseOutput{}, err
	}
	
	if err := a.validator.ValidateRequired("main_objective", analysis.MainObjective, "output"); err != nil {
		a.logger.Error("Analysis output validation failed", "error", err, "main_objective", analysis.MainObjective)
		return core.PhaseOutput{}, err
	}
	
	analysisData, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("marshaling analysis: %w", err)
	}
	
	// Add rollback for file save
	recovery.AddRollback(func() error {
		// In case of later failure, we could clean up the file
		// For now, we'll keep partial results
		return nil
	})
	
	if err := a.storage.Save(ctx, "analysis.json", analysisData); err != nil {
		return core.PhaseOutput{}, &PhaseError{
			Phase:        a.Name(),
			Attempt:      1,
			Cause:        fmt.Errorf("saving analysis: %w", err),
			Retryable:    false,
			RecoveryHint: "Check disk space and permissions",
			Timestamp:    time.Now(),
		}
	}
	
	return core.PhaseOutput{
		Data: analysis,
	}, nil
}

// executeAIAnalysis performs the primary AI-powered analysis with retry logic
func (a *Analyzer) executeAIAnalysis(ctx context.Context, request string) (CodeAnalysis, error) {
	var analysis CodeAnalysis
	
	// Build the analysis prompt
	prompt := a.buildAnalysisPrompt(request)
	
	// Execute with retry logic
	err := a.resilience.ExecuteWithRetry(ctx, func() error {
		// Use the constructed prompt
		response, err := a.agent.ExecuteJSON(ctx, prompt, nil)
		if err != nil {
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 2 * time.Second,
			}
		}
		
		// Parse the response
		if err := json.Unmarshal([]byte(response), &analysis); err != nil {
			a.logger.Error("Failed to parse AI analysis response", "error", err, "response", response)
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 1 * time.Second,
			}
		}
		
		// Basic validation to ensure we got meaningful data
		if analysis.Language == "" || analysis.Language == "Other" {
			return &core.RetryableError{
				Err:        fmt.Errorf("language detection failed: got '%s'", analysis.Language),
				RetryAfter: 1 * time.Second,
			}
		}
		
		return nil
	}, "ai_analysis")
	
	if err != nil {
		return CodeAnalysis{}, err
	}
	
	return analysis, nil
}

// buildAnalysisPrompt creates the analysis prompt with the user request
func (a *Analyzer) buildAnalysisPrompt(request string) string {
	return fmt.Sprintf(`You are a senior software engineer and architect. Analyze the following code request and provide a structured analysis.

User Request: %s

Please analyze this request and return ONLY a JSON response with the following structure:

{
  "language": "go|python|javascript|java|rust|cpp|other",
  "framework": "optional framework name if applicable",
  "complexity": "simple|moderate|complex", 
  "main_objective": "clear description of what needs to be built",
  "requirements": [
    "requirement 1",
    "requirement 2"
  ],
  "constraints": [
    "constraint 1 if any",
    "constraint 2 if any"
  ],
  "potential_risks": [
    "risk 1 if any", 
    "risk 2 if any"
  ]
}

Analysis Guidelines:
1. Language Detection: Identify the programming language from the request. If not explicitly mentioned, infer from context or choose the most appropriate one.
2. Complexity Assessment:
   - Simple: Basic functions, single file solutions, hello world examples
   - Moderate: Multiple files, basic API endpoints, simple data structures
   - Complex: Advanced architectures, multiple services, complex algorithms
3. Objective: Write a clear, one-sentence description of what needs to be built.
4. Requirements: Extract functional and non-functional requirements from the request.
5. Constraints: Identify any technical constraints, performance requirements, or limitations.
6. Risks: Consider potential implementation challenges or risks.

Return ONLY valid JSON, no markdown formatting or explanations.`, request)
}


// GetValidationRules returns validation rules for the analyzer
func (a *Analyzer) GetValidationRules() core.ValidationRules {
	return core.ValidationRules{
		RequiredInputFields:  []string{"request"},
		RequiredOutputFields: []string{"language", "main_objective", "requirements"},
		AllowedDataTypes:     []string{"CodeAnalysis"},
		CustomValidators: []core.ValidationFunc{
			a.validateLanguageDetection,
			a.validateRequirementsExtraction,
		},
	}
}

// validateLanguageDetection ensures language is properly detected
func (a *Analyzer) validateLanguageDetection(ctx context.Context, data interface{}) error {
	analysis, ok := data.(CodeAnalysis)
	if !ok {
		return fmt.Errorf("expected CodeAnalysis, got %T", data)
	}
	
	if analysis.Language == "" {
		return fmt.Errorf("language detection failed")
	}
	
	return nil
}

// validateRequirementsExtraction ensures requirements are extracted
func (a *Analyzer) validateRequirementsExtraction(ctx context.Context, data interface{}) error {
	analysis, ok := data.(CodeAnalysis)
	if !ok {
		return fmt.Errorf("expected CodeAnalysis, got %T", data)
	}
	
	if len(analysis.Requirements) == 0 {
		return fmt.Errorf("no requirements extracted from request")
	}
	
	return nil
}