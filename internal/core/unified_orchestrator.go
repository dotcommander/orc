package core

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/config"
)

// The Orchestrator interface is already defined in orchestrator.go

// UnifiedOrchestrator combines the best of Standard, Fluid, Goal-Aware, and Iterator architectures
type UnifiedOrchestrator struct {
	phases          []Phase
	storage         Storage
	logger          *slog.Logger
	checkpoint      *CheckpointManager
	verifier        *StageVerifier
	iteratorAgent   *IteratorAgent
	config          *config.Config
	executionMode   ExecutionMode
	goals           []*Goal
	criteria        []QualityCriteria
	adaptiveConfig  *AdaptiveConfig
	maxRetries      int
	mu              sync.RWMutex
}

// ExecutionMode determines how the orchestrator behaves
type ExecutionMode int

const (
	// StandardMode - Simple linear execution with basic retries
	StandardMode ExecutionMode = iota
	// AdaptiveMode - Dynamic phase selection with verification loops (Fluid)
	AdaptiveMode
	// QualityMode - Iterative improvement until criteria are met (Iterator)
	QualityMode
	// GoalMode - Target-driven execution with measurable outcomes
	GoalMode
	// UnifiedMode - Intelligently switches between modes based on request
	UnifiedMode
)

// AdaptiveConfig tracks learning and adaptation state
type AdaptiveConfig struct {
	PhasePerformance map[string]*PhaseMetrics `json:"phase_performance"`
	PatternMemory    map[string][]string      `json:"pattern_memory"`
	ErrorPatterns    map[string]int           `json:"error_patterns"`
	LastAdaptation   time.Time                `json:"last_adaptation"`
}

// PhaseMetrics tracks phase execution statistics
type PhaseMetrics struct {
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	AverageDuration time.Duration `json:"average_duration"`
	LastError      string        `json:"last_error,omitempty"`
	LastSuccess    time.Time     `json:"last_success"`
}

// UnifiedOrchestratorConfig extends the base config with unified features
type UnifiedOrchestratorConfig struct {
	// Execution modes
	Mode              ExecutionMode `json:"mode"`
	AutoModeSelection bool          `json:"auto_mode_selection"`
	
	// Quality settings (from Iterator)
	QualityCriteria      []QualityCriteria `json:"quality_criteria"`
	MaxQualityIterations int               `json:"max_quality_iterations"`
	ConvergenceThreshold float64           `json:"convergence_threshold"`
	
	// Adaptive settings (from Fluid)
	EnableAdaptation     bool              `json:"enable_adaptation"`
	VerificationRetries  int               `json:"verification_retries"`
	LearningRate         float64           `json:"learning_rate"`
	
	// Goal settings (from Goal-Aware)
	Goals               []*Goal            `json:"goals"`
	GoalTimeout         time.Duration      `json:"goal_timeout"`
	
	// Performance settings
	EnableParallelism   bool              `json:"enable_parallelism"`
	WorkerPoolSize      int               `json:"worker_pool_size"`
	EnableCaching       bool              `json:"enable_caching"`
}

// NewUnifiedOrchestrator creates a orchestrator that unifies all approaches
func NewUnifiedOrchestrator(
	phases []Phase,
	storage Storage,
	logger *slog.Logger,
	agent Agent,
	cfg *config.Config,
) *UnifiedOrchestrator {
	// Create sub-components
	checkpoint := NewCheckpointManager(storage)
	// TODO: Pass proper session ID and output directory when available
	verifier := NewStageVerifier("", "", logger)
	
	// Create iterator agent for quality mode
	iteratorConfig := IteratorConfig{
		MaxIterations:        5, // Default max iterations
		ConvergenceThreshold: 0.95,
		ParallelCriteria:     true,
		FocusMode:           "worst-first",
		BatchSize:           3,
		MinImprovement:      0.1,
		StagnationThreshold: 3,
		AdaptiveLearning:    true,
	}
	iteratorAgent := NewIteratorAgent(agent, logger, iteratorConfig)
	
	return &UnifiedOrchestrator{
		phases:         phases,
		storage:        storage,
		logger:         logger.With("component", "unified_orchestrator"),
		checkpoint:     checkpoint,
		verifier:       verifier,
		iteratorAgent:  iteratorAgent,
		config:         cfg,
		executionMode:  UnifiedMode,
		maxRetries:     3, // Default retry count
		adaptiveConfig: &AdaptiveConfig{
			PhasePerformance: make(map[string]*PhaseMetrics),
			PatternMemory:    make(map[string][]string),
			ErrorPatterns:    make(map[string]int),
			LastAdaptation:   time.Now(),
		},
	}
}

// Execute runs the unified orchestration pipeline
func (uo *UnifiedOrchestrator) Execute(ctx context.Context, request Request) (*Result, error) {
	uo.logger.Info("Starting unified orchestration",
		"mode", uo.executionMode,
		"request_type", request.Type,
		"session_id", request.SessionID)
	
	// Always use unified mode for automatic adaptation
	return uo.executeUnified(ctx, request)
}


