package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// FluidOrchestrator uses dynamic phase execution with adaptive behavior
type FluidOrchestrator struct {
	phaseFlow        *PhaseFlow
	errorHandler     *AdaptiveErrorHandler
	promptFlow       *PromptFlow
	verifier         *StageVerifier
	storage          Storage
	logger           *slog.Logger
	sessionID        string
	checkpoint       *CheckpointManager
	config           FluidConfig
	learningEnabled  bool
	outputDir        string
	mu               sync.RWMutex
}

// FluidConfig configures fluid orchestrator behavior
type FluidConfig struct {
	// Dynamic phase discovery
	EnablePhaseDiscovery bool
	PhasePatterns        []string
	
	// Adaptive error handling
	EnableLearning      bool
	ErrorRecoveryLevel  int // 0=none, 1=basic, 2=adaptive, 3=aggressive
	
	// Flexible prompting
	EnablePromptFlow    bool
	PromptOptimization  bool
	
	// Runtime configuration
	AllowHotReload      bool
	ConfigWatchInterval time.Duration
	
	// Performance
	MaxConcurrency      int
	AdaptiveConcurrency bool
	
	// Goal awareness
	GoalCheckInterval   time.Duration
	AdaptiveGoals       bool
}

// DefaultFluidConfig returns sensible defaults
func DefaultFluidConfig() FluidConfig {
	return FluidConfig{
		EnablePhaseDiscovery: true,
		EnableLearning:       true,
		ErrorRecoveryLevel:   2, // Adaptive
		EnablePromptFlow:     true,
		PromptOptimization:   true,
		AllowHotReload:       true,
		ConfigWatchInterval:  30 * time.Second,
		MaxConcurrency:       0, // Auto-detect
		AdaptiveConcurrency:  true,
		GoalCheckInterval:    5 * time.Minute,
		AdaptiveGoals:        true,
	}
}

// NewFluidOrchestrator creates an adaptive orchestrator
func NewFluidOrchestrator(storage Storage, sessionID string, outputDir string, logger *slog.Logger, config FluidConfig) *FluidOrchestrator {
	fo := &FluidOrchestrator{
		phaseFlow:       NewPhaseFlow(logger),
		errorHandler:    NewAdaptiveErrorHandler(),
		promptFlow:      NewPromptFlow(),
		verifier:        NewStageVerifier(sessionID, outputDir, logger),
		storage:         storage,
		logger:          logger,
		sessionID:       sessionID,
		outputDir:       outputDir,
		config:          config,
		learningEnabled: config.EnableLearning,
		checkpoint:      NewCheckpointManager(storage),
	}
	
	// Register default verifiers
	fo.verifier.RegisterDefaultVerifiers()
	
	// Start configuration watcher if enabled
	if config.AllowHotReload {
		go fo.watchConfiguration()
	}
	
	return fo
}

// RegisterPhase adds a phase with fluid configuration
func (fo *FluidOrchestrator) RegisterPhase(phase Phase, opts ...PhaseOption) {
	// Add adaptive conditions
	adaptiveOpts := append(opts,
		WithCondition(fo.createAdaptiveCondition(phase)),
		WithPriority(fo.calculatePhasePriority(phase)),
	)
	
	// Register with phase flow
	fo.phaseFlow.RegisterPhase(phase, adaptiveOpts...)
	
	// Register phase-specific prompt templates
	if fo.config.EnablePromptFlow {
		fo.registerPhasePrompts(phase)
	}
}

// RegisterModularPhase adds a modular phase
func (fo *FluidOrchestrator) RegisterModularPhase(phase *ModularPhase) {
	// ModularPhase needs an adapter to implement Phase interface
	// For now, just register the underlying phases individually
	for range phase.components {
		// Each component would need to be wrapped as a Phase
		// This is a placeholder - full implementation would create PhaseAdapter
	}
}

