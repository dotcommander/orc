package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// IterativeImprovementEngine combines inspectors and iterators for continuous quality improvement
type IterativeImprovementEngine struct {
	iterator      *IteratorAgent
	inspector     *InspectorAgent
	logger        *slog.Logger
	config        ImprovementConfig
	learningCache *LearningCache
}

// RegisterInspector adds an inspector to the improvement engine
func (iie *IterativeImprovementEngine) RegisterInspector(inspector Inspector) {
	if iie.inspector != nil {
		iie.inspector.RegisterInspector(inspector)
	}
}

// ImprovementConfig configures the improvement engine
type ImprovementConfig struct {
	MaxIterations        int                    `json:"max_iterations"`
	TargetQuality        float64                `json:"target_quality"`
	ImprovementStrategy  string                 `json:"improvement_strategy"`
	ParallelImprovements bool                   `json:"parallel_improvements"`
	LearningEnabled      bool                   `json:"learning_enabled"`
	CheckpointInterval   int                    `json:"checkpoint_interval"`
	QualityThresholds    map[string]float64     `json:"quality_thresholds"`
	FocusAreas           []string               `json:"focus_areas"`
	AdaptiveMode         bool                   `json:"adaptive_mode"`
	HumanInTheLoop       bool                   `json:"human_in_the_loop"`
}

// ImprovementSession tracks a complete improvement session
type ImprovementSession struct {
	ID                string                  `json:"id"`
	StartTime         time.Time               `json:"start_time"`
	EndTime           time.Time               `json:"end_time"`
	InitialQuality    float64                 `json:"initial_quality"`
	FinalQuality      float64                 `json:"final_quality"`
	TotalIterations   int                     `json:"total_iterations"`
	ImprovementPath   []ImprovementStep       `json:"improvement_path"`
	CriteriaEvolution map[string][]float64    `json:"criteria_evolution"`
	LearningInsights  []LearningInsight       `json:"learning_insights"`
	Checkpoints       []ImprovementCheckpoint `json:"checkpoints"`
	Success           bool                    `json:"success"`
	FailureReason     string                  `json:"failure_reason,omitempty"`
}

// ImprovementStep represents one step in the improvement process
type ImprovementStep struct {
	Iteration        int                          `json:"iteration"`
	Timestamp        time.Time                    `json:"timestamp"`
	ActionTaken      string                       `json:"action_taken"`
	TargetCriteria   []string                     `json:"target_criteria"`
	BeforeScore      float64                      `json:"before_score"`
	AfterScore       float64                      `json:"after_score"`
	Improvement      float64                      `json:"improvement"`
	Changes          []ContentChange              `json:"changes"`
	InspectionResult map[string]InspectionResult  `json:"inspection_result"`
	Success          bool                         `json:"success"`
}

// ImprovementCheckpoint saves state for resume capability
type ImprovementCheckpoint struct {
	Iteration    int                 `json:"iteration"`
	Content      interface{}         `json:"content"`
	Quality      float64             `json:"quality"`
	CriteriaState map[string]CriteriaResult `json:"criteria_state"`
	Timestamp    time.Time           `json:"timestamp"`
}

// LearningInsight captures patterns for future improvements
type LearningInsight struct {
	Pattern          string    `json:"pattern"`
	SuccessRate      float64   `json:"success_rate"`
	AverageImpact    float64   `json:"average_impact"`
	ApplicableTo     []string  `json:"applicable_to"`
	DiscoveredAt     time.Time `json:"discovered_at"`
	TimesApplied     int       `json:"times_applied"`
}

// LearningCache stores successful improvement patterns
type LearningCache struct {
	patterns map[string]*ImprovedPattern
	mu       sync.RWMutex
}

type ImprovedPattern struct {
	Pattern      string
	Improvements []string
	SuccessRate  float64
	LastUsed     time.Time
}

