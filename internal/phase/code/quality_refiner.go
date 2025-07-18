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

// QualityRefiner performs iterative improvement like our contextual editor
type QualityRefiner struct {
	BasePhase
	agent  core.Agent
	logger *slog.Logger
}

// RefinementPlan defines how we'll improve the generated code
type RefinementPlan struct {
	Passes        []RefinementPass       `json:"passes"`
	QualityGoals  QualityMetrics         `json:"quality_goals"`
	Context       map[string]interface{} `json:"context"`
}

type RefinementPass struct {
	Name        string   `json:"name"`
	Focus       string   `json:"focus"`
	Criteria    []string `json:"criteria"`
	Files       []string `json:"files"`
	Order       int      `json:"order"`
}

// RefinementProgress tracks improvement iterations
type RefinementProgress struct {
	CurrentPass     int                    `json:"current_pass"`
	CompletedPasses []string               `json:"completed_passes"`
	Iterations      map[string][]Iteration `json:"iterations"`
	FinalCode       map[string]string      `json:"final_code"`
	QualityMetrics  map[string]float64     `json:"quality_metrics"`
	Context         map[string]interface{} `json:"context"`
}

type Iteration struct {
	Number        int                 `json:"number"`
	Focus         string              `json:"focus"`
	Changes       []CodeChange        `json:"changes"`
	QualityScore  float64             `json:"quality_score"`
	Improvements  []string            `json:"improvements"`
	NextActions   []string            `json:"next_actions"`
	Timestamp     time.Time           `json:"timestamp"`
	Context       map[string]interface{} `json:"context"`
}

type CodeChange struct {
	File        string `json:"file"`
	Type        string `json:"type"` // addition, modification, deletion, refactor
	Description string `json:"description"`
	Before      string `json:"before,omitempty"`
	After       string `json:"after"`
	Reason      string `json:"reason"`
	Impact      string `json:"impact"`
}

func NewQualityRefiner(agent core.Agent, logger *slog.Logger) *QualityRefiner {
	return &QualityRefiner{
		BasePhase: NewBasePhase("QualityRefiner", 5*time.Minute),
		agent:     agent,
		logger:    logger.With("component", "quality_refiner"),
	}
}

func (qr *QualityRefiner) Name() string {
	return "QualityRefiner"
}

func (qr *QualityRefiner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	qr.logger.Info("Starting quality refinement")

	// Extract inputs from previous phases
	exploration, buildPlan, generatedCode, err := qr.extractInputs(input)
	if err != nil {
		return core.PhaseOutput{}, err
	}

	// Create refinement plan based on project quality goals
	refinementPlan, err := qr.createRefinementPlan(ctx, exploration, buildPlan, generatedCode)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("failed to create refinement plan: %w", err)
	}

	// Execute iterative refinement
	progress := &RefinementProgress{
		CurrentPass:     0,
		CompletedPasses: make([]string, 0),
		Iterations:      make(map[string][]Iteration),
		FinalCode:       make(map[string]string),
		QualityMetrics:  make(map[string]float64),
		Context:         make(map[string]interface{}),
	}

	// Initialize with original code
	for path, content := range generatedCode {
		progress.FinalCode[path] = content
		progress.Iterations[path] = make([]Iteration, 0)
	}

	// Execute each refinement pass
	for i, pass := range refinementPlan.Passes {
		qr.logger.Info("Executing refinement pass", "pass", pass.Name, "index", i)
		
		progress.CurrentPass = i
		if err := qr.executeRefinementPass(ctx, pass, refinementPlan, exploration, progress); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to execute refinement pass %s: %w", pass.Name, err)
		}
		
		progress.CompletedPasses = append(progress.CompletedPasses, pass.Name)
	}

	// Calculate final quality metrics
	if err := qr.calculateFinalQualityMetrics(ctx, progress, exploration); err != nil {
		qr.logger.Warn("Failed to calculate final quality metrics", "error", err)
	}

	qr.logger.Info("Quality refinement completed",
		"passes_completed", len(progress.CompletedPasses),
		"total_iterations", qr.getTotalIterations(progress),
		"files_refined", len(progress.FinalCode))

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"exploration":         exploration,    // Pass along exploration data
			"build_plan":          buildPlan,      // Pass along build plan
			"refinement_plan":     refinementPlan,
			"refinement_progress": progress,
			"refined_code":        progress.FinalCode,
			"quality_metrics":     progress.QualityMetrics,
		},
	}, nil
}

