package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GoalAwareOrchestrator extends the base orchestrator with goal-tracking capabilities
type GoalAwareOrchestrator struct {
	*Orchestrator
	goals        *GoalTracker
	strategies   *StrategyManager
	maxAttempts  int
	parseRequest RequestParser
}

// ExecutionResult holds the results of a goal-aware execution
type ExecutionResult struct {
	StartTime    time.Time
	Attempts     int
	GoalsMet     int
	TotalGoals   int
	Success      bool
}

// IterationContext holds the context for a single iteration
type IterationContext struct {
	Attempt     int
	UnmetGoals  []*Goal
	Strategy    Strategy
}

// RequestParser extracts goals from user requests
type RequestParser interface {
	ParseGoals(request string) []*Goal
}

// DefaultRequestParser implements basic goal extraction from requests
type DefaultRequestParser struct{}

func (p *DefaultRequestParser) ParseGoals(request string) []*Goal {
	goals := make([]*Goal, 0)
	
	// Extract specific goals from request
	if goal := p.extractWordCountGoal(request); goal != nil {
		goals = append(goals, goal)
	}
	
	if goal := p.extractChapterCountGoal(request); goal != nil {
		goals = append(goals, goal)
	}
	
	// Add default goals
	goals = append(goals, p.createDefaultQualityGoal())
	goals = append(goals, p.createCompletenessGoal())
	
	return goals
}

// extractWordCountGoal parses word count targets from request text
func (p *DefaultRequestParser) extractWordCountGoal(request string) *Goal {
	wordCountRegex := regexp.MustCompile(`(\d+,?\d*)\s*(?:k|thousand)?\s*word`)
	if match := wordCountRegex.FindStringSubmatch(strings.ToLower(request)); len(match) > 1 {
		// Parse the number, handling both "20,000" and "20000" formats
		numStr := strings.ReplaceAll(match[1], ",", "")
		if wordCount, err := strconv.Atoi(numStr); err == nil {
			// Handle "k" notation (e.g., "20k words")
			if strings.Contains(strings.ToLower(request), "k word") {
				wordCount *= 1000
			}
			
			return &Goal{
				Type:     GoalTypeWordCount,
				Target:   wordCount,
				Current:  0,
				Priority: 10, // Highest priority
				Validator: func(current interface{}) bool {
					if count, ok := current.(int); ok {
						// Accept 90% of target as success
						return count >= int(float64(wordCount)*0.9)
					}
					return false
				},
			}
		}
	}
	return nil
}

// extractChapterCountGoal parses chapter count targets from request text
func (p *DefaultRequestParser) extractChapterCountGoal(request string) *Goal {
	chapterRegex := regexp.MustCompile(`(\d+)\s*chapter`)
	if match := chapterRegex.FindStringSubmatch(strings.ToLower(request)); len(match) > 1 {
		if chapterCount, err := strconv.Atoi(match[1]); err == nil {
			return &Goal{
				Type:     GoalTypeChapterCount,
				Target:   chapterCount,
				Current:  0,
				Priority: 7,
			}
		}
	}
	return nil
}

// createDefaultQualityGoal creates a standard quality goal
func (p *DefaultRequestParser) createDefaultQualityGoal() *Goal {
	return &Goal{
		Type:     GoalTypeQuality,
		Target:   8.0, // Target quality score
		Current:  0.0,
		Priority: 5,
		Validator: func(current interface{}) bool {
			if score, ok := current.(float64); ok {
				return score >= 7.5 // Accept 7.5+ as good quality
			}
			return false
		},
	}
}

// createCompletenessGoal creates a standard completeness goal
func (p *DefaultRequestParser) createCompletenessGoal() *Goal {
	return &Goal{
		Type:     GoalTypeCompleteness,
		Target:   true,
		Current:  false,
		Priority: 8,
	}
}

