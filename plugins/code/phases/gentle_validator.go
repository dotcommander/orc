package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// GentleValidator provides constructive guidance instead of harsh failures
type GentleValidator struct {
	BasePhase
	agent  core.Agent
	logger *slog.Logger
}

// ValidationResult represents the outcome of gentle validation
type ValidationResult struct {
	OverallScore     float64                `json:"overall_score"`
	PassingCriteria  []string               `json:"passing_criteria"`
	ImprovementAreas []ImprovementArea      `json:"improvement_areas"`
	FileResults      map[string]FileResult  `json:"file_results"`
	Recommendations  []Recommendation       `json:"recommendations"`
	NextSteps        []string               `json:"next_steps"`
	ReadyForUse      bool                   `json:"ready_for_use"`
	Context          map[string]interface{} `json:"context"`
}

type ImprovementArea struct {
	Category    string   `json:"category"`
	Priority    string   `json:"priority"` // critical, important, nice-to-have
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Guidance    string   `json:"guidance"`
	Examples    []string `json:"examples,omitempty"`
}

type FileResult struct {
	Path            string             `json:"path"`
	Score           float64            `json:"score"`
	Status          string             `json:"status"` // excellent, good, needs_improvement, needs_attention
	Strengths       []string           `json:"strengths"`
	Improvements    []FileImprovement  `json:"improvements"`
	SecurityCheck   SecurityResult     `json:"security_check"`
	QualityMetrics  QualityAssessment  `json:"quality_metrics"`
}

type FileImprovement struct {
	Line        int    `json:"line,omitempty"`
	Type        string `json:"type"` // security, performance, maintainability, style
	Issue       string `json:"issue"`
	Suggestion  string `json:"suggestion"`
	Priority    string `json:"priority"`
	Example     string `json:"example,omitempty"`
}

type SecurityResult struct {
	Score       float64          `json:"score"`
	Issues      []SecurityIssue  `json:"issues"`
	Compliant   bool             `json:"compliant"`
	Guidance    string           `json:"guidance"`
}

type SecurityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
}

type QualityAssessment struct {
	Readability     float64 `json:"readability"`
	Maintainability float64 `json:"maintainability"`
	Testability     float64 `json:"testability"`
	Performance     float64 `json:"performance"`
	Documentation   float64 `json:"documentation"`
}

type Recommendation struct {
	Category    string   `json:"category"`
	Action      string   `json:"action"`
	Rationale   string   `json:"rationale"`
	Impact      string   `json:"impact"`
	Resources   []string `json:"resources,omitempty"`
}

func NewGentleValidator(agent core.Agent, logger *slog.Logger) *GentleValidator {
	return &GentleValidator{
		BasePhase: NewBasePhase("GentleValidator", 3*time.Minute),
		agent:     agent,
		logger:    logger.With("component", "gentle_validator"),
	}
}

func (gv *GentleValidator) Name() string {
	return "GentleValidator"
}

func (gv *GentleValidator) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	gv.logger.Info("Starting gentle validation")

	// Extract inputs from previous phases
	exploration, refinedCode, qualityMetrics, err := gv.extractInputs(input)
	if err != nil {
		return core.PhaseOutput{}, err
	}

	// Perform comprehensive but gentle validation
	result, err := gv.performGentleValidation(ctx, exploration, refinedCode, qualityMetrics)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("failed to perform gentle validation: %w", err)
	}

	// Always succeed but provide guidance
	gv.logger.Info("Gentle validation completed",
		"overall_score", result.OverallScore,
		"ready_for_use", result.ReadyForUse,
		"improvement_areas", len(result.ImprovementAreas),
		"files_validated", len(result.FileResults))

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"validation_result": result,
			"validated_code":    refinedCode,
			"guidance":          result.Recommendations,
			"ready_for_use":     result.ReadyForUse,
		},
	}, nil
}

func (gv *GentleValidator) extractInputs(input core.PhaseInput) (*ProjectExploration, map[string]string, map[string]float64, error) {
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid input data format")
	}

	explorationData, ok := data["exploration"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("exploration data not found")
	}
	exploration, ok := explorationData.(*ProjectExploration)
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid exploration data type")
	}

	refinedCodeData, ok := data["refined_code"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("refined code not found")
	}
	refinedCode, ok := refinedCodeData.(map[string]string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid refined code type")
	}

	qualityMetricsData, ok := data["quality_metrics"]
	if !ok {
		// Quality metrics are optional
		return exploration, refinedCode, make(map[string]float64), nil
	}
	qualityMetrics, ok := qualityMetricsData.(map[string]float64)
	if !ok {
		return exploration, refinedCode, make(map[string]float64), nil
	}

	return exploration, refinedCode, qualityMetrics, nil
}

