package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
}

// NewFictionPlugin creates a new fiction plugin adapter
func NewFictionPlugin(agent domain.Agent, storage domain.Storage) *FictionPlugin {
	return &FictionPlugin{
		agent:   agent,
		storage: storage,
		config:  getDefaultFictionConfig(),
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
	return "Systematic AI novel generation with word-count accuracy: strategic planning, targeted writing, contextual editing, and polished assembly"
}

// GetPhases returns the ordered phases for fiction generation
func (p *FictionPlugin) GetPhases() []domain.Phase {
	// Convert agent and storage to core interfaces for phase creation
	coreAgent := &domainToCoreAgentAdapter{agent: p.agent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Create core phases using systematic approach for reliable word counts
	corePhases := []core.Phase{
		fiction.NewSystematicPlanner(coreAgent, coreStorage),
		fiction.NewTargetedWriter(coreAgent, coreStorage),
		fiction.NewContextualEditor(coreAgent, coreStorage),
		fiction.NewSystematicAssembler(coreStorage),
	}
	
	// Convert core phases to domain phases
	domainPhases := make([]domain.Phase, len(corePhases))
	for i, corePhase := range corePhases {
		domainPhases[i] = &coreToDomainPhaseAdapter{phase: corePhase}
	}
	
	return domainPhases
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
			"planning":     filepath.Join(getPromptsDir(), "planner.txt"),
			"architecture": filepath.Join(getPromptsDir(), "architect.txt"),
			"writing":      filepath.Join(getPromptsDir(), "writer.txt"),
			"critique":     filepath.Join(getPromptsDir(), "critic.txt"),
		},
		Limits: DomainPluginLimits{
			MaxConcurrentPhases: 4,
			PhaseTimeouts: map[string]time.Duration{
				"planning":     5 * time.Minute,
				"architecture": 10 * time.Minute,
				"writing":      30 * time.Minute,
				"assembly":     2 * time.Minute,
				"critique":     5 * time.Minute,
			},
			MaxRetries:   3,
			TotalTimeout: 60 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"supports_resume":     true,
			"supports_streaming":  false,
			"requires_creativity": true,
			"output_format":       "markdown",
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