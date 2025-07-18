package code

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// IterativeRefiner replaces QualityRefiner with infinite iterative improvement
type IterativeRefiner struct {
	BasePhase
	improvementEngine *core.IterativeImprovementEngine
	logger            *slog.Logger
}

// NewIterativeRefiner creates a new iterative refiner phase
func NewIterativeRefiner(agent core.Agent, logger *slog.Logger) *IterativeRefiner {
	config := core.ImprovementConfig{
		MaxIterations:        100, // High limit, will stop when quality reached
		TargetQuality:        0.95, // 95% quality target
		ImprovementStrategy:  "adaptive",
		ParallelImprovements: true,
		LearningEnabled:      true,
		CheckpointInterval:   5,
		QualityThresholds: map[string]float64{
			"security":       0.98, // Higher threshold for security
			"performance":    0.90,
			"maintainability": 0.85,
			"readability":    0.90,
		},
		FocusAreas:   []string{"security", "error-handling", "performance"},
		AdaptiveMode: true,
	}

	engine := core.NewIterativeImprovementEngine(agent, logger, config)

	return &IterativeRefiner{
		BasePhase:         NewBasePhase("IterativeRefiner", 20*time.Minute),
		improvementEngine: engine,
		logger:            logger.With("component", "iterative_refiner"),
	}
}

func (ir *IterativeRefiner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	ir.logger.Info("Starting iterative refinement")

	// Extract inputs from previous phases
	exploration, buildPlan, generatedCode, err := ir.extractInputs(input)
	if err != nil {
		return core.PhaseOutput{}, err
	}

	// Register domain-specific inspectors
	ir.registerCodeInspectors(exploration)

	// Process each file through iterative improvement
	refinedCode := make(map[string]string)
	improvementSessions := make(map[string]*core.ImprovementSession)

	for filePath, content := range generatedCode {
		ir.logger.Info("Refining file", "path", filePath)

		// Determine target quality based on file type and importance
		targetQuality := ir.determineTargetQuality(filePath, exploration)

		// Run iterative improvement
		session, err := ir.improvementEngine.ImproveContent(ctx, content, targetQuality)
		if err != nil {
			ir.logger.Error("Iterative improvement failed", 
				"file", filePath, 
				"error", err)
			// Still use the original content if improvement fails
			refinedCode[filePath] = content
			continue
		}

		// Extract final improved content
		finalContent := ir.extractFinalContent(session, content)
		refinedCode[filePath] = finalContent
		improvementSessions[filePath] = session

		ir.logger.Info("File refinement completed",
			"file", filePath,
			"initial_quality", session.InitialQuality,
			"final_quality", session.FinalQuality,
			"iterations", session.TotalIterations,
			"success", session.Success)
	}

	// Generate refinement report
	report := ir.generateRefinementReport(improvementSessions, exploration)

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"exploration":           exploration,
			"build_plan":            buildPlan,
			"refined_code":          refinedCode,
			"improvement_sessions":  improvementSessions,
			"refinement_report":     report,
			"learning_insights":     ir.extractAllLearningInsights(improvementSessions),
		},
	}, nil
}