// NewIterativeImprovementEngine creates a new improvement engine
func NewIterativeImprovementEngine(agent Agent, logger *slog.Logger, config ImprovementConfig) *IterativeImprovementEngine {
	iteratorConfig := IteratorConfig{
		MaxIterations:        config.MaxIterations,
		ConvergenceThreshold: config.TargetQuality,
		ParallelCriteria:     config.ParallelImprovements,
		FocusMode:           "worst-first",
		BatchSize:           3,
		MinImprovement:      0.01,
		StagnationThreshold: 5,
		AdaptiveLearning:    config.AdaptiveMode,
	}

	inspectorConfig := InspectorConfig{
		DeepAnalysis:       true,
		ParallelInspection: config.ParallelImprovements,
		CacheResults:       true,
		MaxDepth:          10,
		TimeoutPerCheck:   30 * time.Second,
	}

	engine := &IterativeImprovementEngine{
		iterator:  NewIteratorAgent(agent, logger, iteratorConfig),
		inspector: NewInspectorAgent(agent, logger, inspectorConfig),
		logger:    logger.With("component", "improvement_engine"),
		config:    config,
	}

	if config.LearningEnabled {
		engine.learningCache = &LearningCache{
			patterns: make(map[string]*ImprovedPattern),
		}
	}

	return engine
}

// ImproveContent performs iterative improvement until quality targets are met
func (ie *IterativeImprovementEngine) ImproveContent(ctx context.Context, content interface{}, targetQuality float64) (*ImprovementSession, error) {
	session := &ImprovementSession{
		ID:                fmt.Sprintf("session_%d", time.Now().Unix()),
		StartTime:         time.Now(),
		ImprovementPath:   make([]ImprovementStep, 0),
		CriteriaEvolution: make(map[string][]float64),
		LearningInsights:  make([]LearningInsight, 0),
		Checkpoints:       make([]ImprovementCheckpoint, 0),
	}

	// Initial inspection to establish baseline
	initialResults, err := ie.inspector.InspectContent(ctx, content)
	if err != nil {
		return session, fmt.Errorf("initial inspection failed: %w", err)
	}

	session.InitialQuality = ie.calculateOverallQuality(initialResults)
	ie.logger.Info("Starting improvement session",
		"initial_quality", session.InitialQuality,
		"target_quality", targetQuality)

	// Generate quality criteria from inspectors
	criteria := ie.inspector.GenerateAllCriteria()

	// Apply learning insights if available
	if ie.config.LearningEnabled {
		criteria = ie.enhanceCriteriaWithLearning(criteria)
	}

	// Main improvement loop
	currentContent := content
	iteration := 0

	for iteration < ie.config.MaxIterations {
		iteration++

		// Create improvement step
		step := ImprovementStep{
			Iteration:        iteration,
			Timestamp:        time.Now(),
			InspectionResult: make(map[string]InspectionResult),
		}

		// Deep inspection
		inspectionResults, err := ie.inspector.InspectContent(ctx, currentContent)
		if err != nil {
			ie.logger.Error("Inspection failed", "iteration", iteration, "error", err)
			continue
		}
		step.InspectionResult = inspectionResults

		// Calculate current quality
		currentQuality := ie.calculateOverallQuality(inspectionResults)
		step.BeforeScore = currentQuality

		// Check if we've reached target quality
		if currentQuality >= targetQuality {
			session.Success = true
			session.FinalQuality = currentQuality
			session.TotalIterations = iteration
			ie.logger.Info("Target quality achieved!",
				"iterations", iteration,
				"final_quality", currentQuality)
			break
		}

		// Identify failing criteria
		failingCriteria := ie.identifyFailingCriteria(inspectionResults, criteria)
		if len(failingCriteria) == 0 {
			ie.logger.Warn("No failing criteria but quality below target",
				"current", currentQuality,
				"target", targetQuality)
			break
		}

		// Select improvement strategy
		strategy := ie.selectImprovementStrategy(failingCriteria, inspectionResults, session)
		step.ActionTaken = strategy

		// Apply improvements
		improvedContent, changes, err := ie.applyTargetedImprovements(ctx, currentContent, failingCriteria, inspectionResults, strategy)
		if err != nil {
			ie.logger.Error("Improvement failed", "iteration", iteration, "error", err)
			continue
		}

		// Verify improvement
		verifyResults, err := ie.inspector.InspectContent(ctx, improvedContent)
		if err != nil {
			ie.logger.Error("Verification failed", "iteration", iteration, "error", err)
			continue
		}

		newQuality := ie.calculateOverallQuality(verifyResults)
		step.AfterScore = newQuality
		step.Improvement = newQuality - currentQuality
		step.Changes = changes
		step.Success = step.Improvement > 0

		// Update state
		if step.Success {
			currentContent = improvedContent
			session.ImprovementPath = append(session.ImprovementPath, step)
			
			// Track criteria evolution
			for criteriaID, result := range verifyResults {
				session.CriteriaEvolution[criteriaID] = append(
					session.CriteriaEvolution[criteriaID], 
					result.Score,
				)
			}

			// Learn from success
			if ie.config.LearningEnabled {
				ie.recordSuccessfulPattern(step, failingCriteria)
			}
		}

		// Checkpoint if needed
		if iteration%ie.config.CheckpointInterval == 0 {
			checkpoint := ImprovementCheckpoint{
				Iteration: iteration,
				Content:   currentContent,
				Quality:   newQuality,
				Timestamp: time.Now(),
			}
			session.Checkpoints = append(session.Checkpoints, checkpoint)
		}

		// Check for stagnation
		if ie.isStagnant(session, 5) {
			ie.logger.Warn("Improvement stagnant, applying adaptive strategies")
			if ie.config.AdaptiveMode {
				currentContent, err = ie.applyAdaptiveStrategies(ctx, currentContent, session)
				if err != nil {
					ie.logger.Error("Adaptive strategies failed", "error", err)
				}
			}
		}

		// Human in the loop
		if ie.config.HumanInTheLoop && iteration%10 == 0 {
			guidance, err := ie.requestHumanGuidance(currentContent, inspectionResults)
			if err == nil && guidance != "" {
				// Apply human guidance
				ie.logger.Info("Applying human guidance", "iteration", iteration)
			}
		}
	}

	// Finalize session
	session.EndTime = time.Now()
	session.TotalIterations = iteration
	
	if !session.Success {
		session.FailureReason = fmt.Sprintf("Failed to reach target quality %.2f after %d iterations (achieved %.2f)", 
			targetQuality, iteration, session.FinalQuality)
	}

	// Extract learning insights
	if ie.config.LearningEnabled {
		session.LearningInsights = ie.extractLearningInsights(session)
	}

	return session, nil
}