func (gv *GentleValidator) performGentleValidation(ctx context.Context, exploration *ProjectExploration, refinedCode map[string]string, qualityMetrics map[string]float64) (*ValidationResult, error) {
	result := &ValidationResult{
		FileResults:      make(map[string]FileResult),
		ImprovementAreas: make([]ImprovementArea, 0),
		Recommendations:  make([]Recommendation, 0),
		PassingCriteria:  make([]string, 0),
		NextSteps:        make([]string, 0),
		Context:          make(map[string]interface{}),
	}

	// Validate each file gently
	totalScore := 0.0
	fileCount := 0

	for filePath, content := range refinedCode {
		fileResult, err := gv.validateFile(ctx, filePath, content, exploration, refinedCode)
		if err != nil {
			gv.logger.Warn("Failed to validate file", "file", filePath, "error", err)
			// Don't fail the entire process, just note the issue
			continue
		}

		result.FileResults[filePath] = fileResult
		totalScore += fileResult.Score
		fileCount++
	}

	// Calculate overall score
	if fileCount > 0 {
		result.OverallScore = totalScore / float64(fileCount)
	}

	// Analyze overall project health
	if err := gv.analyzeProjectHealth(ctx, result, exploration, refinedCode); err != nil {
		gv.logger.Warn("Failed to analyze project health", "error", err)
	}

	// Determine if ready for use (we're gentle - usually yes with guidance)
	result.ReadyForUse = result.OverallScore >= 6.0 // Gentle threshold

	// Add encouragement and next steps
	gv.addConstructiveGuidance(result, exploration)

	return result, nil
}

func (gv *GentleValidator) validateFile(ctx context.Context, filePath, content string, exploration *ProjectExploration, allCode map[string]string) (FileResult, error) {
	prompt := gv.buildFileValidationPrompt(filePath, content, exploration, allCode)
	
	response, err := gv.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return FileResult{}, fmt.Errorf("failed to get file validation: %w", err)
	}

	fileResult, err := gv.parseFileValidation(response)
	if err != nil {
		return FileResult{}, fmt.Errorf("failed to parse file validation: %w", err)
	}

	fileResult.Path = filePath
	return fileResult, nil
}

func (gv *GentleValidator) analyzeProjectHealth(ctx context.Context, result *ValidationResult, exploration *ProjectExploration, refinedCode map[string]string) error {
	prompt := gv.buildProjectAnalysisPrompt(result, exploration, refinedCode)
	
	response, err := gv.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get project analysis: %w", err)
	}

	analysis, err := gv.parseProjectAnalysis(response)
	if err != nil {
		return fmt.Errorf("failed to parse project analysis: %w", err)
	}

	// Merge analysis into result
	result.ImprovementAreas = append(result.ImprovementAreas, analysis.ImprovementAreas...)
	result.Recommendations = append(result.Recommendations, analysis.Recommendations...)
	result.NextSteps = analysis.NextSteps
	result.PassingCriteria = analysis.PassingCriteria

	return nil
}

func (gv *GentleValidator) buildFileValidationPrompt(filePath, content string, exploration *ProjectExploration, allCode map[string]string) string {
	// Build context from other files
	contextFiles := make([]string, 0)
	for path, code := range allCode {
		if path != filePath && len(code) > 0 {
			contextFiles = append(contextFiles, fmt.Sprintf("%s: %d lines", path, len(strings.Split(code, "\n"))))
		}
	}

	return fmt.Sprintf(`You are a senior code reviewer providing gentle, constructive validation for file "%s" in a %s project.

File Content:
` + "```" + `
%s
` + "```" + `

Project Context:
- Language: %s
- Architecture: %s  
- Quality Goals: Security (%s), Performance (%s), Maintainability (%s)
- Related Files: %s

Provide constructive validation that:
1. Acknowledges strengths and good practices
2. Identifies improvement opportunities gently
3. Provides specific, actionable guidance
4. Focuses on learning and growth
5. Considers security without being alarmist
6. Assesses quality metrics fairly

Be encouraging while being thorough. The goal is guidance, not gatekeeping.

Return your response in this JSON format:
{
  "score": 8.5,
  "status": "good",
  "strengths": ["strength1", "strength2"],
  "improvements": [
    {
      "line": 10,
      "type": "security",
      "issue": "Input validation could be stronger",
      "suggestion": "Consider adding sanitization before processing",
      "priority": "important",
      "example": "htmlspecialchars($input, ENT_QUOTES, 'UTF-8')"
    }
  ],
  "security_check": {
    "score": 7.5,
    "issues": [
      {
        "type": "input_validation",
        "severity": "medium",
        "location": "line 15",
        "description": "User input not sanitized",
        "fix": "Add input validation and sanitization"
      }
    ],
    "compliant": true,
    "guidance": "Overall security is good with some minor improvements needed"
  },
  "quality_metrics": {
    "readability": 8.0,
    "maintainability": 7.5,
    "testability": 6.0,
    "performance": 8.5,
    "documentation": 5.0
  }
}`,
		filePath,
		exploration.ProjectType,
		content,
		exploration.TechStack.Language,
		exploration.Architecture.Pattern,
		exploration.QualityGoals.Security,
		exploration.QualityGoals.Performance,
		exploration.QualityGoals.Maintainability,
		strings.Join(contextFiles, ", "))
}

