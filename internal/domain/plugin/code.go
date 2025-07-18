package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/domain"
	"github.com/vampirenirmal/orchestrator/internal/phase/code"
)

// CodePlugin implements the DomainPlugin interface for code assistance
type CodePlugin struct {
	agent   domain.Agent
	storage domain.Storage
	config  DomainPluginConfig
}

// NewCodePlugin creates a new code plugin adapter
func NewCodePlugin(agent domain.Agent, storage domain.Storage) *CodePlugin {
	return &CodePlugin{
		agent:   agent,
		storage: storage,
		config:  getDefaultCodeConfig(),
	}
}

// Name returns the plugin name
func (p *CodePlugin) Name() string {
	return "code"
}

// Description returns a human-readable description
func (p *CodePlugin) Description() string {
	return "AI-powered code generation that flows like water: conversational exploration, incremental building, quality refinement, and gentle validation"
}

// GetPhases returns the ordered phases for code tasks using "Be Like Water" approach
func (p *CodePlugin) GetPhases() []domain.Phase {
	// Convert agent and storage to core interfaces for phase creation
	coreAgent := &domainToCoreAgentAdapter{agent: p.agent}
	coreStorage := &domainToCoreStorageAdapter{storage: p.storage}
	
	// Use default logger with code plugin context
	logger := slog.Default().With("component", "code_plugin")
	
	// Create new "Be Like Water" phases that flow naturally with AI capabilities
	corePhases := []core.Phase{
		code.NewConversationalExplorer(coreAgent, logger),
		code.NewIncrementalBuilder(coreAgent, coreStorage, logger),
		code.NewIterativeRefiner(coreAgent, logger), // Infinite iterative improvement!
		code.NewGentleValidator(coreAgent, logger),
	}
	
	// Convert core phases to domain phases
	domainPhases := make([]domain.Phase, len(corePhases))
	for i, corePhase := range corePhases {
		domainPhases[i] = &coreToDomainPhaseAdapter{phase: corePhase}
	}
	
	return domainPhases
}

// GetDefaultConfig returns default configuration for code tasks
func (p *CodePlugin) GetDefaultConfig() DomainPluginConfig {
	return p.config
}

// ValidateRequest validates if the user request is appropriate for code tasks
func (p *CodePlugin) ValidateRequest(request string) error {
	request = strings.TrimSpace(strings.ToLower(request))
	
	if len(request) < 10 {
		return fmt.Errorf("request too short: please provide a detailed description of the code task")
	}
	
	// Check for code-related keywords
	codeKeywords := []string{
		"code", "function", "class", "api", "database", "server",
		"algorithm", "data structure", "testing", "debug", "refactor",
		"implement", "create", "build", "develop", "program",
		"javascript", "python", "go", "java", "c++", "rust",
		"framework", "library", "module", "package", "app",
		"web", "mobile", "desktop", "backend", "frontend",
		"rest", "graphql", "microservice", "container", "docker",
	}
	
	for _, keyword := range codeKeywords {
		if strings.Contains(request, keyword) {
			return nil // Valid code request
		}
	}
	
	// Check for anti-patterns (fiction requests)
	fictionKeywords := []string{
		"novel", "story", "book", "fiction", "tale", "narrative",
		"character", "plot", "chapter", "romance", "thriller",
		"drama", "adventure", "horror", "comedy", "fantasy",
	}
	
	for _, keyword := range fictionKeywords {
		if strings.Contains(request, keyword) {
			return fmt.Errorf("request appears to be for fiction writing, not code")
		}
	}
	
	// If no clear code keywords, give a warning but allow
	return nil
}

// GetOutputSpec returns the expected output structure for code tasks
func (p *CodePlugin) GetOutputSpec() DomainOutputSpec {
	return DomainOutputSpec{
		PrimaryOutput: "code_output.md",
		SecondaryOutputs: []string{
			"exploration.json",
			"build_plan.json",
			"generated_code/",
			"refinement_progress.json",
			"validation_result.json",
		},
		Descriptions: map[string]string{
			"code_output.md":           "📝 Complete code output with explanations",
			"exploration.json":         "🗣️ Conversational project exploration results",
			"build_plan.json":          "🔧 Systematic incremental build plan",
			"generated_code/":          "💻 Generated code files with full context",
			"refinement_progress.json": "✨ Quality refinement iterations and improvements",
			"validation_result.json":   "✅ Gentle validation with constructive guidance",
		},
	}
}

