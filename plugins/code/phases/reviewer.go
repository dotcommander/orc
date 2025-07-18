package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// Reviewer phase reviews generated code
type Reviewer struct {
	BasePhase
	agent      core.Agent
	storage    core.Storage
	promptPath string
	logger       *slog.Logger
	validator    core.Validator
	errorFactory core.ErrorFactory
	resilience   *core.PhaseResilience
}

// NewReviewer creates a new reviewer phase
func NewReviewer(agent core.Agent, storage core.Storage, promptPath string, logger *slog.Logger) *Reviewer {
	return &Reviewer{
		BasePhase:    NewBasePhase("Review", 5*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		logger:       logger,
		validator:    core.NewBaseValidator("Review"),
		errorFactory: core.NewDefaultErrorFactory(),
		resilience:   core.NewPhaseResilience(),
	}
}

// Execute reviews the generated code
func (r *Reviewer) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	// First run consolidated validation
	validator := core.NewBaseValidator("Reviewer")
	if err := validator.ValidateRequired("request", input.Request, "input"); err != nil {
		return err
	}
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return r.errorFactory.NewValidationError(r.Name(), "input", "data", 
			"input data must be a map containing generated code", fmt.Sprintf("%T", input.Data))
	}
	
	// Extract all context for comprehensive review
	var generated GeneratedCode
	if genData, ok := data["generated"].(GeneratedCode); ok {
		generated = genData
	} else if genMap, ok := data["generated"].(map[string]interface{}); ok {
		// Convert from map if needed
		jsonData, _ := json.Marshal(genMap)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return r.errorFactory.NewValidationError(r.Name(), "input", "generated", 
				fmt.Sprintf("invalid generated code data: %v", err), genMap)
		}
	} else {
		return r.errorFactory.NewValidationError(r.Name(), "input", "generated", 
			"generated code missing from input", "missing")
	}
	
	// Validate generated code has files
	if len(generated.Files) == 0 {
		return r.errorFactory.NewValidationError(r.Name(), "input", "files", 
			"no code files to review", generated.Files)
	}
	
	return nil
}

func (r *Reviewer) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	// First run consolidated validation
	validator := core.NewBaseValidator("Reviewer")
	if output.Data == nil {
		return r.errorFactory.NewValidationError(r.Name(), "output", "data", "output data cannot be nil", nil)
	}
	if err := validator.ValidateJSON("data", output.Data, "output"); err != nil {
		return err
	}
	
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		return r.errorFactory.NewValidationError(r.Name(), "output", "data", 
			"output data must be a map containing review results", fmt.Sprintf("%T", output.Data))
	}
	
	// Validate review results exist
	reviewData, hasReview := outputMap["review"]
	if !hasReview {
		return r.errorFactory.NewValidationError(r.Name(), "output", "review", 
			"review results missing from output", "missing")
	}
	
	// Validate review structure
	var review CodeReview
	switch v := reviewData.(type) {
	case CodeReview:
		review = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &review); err != nil {
			return r.errorFactory.NewValidationError(r.Name(), "output", "review", 
				fmt.Sprintf("failed to parse review: %v", err), reviewData)
		}
	default:
		return r.errorFactory.NewValidationError(r.Name(), "output", "review", 
			"review must be CodeReview type", fmt.Sprintf("%T", reviewData))
	}
	
	// Validate review has summary
	if err := r.validator.ValidateRequired("summary", review.Summary, "output"); err != nil {
		return err
	}
	
	return nil
}