// executeStandard runs simple linear phase execution
func (uo *UnifiedOrchestrator) executeStandard(ctx context.Context, request Request) (*Result, error) {
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// Load checkpoint if resuming
	checkpoint, err := uo.checkpoint.Load(ctx, request.SessionID)
	startPhase := 0
	if err == nil && checkpoint != nil {
		uo.logger.Info("Resuming from checkpoint", "phase_index", checkpoint.PhaseIndex)
		startPhase = checkpoint.PhaseIndex
	}
	
	// Execute phases sequentially
	var lastOutput PhaseOutput
	for i, phase := range uo.phases {
		// Skip if already completed
		if i < startPhase {
			continue
		}
		
		// Execute phase with circuit breaker
		phaseResult, err := uo.executePhaseWithRetry(ctx, phase, request, lastOutput)
		if err != nil {
			return result, fmt.Errorf("phase %s failed: %w", phase.Name(), err)
		}
		
		result.Phases = append(result.Phases, *phaseResult)
		lastOutput = phaseResult.Output
		
		// Save checkpoint
		phaseIndex := len(result.Phases)
		if err := uo.checkpoint.Save(ctx, request.SessionID, phaseIndex, phase.Name(), lastOutput); err != nil {
			uo.logger.Error("Failed to save checkpoint", "error", err)
		}
	}
	
	result.EndTime = time.Now()
	result.Success = true
	return result, nil
}

// executeAdaptive runs with dynamic phase selection and verification (Fluid approach)
func (uo *UnifiedOrchestrator) executeAdaptive(ctx context.Context, request Request) (*Result, error) {
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// Analyze request to determine optimal phase sequence
	phaseSequence := uo.determineAdaptiveSequence(request)
	uo.logger.Info("Adaptive phase sequence determined", "phases", phaseSequence)
	
	// Execute with verification loops
	var lastOutput PhaseOutput
	for _, phaseName := range phaseSequence {
		phase := uo.findPhase(phaseName)
		if phase == nil {
			return result, fmt.Errorf("phase %s not found", phaseName)
		}
		
		// Execute with verification
		verified := false
		attempts := 0
		maxAttempts := 3
		
		for !verified && attempts < maxAttempts {
			attempts++
			
			phaseResult, err := uo.executePhaseWithRetry(ctx, phase, request, lastOutput)
			if err != nil {
				uo.recordError(phase.Name(), err)
				if attempts == maxAttempts {
					return result, fmt.Errorf("phase %s failed after %d attempts: %w", phase.Name(), attempts, err)
				}
				continue
			}
			
			// Verify the output
			// TODO: Make verification configurable
			verified = true // For now, skip verification
			
			if verified {
				result.Phases = append(result.Phases, *phaseResult)
				lastOutput = phaseResult.Output
				uo.recordSuccess(phase.Name(), phaseResult.Duration)
			}
		}
	}
	
	result.EndTime = time.Now()
	result.Success = true
	return result, nil
}

// executeQuality runs with iterative improvement until quality criteria are met
func (uo *UnifiedOrchestrator) executeQuality(ctx context.Context, request Request) (*Result, error) {
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// First, run standard execution to get initial content
	standardResult, err := uo.executeStandard(ctx, request)
	if err != nil {
		return result, fmt.Errorf("initial execution failed: %w", err)
	}
	
	// Extract content from final phase
	if len(standardResult.Phases) == 0 {
		return standardResult, nil
	}
	
	finalOutput := standardResult.Phases[len(standardResult.Phases)-1].Output
	content := uo.extractContent(finalOutput)
	
	// Define or use provided quality criteria
	criteria := uo.criteria
	if len(criteria) == 0 {
		criteria = uo.generateDefaultCriteria(request.Type)
	}
	
	// Run iterator until convergence
	iteratorConfig := IteratorConfig{
		MaxIterations:        5, // Default max iterations
		ConvergenceThreshold: 0.95,
		ParallelCriteria:     true,
		FocusMode:           "worst-first",
		BatchSize:           3,
		MinImprovement:      0.1,
		StagnationThreshold: 3,
		AdaptiveLearning:    true,
	}
	
	iterationState, err := uo.iteratorAgent.IterateUntilConvergence(ctx, content, criteria, iteratorConfig)
	if err != nil {
		return result, fmt.Errorf("quality iteration failed: %w", err)
	}
	
	// Create quality phase result
	qualityResult := PhaseResult{
		PhaseName: "QualityIterator",
		StartTime: standardResult.StartTime,
		EndTime:   time.Now(),
		Duration:  time.Since(standardResult.StartTime),
		Success:   iterationState.PassingCriteria == iterationState.TotalCriteria,
		Output: PhaseOutput{
			Data: iterationState.Content,
			Metadata: map[string]interface{}{
				"summary": fmt.Sprintf("Quality iterations completed. Passing criteria: %d/%d, Convergence: %.2f",
					iterationState.PassingCriteria, iterationState.TotalCriteria, iterationState.ConvergenceScore),
				"iteration_count":  iterationState.Iteration,
				"convergence_score": iterationState.ConvergenceScore,
				"criteria_results": iterationState.CriteriaResults,
			},
		},
	}
	
	// Combine results
	result.Phases = standardResult.Phases
	result.Phases = append(result.Phases, qualityResult)
	result.EndTime = time.Now()
	result.Success = qualityResult.Success
	
	return result, nil
}