// Helper methods

func (ie *IterativeImprovementEngine) calculateOverallQuality(results map[string]InspectionResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, result := range results {
		totalScore += result.Score
	}

	return totalScore / float64(len(results))
}

func (ie *IterativeImprovementEngine) identifyFailingCriteria(results map[string]InspectionResult, allCriteria []QualityCriteria) []QualityCriteria {
	failing := make([]QualityCriteria, 0)
	
	// Check each criteria against inspection results
	for _, criteria := range allCriteria {
		// Find corresponding inspection result
		for _, result := range results {
			if !result.Passed && result.Category == criteria.Category {
				failing = append(failing, criteria)
				break
			}
		}
	}

	return failing
}

func (ie *IterativeImprovementEngine) selectImprovementStrategy(failingCriteria []QualityCriteria, results map[string]InspectionResult, session *ImprovementSession) string {
	// Analyze patterns in session history
	if len(session.ImprovementPath) > 3 {
		// Look for recurring issues
		recurringCount := 0
		for _, step := range session.ImprovementPath[len(session.ImprovementPath)-3:] {
			for _, target := range step.TargetCriteria {
				for _, failing := range failingCriteria {
					if target == failing.ID {
						recurringCount++
					}
				}
			}
		}

		if recurringCount > len(failingCriteria)/2 {
			return "aggressive-refactor" // Same issues keep coming back
		}
	}

	// Check severity of failures
	criticalCount := 0
	for _, criteria := range failingCriteria {
		if criteria.Priority == CriticalPriority {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		return "focus-critical" // Fix critical issues first
	}

	// Default strategy based on number of failures
	if len(failingCriteria) > 5 {
		return "batch-improvements" // Many issues, fix in batches
	}

	return "incremental" // Few issues, fix one by one
}

func (ie *IterativeImprovementEngine) applyTargetedImprovements(ctx context.Context, content interface{}, failingCriteria []QualityCriteria, results map[string]InspectionResult, strategy string) (interface{}, []ContentChange, error) {
	changes := make([]ContentChange, 0)
	improvedContent := content

	switch strategy {
	case "focus-critical":
		// Only fix critical issues
		for _, criteria := range failingCriteria {
			if criteria.Priority != CriticalPriority {
				continue
			}
			
			improved, change, err := ie.improveSingleCriteria(ctx, improvedContent, criteria, results)
			if err != nil {
				ie.logger.Error("Failed to improve critical criteria", "criteria", criteria.Name, "error", err)
				continue
			}
			
			improvedContent = improved
			changes = append(changes, change)
		}

	case "batch-improvements":
		// Group related improvements
		grouped := ie.groupRelatedCriteria(failingCriteria)
		for category, group := range grouped {
			improved, batchChanges, err := ie.improveBatch(ctx, improvedContent, group, results)
			if err != nil {
				ie.logger.Error("Batch improvement failed", "category", category, "error", err)
				continue
			}
			
			improvedContent = improved
			changes = append(changes, batchChanges...)
		}

	case "aggressive-refactor":
		// Major restructuring
		improved, refactorChanges, err := ie.performMajorRefactor(ctx, improvedContent, failingCriteria, results)
		if err != nil {
			return content, changes, fmt.Errorf("refactor failed: %w", err)
		}
		
		improvedContent = improved
		changes = refactorChanges

	default: // incremental
		// Fix one by one
		for _, criteria := range failingCriteria {
			improved, change, err := ie.improveSingleCriteria(ctx, improvedContent, criteria, results)
			if err != nil {
				ie.logger.Error("Failed to improve criteria", "criteria", criteria.Name, "error", err)
				continue
			}
			
			improvedContent = improved
			changes = append(changes, change)
		}
	}

	return improvedContent, changes, nil
}

func (ie *IterativeImprovementEngine) improveSingleCriteria(ctx context.Context, content interface{}, criteria QualityCriteria, results map[string]InspectionResult) (interface{}, ContentChange, error) {
	// Find relevant inspection result
	var relevantResult *InspectionResult
	for _, result := range results {
		if result.Category == criteria.Category {
			relevantResult = &result
			break
		}
	}

	if relevantResult == nil {
		return content, ContentChange{}, fmt.Errorf("no inspection result for criteria %s", criteria.Name)
	}

	// Build improvement prompt
	prompt := fmt.Sprintf(`You are an expert at improving content to meet quality criteria.

Current Content:
%v

Criteria to Fix: %s
Description: %s
Current Issues:
%s

Suggestions:
`, content, criteria.Name, criteria.Description, ie.formatFindings(relevantResult.Findings))

	for _, suggestion := range relevantResult.Suggestions {
		prompt += fmt.Sprintf("- %s: %s\n", suggestion.Action, suggestion.Reason)
	}

	prompt += "\nMake targeted improvements to fix ONLY this criteria. Return the improved content."

	// Execute improvement
	response, err := ie.iterator.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return content, ContentChange{}, err
	}

	change := ContentChange{
		Type:      "improvement",
		Target:    criteria.Name,
		Criteria:  criteria.ID,
		Reason:    fmt.Sprintf("Fix %s", criteria.Description),
		Timestamp: time.Now(),
	}

	return response, change, nil
}

