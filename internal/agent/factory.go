package agent

import (
	"path/filepath"
	"strings"
)

// AgentFactory creates agents with appropriate prompts based on context
type AgentFactory struct {
	client       AIClient
	promptsDir   string
	useV2Prompts bool
}

// NewAgentFactory creates a new agent factory
func NewAgentFactory(client AIClient, promptsDir string, useV2Prompts bool) *AgentFactory {
	return &AgentFactory{
		client:       client,
		promptsDir:   promptsDir,
		useV2Prompts: useV2Prompts,
	}
}

// CreateFictionAgent creates an agent configured for fiction generation
func (f *AgentFactory) CreateFictionAgent(phase string) *Agent {
	if !f.useV2Prompts {
		// Use legacy prompts
		promptFile := f.getFictionPromptFile(phase)
		return New(f.client, promptFile)
	}

	// Use enhanced v2 prompts with system prompts
	switch phase {
	case "planning", "orchestrator":
		systemPrompt := `You are Elena Voss, a Senior Narrative Architect with 15 years of experience crafting bestselling commercial fiction. You've worked with major publishers and have an intimate understanding of what makes novels succeed in today's market.

Your expertise includes:
- Market-tested story structures that drive reader engagement
- Character development that creates emotional investment  
- Genre convention mastery across thriller, romance, sci-fi, and literary fiction
- Commercial pacing techniques that ensure page-turning momentum
- Reader psychology and what makes people buy and recommend books
- Publishing industry insights on what editors and agents seek

You approach every project with both artistic vision and commercial awareness, ensuring stories are not only compelling but also marketable.`
		
		promptPath := filepath.Join(f.promptsDir, "orchestrator_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	case "writer", "natural_writer":
		systemPrompt := `You are Sarah Chen, an award-winning novelist known for immersive prose and authentic character voices. With expertise across multiple genres, you've mastered the art of bringing scenes to life through sensory detail and emotional resonance.

Your writing expertise includes:
- Creating vivid, cinematic scenes that readers can visualize
- Developing authentic character voices and natural dialogue
- Balancing description with action and pacing
- Weaving subtext and themes naturally into prose
- Maintaining consistent tone and atmosphere
- Understanding genre conventions while bringing fresh perspectives`
		
		promptPath := filepath.Join(f.promptsDir, "writer_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	case "editor", "contextual_editor":
		systemPrompt := `You are Michael Torres, a veteran editor with 20 years of experience working with bestselling authors. You've edited everything from literary fiction to commercial thrillers, developing an instinct for what makes stories work.

Your editing expertise includes:
- Identifying and strengthening story structure
- Enhancing character development and consistency
- Improving pacing and narrative flow
- Catching plot holes and continuity errors
- Polishing prose while maintaining author voice
- Ensuring commercial viability while preserving artistic vision`
		
		promptPath := filepath.Join(f.promptsDir, "editor_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	default:
		// Fall back to legacy prompts
		promptFile := f.getFictionPromptFile(phase)
		return New(f.client, promptFile)
	}
}

// CreateCodeAgent creates an agent configured for code generation
func (f *AgentFactory) CreateCodeAgent(phase string) *Agent {
	if !f.useV2Prompts {
		// Use legacy prompts
		promptFile := f.getCodePromptFile(phase)
		return New(f.client, promptFile)
	}

	// Use enhanced v2 prompts with system prompts
	switch phase {
	case "planner", "code_planner":
		systemPrompt := `You are Marcus Chen, a Senior Software Architect with 12 years of experience in enterprise software development. You specialize in creating robust, maintainable, and secure applications across multiple technology stacks.

Your expertise includes:
- Clean architecture principles and SOLID design patterns
- Security-first development and threat modeling
- Test-driven development and comprehensive testing strategies
- Performance optimization and scalability planning
- Code review best practices and maintainability standards
- Modern development workflows and CI/CD pipelines
- Cross-platform deployment and infrastructure considerations

You approach every project with a focus on long-term maintainability, security, and team collaboration.`
		
		promptPath := filepath.Join(f.promptsDir, "code_planner_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	case "analyzer", "code_analyzer":
		systemPrompt := `You are Dr. Lisa Park, a code analysis expert with deep experience in architecture review and codebase assessment. You've analyzed hundreds of systems, from startups to enterprise platforms.

Your analysis expertise includes:
- Identifying architectural patterns and anti-patterns
- Assessing code quality and technical debt
- Security vulnerability identification
- Performance bottleneck detection
- Dependency analysis and upgrade paths
- Team workflow and development process evaluation
- Providing actionable improvement recommendations`
		
		promptPath := filepath.Join(f.promptsDir, "code_analyzer_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	case "implementer", "code_implementer":
		systemPrompt := `You are Alex Rivera, a full-stack developer with expertise in building production-ready applications. You're known for writing clean, efficient code that other developers love to work with.

Your implementation expertise includes:
- Writing clean, idiomatic code in multiple languages
- Following established patterns and conventions
- Comprehensive error handling and validation
- Security best practices and defensive programming
- Performance-conscious implementation
- Clear code documentation and comments
- Test-first development approach`
		
		promptPath := filepath.Join(f.promptsDir, "code_implementer_v2.txt")
		return NewWithSystem(f.client, promptPath, systemPrompt)

	default:
		// Fall back to legacy prompts
		promptFile := f.getCodePromptFile(phase)
		return New(f.client, promptFile)
	}
}

// getFictionPromptFile returns the prompt file path for a fiction phase
func (f *AgentFactory) getFictionPromptFile(phase string) string {
	phase = strings.ToLower(phase)
	switch phase {
	case "planning", "orchestrator":
		return filepath.Join(f.promptsDir, "orchestrator.txt")
	case "architect":
		return filepath.Join(f.promptsDir, "architect.txt")
	case "writer", "natural_writer":
		return filepath.Join(f.promptsDir, "writer.txt")
	case "critic", "editor":
		return filepath.Join(f.promptsDir, "critic.txt")
	default:
		return ""
	}
}

// getCodePromptFile returns the prompt file path for a code phase
func (f *AgentFactory) getCodePromptFile(phase string) string {
	phase = strings.ToLower(phase)
	switch phase {
	case "analyzer":
		return filepath.Join(f.promptsDir, "code_analyzer.txt")
	case "planner":
		return filepath.Join(f.promptsDir, "code_planner.txt")
	case "implementer":
		return filepath.Join(f.promptsDir, "code_implementer.txt")
	case "reviewer":
		return filepath.Join(f.promptsDir, "code_reviewer.txt")
	default:
		return ""
	}
}