package core

import (
	"context"
	"log/slog"
	
	"github.com/google/uuid"
)

// OrchestratorConfig consolidates all configuration options
type OrchestratorConfig struct {
	CheckpointingEnabled bool
	MaxRetries          int
	PerformanceEnabled  bool
	MaxConcurrency      int
}

// DefaultConfig returns sensible defaults
func DefaultConfig() OrchestratorConfig {
	return OrchestratorConfig{
		CheckpointingEnabled: true,
		MaxRetries:          3,
		PerformanceEnabled:  true,
		MaxConcurrency:      0, // Auto-detect
	}
}

type Orchestrator struct {
	phases     []Phase
	storage    Storage
	logger     *slog.Logger
	checkpoint *CheckpointManager
	sessionID  string
	engine     *ExecutionEngine
}

type Option func(*Orchestrator)

func WithConfig(config OrchestratorConfig) Option {
	return func(o *Orchestrator) {
		if config.CheckpointingEnabled {
			o.checkpoint = NewCheckpointManager(o.storage)
		}
		
		o.engine = NewExecutionEngine(o.logger, config.MaxRetries)
		
		if config.PerformanceEnabled {
			if config.MaxConcurrency > 0 {
				o.engine.WithCustomConcurrency(config.MaxConcurrency)
			} else {
				o.engine.WithPerformanceOptimization(true)
			}
		}
	}
}

func New(phases []Phase, storage Storage, opts ...Option) *Orchestrator {
	o := &Orchestrator{
		phases:    phases,
		storage:   storage,
		logger:    slog.Default(),
		sessionID: uuid.New().String(),
	}
	
	// Apply default configuration if no options provided
	if len(opts) == 0 {
		WithConfig(DefaultConfig())(o)
	}
	
	for _, opt := range opts {
		opt(o)
	}
	
	return o
}

func (o *Orchestrator) WithLogger(logger *slog.Logger) *Orchestrator {
	o.logger = logger
	// Update engine with new logger if it exists
	if o.engine != nil {
		o.engine = NewExecutionEngine(logger, 3) // Use default retry count
	}
	return o
}

func (o *Orchestrator) WithSessionID(sessionID string) *Orchestrator {
	o.sessionID = sessionID
	return o
}

func (o *Orchestrator) SessionID() string {
	return o.sessionID
}

func (o *Orchestrator) Run(ctx context.Context, request string) error {
	return o.RunWithResume(ctx, request, 0)
}

// RunOptimized executes phases with performance optimizations enabled
func (o *Orchestrator) RunOptimized(ctx context.Context, request string) error {
	return o.engine.ExecutePhases(ctx, o.phases, request, o.sessionID, 0, o.checkpoint)
}

func (o *Orchestrator) RunWithResume(ctx context.Context, request string, startPhase int) error {
	return o.engine.ExecutePhases(ctx, o.phases, request, o.sessionID, startPhase, o.checkpoint)
}

// GetValidationReport returns the validation report for the session
func (o *Orchestrator) GetValidationReport() string {
	if o.engine == nil {
		return "No execution engine available"
	}
	return o.engine.GetValidationReport()
}