func (qr *QualityRefiner) extractInputs(input core.PhaseInput) (*ProjectExploration, *BuildPlan, map[string]string, error) {
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

	buildPlanData, ok := data["build_plan"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("build plan not found")
	}
	buildPlan, ok := buildPlanData.(*BuildPlan)
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid build plan type")
	}

	generatedCodeData, ok := data["generated_code"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("generated code not found")
	}
	generatedCode, ok := generatedCodeData.(map[string]string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("invalid generated code type")
	}

	return exploration, buildPlan, generatedCode, nil
}

func (qr *QualityRefiner) createRefinementPlan(ctx context.Context, exploration *ProjectExploration, buildPlan *BuildPlan, generatedCode map[string]string) (*RefinementPlan, error) {
	prompt := qr.buildRefinementPlanPrompt(exploration, buildPlan, generatedCode)
	
	response, err := qr.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get refinement plan: %w", err)
	}

	plan, err := qr.parseRefinementPlan(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refinement plan: %w", err)
	}

	plan.Context = map[string]interface{}{
		"exploration":         exploration,
		"build_plan":          buildPlan,
		"planning_response":   response,
	}

	return plan, nil
}

func (qr *QualityRefiner) executeRefinementPass(ctx context.Context, pass RefinementPass, plan *RefinementPlan, exploration *ProjectExploration, progress *RefinementProgress) error {
	// Get files to refine in this pass
	filesToRefine := pass.Files
	if len(filesToRefine) == 0 {
		// If no specific files, refine all
		for path := range progress.FinalCode {
			filesToRefine = append(filesToRefine, path)
		}
	}

	for _, filePath := range filesToRefine {
		if err := qr.refineFile(ctx, filePath, pass, plan, exploration, progress); err != nil {
			return fmt.Errorf("failed to refine file %s: %w", filePath, err)
		}
	}

	return nil
}

func (qr *QualityRefiner) refineFile(ctx context.Context, filePath string, pass RefinementPass, plan *RefinementPlan, exploration *ProjectExploration, progress *RefinementProgress) error {
	currentContent := progress.FinalCode[filePath]
	if currentContent == "" {
		return fmt.Errorf("no content found for file %s", filePath)
	}

	prompt := qr.buildFileRefinementPrompt(filePath, currentContent, pass, plan, exploration, progress)
	
	response, err := qr.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get file refinement: %w", err)
	}

	iteration, refinedContent, err := qr.parseFileRefinement(response)
	if err != nil {
		return fmt.Errorf("failed to parse file refinement: %w", err)
	}

	// Update progress
	iteration.Number = len(progress.Iterations[filePath]) + 1
	iteration.Focus = pass.Focus
	iteration.Timestamp = time.Now()
	
	progress.Iterations[filePath] = append(progress.Iterations[filePath], iteration)
	progress.FinalCode[filePath] = refinedContent

	qr.logger.Info("Refined file", 
		"file", filePath, 
		"pass", pass.Name,
		"iteration", iteration.Number,
		"quality_score", iteration.QualityScore)

	return nil
}

func (qr *QualityRefiner) buildRefinementPlanPrompt(exploration *ProjectExploration, buildPlan *BuildPlan, generatedCode map[string]string) string {
	files := make([]string, 0, len(generatedCode))
	for path := range generatedCode {
		files = append(files, path)
	}

	return fmt.Sprintf(`You are a senior software architect creating a systematic refinement plan for a %s project.

Project Quality Goals:
- Security: %s
- Performance: %s  
- Maintainability: %s
- User Experience: %s

Generated Files: %s

Tech Stack: %s with %s architecture

Create a multi-pass refinement plan that systematically improves the code quality. Each pass should focus on specific aspects and build upon previous improvements.

Suggested pass focuses:
1. Code Structure & Organization
2. Security & Validation
3. Performance & Efficiency
4. Error Handling & Robustness
5. Documentation & Maintainability
6. Integration & Testing

Return your response in this JSON format:
{
  "passes": [
    {
      "name": "Pass name",
      "focus": "What this pass focuses on",
      "criteria": ["quality criteria for this pass"],
      "files": ["specific files to focus on, or empty for all"],
      "order": 1
    }
  ],
  "quality_goals": {
    "security": "Security improvement targets",
    "performance": "Performance improvement targets",
    "maintainability": "Maintainability improvement targets",
    "user_experience": "UX improvement targets"
  }
}`,
		exploration.ProjectType,
		exploration.QualityGoals.Security,
		exploration.QualityGoals.Performance,
		exploration.QualityGoals.Maintainability,
		exploration.QualityGoals.UserExperience,
		strings.Join(files, ", "),
		exploration.TechStack.Language,
		exploration.Architecture.Pattern)
}

