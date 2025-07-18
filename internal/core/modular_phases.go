package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ModularPhase breaks down monolithic phases into composable components
type ModularPhase struct {
	name       string
	components map[string]PhaseComponent
	pipeline   []string
	router     *ComponentRouter
	state      *PhaseState
	logger     Logger
	mu         sync.RWMutex
}

// PhaseComponent is a small, focused unit of phase functionality
type PhaseComponent interface {
	Name() string
	Execute(ctx context.Context, input ComponentInput) (ComponentOutput, error)
	CanHandle(input ComponentInput) bool
	EstimatedDuration() time.Duration
}

// ComponentInput provides data to components
type ComponentInput struct {
	Data     interface{}
	State    *PhaseState
	Context  map[string]interface{}
}

// ComponentOutput contains component results
type ComponentOutput struct {
	Data     interface{}
	State    StateUpdates
	Next     []string // Suggested next components
	Metadata map[string]interface{}
}

// StateUpdates tracks state changes
type StateUpdates map[string]interface{}

// PhaseState maintains shared state across components
type PhaseState struct {
	data   map[string]interface{}
	mu     sync.RWMutex
}

// ComponentRouter intelligently routes between components
type ComponentRouter struct {
	rules      []RoutingRule
	conditions map[string]RoutingCondition
	learning   *RoutingLearner
}

// RoutingRule defines component routing logic
type RoutingRule struct {
	From      string
	To        string
	Condition RoutingCondition
	Priority  int
}

// RoutingCondition determines if routing should occur
type RoutingCondition func(state *PhaseState, output ComponentOutput) bool

// RoutingLearner learns optimal routing patterns
type RoutingLearner struct {
	history  []RoutingDecision
	patterns map[string]*RoutingPattern
	mu       sync.RWMutex
}

// RoutingDecision records routing choices
type RoutingDecision struct {
	From      string
	To        string
	Success   bool
	Duration  time.Duration
	Quality   float64
	Timestamp time.Time
}

// RoutingPattern represents learned routing patterns
type RoutingPattern struct {
	Pattern      string
	SuccessRate  float64
	AverageTime  time.Duration
	OptimalPaths []string
}

// NewModularPhase creates a phase from components
func NewModularPhase(name string, logger Logger) *ModularPhase {
	return &ModularPhase{
		name:       name,
		components: make(map[string]PhaseComponent),
		pipeline:   make([]string, 0),
		router:     NewComponentRouter(),
		state:      NewPhaseState(),
		logger:     logger,
	}
}

// RegisterComponent adds a component to the phase
func (mp *ModularPhase) RegisterComponent(component PhaseComponent) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.components[component.Name()] = component
}

// SetPipeline defines default component execution order
func (mp *ModularPhase) SetPipeline(componentNames ...string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Validate components exist
	for _, name := range componentNames {
		if _, exists := mp.components[name]; !exists {
			return fmt.Errorf("component %s not found", name)
		}
	}

	mp.pipeline = componentNames
	return nil
}

// Execute runs the modular phase
func (mp *ModularPhase) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	mp.logger.Info("Executing modular phase", "phase", mp.name)

	// Initialize component input
	componentInput := ComponentInput{
		Data:    input,
		State:   mp.state,
		Context: make(map[string]interface{}),
	}

	// Determine execution path
	executionPath := mp.determineExecutionPath(componentInput)

	// Execute components
	var lastOutput ComponentOutput
	for _, componentName := range executionPath {
		component, exists := mp.components[componentName]
		if !exists {
			return nil, fmt.Errorf("component %s not found", componentName)
		}

		// Check if component can handle input
		if !component.CanHandle(componentInput) {
			mp.logger.Info("Skipping component", "component", componentName)
			continue
		}

		// Execute component
		output, err := mp.executeComponent(ctx, component, componentInput)
		if err != nil {
			return nil, fmt.Errorf("component %s failed: %w", componentName, err)
		}

		// Update state
		mp.state.Update(output.State)

		// Prepare input for next component
		componentInput.Data = output.Data
		lastOutput = output

		// Dynamic routing based on output
		if len(output.Next) > 0 {
			// Modify execution path based on component suggestion
			executionPath = mp.router.Route(componentName, output, executionPath)
		}
	}

	return lastOutput.Data, nil
}