// NewGoalAwareOrchestrator creates an orchestrator that tracks and achieves goals
func NewGoalAwareOrchestrator(base *Orchestrator, agent Agent) *GoalAwareOrchestrator {
	return &GoalAwareOrchestrator{
		Orchestrator: base,
		goals:        NewGoalTracker(),
		strategies:   NewStrategyManager(agent, base.storage, base.logger),
		maxAttempts:  5,
		parseRequest: &DefaultRequestParser{},
	}
}

// RunUntilGoalsMet executes phases iteratively until all goals are achieved
func (o *GoalAwareOrchestrator) RunUntilGoalsMet(ctx context.Context, request string) error {
	// Setup goals and execute initial run
	if err := o.executeInitialRun(ctx, request); err != nil {
		return err
	}
	
	// Run improvement iterations
	executionResult := o.executeImprovementCycle(ctx)
	
	// Log final summary
	o.logExecutionSummary(executionResult)
	
	return nil
}

// executeInitialRun sets up goals and runs the first execution
func (o *GoalAwareOrchestrator) executeInitialRun(ctx context.Context, request string) error {
	// Parse goals from request
	goals := o.parseRequest.ParseGoals(request)
	for _, goal := range goals {
		o.goals.AddGoal(goal)
		o.logger.Info("Goal identified", 
			"type", goal.Type,
			"target", goal.Target,
			"priority", goal.Priority)
	}
	
	// Initial execution
	o.logger.Info("Starting initial execution")
	if err := o.Run(ctx, request); err != nil {
		// Don't fail immediately - some phases might have succeeded
		o.logger.Warn("Initial execution had errors", "error", err)
	}
	
	// Update goals based on initial results
	o.updateGoalsFromOutput()
	o.logger.Info("Initial execution complete", "progress", o.goals.Progress())
	
	return nil
}

// executeImprovementCycle runs improvement iterations until goals are met
func (o *GoalAwareOrchestrator) executeImprovementCycle(ctx context.Context) ExecutionResult {
	result := ExecutionResult{
		StartTime:  time.Now(),
		Attempts:   0,
		TotalGoals: o.goals.TotalCount(),
	}
	
	for result.Attempts < o.maxAttempts {
		// Check if all goals are met
		if o.goals.AllMet() {
			result.Success = true
			break
		}
		
		// Prepare iteration context
		iterCtx := o.prepareIteration(result.Attempts + 1)
		if iterCtx == nil {
			break // No unmet goals or no strategy available
		}
		
		// Execute the iteration
		if !o.executeIteration(ctx, iterCtx) {
			break // Critical failure or no progress
		}
		
		result.Attempts++
	}
	
	result.GoalsMet = o.goals.MetCount()
	result.Success = o.goals.AllMet()
	return result
}

// prepareIteration sets up the context for a single iteration
func (o *GoalAwareOrchestrator) prepareIteration(attemptNum int) *IterationContext {
	unmetGoals := o.goals.GetUnmetGoals()
	if len(unmetGoals) == 0 {
		return nil
	}
	
	o.logIterationStart(attemptNum, unmetGoals)
	
	strategy := o.strategies.SelectOptimal(unmetGoals)
	if strategy == nil {
		o.logger.Warn("No strategy available for unmet goals")
		return nil
	}
	
	return &IterationContext{
		Attempt:    attemptNum,
		UnmetGoals: unmetGoals,
		Strategy:   strategy,
	}
}

// logIterationStart logs the beginning of an iteration with goal details
func (o *GoalAwareOrchestrator) logIterationStart(attempt int, unmetGoals []*Goal) {
	o.logger.Info("Starting improvement iteration", 
		"attempt", attempt,
		"goals_met", o.goals.MetCount(),
		"goals_total", o.goals.TotalCount())
	
	for _, goal := range unmetGoals {
		o.logger.Info("Unmet goal",
			"type", goal.Type,
			"current", goal.Current,
			"target", goal.Target,
			"gap", goal.Gap(),
			"progress", fmt.Sprintf("%.1f%%", goal.Progress()))
	}
}