// executeGoal runs with specific goal tracking and achievement
func (uo *UnifiedOrchestrator) executeGoal(ctx context.Context, request Request) (*Result, error) {
	// Parse goals from request
	goals := uo.parseGoals(request)
	if len(goals) == 0 {
		// Fall back to standard execution
		return uo.executeStandard(ctx, request)
	}
	
	uo.logger.Info("Executing with goals", "goal_count", len(goals))
	
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// Run initial execution
	standardResult, err := uo.executeStandard(ctx, request)
	if err != nil {
		return result, fmt.Errorf("initial execution failed: %w", err)
	}
	
	result.Phases = standardResult.Phases
	
	// Check goals
	unmetGoals := uo.checkGoals(goals, standardResult)
	attempts := 0
	maxAttempts := 5
	
	// Iterate until goals are met or max attempts
	for len(unmetGoals) > 0 && attempts < maxAttempts {
		attempts++
		uo.logger.Info("Attempting to meet unmet goals", 
			"attempt", attempts, 
			"unmet_count", len(unmetGoals))
		
		// Generate improvement strategy
		strategy := uo.generateGoalStrategy(unmetGoals, standardResult)
		
		// Execute improvement
		improvementResult, err := uo.executeImprovement(ctx, strategy, standardResult)
		if err != nil {
			uo.logger.Error("Goal improvement failed", "attempt", attempts, "error", err)
			continue
		}
		
		// Add improvement phase
		result.Phases = append(result.Phases, *improvementResult)
		
		// Re-check goals
		unmetGoals = uo.checkGoals(goals, result)
	}
	
	// Final goal assessment
	goalResult := PhaseResult{
		PhaseName: "GoalAssessment",
		StartTime: result.StartTime,
		EndTime:   time.Now(),
		Duration:  time.Since(result.StartTime),
		Success:   len(unmetGoals) == 0,
		Output: PhaseOutput{
			Data: fmt.Sprintf("Goal execution completed. Met: %d/%d goals after %d attempts",
				len(goals)-len(unmetGoals), len(goals), attempts),
			Metadata: map[string]interface{}{
				"total_goals": len(goals),
				"met_goals":   len(goals) - len(unmetGoals),
				"unmet_goals": unmetGoals,
				"attempts":    attempts,
			},
		},
	}
	
	result.Phases = append(result.Phases, goalResult)
	result.EndTime = time.Now()
	result.Success = len(unmetGoals) == 0
	
	return result, nil
}

// executeUnified intelligently combines all approaches - both goal-aware AND fluid
func (uo *UnifiedOrchestrator) executeUnified(ctx context.Context, request Request) (*Result, error) {
	uo.logger.Info("🧠 Unified orchestrator starting - combining fluid adaptation with goal awareness")
	
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// Comprehensive request analysis
	goals := uo.parseGoals(request)
	hasQualityRequirements := uo.detectQualityRequirements(request)
	isComplex := uo.detectComplexity(request)
	needsAdaptation := uo.detectAdaptationNeed(request)
	
	// Create execution context that tracks both goals and adaptation
	execCtx := &UnifiedExecutionContext{
		Goals:                  goals,
		QualityRequirements:    hasQualityRequirements,
		IsComplex:             isComplex,
		NeedsAdaptation:       needsAdaptation,
		CurrentQualityScore:   0.0,
		PhaseAdaptations:      make(map[string]int),
		GoalProgress:          make(map[string]float64),
	}
	
	// Log our comprehensive execution plan
	uo.logger.Info("📋 Execution plan created",
		"goals", len(goals),
		"quality_focus", hasQualityRequirements,
		"complexity", isComplex,
		"adaptive", needsAdaptation || isComplex)
	
	if len(goals) > 0 {
		uo.logger.Info("🎯 Goals to track throughout execution:")
		for _, goal := range goals {
			uo.logger.Info("  • Goal", "type", goal.Type, "target", goal.Target, "priority", goal.Priority)
			execCtx.GoalProgress[string(goal.Type)] = 0.0
		}
	}
	
	// Execute with fluid adaptation AND continuous goal tracking
	result, err := uo.executeFluidWithGoals(ctx, request, execCtx)
	if err != nil {
		return result, fmt.Errorf("fluid goal-aware execution failed: %w", err)
	}
	
	// Final assessment and reporting
	finalDuration := time.Since(result.StartTime)
	
	// Check final goal status
	if len(goals) > 0 {
		finalUnmet := uo.checkGoals(goals, result)
		allGoalsMet := len(finalUnmet) == 0
		
		if allGoalsMet {
			uo.logger.Info("🎉 All goals successfully achieved through adaptive execution!")
		} else {
			uo.logger.Info("📊 Final goal status", "met", len(goals)-len(finalUnmet), "total", len(goals))
		}
		result.Success = result.Success && allGoalsMet
	}
	
	// Final quality assessment
	if hasQualityRequirements {
		finalQuality := uo.assessQuality(result)
		uo.logger.Info("✨ Final quality score", "score", fmt.Sprintf("%.2f", finalQuality))
		result.Success = result.Success && finalQuality >= 0.85
	}
	
	uo.logger.Info("🏁 Unified orchestration complete", 
		"duration", finalDuration.Round(time.Second),
		"phases_executed", len(result.Phases),
		"adaptations_made", uo.countAdaptations(execCtx),
		"success", result.Success)
	
	// Learn from this execution
	uo.learnFromExecution(request, result)
	
	return result, nil
}

