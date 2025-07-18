package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// IteratorAgent represents an agent that iteratively improves until all criteria are met
type IteratorAgent struct {
	agent           Agent
	logger          *slog.Logger
	maxIterations   int
	convergenceRate float64
	parallelism     int
}

// QualityCriteria represents a single quality check that can pass or fail
type QualityCriteria struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Priority    CriteriaPriority       `json:"priority"`
	Validator   CriteriaValidator      `json:"-"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

type CriteriaPriority int

const (
	CriticalPriority CriteriaPriority = iota
	HighPriority
	MediumPriority
	LowPriority
)

// CriteriaValidator checks if a criteria passes for given content
type CriteriaValidator func(ctx context.Context, content interface{}) (CriteriaResult, error)

// CriteriaResult represents the outcome of checking a criteria
type CriteriaResult struct {
	Passed      bool                   `json:"passed"`
	Score       float64                `json:"score"`
	Details     string                 `json:"details"`
	Suggestions []ImprovementSuggestion `json:"suggestions"`
	Evidence    []string               `json:"evidence,omitempty"`
}

// ImprovementSuggestion guides the agent on how to fix a failing criteria
type ImprovementSuggestion struct {
	Target      string `json:"target"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Example     string `json:"example,omitempty"`
	Complexity  string `json:"complexity"`
}

// IterationState tracks the current state of iterative improvement
type IterationState struct {
	Iteration        int                          `json:"iteration"`
	TotalCriteria    int                          `json:"total_criteria"`
	PassingCriteria  int                          `json:"passing_criteria"`
	FailingCriteria  []string                     `json:"failing_criteria"`
	CriteriaResults  map[string]CriteriaResult    `json:"criteria_results"`
	Content          interface{}                  `json:"content"`
	History          []IterationSnapshot          `json:"history"`
	ConvergenceScore float64                      `json:"convergence_score"`
	StartTime        time.Time                    `json:"start_time"`
	LastImprovement  time.Time                    `json:"last_improvement"`
	Context          map[string]interface{}       `json:"context"`
}

// IterationSnapshot captures state at a point in time
type IterationSnapshot struct {
	Iteration       int                       `json:"iteration"`
	PassingCriteria int                       `json:"passing_criteria"`
	Changes         []ContentChange           `json:"changes"`
	Timestamp       time.Time                 `json:"timestamp"`
	Duration        time.Duration             `json:"duration"`
}

// ContentChange represents a single change made during iteration
type ContentChange struct {
	Type        string    `json:"type"`
	Target      string    `json:"target"`
	Before      string    `json:"before,omitempty"`
	After       string    `json:"after"`
	Criteria    string    `json:"criteria"`
	Reason      string    `json:"reason"`
	Impact      float64   `json:"impact"`
	Timestamp   time.Time `json:"timestamp"`
}

// IteratorConfig configures the iterator behavior
type IteratorConfig struct {
	MaxIterations          int           `json:"max_iterations"`
	ConvergenceThreshold   float64       `json:"convergence_threshold"`
	ParallelCriteria       bool          `json:"parallel_criteria"`
	BackoffStrategy        string        `json:"backoff_strategy"`
	FocusMode              string        `json:"focus_mode"` // "worst-first", "priority", "random"
	BatchSize              int           `json:"batch_size"`
	MinImprovement         float64       `json:"min_improvement"`
	StagnationThreshold    int           `json:"stagnation_threshold"`
	AdaptiveLearning       bool          `json:"adaptive_learning"`
}

// NewIteratorAgent creates a new iterator agent
func NewIteratorAgent(agent Agent, logger *slog.Logger, config IteratorConfig) *IteratorAgent {
	return &IteratorAgent{
		agent:           agent,
		logger:          logger.With("component", "iterator_agent"),
		maxIterations:   config.MaxIterations,
		convergenceRate: config.ConvergenceThreshold,
		parallelism:     config.BatchSize,
	}
}

