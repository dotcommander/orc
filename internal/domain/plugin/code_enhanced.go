package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/agent"
	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/domain"
	"github.com/vampirenirmal/orchestrator/internal/phase/code"
)

// EnhancedCodePlugin uses V2 enhanced prompts for code generation
type EnhancedCodePlugin struct {
	*CodePlugin
	agentFactory *agent.AgentFactory
	logger       *slog.Logger
}

// NewEnhancedCodePlugin creates a code plugin with enhanced prompts
func NewEnhancedCodePlugin(domainAgent domain.Agent, storage domain.Storage, promptsDir string, aiClient agent.AIClient, logger *slog.Logger) *EnhancedCodePlugin {
	// Create agent factory with V2 prompts enabled
	factory := agent.NewAgentFactory(aiClient, promptsDir, true)
	
	return &EnhancedCodePlugin{
		CodePlugin:   NewCodePlugin(domainAgent, storage),
		agentFactory: factory,
		logger:       logger,
	}
}

// GetPhases returns enhanced phases with V2 prompts
func (p *EnhancedCodePlugin) GetPhases() []domain.Phase {
	// Create enhanced phases that use the agent factory
	enhancedPhases := []domain.Phase{
		&enhancedConversationalExplorerPhase{
			factory: p.agentFactory,
			storage: p.storage,
			logger:  p.logger,
		},
		&enhancedCodePlannerPhase{
			factory: p.agentFactory,
			storage: p.storage,
			logger:  p.logger,
		},
		&enhancedCodeImplementerPhase{
			factory: p.agentFactory,
			storage: p.storage,
			logger:  p.logger,
		},
		&enhancedCodeRefinerPhase{
			factory: p.agentFactory,
			storage: p.storage,
			logger:  p.logger,
		},
	}
	
	return enhancedPhases
}

// Enhanced conversational explorer phase
type enhancedConversationalExplorerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
	logger  *slog.Logger
}

func (p *enhancedConversationalExplorerPhase) Name() string {
	return "Enhanced Conversational Explorer"
}

func (p *enhancedConversationalExplorerPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced code analyzer agent
	analyzerAgent := p.factory.CreateCodeAgent("analyzer")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: analyzerAgent}
	
	// Use the conversational explorer with enhanced agent
	explorer := code.NewConversationalExplorer(coreAgent, p.logger)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Request:   input.Request,
		SessionID: getSessionID(input.Metadata),
	}
	
	coreOutput, err := explorer.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedConversationalExplorerPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if strings.TrimSpace(input.Request) == "" {
		return fmt.Errorf("request cannot be empty")
	}
	return nil
}

func (p *enhancedConversationalExplorerPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedConversationalExplorerPhase) EstimatedDuration() time.Duration {
	return 10 * time.Minute
}

func (p *enhancedConversationalExplorerPhase) CanRetry(err error) bool {
	return true
}

// Enhanced code planner phase
type enhancedCodePlannerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
	logger  *slog.Logger
}

func (p *enhancedCodePlannerPhase) Name() string {
	return "Enhanced Code Planning"
}

func (p *enhancedCodePlannerPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced code planner agent
	plannerAgent := p.factory.CreateCodeAgent("planner")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: plannerAgent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use a custom planner that leverages the enhanced prompts
	// For now, we'll use the incremental builder as a planner
	planner := code.NewIncrementalBuilder(coreAgent, coreStorage, p.logger)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Data:      input.Data,
		SessionID: getSessionID(input.Metadata),
	}
	
	// Override to use planning mode
	if coreInput.Data != nil {
		if reqData, ok := coreInput.Data.(map[string]interface{}); ok {
			reqData["planning_mode"] = true
		}
	}
	
	coreOutput, err := planner.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedCodePlannerPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("planner requires requirements data")
	}
	return nil
}

func (p *enhancedCodePlannerPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedCodePlannerPhase) EstimatedDuration() time.Duration {
	return 15 * time.Minute
}

func (p *enhancedCodePlannerPhase) CanRetry(err error) bool {
	return true
}

// Enhanced code implementer phase
type enhancedCodeImplementerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
	logger  *slog.Logger
}

func (p *enhancedCodeImplementerPhase) Name() string {
	return "Enhanced Code Implementation"
}

func (p *enhancedCodeImplementerPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced code implementer agent
	implementerAgent := p.factory.CreateCodeAgent("implementer")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: implementerAgent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use the incremental builder with enhanced agent
	builder := code.NewIncrementalBuilder(coreAgent, coreStorage, p.logger)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Data:      input.Data,
		SessionID: getSessionID(input.Metadata),
	}
	
	coreOutput, err := builder.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedCodeImplementerPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("implementer requires plan data")
	}
	return nil
}

func (p *enhancedCodeImplementerPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedCodeImplementerPhase) EstimatedDuration() time.Duration {
	return 30 * time.Minute
}

func (p *enhancedCodeImplementerPhase) CanRetry(err error) bool {
	return true
}

// Enhanced code refiner phase
type enhancedCodeRefinerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
	logger  *slog.Logger
}

func (p *enhancedCodeRefinerPhase) Name() string {
	return "Enhanced Code Refinement"
}

func (p *enhancedCodeRefinerPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced code reviewer agent (using analyzer for refinement)
	reviewerAgent := p.factory.CreateCodeAgent("analyzer")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: reviewerAgent}
	
	// Use the iterative refiner with enhanced agent
	refiner := code.NewIterativeRefiner(coreAgent, p.logger)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Data:      input.Data,
		SessionID: getSessionID(input.Metadata),
	}
	
	coreOutput, err := refiner.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedCodeRefinerPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("refiner requires implementation data")
	}
	return nil
}

func (p *enhancedCodeRefinerPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedCodeRefinerPhase) EstimatedDuration() time.Duration {
	return 20 * time.Minute
}

func (p *enhancedCodeRefinerPhase) CanRetry(err error) bool {
	return true
}