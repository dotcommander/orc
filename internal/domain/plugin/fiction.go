package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/agent"
	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/domain"
	"github.com/vampirenirmal/orchestrator/internal/phase/fiction"
)

// FictionPlugin implements the DomainPlugin interface for fiction generation
type FictionPlugin struct {
	agent         domain.Agent
	storage       domain.Storage
	config        DomainPluginConfig
	checkpointMgr domain.CheckpointManager
	sessionID     string
	agentFactory  *agent.AgentFactory
}

// NewFictionPlugin creates a new fiction plugin with enhanced prompts
func NewFictionPlugin(domainAgent domain.Agent, storage domain.Storage, promptsDir string, aiClient agent.AIClient) *FictionPlugin {
	// Create agent factory
	factory := agent.NewAgentFactory(aiClient, promptsDir)
	
	return &FictionPlugin{
		agent:        domainAgent,
		storage:      storage,
		config:       getDefaultFictionConfig(),
		agentFactory: factory,
	}
}

// WithCheckpointing enables checkpointing for the fiction plugin
func (p *FictionPlugin) WithCheckpointing(mgr domain.CheckpointManager, sessionID string) *FictionPlugin {
	p.checkpointMgr = mgr
	p.sessionID = sessionID
	return p
}

// Name returns the plugin name
func (p *FictionPlugin) Name() string {
	return "fiction"
}

// Description returns a human-readable description
func (p *FictionPlugin) Description() string {
	return "Professional AI novel generation with enhanced prompts: strategic planning, targeted writing, contextual editing, and polished assembly"
}

// GetPhases returns enhanced phases with professional prompts
func (p *FictionPlugin) GetPhases() []domain.Phase {
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

// GetDefaultConfig returns default configuration for fiction generation
func (p *FictionPlugin) GetDefaultConfig() DomainPluginConfig {
	return p.config
}

// ValidateRequest validates if the user request is appropriate for fiction generation
func (p *FictionPlugin) ValidateRequest(request string) error {
	request = strings.TrimSpace(strings.ToLower(request))
	
	if len(request) < 10 {
		return fmt.Errorf("request too short (minimum 10 characters)")
	}
	
	// Check for fiction-related keywords
	fictionKeywords := []string{
		"novel", "story", "book", "fiction", "tale", "narrative",
		"character", "plot", "chapter", "write", "create",
		"sci-fi", "fantasy", "mystery", "romance", "thriller",
		"drama", "adventure", "horror", "comedy",
	}
	
	for _, keyword := range fictionKeywords {
		if strings.Contains(request, keyword) {
			return nil // Valid fiction request
		}
	}
	
	// Check for anti-patterns (non-fiction requests)
	nonFictionKeywords := []string{
		"code", "function", "class", "api", "database", "server",
		"documentation", "manual", "guide", "tutorial", "readme",
		"algorithm", "data structure", "testing", "debug",
	}
	
	for _, keyword := range nonFictionKeywords {
		if strings.Contains(request, keyword) {
			return fmt.Errorf("request appears to be for code or documentation, not fiction")
		}
	}
	
	// If no clear fiction keywords, give a warning but allow
	return nil
}

// GetOutputSpec returns the expected output structure for fiction
func (p *FictionPlugin) GetOutputSpec() DomainOutputSpec {
	return DomainOutputSpec{
		PrimaryOutput: "complete_novel.md",
		SecondaryOutputs: []string{
			"systematic_plan.json",
			"novel_metadata.json",
			"generation_statistics.json",
			"final_manuscript.md",
			"chapters/",
		},
		Descriptions: map[string]string{
			"complete_novel.md":       "📖 Complete polished novel (ready to read)",
			"systematic_plan.json":    "📋 Detailed systematic plan with word budgets",
			"novel_metadata.json":     "📊 Complete novel data and statistics",
			"generation_statistics.json": "📈 Word count accuracy and quality metrics",
			"final_manuscript.md":     "✍️  Final edited manuscript",
			"chapters/":               "📚 Individual chapter files",
		},
	}
}

// GetDomainValidator returns fiction-specific validation
func (p *FictionPlugin) GetDomainValidator() domain.DomainValidator {
	return &FictionValidator{}
}

// FictionValidator provides fiction-specific validation
type FictionValidator struct{}

// ValidateRequest validates a user request for fiction
func (v *FictionValidator) ValidateRequest(request string) error {
	if len(strings.TrimSpace(request)) == 0 {
		return fmt.Errorf("fiction request cannot be empty")
	}
	
	// Check for fiction keywords to validate this is a fiction request
	fictionKeywords := []string{
		"story", "novel", "character", "plot", "fiction",
		"narrative", "dialogue", "scene", "chapter", "protagonist",
		"fantasy", "sci-fi", "romance", "mystery", "thriller",
		"drama", "adventure", "write", "book", "tale",
	}
	
	lowerRequest := strings.ToLower(request)
	for _, keyword := range fictionKeywords {
		if strings.Contains(lowerRequest, keyword) {
			return nil // Valid fiction request
		}
	}
	
	// Check for anti-patterns (non-fiction requests)
	nonFictionKeywords := []string{
		"code", "function", "class", "api", "database", "server",
		"documentation", "manual", "guide", "tutorial", "readme",
		"algorithm", "data structure", "testing", "debug",
	}
	
	for _, keyword := range nonFictionKeywords {
		if strings.Contains(lowerRequest, keyword) {
			return fmt.Errorf("request appears to be for code or documentation, not fiction")
		}
	}
	
	// If no clear fiction keywords, give a warning but allow
	return nil
}

// ValidatePhaseTransition validates data between fiction phases
func (v *FictionValidator) ValidatePhaseTransition(from, to string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("phase transition data cannot be nil")
	}
	
	// Validate specific phase transitions
	switch from + "->" + to {
	case "Planning->Architecture":
		// Validate plan data contains required fields
		if planData, ok := data.(map[string]interface{}); ok {
			if _, hasTitle := planData["title"]; !hasTitle {
				return fmt.Errorf("planning phase must produce a title")
			}
			if _, hasPlot := planData["plot"]; !hasPlot {
				return fmt.Errorf("planning phase must produce a plot outline")
			}
		}
	case "Architecture->Writing":
		// Validate architecture data contains characters and settings
		if archData, ok := data.(map[string]interface{}); ok {
			if _, hasCharacters := archData["characters"]; !hasCharacters {
				return fmt.Errorf("architecture phase must define characters")
			}
			if _, hasSettings := archData["settings"]; !hasSettings {
				return fmt.Errorf("architecture phase must define settings")
			}
		}
	case "Writing->Assembly":
		// Validate writing data contains scenes
		if writeData, ok := data.(map[string]interface{}); ok {
			if scenes, hasScenes := writeData["scenes"]; hasScenes {
				if sceneList, ok := scenes.([]interface{}); ok {
					if len(sceneList) == 0 {
						return fmt.Errorf("writing phase must produce at least one scene")
					}
				}
			}
		}
	}
	
	return nil
}