// executeFluidWithGoals performs fluid execution while continuously tracking goals
func (uo *UnifiedOrchestrator) executeFluidWithGoals(ctx context.Context, request Request, execCtx *UnifiedExecutionContext) (*Result, error) {
	result := &Result{
		SessionID: request.SessionID,
		StartTime: time.Now(),
		Phases:    make([]PhaseResult, 0),
	}
	
	// Determine initial phase sequence based on request analysis
	phaseSequence := uo.determineAdaptiveSequence(request)
	uo.logger.Info("🌊 Starting fluid execution with goal awareness", "initial_phases", len(phaseSequence))
	
	var lastOutput PhaseOutput
	phaseIndex := 0
	
	// Execute phases with continuous adaptation and goal tracking
	for phaseIndex < len(phaseSequence) {
		phaseName := phaseSequence[phaseIndex]
		phase := uo.findPhase(phaseName)
		if phase == nil {
			uo.logger.Error("Phase not found", "phase", phaseName)
			phaseIndex++
			continue
		}
		
		uo.logger.Info("🔄 Executing phase", "phase", phaseName, "index", phaseIndex)
		
		// Execute phase with monitoring
		phaseResult, err := uo.executePhaseWithMonitoring(ctx, phase, request, lastOutput, execCtx)
		if err != nil {
			// Adaptive error handling
			uo.logger.Warn("Phase encountered issues", "phase", phaseName, "error", err)
			
			// Try adaptive recovery
			if recoveryPhase := uo.determineRecoveryPhase(phaseName, err, execCtx); recoveryPhase != "" {
				uo.logger.Info("🔧 Adapting with recovery phase", "recovery", recoveryPhase)
				// Insert recovery phase
				phaseSequence = append(phaseSequence[:phaseIndex+1], append([]string{recoveryPhase}, phaseSequence[phaseIndex+1:]...)...)
			} else if !phase.CanRetry(err) {
				return result, fmt.Errorf("phase %s failed without recovery: %w", phaseName, err)
			}
			
			execCtx.PhaseAdaptations[phaseName]++
			phaseIndex++
			continue
		}
		
		// Add successful phase result
		result.Phases = append(result.Phases, *phaseResult)
		lastOutput = phaseResult.Output
		
		// Update goal progress after each phase
		if len(execCtx.Goals) > 0 {
			uo.updateGoalProgress(execCtx, result)
			
			// Log progress
			for goalType, progress := range execCtx.GoalProgress {
				if progress > 0 {
					uo.logger.Info("📈 Goal progress", "type", goalType, "progress", fmt.Sprintf("%.1f%%", progress*100))
				}
			}
			
			// Check if we need to adapt based on goal progress
			if adaptation := uo.checkGoalAdaptation(execCtx, phaseIndex, phaseSequence); adaptation != "" {
				uo.logger.Info("🎯 Adapting for better goal achievement", "adding_phase", adaptation)
				phaseSequence = append(phaseSequence[:phaseIndex+1], append([]string{adaptation}, phaseSequence[phaseIndex+1:]...)...)
			}
		}
		
		// Quality monitoring and adaptation
		if execCtx.QualityRequirements {
			currentQuality := uo.assessPhaseQuality(phaseResult)
			execCtx.CurrentQualityScore = currentQuality
			
			if currentQuality < 0.7 && phaseIndex < len(phaseSequence)-1 {
				uo.logger.Info("🔧 Quality below threshold, inserting refinement phase", "score", fmt.Sprintf("%.2f", currentQuality))
				// Insert quality refinement phase
				phaseSequence = append(phaseSequence[:phaseIndex+1], append([]string{"QualityRefinement"}, phaseSequence[phaseIndex+1:]...)...)
			}
		}
		
		// Check if we should add more phases based on current state
		if newPhases := uo.determineAdditionalPhases(execCtx, result, phaseIndex); len(newPhases) > 0 {
			uo.logger.Info("🌊 Fluidly adding phases based on current progress", "new_phases", newPhases)
			phaseSequence = append(phaseSequence, newPhases...)
		}
		
		phaseIndex++
	}
	
	// Final goal-aware quality iteration if needed
	if execCtx.QualityRequirements || len(execCtx.Goals) > 0 {
		needsFinalIteration := false
		
		// Check goals
		if len(execCtx.Goals) > 0 {
			unmetGoals := uo.checkGoals(execCtx.Goals, result)
			needsFinalIteration = len(unmetGoals) > 0
			
			if needsFinalIteration {
				uo.logger.Info("🎯 Final iteration needed for unmet goals", "unmet", len(unmetGoals))
			}
		}
		
		// Check quality
		if execCtx.QualityRequirements && !needsFinalIteration {
			finalQuality := uo.assessQuality(result)
			needsFinalIteration = finalQuality < 0.85
			
			if needsFinalIteration {
				uo.logger.Info("✨ Final iteration needed for quality", "current", fmt.Sprintf("%.2f", finalQuality))
			}
		}
		
		// Run unified final iteration combining quality and goal achievement
		if needsFinalIteration {
			uo.logger.Info("🔄 Running final unified iteration for goals and quality")
			
			iterationCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			
			finalPhase := uo.createUnifiedIterationPhase(execCtx, result)
			finalResult, err := uo.executePhaseWithMonitoring(iterationCtx, finalPhase, request, lastOutput, execCtx)
			if err != nil {
				uo.logger.Error("Final iteration failed", "error", err)
			} else {
				result.Phases = append(result.Phases, *finalResult)
				uo.logger.Info("✅ Final iteration completed successfully")
			}
		}
	}
	
	result.EndTime = time.Now()
	result.Success = true
	
	return result, nil
}

// Supporting structures and methods

type UnifiedExecutionContext struct {
	Goals               []*Goal
	QualityRequirements bool
	IsComplex          bool
	NeedsAdaptation    bool
	CurrentQualityScore float64
	PhaseAdaptations   map[string]int
	GoalProgress       map[string]float64
}

func (uo *UnifiedOrchestrator) executePhaseWithMonitoring(ctx context.Context, phase Phase, request Request, lastOutput PhaseOutput, execCtx *UnifiedExecutionContext) (*PhaseResult, error) {
	// Execute phase with enhanced monitoring
	startTime := time.Now()
	
	input := PhaseInput{
		Request:   request.Content,
		Data:      lastOutput.Data,
		SessionID: request.SessionID,
		Metadata:  map[string]interface{}{
			"goals":           execCtx.Goals,
			"quality_focused": execCtx.QualityRequirements,
			"adaptive_mode":   true,
		},
	}
	
	output, err := phase.Execute(ctx, input)
	if err != nil {
		return nil, err
	}
	
	result := &PhaseResult{
		PhaseName: phase.Name(),
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(startTime),
		Success:   true,
		Output:    output,
	}
	
	return result, nil
}

