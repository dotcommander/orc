package core

import (
	"log/slog"

	"github.com/dotcommander/orc/internal/config"
)

// OrchestratorFactory creates the appropriate orchestrator based on configuration
type OrchestratorFactory struct {
	logger *slog.Logger
}

// NewOrchestratorFactory creates a new orchestrator factory
func NewOrchestratorFactory(logger *slog.Logger) *OrchestratorFactory {
	return &OrchestratorFactory{
		logger: logger.With("component", "orchestrator_factory"),
	}
}

// CreateOrchestrator creates a unified orchestrator that automatically adapts to requests
func (of *OrchestratorFactory) CreateOrchestrator(
	phases []Phase,
	storage Storage,
	agent Agent,
	cfg *config.Config,
) *UnifiedOrchestrator {
	of.logger.Info("Creating unified orchestrator")
	
	// Always create a unified orchestrator in automatic mode
	unified := NewUnifiedOrchestrator(phases, storage, of.logger, agent, cfg)
	
	// Always use UnifiedMode for automatic adaptation
	unified.executionMode = UnifiedMode
	
	of.logger.Info("Unified orchestrator created", 
		"mode", "automatic",
		"features", "adaptive, quality-focused, goal-aware")
	
	return unified
}