// IterateUntilConvergence keeps improving content until all criteria pass
func (ia *IteratorAgent) IterateUntilConvergence(ctx context.Context, content interface{}, criteria []QualityCriteria, config IteratorConfig) (*IterationState, error) {
	state := &IterationState{
		Iteration:       0,
		TotalCriteria:   len(criteria),
		CriteriaResults: make(map[string]CriteriaResult),
		Content:         content,
		History:         make([]IterationSnapshot, 0),
		StartTime:       time.Now(),
		Context:         make(map[string]interface{}),
	}

	// Initial assessment
	if err := ia.assessAllCriteria(ctx, state, criteria); err != nil {
		return state, fmt.Errorf("initial assessment failed: %w", err)
	}

	// Iterate until convergence or limits reached
	for state.Iteration < config.MaxIterations {
		if state.PassingCriteria == state.TotalCriteria {
			ia.logger.Info("All criteria met!", 
				"iterations", state.Iteration,
				"duration", time.Since(state.StartTime))
			break
		}

		// Check for stagnation
		if ia.isStagnant(state, config.StagnationThreshold) {
			ia.logger.Warn("Iteration stagnant, applying adaptive strategies")
			if err := ia.applyAdaptiveStrategies(ctx, state, criteria); err != nil {
				return state, fmt.Errorf("adaptive strategies failed: %w", err)
			}
		}

		// Perform iteration
		improved, err := ia.performIteration(ctx, state, criteria, config)
		if err != nil {
			return state, fmt.Errorf("iteration %d failed: %w", state.Iteration, err)
		}

		if !improved && config.MinImprovement > 0 {
			ia.logger.Warn("No improvement in iteration", "iteration", state.Iteration)
		}

		state.Iteration++
	}

	// Calculate final convergence score
	state.ConvergenceScore = ia.calculateConvergence(state)

	return state, nil
}

// performIteration executes one improvement iteration
func (ia *IteratorAgent) performIteration(ctx context.Context, state *IterationState, criteria []QualityCriteria, config IteratorConfig) (bool, error) {
	startTime := time.Now()
	
	// Identify failing criteria
	failingCriteria := ia.getFailingCriteria(state, criteria)
	if len(failingCriteria) == 0 {
		return false, nil
	}

	// Sort by priority and score
	failingCriteria = ia.prioritizeCriteria(failingCriteria, state, config.FocusMode)

	// Determine batch size
	batchSize := config.BatchSize
	if batchSize == 0 || batchSize > len(failingCriteria) {
		batchSize = len(failingCriteria)
	}

	// Process batch of improvements
	improved := false
	if config.ParallelCriteria && batchSize > 1 {
		improved = ia.processParallel(ctx, state, failingCriteria[:batchSize])
	} else {
		improved = ia.processSequential(ctx, state, failingCriteria[:batchSize])
	}

	// Reassess all criteria after changes
	previousPassing := state.PassingCriteria
	if err := ia.assessAllCriteria(ctx, state, criteria); err != nil {
		return false, fmt.Errorf("reassessment failed: %w", err)
	}

	// Record snapshot
	snapshot := IterationSnapshot{
		Iteration:       state.Iteration,
		PassingCriteria: state.PassingCriteria,
		Timestamp:       time.Now(),
		Duration:        time.Since(startTime),
	}
	state.History = append(state.History, snapshot)

	// Check if we made progress
	if state.PassingCriteria > previousPassing {
		state.LastImprovement = time.Now()
		improved = true
	}

	return improved, nil
}