func (uo *UnifiedOrchestrator) updateGoalProgress(execCtx *UnifiedExecutionContext, result *Result) {
	// Update progress for each goal based on current results
	for _, goal := range execCtx.Goals {
		currentProgress := uo.calculateGoalProgress(goal, result)
		execCtx.GoalProgress[string(goal.Type)] = currentProgress
	}
}

func (uo *UnifiedOrchestrator) calculateGoalProgress(goal *Goal, result *Result) float64 {
	// Extract current content
	if len(result.Phases) == 0 {
		return 0.0
	}
	
	lastPhase := result.Phases[len(result.Phases)-1]
	content := uo.extractContent(lastPhase.Output)
	
	switch goal.Type {
	case GoalTypeWordCount:
		if targetWords, ok := goal.Target.(int); ok && targetWords > 0 {
			currentWords := countWords(fmt.Sprintf("%v", content))
			return math.Min(float64(currentWords)/float64(targetWords), 1.0)
		}
	case GoalTypeChapterCount:
		if targetChapters, ok := goal.Target.(int); ok && targetChapters > 0 {
			currentChapters := countChapters(fmt.Sprintf("%v", content))
			return math.Min(float64(currentChapters)/float64(targetChapters), 1.0)
		}
	}
	
	return 0.0
}

func (uo *UnifiedOrchestrator) checkGoalAdaptation(execCtx *UnifiedExecutionContext, currentPhase int, phaseSequence []string) string {
	// Determine if we need to adapt based on goal progress
	for _, goal := range execCtx.Goals {
		progress := execCtx.GoalProgress[string(goal.Type)]
		
		// If we're past halfway through phases but goal progress is low
		if currentPhase > len(phaseSequence)/2 && progress < 0.3 {
			switch goal.Type {
			case GoalTypeWordCount:
				return "ContentExpansion"
			case GoalTypeChapterCount:
				return "ChapterGeneration"
			}
		}
	}
	
	return ""
}

func (uo *UnifiedOrchestrator) determineRecoveryPhase(failedPhase string, err error, execCtx *UnifiedExecutionContext) string {
	// Intelligently determine recovery strategy
	errorStr := err.Error()
	
	if strings.Contains(errorStr, "timeout") {
		return "QuickGeneration"
	}
	
	if strings.Contains(errorStr, "quality") && execCtx.QualityRequirements {
		return "QualityRefinement"
	}
	
	if strings.Contains(errorStr, "incomplete") && len(execCtx.Goals) > 0 {
		return "GoalCompletion"
	}
	
	return ""
}

func (uo *UnifiedOrchestrator) assessPhaseQuality(result *PhaseResult) float64 {
	// Simple quality assessment of a single phase
	if !result.Success {
		return 0.0
	}
	
	// Base quality on execution time and retries
	quality := 0.8
	
	if result.Duration > 5*time.Minute {
		quality -= 0.1
	}
	
	if result.Retries > 0 {
		quality -= float64(result.Retries) * 0.1
	}
	
	return math.Max(quality, 0.0)
}

func (uo *UnifiedOrchestrator) determineAdditionalPhases(execCtx *UnifiedExecutionContext, result *Result, currentPhase int) []string {
	// Dynamically determine if we need additional phases
	additionalPhases := []string{}
	
	// Check if goals need more work
	if len(execCtx.Goals) > 0 {
		avgProgress := 0.0
		for _, progress := range execCtx.GoalProgress {
			avgProgress += progress
		}
		avgProgress /= float64(len(execCtx.GoalProgress))
		
		// If average progress is low, add expansion phases
		if avgProgress < 0.5 && currentPhase > 2 {
			additionalPhases = append(additionalPhases, "ContentExpansion")
		}
	}
	
	// Check if quality needs improvement
	if execCtx.QualityRequirements && execCtx.CurrentQualityScore < 0.8 {
		additionalPhases = append(additionalPhases, "QualityEnhancement")
	}
	
	return additionalPhases
}

func (uo *UnifiedOrchestrator) createUnifiedIterationPhase(execCtx *UnifiedExecutionContext, result *Result) Phase {
	// Create a dynamic phase that addresses both goals and quality
	return &UnifiedIterationPhase{
		name:     "UnifiedIteration",
		goals:    execCtx.Goals,
		quality:  execCtx.QualityRequirements,
		agent:    uo.iteratorAgent.agent,
		logger:   uo.logger,
	}
}

func (uo *UnifiedOrchestrator) countAdaptations(execCtx *UnifiedExecutionContext) int {
	count := 0
	for _, adaptations := range execCtx.PhaseAdaptations {
		count += adaptations
	}
	return count
}

// UnifiedIterationPhase is a dynamic phase for final iterations
type UnifiedIterationPhase struct {
	name    string
	goals   []*Goal
	quality bool
	agent   Agent
	logger  *slog.Logger
}

func (p *UnifiedIterationPhase) Name() string { return p.name }

func (p *UnifiedIterationPhase) Execute(ctx context.Context, input PhaseInput) (PhaseOutput, error) {
	// Execute unified iteration focusing on both goals and quality
	prompt := "Improve the following content to ensure:\n"
	
	if p.quality {
		prompt += "- Professional quality and polish\n"
	}
	
	for _, goal := range p.goals {
		switch goal.Type {
		case GoalTypeWordCount:
			prompt += fmt.Sprintf("- Reach exactly %v words\n", goal.Target)
		case GoalTypeChapterCount:
			prompt += fmt.Sprintf("- Include exactly %v chapters\n", goal.Target)
		}
	}
	
	prompt += "\nCurrent content:\n" + fmt.Sprintf("%v", input.Data)
	
	response, err := p.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return PhaseOutput{}, err
	}
	
	return PhaseOutput{
		Data: response,
		Metadata: map[string]interface{}{
			"iteration_type": "unified",
			"goals":         len(p.goals),
			"quality_focus": p.quality,
		},
	}, nil
}

