package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
	"github.com/dotcommander/orc/internal/phase"
)

// ConversationalExplorer conducts natural dialogue to understand project requirements
type ConversationalExplorer struct {
	BasePhase
	agent  core.Agent
	logger *slog.Logger
}

// ProjectExploration represents the discovered project understanding
type ProjectExploration struct {
	ProjectType     string                 `json:"project_type"`
	Requirements    []string               `json:"requirements"`
	TechStack       TechStackChoice        `json:"tech_stack"`
	Architecture    ArchitecturePattern    `json:"architecture"`
	Features        []FeatureSpec          `json:"features"`
	Constraints     []string               `json:"constraints"`
	QualityGoals    QualityMetrics         `json:"quality_goals"`
	Context         map[string]interface{} `json:"context"`
	DialogueHistory []DialogueExchange     `json:"dialogue_history"`
}

type TechStackChoice struct {
	Language    string   `json:"language"`
	Framework   string   `json:"framework,omitempty"`
	Database    string   `json:"database,omitempty"`
	Libraries   []string `json:"libraries,omitempty"`
	Rationale   string   `json:"rationale"`
}

type ArchitecturePattern struct {
	Pattern     string   `json:"pattern"`
	Components  []string `json:"components"`
	Structure   string   `json:"structure"`
	Rationale   string   `json:"rationale"`
}

type FeatureSpec struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Complexity  string   `json:"complexity"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type QualityMetrics struct {
	Security     string `json:"security"`
	Performance  string `json:"performance"`
	Maintainability string `json:"maintainability"`
	UserExperience string `json:"user_experience"`
}

type DialogueExchange struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Insights []string `json:"insights"`
	Timestamp time.Time `json:"timestamp"`
}

func NewConversationalExplorer(agent core.Agent, logger *slog.Logger) *ConversationalExplorer {
	return &ConversationalExplorer{
		BasePhase: NewBasePhase("ConversationalExplorer", 8*time.Minute),
		agent:     agent,
		logger:    logger.With("component", "conversational_explorer"),
	}
}

func (ce *ConversationalExplorer) Name() string {
	return "ConversationalExplorer"
}

func (ce *ConversationalExplorer) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	ce.logger.Info("Starting conversational exploration")
	
	// Extract request from input
	request := input.Request
	if request == "" {
		return core.PhaseOutput{}, fmt.Errorf("request not found in input")
	}

	// Begin natural conversation to understand the project
	exploration := &ProjectExploration{
		Context:         make(map[string]interface{}),
		DialogueHistory: make([]DialogueExchange, 0),
	}

	// Phase 1: Initial Understanding
	if err := ce.conductInitialDiscovery(ctx, request, exploration); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("initial discovery failed: %w", err)
	}

	// Phase 2: Deep Dive into Requirements
	if err := ce.conductRequirementsAnalysis(ctx, exploration); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("requirements analysis failed: %w", err)
	}

	// Phase 3: Technical Architecture Discussion
	if err := ce.conductTechnicalDiscussion(ctx, exploration); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("technical discussion failed: %w", err)
	}

	// Phase 4: Quality and Constraints Alignment
	if err := ce.conductQualityAlignment(ctx, exploration); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("quality alignment failed: %w", err)
	}

	ce.logger.Info("Conversational exploration completed",
		"project_type", exploration.ProjectType,
		"features_count", len(exploration.Features),
		"dialogue_exchanges", len(exploration.DialogueHistory))

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"exploration": exploration,
		},
	}, nil
}

func (ce *ConversationalExplorer) conductInitialDiscovery(ctx context.Context, request string, exploration *ProjectExploration) error {
	prompt := ce.buildInitialDiscoveryPrompt(request)
	
	response, err := ce.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get initial discovery response: %w", err)
	}

	// Parse the conversational response
	discovery, err := ce.parseInitialDiscovery(response)
	if err != nil {
		return fmt.Errorf("failed to parse initial discovery: %w", err)
	}

	exploration.ProjectType = discovery.ProjectType
	exploration.Requirements = discovery.Requirements
	exploration.Context["initial_understanding"] = discovery
	
	// Record the dialogue
	exchange := DialogueExchange{
		Question:  "What type of project are we building and what are the core requirements?",
		Answer:    response,
		Insights:  discovery.Requirements,
		Timestamp: time.Now(),
	}
	exploration.DialogueHistory = append(exploration.DialogueHistory, exchange)

	return nil
}

func (ce *ConversationalExplorer) conductRequirementsAnalysis(ctx context.Context, exploration *ProjectExploration) error {
	prompt := ce.buildRequirementsPrompt(exploration)
	
	response, err := ce.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get requirements analysis: %w", err)
	}

	features, err := ce.parseFeatureSpecs(response)
	if err != nil {
		return fmt.Errorf("failed to parse feature specs: %w", err)
	}

	exploration.Features = features
	exploration.Context["requirements_analysis"] = response

	exchange := DialogueExchange{
		Question:  "Let's break down the features and prioritize them based on user value and complexity",
		Answer:    response,
		Insights:  ce.extractFeatureInsights(features),
		Timestamp: time.Now(),
	}
	exploration.DialogueHistory = append(exploration.DialogueHistory, exchange)

	return nil
}

func (ce *ConversationalExplorer) conductTechnicalDiscussion(ctx context.Context, exploration *ProjectExploration) error {
	prompt := ce.buildTechnicalPrompt(exploration)
	
	response, err := ce.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get technical discussion: %w", err)
	}

	techChoices, err := ce.parseTechnicalChoices(response)
	if err != nil {
		return fmt.Errorf("failed to parse technical choices: %w", err)
	}

	exploration.TechStack = techChoices.TechStack
	exploration.Architecture = techChoices.Architecture
	exploration.Context["technical_discussion"] = response

	exchange := DialogueExchange{
		Question:  "What's the best technical approach and architecture for this project?",
		Answer:    response,
		Insights:  []string{techChoices.TechStack.Rationale, techChoices.Architecture.Rationale},
		Timestamp: time.Now(),
	}
	exploration.DialogueHistory = append(exploration.DialogueHistory, exchange)

	return nil
}

func (ce *ConversationalExplorer) conductQualityAlignment(ctx context.Context, exploration *ProjectExploration) error {
	prompt := ce.buildQualityPrompt(exploration)
	
	response, err := ce.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to get quality alignment: %w", err)
	}

	quality, constraints, err := ce.parseQualityAlignment(response)
	if err != nil {
		return fmt.Errorf("failed to parse quality alignment: %w", err)
	}

	exploration.QualityGoals = quality
	exploration.Constraints = constraints
	exploration.Context["quality_alignment"] = response

	exchange := DialogueExchange{
		Question:  "What are our quality goals and constraints for this project?",
		Answer:    response,
		Insights:  constraints,
		Timestamp: time.Now(),
	}
	exploration.DialogueHistory = append(exploration.DialogueHistory, exchange)

	return nil
}

func (ce *ConversationalExplorer) buildInitialDiscoveryPrompt(request string) string {
	return fmt.Sprintf(`You are a senior software architect having a natural conversation with a client about their project needs.