// processSequential improves criteria one at a time
func (ia *IteratorAgent) processSequential(ctx context.Context, state *IterationState, criteria []QualityCriteria) bool {
	improved := false
	
	for _, criterion := range criteria {
		result := state.CriteriaResults[criterion.ID]
		if result.Passed {
			continue
		}

		// Generate improvement for this specific criteria
		newContent, _, err := ia.generateImprovement(ctx, state.Content, criterion, result)
		if err != nil {
			ia.logger.Error("Failed to generate improvement", 
				"criteria", criterion.Name,
				"error", err)
			continue
		}

		// Validate the improvement
		newResult, err := criterion.Validator(ctx, newContent)
		if err != nil {
			ia.logger.Error("Failed to validate improvement",
				"criteria", criterion.Name,
				"error", err)
			continue
		}

		// Accept improvement if it's better
		if newResult.Score > result.Score {
			state.Content = newContent
			state.CriteriaResults[criterion.ID] = newResult
			improved = true
			
			ia.logger.Info("Criteria improved",
				"criteria", criterion.Name,
				"before", result.Score,
				"after", newResult.Score)
		}
	}

	return improved
}

// processParallel improves multiple criteria simultaneously
func (ia *IteratorAgent) processParallel(ctx context.Context, state *IterationState, criteria []QualityCriteria) bool {
	var wg sync.WaitGroup
	improvements := make(chan struct {
		criterion QualityCriteria
		content   interface{}
		change    ContentChange
		result    CriteriaResult
	}, len(criteria))

	// Generate improvements in parallel
	for _, criterion := range criteria {
		wg.Add(1)
		go func(c QualityCriteria) {
			defer wg.Done()
			
			result := state.CriteriaResults[c.ID]
			if result.Passed {
				return
			}

			newContent, change, err := ia.generateImprovement(ctx, state.Content, c, result)
			if err != nil {
				ia.logger.Error("Parallel improvement failed", "criteria", c.Name, "error", err)
				return
			}

			newResult, err := c.Validator(ctx, newContent)
			if err != nil {
				ia.logger.Error("Parallel validation failed", "criteria", c.Name, "error", err)
				return
			}

			if newResult.Score > result.Score {
				improvements <- struct {
					criterion QualityCriteria
					content   interface{}
					change    ContentChange
					result    CriteriaResult
				}{c, newContent, change, newResult}
			}
		}(criterion)
	}

	wg.Wait()
	close(improvements)

	// Merge improvements
	improved := false
	for imp := range improvements {
		// In parallel mode, we need conflict resolution
		if ia.canApplyImprovement(state, imp.change) {
			state.Content = imp.content
			state.CriteriaResults[imp.criterion.ID] = imp.result
			improved = true
		}
	}

	return improved
}

// generateImprovement creates targeted improvement for a specific criteria
func (ia *IteratorAgent) generateImprovement(ctx context.Context, content interface{}, criterion QualityCriteria, result CriteriaResult) (interface{}, ContentChange, error) {
	// Build focused prompt for improvement
	prompt := ia.buildImprovementPrompt(content, criterion, result)
	
	// Execute improvement
	response, err := ia.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, ContentChange{}, fmt.Errorf("agent execution failed: %w", err)
	}

	// Parse and apply improvement
	improved, change, err := ia.parseAndApplyImprovement(content, response, criterion)
	if err != nil {
		return nil, ContentChange{}, fmt.Errorf("failed to apply improvement: %w", err)
	}

	return improved, change, nil
}

// buildImprovementPrompt creates a focused prompt for fixing specific criteria
func (ia *IteratorAgent) buildImprovementPrompt(content interface{}, criterion QualityCriteria, result CriteriaResult) string {
	prompt := fmt.Sprintf(`You are an expert editor focused on improving content to meet specific quality criteria.

Current Content:
%v

Failing Criteria: %s
Description: %s
Current Score: %.2f/1.0
Issues: %s

Improvement Suggestions:
`, content, criterion.Name, criterion.Description, result.Score, result.Details)

	for i, suggestion := range result.Suggestions {
		prompt += fmt.Sprintf("\n%d. %s\n   - Action: %s\n   - Reason: %s\n", 
			i+1, suggestion.Target, suggestion.Action, suggestion.Reason)
		if suggestion.Example != "" {
			prompt += fmt.Sprintf("   - Example: %s\n", suggestion.Example)
		}
	}

	prompt += `
Make the MINIMUM changes necessary to fix this specific criteria.
Return the improved content with clear indication of what changed.
Focus ONLY on addressing the failing criteria - do not change anything else.`

	return prompt
}