func (p *UnifiedIterationPhase) ValidateInput(ctx context.Context, input PhaseInput) error { return nil }
func (p *UnifiedIterationPhase) ValidateOutput(ctx context.Context, output PhaseOutput) error { return nil }
func (p *UnifiedIterationPhase) EstimatedDuration() time.Duration { return 5 * time.Minute }
func (p *UnifiedIterationPhase) CanRetry(err error) bool { return true }

// Helper methods

func (uo *UnifiedOrchestrator) executePhaseWithRetry(ctx context.Context, phase Phase, request Request, lastOutput PhaseOutput) (*PhaseResult, error) {
	input := PhaseInput{
		Request:   request.Content,
		Data:      lastOutput.Data,
		SessionID: request.SessionID,
		Metadata:  make(map[string]interface{}),
	}
	
	result := &PhaseResult{
		PhaseName: phase.Name(),
		StartTime: time.Now(),
	}
	
	var err error
	for attempt := 0; attempt <= uo.maxRetries; attempt++ {
		if attempt > 0 {
			uo.logger.Info("Retrying phase", "phase", phase.Name(), "attempt", attempt)
		}
		
		output, phaseErr := phase.Execute(ctx, input)
		if phaseErr == nil {
			result.Success = true
			result.Output = output
			result.Retries = attempt
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, nil
		}
		
		err = phaseErr
		if !phase.CanRetry(phaseErr) {
			break
		}
		
		// Exponential backoff
		backoff := time.Duration(attempt+1) * time.Second
		time.Sleep(backoff)
	}
	
	result.Success = false
	result.Error = err
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result, err
}

func (uo *UnifiedOrchestrator) findPhase(name string) Phase {
	for _, phase := range uo.phases {
		if phase.Name() == name {
			return phase
		}
	}
	return nil
}

func (uo *UnifiedOrchestrator) determineAdaptiveSequence(request Request) []string {
	// Analyze request patterns and history to determine optimal phase sequence
	// This is a simplified version - real implementation would be more sophisticated
	
	baseSequence := make([]string, len(uo.phases))
	for i, phase := range uo.phases {
		baseSequence[i] = phase.Name()
	}
	
	// Apply learned optimizations
	uo.mu.RLock()
	if patterns, exists := uo.adaptiveConfig.PatternMemory[request.Type]; exists && len(patterns) > 0 {
		// Use the most successful pattern
		uo.mu.RUnlock()
		return patterns
	}
	uo.mu.RUnlock()
	
	return baseSequence
}

func (uo *UnifiedOrchestrator) detectQualityRequirements(request Request) bool {
	// Check for quality indicators in request
	qualityKeywords := []string{"quality", "polish", "refine", "perfect", "professional", "production"}
	for _, keyword := range qualityKeywords {
		if containsIgnoreCase(request.Content, keyword) {
			return true
		}
	}
	return false
}

func (uo *UnifiedOrchestrator) detectMeasurableGoals(request Request) bool {
	// Check for measurable goals
	goalPatterns := []string{"chapter", "word", "page", "feature", "function", "endpoint"}
	for _, pattern := range goalPatterns {
		if containsIgnoreCase(request.Content, pattern) {
			return true
		}
	}
	return false
}

func (uo *UnifiedOrchestrator) detectComplexity(request Request) bool {
	// Simple complexity detection based on request length and keywords
	if len(request.Content) > 500 {
		return true
	}
	
	complexKeywords := []string{"complex", "advanced", "sophisticated", "comprehensive", "detailed"}
	for _, keyword := range complexKeywords {
		if containsIgnoreCase(request.Content, keyword) {
			return true
		}
	}
	
	return false
}

func (uo *UnifiedOrchestrator) detectAdaptationNeed(request Request) bool {
	// Check if this type of request has had failures before
	uo.mu.RLock()
	defer uo.mu.RUnlock()
	
	if errorCount, exists := uo.adaptiveConfig.ErrorPatterns[request.Type]; exists && errorCount > 2 {
		return true
	}
	
	return false
}

func (uo *UnifiedOrchestrator) recordError(phaseName string, err error) {
	uo.mu.Lock()
	defer uo.mu.Unlock()
	
	if metrics, exists := uo.adaptiveConfig.PhasePerformance[phaseName]; exists {
		metrics.FailureCount++
		metrics.LastError = err.Error()
	} else {
		uo.adaptiveConfig.PhasePerformance[phaseName] = &PhaseMetrics{
			FailureCount: 1,
			LastError:    err.Error(),
		}
	}
}

func (uo *UnifiedOrchestrator) recordSuccess(phaseName string, duration time.Duration) {
	uo.mu.Lock()
	defer uo.mu.Unlock()
	
	if metrics, exists := uo.adaptiveConfig.PhasePerformance[phaseName]; exists {
		metrics.SuccessCount++
		metrics.LastSuccess = time.Now()
		// Update average duration
		total := metrics.AverageDuration * time.Duration(metrics.SuccessCount-1)
		metrics.AverageDuration = (total + duration) / time.Duration(metrics.SuccessCount)
	} else {
		uo.adaptiveConfig.PhasePerformance[phaseName] = &PhaseMetrics{
			SuccessCount:    1,
			AverageDuration: duration,
			LastSuccess:     time.Now(),
		}
	}
}