The client has made this request: "%s"

Your goal is to understand what they really want to build through natural dialogue. Ask clarifying questions and explore their vision.

Please respond in a conversational way that demonstrates your understanding and asks thoughtful follow-up questions. Focus on:

1. Project type and domain
2. Core functionality they envision
3. Who will use this and how
4. Success criteria from their perspective

Respond as if you're having a natural conversation, not filling out a form. Be curious and insightful.

Return your response in this JSON format:
{
  "conversational_response": "Your natural response to the client",
  "project_type": "Identified project type",
  "requirements": ["requirement1", "requirement2", "requirement3"],
  "follow_up_questions": ["question1", "question2"]
}`, request)
}

func (ce *ConversationalExplorer) buildRequirementsPrompt(exploration *ProjectExploration) string {
	return fmt.Sprintf(`Continuing our conversation about the %s project...

Based on our initial discussion, I understand you want to build %s.

Let's dive deeper into the specific features and functionality. I want to make sure we prioritize what's most valuable to your users.

Here's what I'm thinking for the core features:
%s

Let's break these down further and talk about:
1. Which features are absolutely essential vs nice-to-have
2. How complex each feature might be to implement
3. Any dependencies between features
4. User workflow and experience

What's your perspective on this breakdown? What am I missing?

Return your response in this JSON format:
{
  "conversational_response": "Your natural dialogue response",
  "features": [
    {
      "name": "Feature name",
      "description": "Detailed description",
      "priority": "high|medium|low",
      "complexity": "simple|moderate|complex",
      "dependencies": ["other features"]
    }
  ],
  "user_workflow": "Description of how users will interact with the system"
}`, 
		exploration.ProjectType,
		strings.Join(exploration.Requirements, ", "),
		strings.Join(exploration.Requirements, "\n- "))
}

func (ce *ConversationalExplorer) buildTechnicalPrompt(exploration *ProjectExploration) string {
	features := make([]string, len(exploration.Features))
	for i, f := range exploration.Features {
		features[i] = f.Name
	}

	return fmt.Sprintf(`Now let's talk about the technical approach for your %s project.

We've identified these key features: %s

I need to recommend the best technology stack and architecture that will:
1. Deliver these features effectively
2. Be maintainable and scalable
3. Match your team's capabilities
4. Stay within reasonable complexity bounds

Let me think through the options...

For a project like this, I'm considering:
- Language and framework choices
- Database and storage needs
- Architecture patterns that fit
- Development and deployment approach

What are your thoughts on technology preferences? Any constraints I should know about?