func (r *Reviewer) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid input data")
	}
	
	// Extract all context for comprehensive review
	var generated GeneratedCode
	if genData, ok := data["generated"].(GeneratedCode); ok {
		generated = genData
	} else if genMap, ok := data["generated"].(map[string]interface{}); ok {
		// Convert from map if needed
		jsonData, _ := json.Marshal(genMap)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("invalid generated code data: %w", err)
		}
	} else {
		return core.PhaseOutput{}, fmt.Errorf("missing generated code in input")
	}
	
	// Include full context for better review
	reviewContext := map[string]interface{}{
		"generated": generated,
		"plan":      data["plan"],
		"analysis":  data["analysis"],
	}
	
	contextJSON, _ := json.Marshal(reviewContext)
	
	// Use resilient AI call with retry and fallback mechanisms
	var review CodeReview
	
	result, err := r.resilience.ExecuteWithFallbacks(ctx, "review", func() (interface{}, error) {
		return r.executeReviewWithRetry(ctx, string(contextJSON))
	}, string(contextJSON))
	
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("review failed after retries and fallbacks: %w", err)
	}
	
	// Convert result to CodeReview
	switch v := result.(type) {
	case CodeReview:
		review = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &review); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to convert review result: %w", err)
		}
	default:
		return core.PhaseOutput{}, fmt.Errorf("unexpected review result type: %T", result)
	}
	
	reviewData, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("marshaling review: %w", err)
	}
	
	if err := r.storage.Save(ctx, "review_report.json", reviewData); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("saving review: %w", err)
	}
	
	// Create final output markdown
	output := r.createOutputMarkdown(data, review)
	if err := r.storage.Save(ctx, "code_output.md", []byte(output)); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("saving output: %w", err)
	}
	
	return core.PhaseOutput{
		Data: review,
	}, nil
}

// executeReviewWithRetry performs the primary AI-powered review with retry logic
func (r *Reviewer) executeReviewWithRetry(ctx context.Context, contextJSON string) (CodeReview, error) {
	var review CodeReview
	
	// Execute with retry logic
	err := r.resilience.ExecuteWithRetry(ctx, func() error {
		// Build review prompt with context data
		prompt := r.buildReviewPrompt(contextJSON)
		response, err := r.agent.ExecuteJSON(ctx, prompt, nil)
		if err != nil {
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 2 * time.Second,
			}
		}
		
		// Parse the response
		if err := json.Unmarshal([]byte(response), &review); err != nil {
			r.logger.Error("Failed to parse AI review response", "error", err, "response", response)
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 1 * time.Second,
			}
		}
		
		// Basic validation to ensure we got meaningful data
		if len(review.Improvements) == 0 && review.Score < 1 {
			return &core.RetryableError{
				Err:        fmt.Errorf("review appears incomplete"),
				RetryAfter: 1 * time.Second,
			}
		}
		
		return nil
	}, "ai_review")
	
	if err != nil {
		return CodeReview{}, err
	}
	
	return review, nil
}

// buildReviewPrompt creates the review prompt with implementation context
func (r *Reviewer) buildReviewPrompt(contextJSON string) string {
	return fmt.Sprintf(`You are a senior software engineer conducting a thorough code review. Analyze the implemented code and provide constructive feedback.

Implementation to Review:
%s

Please review the code and return ONLY a JSON response with the following structure:

{
  "score": 8.5,
  "summary": "Overall assessment of the code quality and implementation",
  "strengths": [
    "Well-structured and readable code",
    "Proper error handling implemented",
    "Good separation of concerns"
  ],
  "improvements": [
    {
      "priority": "high|medium|low",
      "description": "Description of the improvement needed",
      "location": "file.php:line 25",
      "suggestion": "Specific suggestion for improvement"
    }
  ],
  "security_issues": [
    "Security concern 1 if any",
    "Security concern 2 if any"
  ],
  "best_practices": [
    "Best practice recommendation 1",
    "Best practice recommendation 2"
  ]
}

Review Criteria:
1. Code Quality (1-10 scale): Readability, maintainability, naming conventions
2. Functionality: Does the code meet the requirements correctly?
3. Security: Input validation, potential vulnerabilities, safe data handling
4. Performance: Efficient algorithms, resource usage, scalability
5. Best Practices: Language-specific conventions, design patterns

Return ONLY valid JSON, no markdown formatting or explanations.`, contextJSON)
}