// determineExecutionPath creates dynamic execution path
func (mp *ModularPhase) determineExecutionPath(input ComponentInput) []string {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Start with default pipeline
	path := make([]string, len(mp.pipeline))
	copy(path, mp.pipeline)

	// Apply routing rules
	path = mp.router.OptimizePath(path, mp.state)

	// Apply learning if available
	if learned := mp.router.learning.SuggestPath(mp.name, input); len(learned) > 0 {
		mp.logger.Info("Using learned path", "path", learned)
		return learned
	}

	return path
}

// executeComponent runs a single component with monitoring
func (mp *ModularPhase) executeComponent(ctx context.Context, component PhaseComponent, input ComponentInput) (ComponentOutput, error) {
	// Create timeout context
	timeout := component.EstimatedDuration()
	componentCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Monitor execution
	start := time.Now()
	
	// Execute
	output, err := component.Execute(componentCtx, input)
	
	duration := time.Since(start)

	// Record for learning
	mp.recordExecution(component.Name(), duration, err == nil)

	if err != nil {
		return ComponentOutput{}, err
	}

	return output, nil
}

// recordExecution tracks component performance
func (mp *ModularPhase) recordExecution(componentName string, duration time.Duration, success bool) {
	// Record for learning and optimization
	decision := RoutingDecision{
		From:      mp.name,
		To:        componentName,
		Success:   success,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	mp.router.learning.Record(decision)
}

// Example Components

// DataValidationComponent validates input data
type DataValidationComponent struct {
	validators []DataValidator
}

func (dvc *DataValidationComponent) Name() string { return "DataValidation" }

func (dvc *DataValidationComponent) Execute(ctx context.Context, input ComponentInput) (ComponentOutput, error) {
	// Validate data
	for _, validator := range dvc.validators {
		if err := validator.Validate(input.Data); err != nil {
			return ComponentOutput{}, fmt.Errorf("validation failed: %w", err)
		}
	}

	return ComponentOutput{
		Data: input.Data,
		State: StateUpdates{
			"validated": true,
		},
	}, nil
}

func (dvc *DataValidationComponent) CanHandle(input ComponentInput) bool {
	return input.Data != nil
}

func (dvc *DataValidationComponent) EstimatedDuration() time.Duration {
	return 1 * time.Second
}

// TransformationComponent transforms data
type TransformationComponent struct {
	transformer func(interface{}) (interface{}, error)
}

func (tc *TransformationComponent) Name() string { return "Transformation" }

func (tc *TransformationComponent) Execute(ctx context.Context, input ComponentInput) (ComponentOutput, error) {
	transformed, err := tc.transformer(input.Data)
	if err != nil {
		return ComponentOutput{}, err
	}

	return ComponentOutput{
		Data: transformed,
		State: StateUpdates{
			"transformed": true,
		},
	}, nil
}

func (tc *TransformationComponent) CanHandle(input ComponentInput) bool {
	return true
}

func (tc *TransformationComponent) EstimatedDuration() time.Duration {
	return 2 * time.Second
}

// AIProcessingComponent handles AI operations
type AIProcessingComponent struct {
	agent  Agent
	prompt string
}

func (apc *AIProcessingComponent) Name() string { return "AIProcessing" }

func (apc *AIProcessingComponent) Execute(ctx context.Context, input ComponentInput) (ComponentOutput, error) {
	// Build prompt with context
	fullPrompt := fmt.Sprintf("%s\n\nInput: %v", apc.prompt, input.Data)

	// Execute AI call
	result, err := apc.agent.Execute(ctx, fullPrompt, input.Data)
	if err != nil {
		return ComponentOutput{}, err
	}

	return ComponentOutput{
		Data: result,
		State: StateUpdates{
			"ai_processed": true,
		},
		Next: []string{"PostProcessing"}, // Suggest next component
	}, nil
}

func (apc *AIProcessingComponent) CanHandle(input ComponentInput) bool {
	// Check if input is suitable for AI processing
	validated, _ := input.State.Get("validated").(bool)
	return validated
}

func (apc *AIProcessingComponent) EstimatedDuration() time.Duration {
	return 30 * time.Second
}

// ComponentRouter implementation

func NewComponentRouter() *ComponentRouter {
	return &ComponentRouter{
		rules:      make([]RoutingRule, 0),
		conditions: make(map[string]RoutingCondition),
		learning:   NewRoutingLearner(),
	}
}

// AddRule adds a routing rule
func (cr *ComponentRouter) AddRule(from, to string, condition RoutingCondition, priority int) {
	rule := RoutingRule{
		From:      from,
		To:        to,
		Condition: condition,
		Priority:  priority,
	}

	cr.rules = append(cr.rules, rule)
}

// Route determines next components based on output
func (cr *ComponentRouter) Route(current string, output ComponentOutput, remainingPath []string) []string {
	// Apply routing rules
	for _, rule := range cr.rules {
		if rule.From == current && rule.Condition(nil, output) {
			// Insert component into path
			return cr.insertComponent(rule.To, remainingPath)
		}
	}

	// Use suggested next components
	if len(output.Next) > 0 {
		return append(output.Next, remainingPath...)
	}

	return remainingPath
}

// OptimizePath optimizes execution path based on state
func (cr *ComponentRouter) OptimizePath(path []string, state *PhaseState) []string {
	// Apply learned optimizations
	if optimized := cr.learning.OptimizePath(path); len(optimized) > 0 {
		return optimized
	}

	return path
}

func (cr *ComponentRouter) insertComponent(component string, path []string) []string {
	// Insert component at beginning of remaining path
	return append([]string{component}, path...)
}

// RoutingLearner implementation

func NewRoutingLearner() *RoutingLearner {
	return &RoutingLearner{
		history:  make([]RoutingDecision, 0),
		patterns: make(map[string]*RoutingPattern),
	}
}

func (rl *RoutingLearner) Record(decision RoutingDecision) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.history = append(rl.history, decision)
	
	// Update patterns
	rl.updatePatterns(decision)
}