func (ir *IterativeRefiner) extractInputs(input core.PhaseInput) (*ProjectExploration, *BuildPlan, map[string]string, error) {
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

func (ir *IterativeRefiner) registerCodeInspectors(exploration *ProjectExploration) {
	// Register language-specific inspectors based on tech stack
	switch exploration.TechStack.Language {
	case "PHP":
		ir.improvementEngine.RegisterInspector(NewPHPSecurityInspector(ir.logger))
		ir.improvementEngine.RegisterInspector(NewPHPPerformanceInspector(ir.logger))
	case "JavaScript", "TypeScript":
		ir.improvementEngine.RegisterInspector(NewJSSecurityInspector(ir.logger))
		ir.improvementEngine.RegisterInspector(NewJSPerformanceInspector(ir.logger))
	case "Go":
		ir.improvementEngine.RegisterInspector(NewGoInspector(ir.logger))
	case "Python":
		ir.improvementEngine.RegisterInspector(NewPythonInspector(ir.logger))
	}

	// Register universal inspectors
	ir.improvementEngine.RegisterInspector(NewAccessibilityInspector(ir.logger))
	ir.improvementEngine.RegisterInspector(NewDocumentationInspector(ir.logger))
	ir.improvementEngine.RegisterInspector(NewTestCoverageInspector(ir.logger))
}

func (ir *IterativeRefiner) determineTargetQuality(filePath string, exploration *ProjectExploration) float64 {
	// Base target quality
	baseQuality := 0.85

	// Adjust based on file importance
	if strings.Contains(filePath, "security") || 
	   strings.Contains(filePath, "auth") ||
	   strings.Contains(filePath, "crypto") {
		baseQuality = 0.98 // Critical security files need high quality
	} else if strings.Contains(filePath, "test") {
		baseQuality = 0.80 // Test files can have slightly lower bar
	} else if strings.Contains(filePath, "config") {
		baseQuality = 0.90 // Config files need good quality
	}

	// Adjust based on project requirements
	for _, requirement := range exploration.Requirements {
		requirement = strings.ToLower(requirement)
		if strings.Contains(requirement, "high security") ||
		   strings.Contains(requirement, "financial") ||
		   strings.Contains(requirement, "medical") {
			baseQuality = max(baseQuality, 0.95)
		}
	}

	return baseQuality
}

func (ir *IterativeRefiner) extractFinalContent(session *core.ImprovementSession, originalContent interface{}) string {
	// Extract the final improved content from the session
	if len(session.Checkpoints) > 0 {
		// Use the last checkpoint
		lastCheckpoint := session.Checkpoints[len(session.Checkpoints)-1]
		if content, ok := lastCheckpoint.Content.(string); ok {
			return content
		}
	}

	// Fallback to original if no improvements
	if content, ok := originalContent.(string); ok {
		return content
	}

	return ""
}

func (ir *IterativeRefiner) generateRefinementReport(sessions map[string]*core.ImprovementSession, exploration *ProjectExploration) RefinementReport {
	report := RefinementReport{
		ProjectType:      exploration.ProjectType,
		TotalFiles:       len(sessions),
		Timestamp:        time.Now(),
		FileReports:      make([]FileRefinementReport, 0),
		OverallMetrics:   make(map[string]float64),
		LearningInsights: make([]string, 0),
	}

	totalInitialQuality := 0.0
	totalFinalQuality := 0.0
	totalIterations := 0
	successCount := 0

	for filePath, session := range sessions {
		fileReport := FileRefinementReport{
			FilePath:        filePath,
			InitialQuality:  session.InitialQuality,
			FinalQuality:    session.FinalQuality,
			Improvement:     session.FinalQuality - session.InitialQuality,
			Iterations:      session.TotalIterations,
			Success:         session.Success,
			ImprovementPath: ir.summarizeImprovementPath(session.ImprovementPath),
		}

		report.FileReports = append(report.FileReports, fileReport)

		totalInitialQuality += session.InitialQuality
		totalFinalQuality += session.FinalQuality
		totalIterations += session.TotalIterations
		if session.Success {
			successCount++
		}
	}

	// Calculate overall metrics
	fileCount := float64(len(sessions))
	report.OverallMetrics["average_initial_quality"] = totalInitialQuality / fileCount
	report.OverallMetrics["average_final_quality"] = totalFinalQuality / fileCount
	report.OverallMetrics["average_improvement"] = (totalFinalQuality - totalInitialQuality) / fileCount
	report.OverallMetrics["average_iterations"] = float64(totalIterations) / fileCount
	report.OverallMetrics["success_rate"] = float64(successCount) / fileCount

	return report
}

func (ir *IterativeRefiner) summarizeImprovementPath(path []core.ImprovementStep) []string {
	summary := make([]string, 0)
	
	for _, step := range path {
		if step.Success {
			summary = append(summary, fmt.Sprintf("Iteration %d: %s (%.2f%% improvement)",
				step.Iteration,
				step.ActionTaken,
				step.Improvement*100))
		}
	}

	return summary
}

func (ir *IterativeRefiner) extractAllLearningInsights(sessions map[string]*core.ImprovementSession) []core.LearningInsight {
	allInsights := make([]core.LearningInsight, 0)
	
	for _, session := range sessions {
		allInsights = append(allInsights, session.LearningInsights...)
	}

	// Deduplicate and aggregate insights
	insightMap := make(map[string]*core.LearningInsight)
	for _, insight := range allInsights {
		if existing, exists := insightMap[insight.Pattern]; exists {
			// Aggregate statistics
			existing.TimesApplied += insight.TimesApplied
			existing.SuccessRate = (existing.SuccessRate + insight.SuccessRate) / 2
			existing.AverageImpact = (existing.AverageImpact + insight.AverageImpact) / 2
		} else {
			insightCopy := insight
			insightMap[insight.Pattern] = &insightCopy
		}
	}

	// Convert back to slice
	aggregated := make([]core.LearningInsight, 0, len(insightMap))
	for _, insight := range insightMap {
		aggregated = append(aggregated, *insight)
	}

	return aggregated
}

func (ir *IterativeRefiner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("input data is required for iterative refinement")
	}
	return nil
}

