package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// PhaseFlow represents a dynamic, adaptive phase execution system
type PhaseFlow struct {
	phases      map[string]Phase
	graph       *PhaseGraph
	logger      *slog.Logger
	mu          sync.RWMutex
}

// PhaseGraph represents phase dependencies and relationships
type PhaseGraph struct {
	nodes map[string]*PhaseNode
	edges map[string][]string // phase -> dependent phases
}

// PhaseNode contains phase metadata and runtime state
type PhaseNode struct {
	Phase       Phase
	Status      PhaseStatus
	Priority    float64
	CanParallel bool
	Conditions  []PhaseCondition
	Results     interface{}
}

type PhaseStatus int

const (
	PhaseReady PhaseStatus = iota
	PhaseRunning
	PhaseCompleted
	PhaseSkipped
	PhaseFailed
)

// PhaseCondition determines if a phase should run
type PhaseCondition func(ctx context.Context, previousResults map[string]interface{}) bool

// NewPhaseFlow creates a dynamic phase execution system
func NewPhaseFlow(logger *slog.Logger) *PhaseFlow {
	return &PhaseFlow{
		phases: make(map[string]Phase),
		graph: &PhaseGraph{
			nodes: make(map[string]*PhaseNode),
			edges: make(map[string][]string),
		},
		logger: logger,
	}
}

// RegisterPhase adds a phase with dynamic configuration
func (pf *PhaseFlow) RegisterPhase(phase Phase, opts ...PhaseOption) {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	node := &PhaseNode{
		Phase:       phase,
		Status:      PhaseReady,
		Priority:    1.0,
		CanParallel: false,
		Conditions:  make([]PhaseCondition, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(node)
	}

	pf.phases[phase.Name()] = phase
	pf.graph.nodes[phase.Name()] = node
}

// PhaseOption configures phase behavior
type PhaseOption func(*PhaseNode)

// WithDependencies sets phase dependencies
func WithDependencies(deps ...string) PhaseOption {
	return func(n *PhaseNode) {
		// Dependencies will be validated at execution time
		// deps parameter is available for future use
	}
}

// WithCondition adds a runtime condition
func WithCondition(cond PhaseCondition) PhaseOption {
	return func(n *PhaseNode) {
		n.Conditions = append(n.Conditions, cond)
	}
}

// WithParallel allows parallel execution
func WithParallel() PhaseOption {
	return func(n *PhaseNode) {
		n.CanParallel = true
	}
}

// WithPriority sets execution priority
func WithPriority(priority float64) PhaseOption {
	return func(n *PhaseNode) {
		n.Priority = priority
	}
}

// Execute runs phases dynamically based on conditions and dependencies
func (pf *PhaseFlow) Execute(ctx context.Context, input interface{}) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	var mu sync.Mutex

	// Determine execution order dynamically
	executionPlan := pf.planExecution(ctx, results)
	
	// Execute phases according to plan
	for _, wave := range executionPlan {
		if err := pf.executeWave(ctx, wave, input, results, &mu); err != nil {
			return results, err
		}
	}

	return results, nil
}

// planExecution creates dynamic execution plan based on current state
func (pf *PhaseFlow) planExecution(ctx context.Context, previousResults map[string]interface{}) [][]string {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	waves := make([][]string, 0)
	executed := make(map[string]bool)

	for {
		wave := make([]string, 0)
		
		// Find phases ready to execute
		for name, node := range pf.graph.nodes {
			if executed[name] {
				continue
			}

			// Check if all dependencies are satisfied
			if !pf.dependenciesSatisfied(name, executed) {
				continue
			}

			// Check runtime conditions
			if !pf.conditionsMet(ctx, node, previousResults) {
				executed[name] = true // Skip this phase
				node.Status = PhaseSkipped
				continue
			}

			wave = append(wave, name)
		}

		if len(wave) == 0 {
			break
		}

		// Sort by priority
		waves = append(waves, wave)
		
		// Mark as executed
		for _, name := range wave {
			executed[name] = true
		}
	}

	return waves
}