// assessAllCriteria evaluates content against all criteria
func (ia *IteratorAgent) assessAllCriteria(ctx context.Context, state *IterationState, criteria []QualityCriteria) error {
	state.PassingCriteria = 0
	state.FailingCriteria = make([]string, 0)

	for _, criterion := range criteria {
		result, err := criterion.Validator(ctx, state.Content)
		if err != nil {
			return fmt.Errorf("failed to validate %s: %w", criterion.Name, err)
		}

		state.CriteriaResults[criterion.ID] = result
		
		if result.Passed {
			state.PassingCriteria++
		} else {
			state.FailingCriteria = append(state.FailingCriteria, criterion.ID)
		}
	}

	return nil
}

// Helper methods
func (ia *IteratorAgent) getFailingCriteria(state *IterationState, allCriteria []QualityCriteria) []QualityCriteria {
	failing := make([]QualityCriteria, 0)
	for _, criterion := range allCriteria {
		if result, exists := state.CriteriaResults[criterion.ID]; exists && !result.Passed {
			failing = append(failing, criterion)
		}
	}
	return failing
}

func (ia *IteratorAgent) prioritizeCriteria(criteria []QualityCriteria, state *IterationState, focusMode string) []QualityCriteria {
	// Implementation depends on focus mode
	// "worst-first": Sort by lowest scores
	// "priority": Sort by priority level
	// "random": Randomize for variety
	return criteria
}

func (ia *IteratorAgent) isStagnant(state *IterationState, threshold int) bool {
	if len(state.History) < threshold {
		return false
	}
	
	// Check if passing criteria hasn't changed in last N iterations
	recent := state.History[len(state.History)-threshold:]
	firstPassing := recent[0].PassingCriteria
	
	for _, snapshot := range recent[1:] {
		if snapshot.PassingCriteria != firstPassing {
			return false
		}
	}
	
	return true
}

func (ia *IteratorAgent) applyAdaptiveStrategies(ctx context.Context, state *IterationState, criteria []QualityCriteria) error {
	// Implement adaptive strategies when stuck
	// - Relax criteria temporarily
	// - Try alternative approaches
	// - Combine multiple small improvements
	// - Request human guidance
	return nil
}

func (ia *IteratorAgent) calculateConvergence(state *IterationState) float64 {
	if state.TotalCriteria == 0 {
		return 1.0
	}
	
	// Simple convergence: percentage of passing criteria
	basicScore := float64(state.PassingCriteria) / float64(state.TotalCriteria)
	
	// Advanced: Weight by criteria scores and priorities
	totalScore := 0.0
	for _, result := range state.CriteriaResults {
		totalScore += result.Score
	}
	
	averageScore := totalScore / float64(len(state.CriteriaResults))
	
	// Combine both metrics
	return (basicScore + averageScore) / 2.0
}

func (ia *IteratorAgent) canApplyImprovement(state *IterationState, change ContentChange) bool {
	// Check if this improvement conflicts with other changes
	// In a real implementation, this would check for overlapping changes
	return true
}

func (ia *IteratorAgent) parseAndApplyImprovement(content interface{}, response string, criterion QualityCriteria) (interface{}, ContentChange, error) {
	// Parse the AI response and apply changes to content
	// This is domain-specific and would need proper implementation
	
	change := ContentChange{
		Type:      "improvement",
		Target:    criterion.Name,
		Criteria:  criterion.ID,
		Reason:    "AI-generated improvement",
		Timestamp: time.Now(),
	}
	
	// Return improved content
	return response, change, nil
}