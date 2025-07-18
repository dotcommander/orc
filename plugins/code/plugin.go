package code

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotcommander/orc/internal/core"
	"github.com/dotcommander/orc/internal/domain"
	"github.com/dotcommander/orc/internal/phase/code"
)

// Plugin implements the domain plugin interface for code generation
type Plugin struct {
	agentFactory *core.AgentFactory
	storage      core.Storage
	logger       *slog.Logger
}

// NewPlugin creates a new code plugin instance
func NewPlugin(agentFactory *core.AgentFactory, storage core.Storage, logger *slog.Logger) *Plugin {
	return &Plugin{
		agentFactory: agentFactory,
		storage:      storage,
		logger:       logger.With("plugin", "code"),
	}
}

// GetInfo returns plugin metadata
func (p *Plugin) GetInfo() domain.PluginInfo {
	return domain.PluginInfo{
		Name:        "code",
		Version:     "1.0.0",
		Description: "AI-powered code generation with best practices",
		Author:      "Orc Team",
		Domains:     []string{"code"},
		MinOrcVersion: "0.1.0",
	}
}

// CreatePhases returns the phases for code generation
func (p *Plugin) CreatePhases() ([]core.Phase, error) {
	// Create agents for each phase
	explorerAgent := p.agentFactory.CreateAgent("explorer", "prompts/code_explorer.txt")
	plannerAgent := p.agentFactory.CreateAgent("planner", "prompts/code_planner.txt")
	builderAgent := p.agentFactory.CreateAgent("builder", "prompts/code_builder.txt")
	refinerAgent := p.agentFactory.CreateAgent("refiner", "prompts/code_refiner.txt")

	// Return phase pipeline
	return []core.Phase{
		code.NewConversationalExplorer(explorerAgent, p.logger),
		code.NewIncrementalBuilder(builderAgent, p.storage, p.logger),
		code.NewIterativeRefiner(refinerAgent, p.logger),
	}, nil
}

// ValidateRequest checks if the request is suitable for code generation
func (p *Plugin) ValidateRequest(request string) error {
	if len(request) < 10 {
		return fmt.Errorf("request too short for meaningful code generation")
	}
	return nil
}

// GetOutputSpec returns expected outputs for code
func (p *Plugin) GetOutputSpec() domain.OutputSpec {
	return domain.OutputSpec{
		PrimaryOutput: "src/",
		SecondaryOutputs: []string{
			"README.md",
			"requirements.txt",
			"package.json",
			"go.mod",
			"tests/",
			"docs/",
		},
		FilePatterns: map[string]string{
			"source": "src/**/*",
			"tests":  "tests/**/*",
			"docs":   "docs/**/*.md",
		},
	}
}

// GetPhaseTimeouts returns recommended timeouts
func (p *Plugin) GetPhaseTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"ConversationalExplorer": 8 * time.Minute,
		"IncrementalBuilder":    15 * time.Minute,
		"IterativeRefiner":      20 * time.Minute,
	}
}

// GetRequiredConfig returns required configuration keys
func (p *Plugin) GetRequiredConfig() []string {
	return []string{
		"ai.model",
		"ai.api_key",
	}
}

// GetDefaultConfig returns default configuration for code
func (p *Plugin) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"code": map[string]interface{}{
			"language_detection": true,
			"quality_checks":     true,
			"test_generation":    true,
			"documentation":      true,
			"max_iterations":     10,
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