// Run executes with full fluid behavior
func (fo *FluidOrchestrator) Run(ctx context.Context, request string) error {
	fo.logger.Info("starting fluid orchestration",
		"request", request,
		"session", fo.sessionID,
		"learning", fo.learningEnabled)
	
	// Discover additional phases if enabled
	if fo.config.EnablePhaseDiscovery {
		fo.discoverAndRegisterPhases(request)
	}
	
	// Create execution context with adaptive behavior
	execCtx := fo.createExecutionContext(ctx, request)
	
	// Execute with error recovery
	results, err := fo.executeWithRecovery(execCtx, request)
	if err != nil {
		return err
	}
	
	// Learn from execution
	if fo.learningEnabled {
		fo.learnFromExecution(results)
	}
	
	fo.logger.Info("fluid orchestration completed",
		"session", fo.sessionID,
		"phases_executed", len(results))
	
	return nil
}

// executeWithRecovery handles execution with adaptive error recovery and verification
func (fo *FluidOrchestrator) executeWithRecovery(ctx context.Context, request string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	// Get ordered phases for execution
	phases := fo.getOrderedPhases()
	
	// Execute each phase with verification
	for _, phaseName := range phases {
		phase, exists := fo.phaseFlow.phases[phaseName]
		if !exists {
			continue
		}
		
		// Create execution function for verification
		executeFunc := func() (interface{}, error) {
			input := PhaseInput{
				Request: request,
				Data:    results, // Pass accumulated results
			}
			
			output, err := phase.Execute(ctx, input)
			if err != nil {
				return nil, err
			}
			
			return output.Data, nil
		}
		
		// Execute with verification and retry
		stageResult, err := fo.verifier.VerifyStageWithRetry(ctx, phaseName, executeFunc)
		
		if err != nil {
			// Stage failed after all retries
			fo.logger.Error("stage failed verification", 
				"stage", phaseName,
				"attempts", stageResult.Attempts,
				"issues", len(stageResult.Issues))
			
			// Try adaptive recovery if enabled
			if fo.config.ErrorRecoveryLevel > 0 {
				adaptiveErr := fo.errorHandler.HandleError(ctx, err, map[string]interface{}{
					"stage":   phaseName,
					"result":  stageResult,
					"request": request,
				})
				
				if len(adaptiveErr.RecoveryHints) > 0 {
					fo.logger.Info("attempting adaptive recovery",
						"stage", phaseName,
						"strategies", len(adaptiveErr.RecoveryHints))
					
					_, recoveryErr := fo.errorHandler.RecoverWithLearning(ctx, adaptiveErr, stageResult)
					if recoveryErr == nil {
						// Recovery succeeded, mark as partial success
						results[phaseName] = map[string]interface{}{
							"status": "recovered",
							"output": stageResult.Output,
						}
						continue
					}
				}
			}
			
			// Stage definitively failed
			return results, fmt.Errorf("stage %s failed verification: %w", phaseName, err)
		}
		
		// Stage succeeded
		results[phaseName] = stageResult.Output
		fo.logger.Info("stage completed and verified",
			"stage", phaseName,
			"attempts", stageResult.Attempts)
		
		// Save checkpoint
		if fo.checkpoint != nil {
			fo.checkpoint.Save(ctx, fo.sessionID, len(results), phaseName, stageResult.Output)
		}
	}
	
	return results, nil
}

// createAdaptiveCondition creates a condition that learns from patterns
func (fo *FluidOrchestrator) createAdaptiveCondition(phase Phase) PhaseCondition {
	return func(ctx context.Context, previousResults map[string]interface{}) bool {
		// Basic condition - always run unless we learn otherwise
		if !fo.learningEnabled {
			return true
		}
		
		// Check if we've learned this phase should be skipped
		skipPatterns := fo.getLearnedSkipPatterns(phase.Name())
		for _, pattern := range skipPatterns {
			if fo.matchesPattern(previousResults, pattern) {
				fo.logger.Info("skipping phase based on learned pattern",
					"phase", phase.Name(),
					"pattern", pattern)
				return false
			}
		}
		
		return true
	}
}