func (ie *IterativeImprovementEngine) improveBatch(ctx context.Context, content interface{}, criteria []QualityCriteria, results map[string]InspectionResult) (interface{}, []ContentChange, error) {
	// Implement batch improvement logic
	changes := make([]ContentChange, 0)
	
	// Build comprehensive prompt for multiple improvements
	prompt := fmt.Sprintf(`Improve the following content to fix multiple related issues:

Content:
%v

Issues to fix:
`, content)

	for _, c := range criteria {
		prompt += fmt.Sprintf("- %s: %s\n", c.Name, c.Description)
	}

	prompt += "\nApply all improvements while maintaining consistency."

	response, err := ie.iterator.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return content, changes, err
	}

	for _, c := range criteria {
		changes = append(changes, ContentChange{
			Type:      "batch-improvement",
			Target:    c.Name,
			Criteria:  c.ID,
			Timestamp: time.Now(),
		})
	}

	return response, changes, nil
}

func (ie *IterativeImprovementEngine) performMajorRefactor(ctx context.Context, content interface{}, criteria []QualityCriteria, results map[string]InspectionResult) (interface{}, []ContentChange, error) {
	// Implement major refactoring logic
	prompt := fmt.Sprintf(`The current content has recurring quality issues that require major refactoring.

Content:
%v

Recurring Issues:
`, content)

	for _, c := range criteria {
		prompt += fmt.Sprintf("- %s\n", c.Description)
	}

	prompt += `
Perform a comprehensive refactor to address these systemic issues.
Focus on structural improvements that prevent these issues from recurring.`

	response, err := ie.iterator.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return content, nil, err
	}

	changes := []ContentChange{{
		Type:      "major-refactor",
		Target:    "entire-content",
		Reason:    "Systemic quality issues",
		Timestamp: time.Now(),
	}}

	return response, changes, nil
}