// GetDomainValidator returns code-specific validation
func (p *CodePlugin) GetDomainValidator() domain.DomainValidator {
	return &CodeValidator{}
}

// CodeValidator provides code-specific validation
type CodeValidator struct{}

// ValidateRequest validates a user request for code tasks
func (v *CodeValidator) ValidateRequest(request string) error {
	if len(strings.TrimSpace(request)) == 0 {
		return fmt.Errorf("code request cannot be empty")
	}
	
	// Check for code-related keywords
	codeKeywords := []string{
		"code", "function", "class", "api", "database", "server",
		"algorithm", "refactor", "debug", "test", "implement",
		"library", "framework", "programming", "script", "application",
		"module", "package", "dependency", "build", "deploy",
	}
	
	lowerRequest := strings.ToLower(request)
	for _, keyword := range codeKeywords {
		if strings.Contains(lowerRequest, keyword) {
			return nil // Valid code request
		}
	}
	
	// Check for programming language names
	programmingLanguages := []string{
		"python", "javascript", "java", "go", "rust", "c++", "c#",
		"php", "ruby", "kotlin", "swift", "typescript", "scala",
		"html", "css", "sql", "bash", "powershell",
	}
	
	for _, lang := range programmingLanguages {
		if strings.Contains(lowerRequest, lang) {
			return nil // Valid code request
		}
	}
	
	// Check for anti-patterns (non-code requests)
	nonCodeKeywords := []string{
		"story", "novel", "character", "plot", "fiction",
		"narrative", "dialogue", "scene", "chapter", "protagonist",
		"fantasy", "romance", "mystery", "thriller", "drama",
	}
	
	for _, keyword := range nonCodeKeywords {
		if strings.Contains(lowerRequest, keyword) {
			return fmt.Errorf("request appears to be for fiction writing, not code")
		}
	}
	
	// If no clear code keywords, give a warning but allow
	return nil
}

// ValidatePhaseTransition validates data between code phases
func (v *CodeValidator) ValidatePhaseTransition(from, to string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("phase transition data cannot be nil")
	}
	
	// Validate specific phase transitions
	switch from + "->" + to {
	case "Analysis->Planning":
		// Validate analysis data contains required fields
		if analysisData, ok := data.(map[string]interface{}); ok {
			if _, hasComplexity := analysisData["complexity"]; !hasComplexity {
				return fmt.Errorf("analysis phase must assess complexity")
			}
			if _, hasLanguage := analysisData["language"]; !hasLanguage {
				return fmt.Errorf("analysis phase must identify programming language")
			}
		}
	case "Planning->Implementation":
		// Validate planning data contains implementation steps
		if planData, ok := data.(map[string]interface{}); ok {
			if _, hasSteps := planData["steps"]; !hasSteps {
				return fmt.Errorf("planning phase must define implementation steps")
			}
			if _, hasFiles := planData["files"]; !hasFiles {
				return fmt.Errorf("planning phase must specify files to create")
			}
		}
	case "Implementation->Review":
		// Validate implementation data contains code
		if implData, ok := data.(map[string]interface{}); ok {
			if files, hasFiles := implData["files"]; hasFiles {
				if fileList, ok := files.([]interface{}); ok {
					if len(fileList) == 0 {
						return fmt.Errorf("implementation phase must produce at least one file")
					}
				}
			}
		}
	}
	
	return nil
}

// ValidateOldOutput validates code-specific output data (deprecated)
func (v *CodeValidator) ValidateOldOutput(output interface{}) error {
	if output == nil {
		return fmt.Errorf("code output cannot be nil")
	}
	
	switch typed := output.(type) {
	case string:
		if len(strings.TrimSpace(typed)) == 0 {
			return fmt.Errorf("code output cannot be empty")
		}
	case map[string]interface{}:
		// Validate structured code output
		if content, hasContent := typed["content"]; hasContent {
			if contentStr, ok := content.(string); ok {
				if len(strings.TrimSpace(contentStr)) == 0 {
					return fmt.Errorf("code content cannot be empty")
				}
			}
		}
		
		// Validate code-specific output fields
		if analysis, hasAnalysis := typed["analysis"]; hasAnalysis {
			if analysisMap, ok := analysis.(map[string]interface{}); ok {
				if len(analysisMap) == 0 {
					return fmt.Errorf("code analysis cannot be empty")
				}
			}
		}
	}
	
	return nil
}