func (ir *IterativeRefiner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	if output.Data == nil {
		return fmt.Errorf("refinement data is required")
	}
	return nil
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// RefinementReport summarizes the iterative refinement results
type RefinementReport struct {
	ProjectType      string                   `json:"project_type"`
	TotalFiles       int                      `json:"total_files"`
	FileReports      []FileRefinementReport   `json:"file_reports"`
	OverallMetrics   map[string]float64       `json:"overall_metrics"`
	LearningInsights []string                 `json:"learning_insights"`
	Timestamp        time.Time                `json:"timestamp"`
}

type FileRefinementReport struct {
	FilePath        string   `json:"file_path"`
	InitialQuality  float64  `json:"initial_quality"`
	FinalQuality    float64  `json:"final_quality"`
	Improvement     float64  `json:"improvement"`
	Iterations      int      `json:"iterations"`
	Success         bool     `json:"success"`
	ImprovementPath []string `json:"improvement_path"`
}

// Domain-specific inspectors

// PHPSecurityInspector checks PHP-specific security issues
type PHPSecurityInspector struct {
	logger *slog.Logger
}

func NewPHPSecurityInspector(logger *slog.Logger) *PHPSecurityInspector {
	return &PHPSecurityInspector{
		logger: logger.With("inspector", "php_security"),
	}
}

func (psi *PHPSecurityInspector) Name() string { return "PHPSecurity" }
func (psi *PHPSecurityInspector) Category() string { return "security" }

func (psi *PHPSecurityInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	code, ok := content.(string)
	if !ok {
		return core.InspectionResult{}, fmt.Errorf("content must be string for PHP inspection")
	}

	result := core.InspectionResult{
		InspectorName: psi.Name(),
		Category:      psi.Category(),
		Findings:      make([]core.Finding, 0),
		Metrics:       make(map[string]float64),
		Suggestions:   make([]core.ImprovementSuggestion, 0),
		Timestamp:     time.Now(),
	}

	// Check for SQL injection vulnerabilities
	if strings.Contains(code, "$_POST") || strings.Contains(code, "$_GET") {
		if !strings.Contains(code, "filter_var") && !strings.Contains(code, "htmlspecialchars") {
			result.Findings = append(result.Findings, core.Finding{
				ID:          "php-no-input-sanitization",
				Type:        core.ErrorFinding,
				Severity:    core.CriticalSeverity,
				Description: "User input not sanitized",
				Impact:      "XSS and SQL injection vulnerabilities",
			})
			
			result.Suggestions = append(result.Suggestions, core.ImprovementSuggestion{
				Target: "User input handling",
				Action: "Add input sanitization using filter_var() and htmlspecialchars()",
				Reason: "Prevent XSS and injection attacks",
				Example: "htmlspecialchars($_POST['email'], ENT_QUOTES, 'UTF-8')",
				Complexity: "low",
			})
		}
	}

	// Check for file operation security
	if strings.Contains(code, "fopen") && !strings.Contains(code, "fopen(") {
		result.Findings = append(result.Findings, core.Finding{
			ID:          "php-unsafe-file-ops",
			Type:        core.WarningFinding,
			Severity:    core.HighSeverity,
			Description: "Unsafe file operations",
			Impact:      "Potential file access vulnerabilities",
		})
	}

	// Calculate security score
	securityScore := 1.0
	for _, finding := range result.Findings {
		if finding.Severity == core.CriticalSeverity {
			securityScore -= 0.3
		} else if finding.Severity == core.HighSeverity {
			securityScore -= 0.2
		}
	}

	result.Score = max(0.0, securityScore)
	result.Passed = result.Score >= 0.8

	return result, nil
}

func (psi *PHPSecurityInspector) GenerateCriteria() []core.QualityCriteria {
	return []core.QualityCriteria{
		{
			ID:          "php-input-sanitization",
			Name:        "Input Sanitization",
			Description: "All user input must be properly sanitized",
			Category:    "security",
			Priority:    core.CriticalPriority,
			Validator: func(ctx context.Context, content interface{}) (core.CriteriaResult, error) {
				inspection, _ := psi.Inspect(ctx, content)
				
				hasSanitization := true
				for _, finding := range inspection.Findings {
					if finding.ID == "php-no-input-sanitization" {
						hasSanitization = false
						break
					}
				}
				
				return core.CriteriaResult{
					Passed:      hasSanitization,
					Score:       inspection.Score,
					Details:     "Input sanitization check",
					Suggestions: inspection.Suggestions,
				}, nil
			},
		},
	}
}

func (psi *PHPSecurityInspector) CanInspect(content interface{}) bool {
	if code, ok := content.(string); ok {
		return strings.Contains(code, "<?php") || strings.Contains(code, ".php")
	}
	return false
}

// Additional inspector stubs to demonstrate extensibility

type PHPPerformanceInspector struct{ logger *slog.Logger }
func NewPHPPerformanceInspector(logger *slog.Logger) *PHPPerformanceInspector {
	return &PHPPerformanceInspector{logger: logger}
}
func (p *PHPPerformanceInspector) Name() string { return "PHPPerformance" }
func (p *PHPPerformanceInspector) Category() string { return "performance" }
func (p *PHPPerformanceInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (p *PHPPerformanceInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (p *PHPPerformanceInspector) CanInspect(content interface{}) bool { return true }

type JSSecurityInspector struct{ logger *slog.Logger }
func NewJSSecurityInspector(logger *slog.Logger) *JSSecurityInspector {
	return &JSSecurityInspector{logger: logger}
}
func (j *JSSecurityInspector) Name() string { return "JSSecurity" }
func (j *JSSecurityInspector) Category() string { return "security" }
func (j *JSSecurityInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (j *JSSecurityInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (j *JSSecurityInspector) CanInspect(content interface{}) bool { return true }

type JSPerformanceInspector struct{ logger *slog.Logger }
func NewJSPerformanceInspector(logger *slog.Logger) *JSPerformanceInspector {
	return &JSPerformanceInspector{logger: logger}
}
func (j *JSPerformanceInspector) Name() string { return "JSPerformance" }
func (j *JSPerformanceInspector) Category() string { return "performance" }
func (j *JSPerformanceInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (j *JSPerformanceInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (j *JSPerformanceInspector) CanInspect(content interface{}) bool { return true }

type GoInspector struct{ logger *slog.Logger }
func NewGoInspector(logger *slog.Logger) *GoInspector {
	return &GoInspector{logger: logger}
}
func (g *GoInspector) Name() string { return "GoQuality" }
func (g *GoInspector) Category() string { return "quality" }
func (g *GoInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (g *GoInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (g *GoInspector) CanInspect(content interface{}) bool { return true }

type PythonInspector struct{ logger *slog.Logger }
func NewPythonInspector(logger *slog.Logger) *PythonInspector {
	return &PythonInspector{logger: logger}
}
func (py *PythonInspector) Name() string { return "PythonQuality" }
func (py *PythonInspector) Category() string { return "quality" }
func (py *PythonInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (py *PythonInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (py *PythonInspector) CanInspect(content interface{}) bool { return true }

type AccessibilityInspector struct{ logger *slog.Logger }
func NewAccessibilityInspector(logger *slog.Logger) *AccessibilityInspector {
	return &AccessibilityInspector{logger: logger}
}
func (a *AccessibilityInspector) Name() string { return "Accessibility" }
func (a *AccessibilityInspector) Category() string { return "accessibility" }
func (a *AccessibilityInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (a *AccessibilityInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (a *AccessibilityInspector) CanInspect(content interface{}) bool { return true }

type DocumentationInspector struct{ logger *slog.Logger }
func NewDocumentationInspector(logger *slog.Logger) *DocumentationInspector {
	return &DocumentationInspector{logger: logger}
}
func (d *DocumentationInspector) Name() string { return "Documentation" }
func (d *DocumentationInspector) Category() string { return "documentation" }
func (d *DocumentationInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (d *DocumentationInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (d *DocumentationInspector) CanInspect(content interface{}) bool { return true }

type TestCoverageInspector struct{ logger *slog.Logger }
func NewTestCoverageInspector(logger *slog.Logger) *TestCoverageInspector {
	return &TestCoverageInspector{logger: logger}
}
func (t *TestCoverageInspector) Name() string { return "TestCoverage" }
func (t *TestCoverageInspector) Category() string { return "testing" }
func (t *TestCoverageInspector) Inspect(ctx context.Context, content interface{}) (core.InspectionResult, error) {
	return core.InspectionResult{Score: 0.9, Passed: true}, nil
}
func (t *TestCoverageInspector) GenerateCriteria() []core.QualityCriteria { return nil }
func (t *TestCoverageInspector) CanInspect(content interface{}) bool { return true }