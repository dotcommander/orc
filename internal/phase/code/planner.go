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

// Planner phase creates implementation plan
type Planner struct {
	BasePhase
	agent      core.Agent
	storage    core.Storage
	promptPath string
	logger       *slog.Logger
	validator    core.Validator
	errorFactory core.ErrorFactory
	resilience   *core.PhaseResilience
}

// NewPlanner creates a new planner phase
func NewPlanner(agent core.Agent, storage core.Storage, promptPath string, logger *slog.Logger) *Planner {
	return &Planner{
		BasePhase:    NewBasePhase("Planning", 5*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		logger:       logger,
		validator:    core.NewBaseValidator("Planning"),
		errorFactory: core.NewDefaultErrorFactory(),
		resilience:   core.NewPhaseResilience(),
	}
}

// Execute creates an implementation plan based on analysis
func (p *Planner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	var analysis CodeAnalysis
	
	// Handle both direct type and map from JSON marshaling
	switch v := input.Data.(type) {
	case CodeAnalysis:
		analysis = v
	case map[string]interface{}:
		// Convert map back to CodeAnalysis
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &analysis); err != nil {
			p.logger.Error("Planning input validation failed", "error", err)
			return core.PhaseOutput{}, fmt.Errorf("invalid analysis data: %w", err)
		}
	default:
		return core.PhaseOutput{}, fmt.Errorf("invalid input: expected CodeAnalysis, got %T", input.Data)
	}
	
	// Validate the analysis data using validation system
	if err := p.validator.ValidateLanguage(analysis.Language, "input"); err != nil {
		p.logger.Error("Planning input validation failed", "error", err, "language", analysis.Language)
		return core.PhaseOutput{}, err
	}
	
	if err := p.validator.ValidateRequired("main_objective", analysis.MainObjective, "input"); err != nil {
		p.logger.Error("Planning input validation failed", "error", err, "main_objective", analysis.MainObjective)
		return core.PhaseOutput{}, err
	}
	
	analysisJSON, _ := json.Marshal(analysis)
	
	// Use resilient AI call with retry and fallback mechanisms
	var plan ImplementationPlan
	
	result, err := p.resilience.ExecuteWithFallbacks(ctx, "planning", func() (interface{}, error) {
		return p.executePlanningWithRetry(ctx, string(analysisJSON))
	}, string(analysisJSON))
	
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("planning failed after retries and fallbacks: %w", err)
	}
	
	// Convert result to ImplementationPlan
	switch v := result.(type) {
	case ImplementationPlan:
		plan = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &plan); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to convert planning result: %w", err)
		}
	default:
		return core.PhaseOutput{}, fmt.Errorf("unexpected planning result type: %T", result)
	}
	
	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("marshaling plan: %w", err)
	}
	
	if err := p.storage.Save(ctx, "implementation_plan.json", planData); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("saving plan: %w", err)
	}
	
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"analysis": analysis,
			"plan":     plan,
		},
	}, nil
}

// executePlanningWithRetry performs the primary AI-powered planning with retry logic
func (p *Planner) executePlanningWithRetry(ctx context.Context, analysisJSON string) (ImplementationPlan, error) {
	var plan ImplementationPlan
	
	// Execute with retry logic
	err := p.resilience.ExecuteWithRetry(ctx, func() error {
		// Build planning prompt with analysis data
		prompt := p.buildPlanningPrompt(analysisJSON)
		response, err := p.agent.ExecuteJSON(ctx, prompt, nil)
		if err != nil {
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 2 * time.Second,
			}
		}
		
		// Parse the response
		if err := json.Unmarshal([]byte(response), &plan); err != nil {
			p.logger.Error("Failed to parse AI planning response", "error", err, "response", response)
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 1 * time.Second,
			}
		}
		
		// Basic validation to ensure we got meaningful data
		if plan.Overview == "" {
			return &core.RetryableError{
				Err:        fmt.Errorf("planning overview is empty"),
				RetryAfter: 1 * time.Second,
			}
		}
		
		return nil
	}, "ai_planning")
	
	if err != nil {
		return ImplementationPlan{}, err
	}
	
	return plan, nil
}

// buildPlanningPrompt creates the planning prompt with analysis data
func (p *Planner) buildPlanningPrompt(analysisJSON string) string {
	return fmt.Sprintf(`You are a senior software engineer creating an implementation plan. Based on the code analysis provided, create a detailed implementation plan.

Code Analysis:
%s

Please create an implementation plan and return ONLY a JSON response with the following structure:

{
  "overview": "Brief overview of the implementation approach",
  "steps": [
    {
      "order": 1,
      "description": "Step description",
      "code_files": ["file1.go", "file2.go"],
      "rationale": "Why this step is important",
      "time_estimate": "5-10 minutes"
    }
  ],
  "testing": {
    "unit_tests": ["test1.go", "test2.go"],
    "integration_tests": ["integration_test.go"],
    "edge_cases": ["edge case 1", "edge case 2"]
  }
}

Planning Guidelines:
1. Overview: Provide a 2-3 sentence summary of the implementation approach.
2. Steps: Break down implementation into logical, ordered steps.
3. Testing Strategy: Plan comprehensive testing.
4. File Organization: Follow language-specific conventions.

Return ONLY valid JSON, no markdown formatting or explanations.`, analysisJSON)
}

