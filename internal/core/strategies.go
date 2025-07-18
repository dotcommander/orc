package core

import (
	"context"
	"log/slog"
)

// Strategy defines an approach to meet unmet goals
type Strategy interface {
	// Name returns the strategy identifier
	Name() string
	
	// CanHandle checks if this strategy can handle the given goals
	CanHandle(goals []*Goal) bool
	
	// Execute applies the strategy to achieve the goals
	Execute(ctx context.Context, input interface{}, goals []*Goal) (interface{}, error)
	
	// EstimateEffectiveness returns a score 0-1 for how well this strategy fits
	EstimateEffectiveness(goals []*Goal) float64
}

// StrategyManager manages available strategies and selects optimal ones
type StrategyManager struct {
	strategies map[string]Strategy
	agent      Agent
	storage    Storage
	logger     *slog.Logger
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager(agent Agent, storage Storage, logger *slog.Logger) *StrategyManager {
	sm := &StrategyManager{
		strategies: make(map[string]Strategy),
		agent:      agent,
		storage:    storage,
		logger:     logger,
	}
	
	// Register default strategies
	sm.Register(NewExpansionStrategy(agent, logger))
	sm.Register(NewAdditionStrategy(agent, logger))
	sm.Register(NewRegenerationStrategy(agent, storage, logger))
	sm.Register(NewQualityEnhancementStrategy(agent, logger))
	
	return sm
}

// Register adds a strategy to the manager
func (sm *StrategyManager) Register(strategy Strategy) {
	sm.strategies[strategy.Name()] = strategy
}

// SelectOptimal chooses the best strategy for the given goals
func (sm *StrategyManager) SelectOptimal(goals []*Goal) Strategy {
	var bestStrategy Strategy
	bestScore := 0.0
	
	for _, strategy := range sm.strategies {
		if strategy.CanHandle(goals) {
			score := strategy.EstimateEffectiveness(goals)
			if score > bestScore {
				bestScore = score
				bestStrategy = strategy
			}
		}
	}
	
	if bestStrategy == nil {
		sm.logger.Warn("No suitable strategy found for goals")
		return nil
	}
	
	sm.logger.Info("Selected strategy", 
		"name", bestStrategy.Name(),
		"effectiveness", bestScore,
		"goals", len(goals))
	
	return bestStrategy
}
