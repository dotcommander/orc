package plugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/agent"
	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/domain"
	"github.com/vampirenirmal/orchestrator/internal/phase/fiction"
)

// EnhancedFictionPlugin uses V2 enhanced prompts
type EnhancedFictionPlugin struct {
	*FictionPlugin
	agentFactory *agent.AgentFactory
}

// NewEnhancedFictionPlugin creates a fiction plugin with enhanced prompts
func NewEnhancedFictionPlugin(domainAgent domain.Agent, storage domain.Storage, promptsDir string, aiClient agent.AIClient) *EnhancedFictionPlugin {
	// Create agent factory with V2 prompts enabled
	factory := agent.NewAgentFactory(aiClient, promptsDir, true)
	
	return &EnhancedFictionPlugin{
		FictionPlugin: NewFictionPlugin(domainAgent, storage),
		agentFactory:  factory,
	}
}

// GetPhases returns enhanced phases with V2 prompts
func (p *EnhancedFictionPlugin) GetPhases() []domain.Phase {
	// Create enhanced phases that use the agent factory
	enhancedPhases := []domain.Phase{
		&enhancedPlannerPhase{
			factory: p.agentFactory,
			storage: p.storage,
		},
		&enhancedWriterPhase{
			factory: p.agentFactory,
			storage: p.storage,
		},
		&enhancedEditorPhase{
			factory: p.agentFactory,
			storage: p.storage,
		},
		// Assembler doesn't need AI, so use the standard one
		&coreToDomainPhaseAdapter{
			phase: fiction.NewSystematicAssembler(&domainToCoreStorageAdapter{storage: p.storage}),
		},
	}
	
	return enhancedPhases
}

// Enhanced phase implementations
type enhancedPlannerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
}

func (p *enhancedPlannerPhase) Name() string {
	return "Enhanced Planning"
}

func (p *enhancedPlannerPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced planning agent
	plannerAgent := p.factory.CreateFictionAgent("planning")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: plannerAgent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use the systematic planner with enhanced agent
	planner := fiction.NewSystematicPlanner(coreAgent, coreStorage)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Request:   input.Request,
		SessionID: getSessionID(input.Metadata),
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

func (p *enhancedPlannerPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if strings.TrimSpace(input.Request) == "" {
		return fmt.Errorf("request cannot be empty")
	}
	return nil
}

func (p *enhancedPlannerPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedPlannerPhase) EstimatedDuration() time.Duration {
	return 20 * time.Minute
}

func (p *enhancedPlannerPhase) CanRetry(err error) bool {
	return true
}

// Enhanced writer phase
type enhancedWriterPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
}

func (p *enhancedWriterPhase) Name() string {
	return "Enhanced Writing"
}

func (p *enhancedWriterPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced writing agent
	writerAgent := p.factory.CreateFictionAgent("writer")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: writerAgent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use the targeted writer with enhanced agent
	writer := fiction.NewTargetedWriter(coreAgent, coreStorage)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Data:      input.Data,
		SessionID: getSessionID(input.Metadata),
	}
	
	coreOutput, err := writer.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedWriterPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("writer requires plan data")
	}
	return nil
}

func (p *enhancedWriterPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedWriterPhase) EstimatedDuration() time.Duration {
	return 30 * time.Minute
}

func (p *enhancedWriterPhase) CanRetry(err error) bool {
	return true
}

// Enhanced editor phase
type enhancedEditorPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
}

func (p *enhancedEditorPhase) Name() string {
	return "Enhanced Editing"
}

func (p *enhancedEditorPhase) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	// Create enhanced editing agent
	editorAgent := p.factory.CreateFictionAgent("editor")
	
	// Convert to core agent adapter
	coreAgent := &agentToCoreAdapter{agent: editorAgent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use the contextual editor with enhanced agent
	editor := fiction.NewContextualEditor(coreAgent, coreStorage)
	
	// Convert input and execute
	coreInput := core.PhaseInput{
		Data:      input.Data,
		SessionID: getSessionID(input.Metadata),
	}
	
	coreOutput, err := editor.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{Error: err}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Metadata: input.Metadata,
	}, nil
}

func (p *enhancedEditorPhase) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("editor requires written content")
	}
	return nil
}

func (p *enhancedEditorPhase) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	return nil
}

func (p *enhancedEditorPhase) EstimatedDuration() time.Duration {
	return 15 * time.Minute
}

func (p *enhancedEditorPhase) CanRetry(err error) bool {
	return true
}

// Adapter to convert agent.Agent to core.Agent
type agentToCoreAdapter struct {
	agent *agent.Agent
}

func (a *agentToCoreAdapter) Execute(ctx context.Context, prompt string, input interface{}) (string, error) {
	return a.agent.Execute(ctx, prompt, input)
}

func (a *agentToCoreAdapter) ExecuteJSON(ctx context.Context, prompt string, input interface{}) (string, error) {
	return a.agent.ExecuteJSON(ctx, prompt, input)
}

// Helper to extract session ID from metadata
func getSessionID(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	if sessionID, ok := metadata["session_id"].(string); ok {
		return sessionID
	}
	return ""
}