// getDefaultCodeConfig returns the default configuration for code tasks
func getDefaultCodeConfig() DomainPluginConfig {
	return DomainPluginConfig{
		Prompts: map[string]string{
			"analyzer":    "prompts/code_analyzer.txt",
			"planner":     "prompts/code_planner.txt",
			"implementer": "prompts/code_implementer.txt",
			"reviewer":    "prompts/code_reviewer.txt",
		},
		Limits: DomainPluginLimits{
			MaxConcurrentPhases: 1,
			PhaseTimeouts: map[string]time.Duration{
				"ConversationalExplorer": 3 * time.Minute,
				"IncrementalBuilder":     8 * time.Minute,
				"IterativeRefiner":       10 * time.Minute, // Allows for multiple iterations
				"GentleValidator":        3 * time.Minute,
			},
			MaxRetries:   3,
			TotalTimeout: 30 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"supports_resume":     true,
			"supports_streaming":  false,
			"requires_creativity": false,
			"output_format":       "markdown",
		},
	}
}

// Local adapter types for domain/core conversion

type domainToCoreAgentAdapter struct {
	agent domain.Agent
}

func (a *domainToCoreAgentAdapter) Execute(ctx context.Context, prompt string, input any) (string, error) {
	return a.agent.Execute(ctx, prompt, input)
}

func (a *domainToCoreAgentAdapter) ExecuteJSON(ctx context.Context, prompt string, input any) (string, error) {
	return a.agent.ExecuteJSON(ctx, prompt, input)
}

type domainToCoreStorageAdapter struct {
	storage domain.Storage
}

func (s *domainToCoreStorageAdapter) Save(ctx context.Context, key string, data []byte) error {
	return s.storage.Save(ctx, key, data)
}

func (s *domainToCoreStorageAdapter) Load(ctx context.Context, key string) ([]byte, error) {
	return s.storage.Load(ctx, key)
}

func (s *domainToCoreStorageAdapter) Exists(ctx context.Context, key string) bool {
	return s.storage.Exists(ctx, key)
}

func (s *domainToCoreStorageAdapter) Delete(ctx context.Context, key string) error {
	return s.storage.Delete(ctx, key)
}

func (s *domainToCoreStorageAdapter) List(ctx context.Context, pattern string) ([]string, error) {
	return s.storage.List(ctx, pattern)
}

type coreToDomainPhaseAdapter struct {
	phase core.Phase
}

func (p *coreToDomainPhaseAdapter) Name() string {
	return p.phase.Name()
}

func (p *coreToDomainPhaseAdapter) Execute(ctx context.Context, input domain.PhaseInput) (domain.PhaseOutput, error) {
	coreInput := core.PhaseInput{
		Request: input.Request,
		Data:    input.Data,
	}
	
	coreOutput, err := p.phase.Execute(ctx, coreInput)
	if err != nil {
		return domain.PhaseOutput{}, err
	}
	
	return domain.PhaseOutput{
		Data:     coreOutput.Data,
		Error:    coreOutput.Error,
		Metadata: make(map[string]interface{}),
	}, nil
}

func (p *coreToDomainPhaseAdapter) ValidateInput(ctx context.Context, input domain.PhaseInput) error {
	coreInput := core.PhaseInput{
		Request: input.Request,
		Data:    input.Data,
	}
	return p.phase.ValidateInput(ctx, coreInput)
}

func (p *coreToDomainPhaseAdapter) ValidateOutput(ctx context.Context, output domain.PhaseOutput) error {
	coreOutput := core.PhaseOutput{
		Data:  output.Data,
		Error: output.Error,
	}
	return p.phase.ValidateOutput(ctx, coreOutput)
}

func (p *coreToDomainPhaseAdapter) EstimatedDuration() time.Duration {
	return p.phase.EstimatedDuration()
}

func (p *coreToDomainPhaseAdapter) CanRetry(err error) bool {
	return p.phase.CanRetry(err)
}