// ValidateInput validates input for the planner phase
func (p *Planner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	p.logger.Debug("Validating planner input",
		"has_data", input.Data != nil,
		"data_type", fmt.Sprintf("%T", input.Data))
	
	if input.Data == nil {
		return p.errorFactory.NewValidationError(p.Name(), "input", "data", 
			"planner requires analysis data from previous phase", nil)
	}
	
	// Try to extract analysis from different data formats
	var analysis CodeAnalysis
	var extractErr error
	
	switch v := input.Data.(type) {
	case CodeAnalysis:
		analysis = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		extractErr = json.Unmarshal(jsonData, &analysis)
	default:
		return p.errorFactory.NewValidationError(p.Name(), "input", "data", 
			"invalid data type for planner input", fmt.Sprintf("%T", input.Data))
	}
	
	if extractErr != nil {
		return p.errorFactory.NewValidationError(p.Name(), "input", "data", 
			fmt.Sprintf("failed to extract analysis: %v", extractErr), input.Data)
	}
	
	// Validate analysis content
	if err := p.validator.ValidateRequired("language", analysis.Language, "input"); err != nil {
		return err
	}
	
	if err := p.validator.ValidateRequired("main_objective", analysis.MainObjective, "input"); err != nil {
		return err
	}
	
	if len(analysis.Requirements) == 0 {
		return p.errorFactory.NewValidationError(p.Name(), "input", "requirements", 
			"no requirements found in analysis", analysis.Requirements)
	}
	
	return nil
}

// ValidateOutput validates output from the planner phase
func (p *Planner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	p.logger.Debug("Validating planner output",
		"has_error", output.Error != nil,
		"data_type", fmt.Sprintf("%T", output.Data))
	
	if output.Error != nil {
		return output.Error
	}
	
	if output.Data == nil {
		return p.errorFactory.NewValidationError(p.Name(), "output", "data", 
			"planner output cannot be nil", nil)
	}
	
	// Validate the output data structure
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		return p.errorFactory.NewValidationError(p.Name(), "output", "data", 
			"output data must be a map containing analysis and plan", fmt.Sprintf("%T", output.Data))
	}
	
	// Check for required keys
	planData, hasPlan := outputMap["plan"]
	if !hasPlan {
		return p.errorFactory.NewValidationError(p.Name(), "output", "plan", 
			"plan key missing from output", "missing")
	}
	
	// Validate plan structure
	var plan ImplementationPlan
	switch v := planData.(type) {
	case ImplementationPlan:
		plan = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &plan); err != nil {
			return p.errorFactory.NewValidationError(p.Name(), "output", "plan", 
				fmt.Sprintf("failed to parse plan: %v", err), planData)
		}
	default:
		return p.errorFactory.NewValidationError(p.Name(), "output", "plan", 
			"plan must be ImplementationPlan type", fmt.Sprintf("%T", planData))
	}
	
	// Validate plan content
	if err := p.validator.ValidateRequired("overview", plan.Overview, "output"); err != nil {
		return err
	}
	
	if len(plan.Steps) == 0 {
		return p.errorFactory.NewValidationError(p.Name(), "output", "steps", 
			"plan must contain at least one step", plan.Steps)
	}
	
	// Validate each step
	for i, step := range plan.Steps {
		if strings.TrimSpace(step.Description) == "" {
			return p.errorFactory.NewValidationError(p.Name(), "output", fmt.Sprintf("steps[%d].description", i), 
				"step description cannot be empty", step.Description)
		}
		
		if len(step.CodeFiles) == 0 {
			return p.errorFactory.NewValidationError(p.Name(), "output", fmt.Sprintf("steps[%d].code_files", i), 
				"step must specify code files to create/modify", step.CodeFiles)
		}
	}
	
	p.logger.Info("Planner output validation passed",
		"overview_length", len(plan.Overview),
		"steps_count", len(plan.Steps),
		"has_testing", plan.Testing.UnitTests != nil || plan.Testing.IntegrationTests != nil)
	
	return nil
}

// GetValidationRules returns validation rules for the planner
func (p *Planner) GetValidationRules() core.ValidationRules {
	return core.ValidationRules{
		RequiredInputFields:  []string{"data"},
		RequiredOutputFields: []string{"plan", "analysis"},
		AllowedDataTypes:     []string{"map[string]interface{}", "ImplementationPlan"},
		CustomValidators: []core.ValidationFunc{
			p.validatePlanStructure,
			p.validateStepSequence,
		},
	}
}

// validatePlanStructure ensures plan has proper structure
func (p *Planner) validatePlanStructure(ctx context.Context, data interface{}) error {
	outputMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map output, got %T", data)
	}
	
	if _, hasPlan := outputMap["plan"]; !hasPlan {
		return fmt.Errorf("plan missing from output")
	}
	
	return nil
}

// validateStepSequence ensures steps are in logical order
func (p *Planner) validateStepSequence(ctx context.Context, data interface{}) error {
	outputMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map output, got %T", data)
	}
	
	planData, exists := outputMap["plan"]
	if !exists {
		return fmt.Errorf("plan not found in output")
	}
	
	var plan ImplementationPlan
	switch v := planData.(type) {
	case ImplementationPlan:
		plan = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &plan); err != nil {
			return fmt.Errorf("failed to parse plan: %w", err)
		}
	default:
		return fmt.Errorf("invalid plan type: %T", planData)
	}
	
	// Check step ordering
	for i, step := range plan.Steps {
		if step.Order != i+1 {
			return fmt.Errorf("step %d has incorrect order %d", i+1, step.Order)
		}
	}
	
	return nil
}