package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotcommander/orc/internal/core"
	"github.com/dotcommander/orc/internal/domain"
	"github.com/dotcommander/orc/internal/phase/fiction"
)

// Plugin implements the domain plugin interface for fiction generation
type Plugin struct {
	agentFactory *core.AgentFactory
	storage      core.Storage
	logger       *slog.Logger
}

// NewPlugin creates a new fiction plugin instance
func NewPlugin(agentFactory *core.AgentFactory, storage core.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		agentFactory: agentFactory,
		storage:      storage,
		logger:       logger.With("plugin", "fiction"),
	}
}

// GetInfo returns plugin metadata
func (p *Plugin) GetInfo() domain.PluginInfo {
	return domain.PluginInfo{
		Name:        "fiction",
		Version:     "1.0.0",
		Description: "AI-powered fiction and novel generation",
		Author:      "Orc Team",
		Domains:     []string{"fiction"},
		MinOrcVersion: "0.1.0",
	}
}

// CreatePhases returns the phases for fiction generation
func (p *Plugin) CreatePhases() ([]core.Phase, error) {
	// Create agents for each phase
	plannerAgent := p.agentFactory.CreateAgent("planner", "prompts/architect.txt")
	writerAgent := p.agentFactory.CreateAgent("writer", "prompts/writer.txt")
	criticAgent := p.agentFactory.CreateAgent("critic", "prompts/critic.txt")

	// Return phase pipeline
	return []core.Phase{
		fiction.NewSystematicPlanner(plannerAgent, p.logger),
		fiction.NewTargetedWriter(writerAgent, p.storage, p.logger),
		fiction.NewContextualEditor(criticAgent, p.storage, p.logger),
		fiction.NewSystematicAssembler(p.storage, p.logger),
	}, nil
}

// ValidateRequest checks if the request is suitable for fiction generation
func (p *Plugin) ValidateRequest(request string) error {
	if len(request) < 10 {
		return fmt.Errorf("request too short for meaningful fiction generation")
	}
	return nil
}

// GetOutputSpec returns expected outputs for fiction
func (p *Plugin) GetOutputSpec() domain.OutputSpec {
	return domain.OutputSpec{
		PrimaryOutput: "manuscript.md",
		SecondaryOutputs: []string{
			"outline.json",
			"characters.json",
			"chapters/",
		},
		FilePatterns: map[string]string{
			"chapters": "chapters/chapter_*.md",
			"metadata": "*.json",
		},
	}
}

// GetPhaseTimeouts returns recommended timeouts
func (p *Plugin) GetPhaseTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"SystematicPlanner":   5 * time.Minute,
		"TargetedWriter":     30 * time.Minute,
		"ContextualEditor":   20 * time.Minute,
		"SystematicAssembler": 2 * time.Minute,
	}
}

// GetRequiredConfig returns required configuration keys
func (p *Plugin) GetRequiredConfig() []string {
	return []string{
		"ai.model",
		"ai.api_key",
	}
}

// GetDefaultConfig returns default configuration for fiction
func (p *Plugin) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"fiction": map[string]interface{}{
			"chapter_word_target": 3000,
			"quality_threshold":   0.8,
			"style_consistency":   true,
			"enable_outlining":    true,
		},
	}
}

// For binary plugin mode
var PluginInstance domain.DomainPlugin

func init() {
	// Plugin will be initialized by the loader with dependencies
}

// InitPlugin is called by the plugin loader to inject dependencies
func InitPlugin(agentFactory *core.AgentFactory, storage core.Storage, logger *slog.Logger) {
	PluginInstance = NewPlugin(agentFactory, storage, logger)
}