func (qr *QualityRefiner) buildFileRefinementPrompt(filePath, currentContent string, pass RefinementPass, plan *RefinementPlan, exploration *ProjectExploration, progress *RefinementProgress) string {
	// Build context from other files
	contextFiles := make([]string, 0)
	for path, content := range progress.FinalCode {
		if path != filePath && len(content) > 0 {
			contextFiles = append(contextFiles, fmt.Sprintf("%s:\n"+"`"+"``"+"\n%s\n"+"`"+"``", path, content))
		}
	}

	// Get previous iterations for this file
	previousIterations := make([]string, 0)
	if iterations, exists := progress.Iterations[filePath]; exists {
		for _, iter := range iterations {
			previousIterations = append(previousIterations, fmt.Sprintf("Iteration %d (%s): %s", 
				iter.Number, iter.Focus, strings.Join(iter.Improvements, "; ")))
		}
	}

	return fmt.Sprintf(`You are performing quality refinement pass "%s" on file "%s".

Pass Focus: %s
Pass Criteria: %s

Current File Content:
` + "```" + `
%s
` + "```" + `

Project Context:
%s

Quality Goals for This Pass:
- Security: %s
- Performance: %s
- Maintainability: %s

Related Files:
%s

Previous Iterations:
%s

Systematically improve this file focusing on "%s". Consider:
1. Code structure and organization
2. Best practices for %s
3. Security and validation
4. Error handling
5. Performance optimizations
6. Maintainability and clarity
7. Integration with other files

Make meaningful improvements while preserving functionality.

Return your response in this JSON format:
{
  "refined_content": "The improved file content",
  "changes": [
    {
      "type": "addition|modification|deletion|refactor",
      "description": "What was changed",
      "before": "original code if modification",
      "after": "new code",
      "reason": "why this change was made",
      "impact": "expected impact of this change"
    }
  ],
  "quality_score": 8.5,
  "improvements": ["improvement1", "improvement2"],
  "next_actions": ["what should be done next"]
}`,
		pass.Name,
		filePath,
		pass.Focus,
		strings.Join(pass.Criteria, "; "),
		currentContent,
		strings.Join(exploration.Requirements, "; "),
		exploration.QualityGoals.Security,
		exploration.QualityGoals.Performance,
		exploration.QualityGoals.Maintainability,
		strings.Join(contextFiles, "\n\n"),
		strings.Join(previousIterations, "\n"),
		pass.Focus,
		exploration.TechStack.Language)
}

func (qr *QualityRefiner) parseRefinementPlan(response string) (*RefinementPlan, error) {
	var plan RefinementPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse refinement plan JSON: %w", err)
	}
	return &plan, nil
}

func (qr *QualityRefiner) parseFileRefinement(response string) (Iteration, string, error) {
	var result struct {
		RefinedContent string       `json:"refined_content"`
		Changes        []CodeChange `json:"changes"`
		QualityScore   float64      `json:"quality_score"`
		Improvements   []string     `json:"improvements"`
		NextActions    []string     `json:"next_actions"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return Iteration{}, "", fmt.Errorf("failed to parse file refinement JSON: %w", err)
	}

	iteration := Iteration{
		Changes:      result.Changes,
		QualityScore: result.QualityScore,
		Improvements: result.Improvements,
		NextActions:  result.NextActions,
	}

	return iteration, result.RefinedContent, nil
}

func (qr *QualityRefiner) calculateFinalQualityMetrics(ctx context.Context, progress *RefinementProgress, exploration *ProjectExploration) error {
	// Calculate aggregate quality metrics
	totalScore := 0.0
	totalIterations := 0

	for _, iterations := range progress.Iterations {
		for _, iteration := range iterations {
			totalScore += iteration.QualityScore
			totalIterations++
		}
	}

	if totalIterations > 0 {
		progress.QualityMetrics["average_quality_score"] = totalScore / float64(totalIterations)
	}

	progress.QualityMetrics["total_iterations"] = float64(totalIterations)
	progress.QualityMetrics["files_refined"] = float64(len(progress.FinalCode))
	progress.QualityMetrics["passes_completed"] = float64(len(progress.CompletedPasses))

	return nil
}

func (qr *QualityRefiner) getTotalIterations(progress *RefinementProgress) int {
	total := 0
	for _, iterations := range progress.Iterations {
		total += len(iterations)
	}
	return total
}

func (qr *QualityRefiner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("input data is required for quality refinement")
	}
	return nil
}

func (qr *QualityRefiner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	if output.Data == nil {
		return fmt.Errorf("refinement data is required")
	}
	return nil
}

func (qr *QualityRefiner) EstimatedDuration() time.Duration {
	return 3 * time.Minute
}

func (qr *QualityRefiner) CanRetry(err error) bool {
	return true
}