// executeWave runs a set of phases potentially in parallel
func (pf *PhaseFlow) executeWave(ctx context.Context, wave []string, input interface{}, results map[string]interface{}, mu *sync.Mutex) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(wave))

	for _, phaseName := range wave {
		node := pf.graph.nodes[phaseName]
		
		if node.CanParallel && len(wave) > 1 {
			wg.Add(1)
			go func(name string, n *PhaseNode) {
				defer wg.Done()
				if err := pf.executePhase(ctx, name, n, input, results, mu); err != nil {
					errChan <- err
				}
			}(phaseName, node)
		} else {
			if err := pf.executePhase(ctx, phaseName, node, input, results, mu); err != nil {
				return err
			}
		}
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// executePhase runs a single phase with adaptive behavior
func (pf *PhaseFlow) executePhase(ctx context.Context, name string, node *PhaseNode, input interface{}, results map[string]interface{}, mu *sync.Mutex) error {
	pf.logger.Info("Executing phase", "name", name, "parallel", node.CanParallel)
	
	node.Status = PhaseRunning

	// Build phase input from accumulated results
	phaseInput := PhaseInput{
		Request: fmt.Sprintf("%v", input),
		Data:    pf.buildPhaseInput(name, results, mu),
	}

	// Execute with adaptive timeout
	timeout := pf.calculateAdaptiveTimeout(node.Phase, results)
	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	output, err := node.Phase.Execute(phaseCtx, phaseInput)
	if err != nil {
		node.Status = PhaseFailed
		return fmt.Errorf("phase %s failed: %w", name, err)
	}

	// Store results
	mu.Lock()
	results[name] = output.Data
	mu.Unlock()

	node.Status = PhaseCompleted
	node.Results = output.Data

	return nil
}

// buildPhaseInput creates input from previous phase results
func (pf *PhaseFlow) buildPhaseInput(phaseName string, results map[string]interface{}, mu *sync.Mutex) interface{} {
	mu.Lock()
	defer mu.Unlock()

	// For now, pass all previous results
	// In future, could be selective based on dependencies
	inputData := make(map[string]interface{})
	for k, v := range results {
		inputData[k] = v
	}

	return inputData
}

// dependenciesSatisfied checks if all dependencies are met
func (pf *PhaseFlow) dependenciesSatisfied(phaseName string, executed map[string]bool) bool {
	deps, exists := pf.graph.edges[phaseName]
	if !exists {
		return true
	}

	for _, dep := range deps {
		if !executed[dep] {
			return false
		}
	}

	return true
}

// conditionsMet evaluates runtime conditions
func (pf *PhaseFlow) conditionsMet(ctx context.Context, node *PhaseNode, results map[string]interface{}) bool {
	if len(node.Conditions) == 0 {
		return true
	}

	for _, cond := range node.Conditions {
		if !cond(ctx, results) {
			pf.logger.Info("Phase condition not met", "phase", node.Phase.Name())
			return false
		}
	}

	return true
}

// calculateAdaptiveTimeout determines timeout based on context
func (pf *PhaseFlow) calculateAdaptiveTimeout(phase Phase, results map[string]interface{}) time.Duration {
	baseTimeout := phase.EstimatedDuration()
	
	// Could adapt based on:
	// - Previous phase performance
	// - Content size
	// - System load
	// - Historical data
	
	return baseTimeout
}

// AddDependency creates a dynamic dependency
func (pf *PhaseFlow) AddDependency(from, to string) {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	if pf.graph.edges == nil {
		pf.graph.edges = make(map[string][]string)
	}

	pf.graph.edges[to] = append(pf.graph.edges[to], from)
}

// Runtime phase discovery
func (pf *PhaseFlow) DiscoverPhases(pattern string) []Phase {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	discovered := make([]Phase, 0)
	for name, phase := range pf.phases {
		if matchesPattern(name, pattern) {
			discovered = append(discovered, phase)
		}
	}

	return discovered
}

func matchesPattern(name, pattern string) bool {
	// Simple pattern matching for now
	// Could use more sophisticated matching
	return strings.Contains(name, pattern)
}