// createOutputMarkdown creates the final output document
func (r *Reviewer) createOutputMarkdown(data map[string]interface{}, review CodeReview) string {
	// Safely extract data with type assertions
	var analysis CodeAnalysis
	if analysisMap, ok := data["analysis"].(map[string]interface{}); ok {
		jsonData, _ := json.Marshal(analysisMap)
		json.Unmarshal(jsonData, &analysis)
	}
	
	var plan ImplementationPlan
	if planMap, ok := data["plan"].(map[string]interface{}); ok {
		jsonData, _ := json.Marshal(planMap)
		json.Unmarshal(jsonData, &plan)
	}
	
	var generated GeneratedCode
	if genData, ok := data["generated"].(GeneratedCode); ok {
		generated = genData
	} else if genMap, ok := data["generated"].(map[string]interface{}); ok {
		jsonData, _ := json.Marshal(genMap)
		json.Unmarshal(jsonData, &generated)
	}
	
	output := fmt.Sprintf(`# Code Generation Report

## Task Overview
**Objective**: %s
**Language**: %s
**Complexity**: %s

## Requirements
%s

## Implementation Plan
%s

## Generated Code
%s

## Code Review
**Score**: %.1f/10
**Summary**: %s

### Strengths
%s

### Suggested Improvements
%s

## How to Use
%s
`,
		analysis.MainObjective,
		analysis.Language,
		analysis.Complexity,
		r.formatList(analysis.Requirements),
		plan.Overview,
		r.formatCodeFiles(generated.Files),
		review.Score,
		review.Summary,
		r.formatList(review.Strengths),
		r.formatImprovements(review.Improvements),
		generated.RunInstructions,
	)
	
	return output
}

func (r *Reviewer) formatList(items []string) string {
	result := ""
	for _, item := range items {
		result += fmt.Sprintf("- %s\n", item)
	}
	return result
}

func (r *Reviewer) formatCodeFiles(files []CodeFile) string {
	result := ""
	for _, file := range files {
		result += fmt.Sprintf("\n### %s\n```%s\n%s\n```\n", file.Path, file.Language, file.Content)
	}
	return result
}

func (r *Reviewer) formatImprovements(improvements []Improvement) string {
	result := ""
	for _, imp := range improvements {
		result += fmt.Sprintf("- **%s**: %s\n  - Location: %s\n  - Suggestion: %s\n", 
			imp.Priority, imp.Description, imp.Location, imp.Suggestion)
	}
	return result
}


// GetValidationRules returns validation rules for the reviewer
func (r *Reviewer) GetValidationRules() core.ValidationRules {
	return core.ValidationRules{
		RequiredInputFields:  []string{"data", "generated"},
		RequiredOutputFields: []string{"score", "summary", "strengths"},
		AllowedDataTypes:     []string{"CodeReview"},
		CustomValidators: []core.ValidationFunc{
			r.validateReviewScore,
			r.validateReviewContent,
		},
	}
}

// validateReviewScore ensures score is within valid range
func (r *Reviewer) validateReviewScore(ctx context.Context, data interface{}) error {
	review, ok := data.(CodeReview)
	if !ok {
		return fmt.Errorf("expected CodeReview, got %T", data)
	}
	
	if review.Score < 0 || review.Score > 10 {
		return fmt.Errorf("score %.1f out of valid range [0-10]", review.Score)
	}
	
	return nil
}

// validateReviewContent ensures review has meaningful content
func (r *Reviewer) validateReviewContent(ctx context.Context, data interface{}) error {
	review, ok := data.(CodeReview)
	if !ok {
		return fmt.Errorf("expected CodeReview, got %T", data)
	}
	
	if len(review.Summary) < 20 {
		return fmt.Errorf("review summary too short (minimum 20 characters)")
	}
	
	if len(review.Strengths) == 0 {
		return fmt.Errorf("review must identify at least one strength")
	}
	
	return nil
}