func (uo *UnifiedOrchestrator) adaptFromIssues(phaseName string, issues []string) {
	// Learn from verification issues to improve future executions
	uo.mu.Lock()
	defer uo.mu.Unlock()
	
	// Simple adaptation: track error patterns
	for _, issue := range issues {
		uo.adaptiveConfig.ErrorPatterns[issue]++
	}
	
	uo.adaptiveConfig.LastAdaptation = time.Now()
}

func (uo *UnifiedOrchestrator) learnFromExecution(request Request, result *Result) {
	// Extract patterns from successful execution
	if !result.Success {
		return
	}
	
	uo.mu.Lock()
	defer uo.mu.Unlock()
	
	// Record successful phase sequence
	phaseNames := make([]string, len(result.Phases))
	for i, phase := range result.Phases {
		phaseNames[i] = phase.PhaseName
	}
	
	uo.adaptiveConfig.PatternMemory[request.Type] = phaseNames
}

func (uo *UnifiedOrchestrator) extractContent(output PhaseOutput) interface{} {
	// Extract the main content from phase output
	artifacts := output.GetArtifacts()
	if content, exists := artifacts["content"]; exists {
		return content
	}
	if manuscript, exists := artifacts["manuscript"]; exists {
		return manuscript
	}
	if output.Data != nil {
		return output.Data
	}
	return output.AsContent()
}

func (uo *UnifiedOrchestrator) generateDefaultCriteria(requestType string) []QualityCriteria {
	// Generate sensible default criteria based on request type
	switch requestType {
	case "fiction":
		return uo.generateFictionCriteria()
	case "code":
		return uo.generateCodeCriteria()
	case "documentation":
		return uo.generateDocumentationCriteria()
	default:
		return uo.generateGenericCriteria()
	}
}

func (uo *UnifiedOrchestrator) generateFictionCriteria() []QualityCriteria {
	return []QualityCriteria{
		{
			ID:          "coherence",
			Name:        "Narrative Coherence",
			Description: "Story maintains logical consistency and flow",
			Category:    "structure",
			Priority:    CriticalPriority,
		},
		{
			ID:          "character_development",
			Name:        "Character Development",
			Description: "Characters are well-developed and consistent",
			Category:    "content",
			Priority:    HighPriority,
		},
		{
			ID:          "engagement",
			Name:        "Reader Engagement",
			Description: "Story maintains reader interest throughout",
			Category:    "quality",
			Priority:    HighPriority,
		},
	}
}

func (uo *UnifiedOrchestrator) generateCodeCriteria() []QualityCriteria {
	return []QualityCriteria{
		{
			ID:          "correctness",
			Name:        "Code Correctness",
			Description: "Code compiles and runs without errors",
			Category:    "functionality",
			Priority:    CriticalPriority,
		},
		{
			ID:          "best_practices",
			Name:        "Best Practices",
			Description: "Code follows language best practices and conventions",
			Category:    "quality",
			Priority:    HighPriority,
		},
		{
			ID:          "documentation",
			Name:        "Code Documentation",
			Description: "Code is well-documented with clear comments",
			Category:    "maintainability",
			Priority:    MediumPriority,
		},
	}
}

func (uo *UnifiedOrchestrator) generateDocumentationCriteria() []QualityCriteria {
	return []QualityCriteria{
		{
			ID:          "clarity",
			Name:        "Documentation Clarity",
			Description: "Documentation is clear and easy to understand",
			Category:    "readability",
			Priority:    CriticalPriority,
		},
		{
			ID:          "completeness",
			Name:        "Documentation Completeness",
			Description: "All necessary topics are covered",
			Category:    "coverage",
			Priority:    HighPriority,
		},
		{
			ID:          "accuracy",
			Name:        "Technical Accuracy",
			Description: "Information is technically correct and up-to-date",
			Category:    "correctness",
			Priority:    CriticalPriority,
		},
	}
}

func (uo *UnifiedOrchestrator) generateGenericCriteria() []QualityCriteria {
	return []QualityCriteria{
		{
			ID:          "completeness",
			Name:        "Content Completeness",
			Description: "All requested elements are present",
			Category:    "coverage",
			Priority:    HighPriority,
		},
		{
			ID:          "quality",
			Name:        "Overall Quality",
			Description: "Content meets professional standards",
			Category:    "quality",
			Priority:    HighPriority,
		},
	}
}

func (uo *UnifiedOrchestrator) parseGoals(request Request) []*Goal {
	// Parse measurable goals from request
	// This is simplified - real implementation would use NLP
	goals := make([]*Goal, 0)
	
	// Word count goals
	if match := parseWordCount(request.Content); match > 0 {
		goals = append(goals, &Goal{
			Type:        GoalTypeWordCount,
			Target:      match,
			Priority:    8,
		})
	}
	
	// Chapter goals
	if match := parseChapterCount(request.Content); match > 0 {
		goals = append(goals, &Goal{
			Type:        GoalTypeChapterCount,
			Target:      match,
			Priority:    7,
		})
	}
	
	return goals
}

func (uo *UnifiedOrchestrator) checkGoals(goals []*Goal, result *Result) []*Goal {
	unmet := make([]*Goal, 0)
	
	// Extract final content
	if len(result.Phases) == 0 {
		return goals // All unmet if no phases
	}
	
	finalOutput := result.Phases[len(result.Phases)-1].Output
	content := uo.extractContent(finalOutput)
	
	// Check each goal
	for _, goal := range goals {
		if !uo.isGoalMet(goal, content) {
			unmet = append(unmet, goal)
		}
	}
	
	return unmet
}