// getDefaultFictionConfig returns the default configuration for fiction generation
func getDefaultFictionConfig() DomainPluginConfig {
	return DomainPluginConfig{
		Prompts: map[string]string{
			"planning":     filepath.Join(getPromptsDir(), "orchestrator.txt"),
			"architecture": filepath.Join(getPromptsDir(), "architect.txt"),
			"writing":      filepath.Join(getPromptsDir(), "writer.txt"),
			"critique":     filepath.Join(getPromptsDir(), "critic.txt"),
			"editor":       filepath.Join(getPromptsDir(), "editor.txt"),
		},
		Limits: DomainPluginLimits{
			MaxConcurrentPhases: 4,
			PhaseTimeouts: map[string]time.Duration{
				"planning":     20 * time.Minute,
				"architecture": 10 * time.Minute,
				"writing":      30 * time.Minute,
				"editing":      15 * time.Minute,
				"assembly":     2 * time.Minute,
			},
			MaxRetries:   3,
			TotalTimeout: 60 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"supports_resume":     true,
			"supports_streaming":  false,
			"requires_creativity": true,
			"output_format":       "markdown",
			"uses_enhanced_prompts": true,
		},
	}
}

// getPromptsDir returns the XDG-compliant prompts directory
func getPromptsDir() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "orchestrator", "prompts")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "orchestrator", "prompts")
}

// Enhanced phase implementations
type enhancedPlannerPhase struct {
	factory *agent.AgentFactory
	storage domain.Storage
}

func (p *enhancedPlannerPhase) Name() string {
	return "Strategic Planning"
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
	return "Targeted Writing"
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
	return "Contextual Editing"
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