func (rl *RoutingLearner) SuggestPath(phase string, input ComponentInput) []string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Find best performing path
	if pattern, exists := rl.patterns[phase]; exists && pattern.SuccessRate > 0.8 {
		return pattern.OptimalPaths
	}

	return nil
}

func (rl *RoutingLearner) OptimizePath(path []string) []string {
	// Apply learned optimizations
	return path
}

func (rl *RoutingLearner) updatePatterns(decision RoutingDecision) {
	// Update routing patterns based on success
	key := fmt.Sprintf("%s->%s", decision.From, decision.To)
	
	pattern, exists := rl.patterns[key]
	if !exists {
		pattern = &RoutingPattern{
			Pattern:      key,
			OptimalPaths: make([]string, 0),
		}
		rl.patterns[key] = pattern
	}

	// Update metrics
	if decision.Success {
		pattern.SuccessRate = (pattern.SuccessRate + 1.0) / 2.0
		pattern.AverageTime = (pattern.AverageTime + decision.Duration) / 2
	}
}

// PhaseState implementation

func NewPhaseState() *PhaseState {
	return &PhaseState{
		data: make(map[string]interface{}),
	}
}

func (ps *PhaseState) Get(key string) interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.data[key]
}

func (ps *PhaseState) Set(key string, value interface{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data[key] = value
}

func (ps *PhaseState) Update(updates StateUpdates) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	for k, v := range updates {
		ps.data[k] = v
	}
}

// Helper interfaces

type DataValidator interface {
	Validate(data interface{}) error
}

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}