func (uo *UnifiedOrchestrator) isGoalMet(goal *Goal, content interface{}) bool {
	// Check if a specific goal is met
	// This is simplified - real implementation would be more sophisticated
	
	contentStr, ok := content.(string)
	if !ok {
		return false
	}
	
	switch goal.Type {
	case GoalTypeWordCount:
		wordCount := countWords(contentStr)
		targetInt, ok := goal.Target.(int)
		if !ok {
			return false
		}
		return wordCount >= targetInt
	case GoalTypeChapterCount:
		chapterCount := countChapters(contentStr)
		targetInt, ok := goal.Target.(int)
		if !ok {
			return false
		}
		return chapterCount >= targetInt
	default:
		return false
	}
}

func (uo *UnifiedOrchestrator) generateGoalStrategy(unmetGoals []*Goal, result *Result) ImprovementStrategy {
	// Generate strategy to meet unmet goals
	suggestions := make([]string, 0)
	
	for _, goal := range unmetGoals {
		switch goal.Type {
		case GoalTypeWordCount:
			suggestions = append(suggestions, "Expand existing content with more detail and description")
		case GoalTypeChapterCount:
			suggestions = append(suggestions, "Add additional chapters to meet the target count")
		}
	}
	
	return ImprovementStrategy{
		Goals:       unmetGoals,
		Suggestions: suggestions,
	}
}

func (uo *UnifiedOrchestrator) executeImprovement(ctx context.Context, strategy ImprovementStrategy, previousResult *Result) (*PhaseResult, error) {
	// Execute improvement based on strategy
	// This would use the agent to generate improvements
	
	result := &PhaseResult{
		PhaseName: "GoalImprovement",
		StartTime: time.Now(),
		Success:   true,
		Output: PhaseOutput{
			Data: "Improvement executed based on strategy",
			Metadata: map[string]interface{}{
				"strategy": strategy,
			},
		},
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result, nil
}

func (uo *UnifiedOrchestrator) hasGeneratedContent(result *Result) bool {
	// Check if the result contains generated content
	for _, phase := range result.Phases {
		artifacts := phase.Output.GetArtifacts()
		if _, hasContent := artifacts["content"]; hasContent {
			return true
		}
		if _, hasManuscript := artifacts["manuscript"]; hasManuscript {
			return true
		}
		if phase.Output.Data != nil {
			return true
		}
	}
	return false
}

func (uo *UnifiedOrchestrator) assessQuality(result *Result) float64 {
	// Simple quality assessment
	// Real implementation would use more sophisticated metrics
	
	if !result.Success {
		return 0.0
	}
	
	// Base score for successful completion
	score := 0.6
	
	// Bonus for no retries
	retriesUsed := 0
	for _, phase := range result.Phases {
		if phase.Retries > 0 {
			retriesUsed += phase.Retries
		}
	}
	
	if retriesUsed == 0 {
		score += 0.2
	} else if retriesUsed < 3 {
		score += 0.1
	}
	
	// Bonus for fast execution
	totalDuration := result.EndTime.Sub(result.StartTime)
	if totalDuration < 5*time.Minute {
		score += 0.2
	} else if totalDuration < 10*time.Minute {
		score += 0.1
	}
	
	return score
}


// Utility functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 len(s) > len(substr) && 
		 containsIgnoreCase(s[1:], substr) ||
		 containsIgnoreCase(s[:len(s)-1], substr))
}

func parseWordCount(content string) int {
	// Simple word count parser
	// Look for patterns like "1000 word", "50,000 words", "50k words"
	
	// Try simple number + "word" pattern
	matches := regexp.MustCompile(`(\d+)\s*word`).FindStringSubmatch(content)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count
		}
	}
	
	// Try number with commas + "words"
	matches = regexp.MustCompile(`([\d,]+)\s*words?`).FindStringSubmatch(content)
	if len(matches) > 1 {
		// Remove commas
		numStr := strings.ReplaceAll(matches[1], ",", "")
		if count, err := strconv.Atoi(numStr); err == nil {
			return count
		}
	}
	
	// Try "k" notation (e.g., "50k words")
	matches = regexp.MustCompile(`(\d+)k\s*words?`).FindStringSubmatch(content)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count * 1000
		}
	}
	
	return 0
}

func parseChapterCount(content string) int {
	// Simple chapter count parser
	// Look for patterns like "20 chapters", "5 chapter"
	
	matches := regexp.MustCompile(`(\d+)\s*chapters?`).FindStringSubmatch(content)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count
		}
	}
	
	return 0
}

// countWords and countChapters are defined in goal_orchestrator.go

// Supporting types

type ImprovementStrategy struct {
	Goals       []*Goal  `json:"goals"`
	Suggestions []string `json:"suggestions"`
}

// Request represents an orchestration request
type Request struct {
	SessionID string
	Type      string
	Content   string
}

// Result represents the orchestration result
type Result struct {
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Phases    []PhaseResult
}

// PhaseResult represents the result of a single phase execution
type PhaseResult struct {
	PhaseName string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Success   bool
	Retries   int
	Output    PhaseOutput
	Error     error
}

// Convert PhaseOutput to simplified content for some methods
func (po PhaseOutput) AsContent() string {
	if po.Data != nil {
		return fmt.Sprintf("%v", po.Data)
	}
	return ""
}

// Get artifacts from PhaseOutput 
func (po PhaseOutput) GetArtifacts() map[string]interface{} {
	if po.Metadata != nil {
		if artifacts, ok := po.Metadata["artifacts"].(map[string]interface{}); ok {
			return artifacts
		}
	}
	return map[string]interface{}{"data": po.Data}
}