func (gv *GentleValidator) buildProjectAnalysisPrompt(result *ValidationResult, exploration *ProjectExploration, refinedCode map[string]string) string {
	fileScores := make([]string, 0)
	for path, fileResult := range result.FileResults {
		fileScores = append(fileScores, fmt.Sprintf("%s: %.1f (%s)", path, fileResult.Score, fileResult.Status))
	}

	return fmt.Sprintf(`You are analyzing the overall health of a %s project for constructive guidance.

Project Overview:
- Overall Score: %.1f
- Files: %s
- Quality Goals: %s

File Results:
%s

Provide encouraging, constructive project-level analysis that:
1. Identifies system-wide improvement opportunities
2. Suggests specific next steps for growth
3. Acknowledges what's working well
4. Provides learning-focused recommendations
5. Considers the project's goals and constraints

Be supportive and focus on continuous improvement rather than criticism.

Return your response in this JSON format:
{
  "improvement_areas": [
    {
      "category": "security",
      "priority": "important",
      "description": "Input validation consistency",
      "files": ["file1.php", "file2.php"],
      "guidance": "Consider implementing a centralized validation system",
      "examples": ["Example implementation approaches"]
    }
  ],
  "recommendations": [
    {
      "category": "architecture",
      "action": "Add error handling middleware",
      "rationale": "Centralized error handling improves maintainability",
      "impact": "Better user experience and easier debugging",
      "resources": ["link1", "link2"]
    }
  ],
  "next_steps": ["step1", "step2", "step3"],
  "passing_criteria": ["criteria1", "criteria2"]
}`,
		exploration.ProjectType,
		result.OverallScore,
		strings.Join(exploration.Requirements, "; "),
		exploration.QualityGoals.Security,
		strings.Join(fileScores, "\n"))
}

func (gv *GentleValidator) parseFileValidation(response string) (FileResult, error) {
	var result FileResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return FileResult{}, fmt.Errorf("failed to parse file validation JSON: %w", err)
	}
	return result, nil
}

func (gv *GentleValidator) parseProjectAnalysis(response string) (*struct {
	ImprovementAreas []ImprovementArea `json:"improvement_areas"`
	Recommendations  []Recommendation  `json:"recommendations"`
	NextSteps        []string          `json:"next_steps"`
	PassingCriteria  []string          `json:"passing_criteria"`
}, error) {
	var analysis struct {
		ImprovementAreas []ImprovementArea `json:"improvement_areas"`
		Recommendations  []Recommendation  `json:"recommendations"`
		NextSteps        []string          `json:"next_steps"`
		PassingCriteria  []string          `json:"passing_criteria"`
	}

	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse project analysis JSON: %w", err)
	}

	return &analysis, nil
}

func (gv *GentleValidator) addConstructiveGuidance(result *ValidationResult, exploration *ProjectExploration) {
	// Add encouraging context
	result.Context["validation_philosophy"] = "This validation focuses on growth and improvement rather than gatekeeping"
	result.Context["project_strengths"] = gv.identifyProjectStrengths(result)
	
	// Ensure we have positive next steps
	if len(result.NextSteps) == 0 {
		result.NextSteps = []string{
			"Run the application in a development environment",
			"Test the core functionality manually",
			"Consider adding automated tests for key features",
			"Review and implement the suggested improvements gradually",
		}
	}

	// Add encouragement for lower scores
	if result.OverallScore < 7.0 {
		result.NextSteps = append([]string{
			"Great foundation! Focus on the priority improvements to enhance the application",
		}, result.NextSteps...)
	}
}

func (gv *GentleValidator) identifyProjectStrengths(result *ValidationResult) []string {
	strengths := make([]string, 0)
	
	// Count strengths across files
	strengthMap := make(map[string]int)
	for _, fileResult := range result.FileResults {
		for _, strength := range fileResult.Strengths {
			strengthMap[strength]++
		}
	}

	// Identify common strengths
	for strength, count := range strengthMap {
		if count > 1 {
			strengths = append(strengths, fmt.Sprintf("%s (consistent across %d files)", strength, count))
		}
	}

	if len(strengths) == 0 {
		strengths = append(strengths, "Functional implementation that meets basic requirements")
	}

	return strengths
}

func (gv *GentleValidator) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("input data is required for gentle validation")
	}
	return nil
}

func (gv *GentleValidator) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	if output.Data == nil {
		return fmt.Errorf("validation data is required")
	}
	return nil
}

func (gv *GentleValidator) EstimatedDuration() time.Duration {
	return 2 * time.Minute
}

func (gv *GentleValidator) CanRetry(err error) bool {
	return true // Gentle validation can always be retried
}