Return your response in this JSON format:
{
  "conversational_response": "Your technical discussion",
  "tech_stack": {
    "language": "Recommended language",
    "framework": "Framework if applicable",
    "database": "Database choice if needed",
    "libraries": ["key libraries"],
    "rationale": "Why this stack makes sense"
  },
  "architecture": {
    "pattern": "Architecture pattern",
    "components": ["main components"],
    "structure": "How components relate",
    "rationale": "Why this architecture fits"
  }
}`,
		exploration.ProjectType,
		strings.Join(features, ", "))
}

func (ce *ConversationalExplorer) buildQualityPrompt(exploration *ProjectExploration) string {
	return fmt.Sprintf(`Let's align on quality expectations for your %s project.

Given the features we've discussed and the technical approach, I want to make sure we're on the same page about:

1. Security requirements and concerns
2. Performance expectations
3. Maintainability and future development
4. User experience priorities

Every project has trade-offs, so I want to understand what matters most to you.

Some questions to consider:
- Will this handle sensitive data?
- How many users do you expect?
- Will other developers need to work on this?
- What's the timeline and budget reality?

Return your response in this JSON format:
{
  "conversational_response": "Your quality discussion",
  "quality_goals": {
    "security": "Security requirements",
    "performance": "Performance expectations", 
    "maintainability": "Maintainability needs",
    "user_experience": "UX priorities"
  },
  "constraints": ["constraint1", "constraint2", "constraint3"]
}`,
		exploration.ProjectType)
}

// Parsing helper methods
func (ce *ConversationalExplorer) parseInitialDiscovery(response string) (*ProjectExploration, error) {
	var result struct {
		ConversationalResponse string   `json:"conversational_response"`
		ProjectType           string   `json:"project_type"`
		Requirements          []string `json:"requirements"`
		FollowUpQuestions     []string `json:"follow_up_questions"`
	}

	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		ce.logger.Error("Failed to parse JSON response",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &ProjectExploration{
		ProjectType:  result.ProjectType,
		Requirements: result.Requirements,
		Context: map[string]interface{}{
			"conversational_response": result.ConversationalResponse,
			"follow_up_questions":     result.FollowUpQuestions,
		},
	}, nil
}

func (ce *ConversationalExplorer) parseFeatureSpecs(response string) ([]FeatureSpec, error) {
	var result struct {
		ConversationalResponse string        `json:"conversational_response"`
		Features              []FeatureSpec `json:"features"`
		UserWorkflow          string        `json:"user_workflow"`
	}

	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		ce.logger.Error("Failed to parse features JSON",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return nil, fmt.Errorf("failed to parse features JSON: %w", err)
	}

	return result.Features, nil
}

func (ce *ConversationalExplorer) parseTechnicalChoices(response string) (*struct {
	TechStack    TechStackChoice     `json:"tech_stack"`
	Architecture ArchitecturePattern `json:"architecture"`
}, error) {
	var result struct {
		ConversationalResponse string              `json:"conversational_response"`
		TechStack             TechStackChoice     `json:"tech_stack"`
		Architecture          ArchitecturePattern `json:"architecture"`
	}

	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		ce.logger.Error("Failed to parse technical choices JSON",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return nil, fmt.Errorf("failed to parse technical choices JSON: %w", err)
	}

	return &struct {
		TechStack    TechStackChoice     `json:"tech_stack"`
		Architecture ArchitecturePattern `json:"architecture"`
	}{
		TechStack:    result.TechStack,
		Architecture: result.Architecture,
	}, nil
}

func (ce *ConversationalExplorer) parseQualityAlignment(response string) (QualityMetrics, []string, error) {
	var result struct {
		ConversationalResponse string         `json:"conversational_response"`
		QualityGoals          QualityMetrics `json:"quality_goals"`
		Constraints           []string       `json:"constraints"`
	}

	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		ce.logger.Error("Failed to parse quality alignment JSON",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return QualityMetrics{}, nil, fmt.Errorf("failed to parse quality alignment JSON: %w", err)
	}

	return result.QualityGoals, result.Constraints, nil
}

func (ce *ConversationalExplorer) extractFeatureInsights(features []FeatureSpec) []string {
	insights := make([]string, len(features))
	for i, f := range features {
		insights[i] = fmt.Sprintf("%s (%s priority, %s complexity)", f.Name, f.Priority, f.Complexity)
	}
	return insights
}

func (ce *ConversationalExplorer) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if input.Request == "" {
		return fmt.Errorf("request is required for conversational exploration")
	}
	return nil
}

func (ce *ConversationalExplorer) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	if output.Data == nil {
		return fmt.Errorf("exploration data is required")
	}
	return nil
}

func (ce *ConversationalExplorer) EstimatedDuration() time.Duration {
	return 2 * time.Minute
}

func (ce *ConversationalExplorer) CanRetry(err error) bool {
	return true // Most exploration failures can be retried
}