// executeIteration runs a single improvement iteration
func (o *GoalAwareOrchestrator) executeIteration(ctx context.Context, iterCtx *IterationContext) bool {
	// Execute the strategy
	result, err := o.executeStrategy(ctx, iterCtx)
	if err != nil {
		return o.handleStrategyError(err, iterCtx)
	}
	
	// Process the results
	if err := o.processStrategyResult(result); err != nil {
		return o.handleProcessingError(err)
	}
	
	// Check progress
	return o.checkIterationProgress(iterCtx.Attempt)
}

// executeStrategy runs the selected strategy and returns the result
func (o *GoalAwareOrchestrator) executeStrategy(ctx context.Context, iterCtx *IterationContext) (interface{}, error) {
	// Prepare input
	input, err := o.prepareStrategyInput()
	if err != nil {
		return nil, fmt.Errorf("prepare input: %w", err)
	}
	
	// Execute
	o.logger.Info("Executing strategy", "name", iterCtx.Strategy.Name())
	return iterCtx.Strategy.Execute(ctx, input, iterCtx.UnmetGoals)
}

// processStrategyResult applies the strategy results and updates goals
func (o *GoalAwareOrchestrator) processStrategyResult(result interface{}) error {
	// Apply results
	if err := o.applyStrategyResults(result); err != nil {
		return fmt.Errorf("apply results: %w", err)
	}
	
	// Update goal progress
	o.updateGoalsFromOutput()
	o.logger.Info("Iteration complete", "progress", o.goals.Progress())
	
	return nil
}

// handleStrategyError decides whether to continue after a strategy error
func (o *GoalAwareOrchestrator) handleStrategyError(err error, iterCtx *IterationContext) bool {
	o.logger.Error("Strategy execution failed", 
		"strategy", iterCtx.Strategy.Name(),
		"error", err)
	
	// Continue to next iteration for non-critical errors
	return true
}

// handleProcessingError decides whether to continue after a processing error
func (o *GoalAwareOrchestrator) handleProcessingError(err error) bool {
	o.logger.Error("Failed to process strategy results", "error", err)
	
	// Continue to next iteration for non-critical errors
	return true
}

// checkIterationProgress determines if we should continue iterating
func (o *GoalAwareOrchestrator) checkIterationProgress(attempt int) bool {
	if !o.isProgressing(attempt) {
		o.logger.Warn("No significant progress detected, stopping iterations")
		return false
	}
	return true
}

// logExecutionSummary logs the completion status and results
func (o *GoalAwareOrchestrator) logExecutionSummary(result ExecutionResult) {
	duration := time.Since(result.StartTime)
	o.logger.Info("Goal-aware execution complete",
		"duration", duration,
		"attempts", result.Attempts,
		"goals_met", result.GoalsMet,
		"goals_total", result.TotalGoals,
		"success", result.Success)
	
	// Log final goal status
	o.logger.Info(o.goals.Progress())
}

// updateGoalsFromOutput analyzes the output and updates goal progress
func (o *GoalAwareOrchestrator) updateGoalsFromOutput() {
	manuscript := o.loadManuscriptForGoals()
	if manuscript == "" {
		return
	}
	
	// Update all goal metrics
	o.updateContentMetrics(manuscript)
	o.updateQualityMetrics()
}

// loadManuscriptForGoals loads the manuscript for goal tracking
func (o *GoalAwareOrchestrator) loadManuscriptForGoals() string {
	data, err := o.storage.Load(context.Background(), "manuscript.md")
	if err != nil {
		o.logger.Warn("Could not load manuscript for goal tracking", "error", err)
		return ""
	}
	return string(data)
}

// updateContentMetrics updates word count, chapter count, and completeness
func (o *GoalAwareOrchestrator) updateContentMetrics(manuscript string) {
	// Word count
	wordCount := countWords(manuscript)
	o.goals.Update(GoalTypeWordCount, wordCount)
	
	// Chapter count
	chapterCount := countChapters(manuscript)
	o.goals.Update(GoalTypeChapterCount, chapterCount)
	
	// Completeness
	isComplete := wordCount > 0 && strings.Contains(manuscript, "## Chapter")
	o.goals.Update(GoalTypeCompleteness, isComplete)
}