// calculatePhasePriority determines dynamic priority
func (fo *FluidOrchestrator) calculatePhasePriority(phase Phase) float64 {
	basePriority := 1.0
	
	// Adjust based on phase characteristics
	switch phase.Name() {
	case "Planning", "Analysis":
		basePriority = 2.0 // Higher priority for initial phases
	case "Validation", "Review":
		basePriority = 0.5 // Lower priority for validation phases
	}
	
	// Adjust based on learning
	if fo.learningEnabled {
		successRate := fo.getPhaseSuccessRate(phase.Name())
		basePriority *= successRate
	}
	
	return basePriority
}

// discoverAndRegisterPhases dynamically discovers phases based on request
func (fo *FluidOrchestrator) discoverAndRegisterPhases(request string) {
	fo.logger.Info("discovering phases for request", "request", request)
	
	// Analyze request to determine needed phases
	patterns := fo.analyzeRequestPatterns(request)
	
	for _, pattern := range patterns {
		// Discover phases matching pattern
		discovered := fo.phaseFlow.DiscoverPhases(pattern)
		
		for _, phase := range discovered {
			fo.logger.Info("discovered phase", "name", phase.Name(), "pattern", pattern)
			// Phase already registered in PhaseFlow
		}
	}
}

// registerPhasePrompts creates flexible prompts for a phase
func (fo *FluidOrchestrator) registerPhasePrompts(phase Phase) {
	phaseName := phase.Name()
	
	// Register base template
	basePrompt := fmt.Sprintf("Execute %s phase with the following context:\n{{.context}}", phaseName)
	fo.promptFlow.RegisterTemplate(
		phaseName,
		basePrompt,
		WithFragments("expert_developer", "think_step_by_step"),
		WithVariables(map[string]interface{}{
			"phase": phaseName,
			"timestamp": time.Now(),
		}),
	)
	
	// Register variations based on context
	fo.promptFlow.RegisterTemplate(
		phaseName+"-detailed",
		basePrompt+"\n{{fragment:explain_reasoning}}",
		WithVariation("verbose", basePrompt+"\nProvide detailed explanation for each decision.", 
			func(ctx context.Context, data interface{}) bool {
				// Use verbose variation when needed
				return fo.config.PromptOptimization
			}),
	)
}

// createExecutionContext creates an adaptive execution context
func (fo *FluidOrchestrator) createExecutionContext(ctx context.Context, request string) context.Context {
	// Add execution metadata
	ctx = context.WithValue(ctx, "session_id", fo.sessionID)
	ctx = context.WithValue(ctx, "request", request)
	ctx = context.WithValue(ctx, "learning_enabled", fo.learningEnabled)
	
	// Add adaptive timeouts
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		// Adjust deadline based on learned patterns
		if fo.learningEnabled {
			avgDuration := fo.getAverageExecutionTime()
			if avgDuration > 0 && avgDuration < remaining {
				// Give 20% buffer
				adjustedDeadline := time.Now().Add(avgDuration * 120 / 100)
				ctx, _ = context.WithDeadline(ctx, adjustedDeadline)
			}
		}
	}
	
	return ctx
}

// watchConfiguration monitors for configuration changes
func (fo *FluidOrchestrator) watchConfiguration() {
	ticker := time.NewTicker(fo.config.ConfigWatchInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		// Check for configuration updates
		if fo.hasConfigurationChanged() {
			fo.logger.Info("configuration change detected, reloading")
			fo.reloadConfiguration()
		}
	}
}

// learnFromExecution updates learning patterns
func (fo *FluidOrchestrator) learnFromExecution(results map[string]interface{}) {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	
	// Extract execution patterns
	duration := fo.extractExecutionDuration(results)
	quality := fo.assessExecutionQuality(results)
	
	// Update learning data
	fo.updateExecutionPatterns(duration, quality, results)
}

// Helper methods for learning and adaptation