func (ie *IterativeImprovementEngine) groupRelatedCriteria(criteria []QualityCriteria) map[string][]QualityCriteria {
	grouped := make(map[string][]QualityCriteria)
	
	for _, c := range criteria {
		grouped[c.Category] = append(grouped[c.Category], c)
	}
	
	return grouped
}

func (ie *IterativeImprovementEngine) formatFindings(findings []Finding) string {
	result := ""
	for _, f := range findings {
		result += fmt.Sprintf("- %s (%s): %s\n", f.Description, f.Severity, f.Impact)
	}
	return result
}

func (ie *IterativeImprovementEngine) isStagnant(session *ImprovementSession, threshold int) bool {
	if len(session.ImprovementPath) < threshold {
		return false
	}
	
	// Check if quality hasn't improved in last N iterations
	recent := session.ImprovementPath[len(session.ImprovementPath)-threshold:]
	totalImprovement := 0.0
	
	for _, step := range recent {
		totalImprovement += step.Improvement
	}
	
	return totalImprovement < 0.01 // Less than 1% improvement
}

func (ie *IterativeImprovementEngine) applyAdaptiveStrategies(ctx context.Context, content interface{}, session *ImprovementSession) (interface{}, error) {
	// Implement adaptive strategies
	ie.logger.Info("Applying adaptive strategies")
	
	// Strategy 1: Relax non-critical criteria temporarily
	// Strategy 2: Try alternative approaches
	// Strategy 3: Request external help
	
	return content, nil
}

func (ie *IterativeImprovementEngine) requestHumanGuidance(content interface{}, results map[string]InspectionResult) (string, error) {
	// Implement human-in-the-loop logic
	ie.logger.Info("Requesting human guidance")
	
	// In a real implementation, this would interact with a UI or API
	return "", nil
}

func (ie *IterativeImprovementEngine) enhanceCriteriaWithLearning(criteria []QualityCriteria) []QualityCriteria {
	// Apply learned patterns to enhance criteria
	if ie.learningCache == nil {
		return criteria
	}
	
	ie.learningCache.mu.RLock()
	defer ie.learningCache.mu.RUnlock()
	
	// Enhance criteria based on successful patterns
	for i, c := range criteria {
		if pattern, exists := ie.learningCache.patterns[c.Category]; exists {
			// Enhance validator with learned patterns
			criteria[i].Context["learned_patterns"] = pattern.Improvements
		}
	}
	
	return criteria
}

func (ie *IterativeImprovementEngine) recordSuccessfulPattern(step ImprovementStep, criteria []QualityCriteria) {
	if ie.learningCache == nil {
		return
	}
	
	ie.learningCache.mu.Lock()
	defer ie.learningCache.mu.Unlock()
	
	// Record successful improvement patterns
	for _, change := range step.Changes {
		key := fmt.Sprintf("%s_%s", change.Type, change.Target)
		
		if pattern, exists := ie.learningCache.patterns[key]; exists {
			pattern.SuccessRate = (pattern.SuccessRate + step.Improvement) / 2
			pattern.LastUsed = time.Now()
		} else {
			ie.learningCache.patterns[key] = &ImprovedPattern{
				Pattern:     change.Reason,
				SuccessRate: step.Improvement,
				LastUsed:    time.Now(),
			}
		}
	}
}

func (ie *IterativeImprovementEngine) extractLearningInsights(session *ImprovementSession) []LearningInsight {
	insights := make([]LearningInsight, 0)
	
	// Analyze improvement patterns
	patternSuccess := make(map[string][]float64)
	
	for _, step := range session.ImprovementPath {
		if step.Success {
			key := step.ActionTaken
			patternSuccess[key] = append(patternSuccess[key], step.Improvement)
		}
	}
	
	// Create insights from patterns
	for pattern, improvements := range patternSuccess {
		if len(improvements) > 2 {
			totalImprovement := 0.0
			for _, imp := range improvements {
				totalImprovement += imp
			}
			
			insight := LearningInsight{
				Pattern:       pattern,
				SuccessRate:   float64(len(improvements)) / float64(session.TotalIterations),
				AverageImpact: totalImprovement / float64(len(improvements)),
				DiscoveredAt:  time.Now(),
				TimesApplied:  len(improvements),
			}
			
			insights = append(insights, insight)
		}
	}
	
	return insights
}