// updateQualityMetrics updates quality score from critique if available
func (o *GoalAwareOrchestrator) updateQualityMetrics() {
	critiqueData, err := o.storage.Load(context.Background(), "critique.json")
	if err != nil {
		return // Quality is optional
	}
	
	rating := o.extractRatingFromCritique(string(critiqueData))
	if rating > 0 {
		o.goals.Update(GoalTypeQuality, rating)
	}
}

// extractRatingFromCritique parses the overall rating from critique JSON
func (o *GoalAwareOrchestrator) extractRatingFromCritique(critique string) float64 {
	match := regexp.MustCompile(`"overall_rating":\s*([\d.]+)`).FindStringSubmatch(critique)
	if len(match) < 2 {
		return 0
	}
	
	rating, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	return rating
}

// prepareStrategyInput creates input data for strategy execution
func (o *GoalAwareOrchestrator) prepareStrategyInput() (interface{}, error) {
	// Try loading manuscript first
	manuscript, manuscriptErr := o.loadManuscript()
	
	// If no manuscript, try scenes
	if manuscriptErr != nil {
		return o.loadScenesAsInput()
	}
	
	// Check if we need to include the plan
	return o.enrichWithPlan(manuscript)
}

// loadManuscript attempts to load the manuscript file
func (o *GoalAwareOrchestrator) loadManuscript() (string, error) {
	data, err := o.storage.Load(context.Background(), "manuscript.md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// loadScenesAsInput loads and combines scene files as input
func (o *GoalAwareOrchestrator) loadScenesAsInput() (interface{}, error) {
	scenes, err := o.loadScenes()
	if err != nil || len(scenes) == 0 {
		return "", fmt.Errorf("no content found")
	}
	return strings.Join(scenes, "\n\n"), nil
}

// enrichWithPlan adds plan data if available
func (o *GoalAwareOrchestrator) enrichWithPlan(manuscript string) (interface{}, error) {
	plan, err := o.storage.Load(context.Background(), "plan.json")
	if err != nil {
		return manuscript, nil // Plan is optional
	}
	
	return map[string]interface{}{
		"manuscript": manuscript,
		"plan":       string(plan),
	}, nil
}

// applyStrategyResults saves the strategy output back to storage
func (o *GoalAwareOrchestrator) applyStrategyResults(result interface{}) error {
	manuscript := o.extractManuscriptFromResult(result)
	if manuscript == "" {
		return fmt.Errorf("no manuscript content in result")
	}
	
	return o.storage.Save(context.Background(), "manuscript.md", []byte(manuscript))
}

// extractManuscriptFromResult gets manuscript content from various result types
func (o *GoalAwareOrchestrator) extractManuscriptFromResult(result interface{}) string {
	switch v := result.(type) {
	case string:
		return v
	case map[string]interface{}:
		if manuscript, ok := v["manuscript"].(string); ok {
			return manuscript
		}
	}
	return ""
}

// loadScenes loads individual scene files
func (o *GoalAwareOrchestrator) loadScenes() ([]string, error) {
	files, err := o.storage.List(context.Background(), "scenes/chapter_*_scene_*.txt")
	if err != nil {
		return nil, err
	}
	
	scenes := make([]string, 0, len(files))
	for _, file := range files {
		data, err := o.storage.Load(context.Background(), file)
		if err != nil {
			continue
		}
		scenes = append(scenes, string(data))
	}
	
	return scenes, nil
}

// isProgressing checks if we're making meaningful progress
func (o *GoalAwareOrchestrator) isProgressing(attempts int) bool {
	// Simple check - in production this would track progress over iterations
	return attempts < 3 // Allow 3 attempts for now
}

// Helper functions

func countWords(text string) int {
	// Simple word count - could be more sophisticated
	words := strings.Fields(text)
	return len(words)
}

func countChapters(text string) int {
	// Count markdown chapter headings
	lines := strings.Split(text, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "## Chapter") {
			count++
		}
	}
	return count
}