func (fo *FluidOrchestrator) getOrderedPhases() []string {
	fo.mu.RLock()
	defer fo.mu.RUnlock()
	
	// Get phases from PhaseFlow in execution order
	phases := []string{}
	for name := range fo.phaseFlow.phases {
		phases = append(phases, name)
	}
	
	// Sort by priority if available
	// For now, return a reasonable default order
	defaultOrder := []string{"Planning", "Architecture", "Writing", "Assembly", "Review", "Implementation", "Validation"}
	
	orderedPhases := []string{}
	for _, phase := range defaultOrder {
		for _, p := range phases {
			if p == phase {
				orderedPhases = append(orderedPhases, p)
				break
			}
		}
	}
	
	// Add any remaining phases not in default order
	for _, p := range phases {
		found := false
		for _, op := range orderedPhases {
			if p == op {
				found = true
				break
			}
		}
		if !found {
			orderedPhases = append(orderedPhases, p)
		}
	}
	
	return orderedPhases
}

func (fo *FluidOrchestrator) getLearnedSkipPatterns(phaseName string) []map[string]interface{} {
	// Return patterns where this phase was successfully skipped
	return []map[string]interface{}{}
}

func (fo *FluidOrchestrator) matchesPattern(data, pattern map[string]interface{}) bool {
	// Simple pattern matching for now
	for key, value := range pattern {
		if data[key] != value {
			return false
		}
	}
	return true
}

func (fo *FluidOrchestrator) getPhaseSuccessRate(phaseName string) float64 {
	// Return historical success rate
	return 0.95 // Default high success rate
}

func (fo *FluidOrchestrator) analyzeRequestPatterns(request string) []string {
	patterns := []string{}
	
	// Analyze request for patterns
	if contains(request, "code") || contains(request, "implement") {
		patterns = append(patterns, "code")
	}
	if contains(request, "document") || contains(request, "docs") {
		patterns = append(patterns, "docs")
	}
	if contains(request, "test") {
		patterns = append(patterns, "test")
	}
	
	return patterns
}

func (fo *FluidOrchestrator) getAverageExecutionTime() time.Duration {
	// Return learned average execution time
	return 30 * time.Minute // Default
}

func (fo *FluidOrchestrator) hasConfigurationChanged() bool {
	// Check if configuration has changed
	return false
}

func (fo *FluidOrchestrator) reloadConfiguration() {
	// Reload configuration dynamically
}

func (fo *FluidOrchestrator) extractExecutionDuration(results map[string]interface{}) time.Duration {
	// Extract duration from results
	return 10 * time.Minute
}

func (fo *FluidOrchestrator) assessExecutionQuality(results map[string]interface{}) float64 {
	// Assess quality of execution
	return 0.9
}

func (fo *FluidOrchestrator) updateExecutionPatterns(duration time.Duration, quality float64, results map[string]interface{}) {
	// Update learning patterns
}

// ModularPhase adapter for Phase interface
func (mp *ModularPhase) Name() string {
	return mp.name
}

func (mp *ModularPhase) EstimatedDuration() time.Duration {
	// Sum component durations
	total := time.Duration(0)
	for _, comp := range mp.components {
		total += comp.EstimatedDuration()
	}
	return total
}

func (mp *ModularPhase) ValidateInput(ctx context.Context, input PhaseInput) error {
	// Validate using first component
	if len(mp.pipeline) > 0 {
		firstComp := mp.components[mp.pipeline[0]]
		if firstComp != nil {
			compInput := ComponentInput{
				Data:    input.Data,
				State:   mp.state,
				Context: make(map[string]interface{}),
			}
			if !firstComp.CanHandle(compInput) {
				return fmt.Errorf("input not suitable for phase %s", mp.name)
			}
		}
	}
	return nil
}

func (mp *ModularPhase) ValidateOutput(ctx context.Context, output PhaseOutput) error {
	// Basic output validation
	if output.Data == nil {
		return fmt.Errorf("phase %s produced no output", mp.name)
	}
	return nil
}

func (mp *ModularPhase) CanRetry(err error) bool {
	// Modular phases can usually retry
	return true
}