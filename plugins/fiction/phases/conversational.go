package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// ConversationalPlanner develops stories through natural dialogue
type ConversationalPlanner struct {
	BasePhase
	agent      core.Agent
	storage    core.Storage
	conversation []ConversationTurn
}

type ConversationTurn struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Context  string `json:"context,omitempty"`
}

type StoryCore struct {
	Premise    string   `json:"premise"`
	MainIdea   string   `json:"main_idea"`
	Characters []string `json:"characters"`
	Setting    string   `json:"setting"`
	Tone       string   `json:"tone"`
	Length     string   `json:"length"`
}

func NewConversationalPlanner(agent core.Agent, storage core.Storage) *ConversationalPlanner {
	return &ConversationalPlanner{
		BasePhase:    NewBasePhase("Conversational Planning", 15*time.Minute),
		agent:        agent,
		storage:      storage,
		conversation: make([]ConversationTurn, 0),
	}
}

func (p *ConversationalPlanner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting conversational story development", 
		"phase", p.Name(),
		"request_preview", truncateString(input.Request, 100))

	// Start with understanding what the user wants
	storyCore, err := p.discoverStoryCore(ctx, input.Request)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("discovering story core: %w", err)
	}

	// Develop the premise through conversation
	refinedCore, err := p.refinePremise(ctx, storyCore)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("refining premise: %w", err)
	}

	// Build chapter structure naturally
	chapters, err := p.buildChapterFlow(ctx, refinedCore)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("building chapters: %w", err)
	}

	// Save conversation history
	if err := p.saveConversation(ctx, input.SessionID); err != nil {
		slog.Warn("Failed to save conversation", "error", err)
	}

	result := map[string]interface{}{
		"story_core":    refinedCore,
		"chapters":      chapters,
		"conversation": p.conversation,
	}

	slog.Info("Conversational planning completed",
		"phase", p.Name(),
		"chapters", len(chapters),
		"conversation_turns", len(p.conversation))

	return core.PhaseOutput{Data: result}, nil
}

// discoverStoryCore extracts the essential story elements through natural questions
func (p *ConversationalPlanner) discoverStoryCore(ctx context.Context, userRequest string) (*StoryCore, error) {
	// Ask: What's this story really about?
	coreQuestion := fmt.Sprintf(`
	A user wants this story: "%s"
	
	In simple, natural language, what do you think this story is really about at its heart? 
	Focus on the main idea, not plot details. Keep it conversational and clear.`, userRequest)

	response, err := p.askAI(ctx, coreQuestion, "understanding the core")
	if err != nil {
		return nil, err
	}

	// Extract setting naturally
	settingQuestion := fmt.Sprintf(`
	For a story about: "%s"
	
	Where and when should this take place? Just describe the setting naturally - 
	don't worry about being comprehensive, just what feels right for this story.`, response)

	settingResponse, err := p.askAI(ctx, settingQuestion, "choosing setting")
	if err != nil {
		return nil, err
	}

	// Extract characters naturally
	characterQuestion := fmt.Sprintf(`
	For this story: "%s"
	Set in: "%s"
	
	Who are the main people we should care about? Just describe them as you'd 
	tell a friend about interesting people you met. Don't make it formal.`, response, settingResponse)

	characterResponse, err := p.askAI(ctx, characterQuestion, "meeting characters")
	if err != nil {
		return nil, err
	}

	return &StoryCore{
		Premise:    response,
		MainIdea:   userRequest,
		Characters: []string{characterResponse}, // We'll expand this later
		Setting:    settingResponse,
		Tone:       "natural", // We could ask about this too
		Length:     "medium",  // Could be inferred from request
	}, nil
}

// refinePremise improves the story through iterative conversation
func (p *ConversationalPlanner) refinePremise(ctx context.Context, core *StoryCore) (*StoryCore, error) {
	refinementQuestion := fmt.Sprintf(`
	We have this story developing:
	
	Core idea: %s
	Setting: %s
	Characters: %s
	
	What's one thing that would make this story more interesting or compelling? 
	Just suggest one improvement - don't rewrite everything.`, 
	core.Premise, core.Setting, strings.Join(core.Characters, ", "))

	improvement, err := p.askAI(ctx, refinementQuestion, "improving the story")
	if err != nil {
		return core, nil // Return original if refinement fails
	}

	// Apply improvement naturally
	applyQuestion := fmt.Sprintf(`
	Original premise: %s
	Suggested improvement: %s
	
	Now describe the improved story premise in one clear paragraph. 
	Make it sound natural and engaging.`, core.Premise, improvement)

	improvedPremise, err := p.askAI(ctx, applyQuestion, "applying improvement")
	if err != nil {
		return core, nil // Return original if application fails
	}

	core.Premise = improvedPremise
	return core, nil
}

// buildChapterFlow creates chapter structure through natural progression
func (p *ConversationalPlanner) buildChapterFlow(ctx context.Context, core *StoryCore) ([]map[string]interface{}, error) {
	flowQuestion := fmt.Sprintf(`
	For this story: "%s"
	
	How should this story unfold? Describe the natural flow from beginning to end.
	Don't worry about exact chapters - just tell me how the story should progress.
	What happens first, then what, then what? Keep it conversational.`, core.Premise)

	flow, err := p.askAI(ctx, flowQuestion, "planning story flow")
	if err != nil {
		return nil, err
	}

	// Convert flow into loose chapters
	chapterQuestion := fmt.Sprintf(`
	Story flow: "%s"
	
	Break this flow into natural chapters. For each chapter, just give me:
	- A simple title
	- What happens in that chapter (one sentence)
	
	Format like:
	Chapter 1: Title - What happens
	Chapter 2: Title - What happens
	
	Keep it simple and natural.`, flow)

	chapters, err := p.askAI(ctx, chapterQuestion, "organizing chapters")
	if err != nil {
		return nil, err
	}

	// Parse into structure (loosely)
	return p.parseChaptersNaturally(chapters), nil
}

// parseChaptersNaturally extracts chapter info without rigid validation
func (p *ConversationalPlanner) parseChaptersNaturally(text string) []map[string]interface{} {
	lines := strings.Split(text, "\n")
	chapters := make([]map[string]interface{}, 0)
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Chapter") && strings.Contains(line, ":") {
			// Extract title and description loosely
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				titlePart := strings.TrimSpace(parts[0])
				content := strings.TrimSpace(parts[1])
				
				chapter := map[string]interface{}{
					"number":      i + 1,
					"title":       titlePart,
					"description": content,
					"scenes":      []map[string]string{{"description": content}},
				}
				chapters = append(chapters, chapter)
			}
		}
	}
	
	// If no chapters found, create simple structure
	if len(chapters) == 0 {
		chapters = append(chapters, map[string]interface{}{
			"number":      1,
			"title":       "Chapter 1",
			"description": text,
			"scenes":      []map[string]string{{"description": text}},
		})
	}
	
	return chapters
}

// askAI handles the conversation with context tracking
func (p *ConversationalPlanner) askAI(ctx context.Context, question, context string) (string, error) {
	slog.Debug("Conversational AI request",
		"context", context,
		"question_length", len(question))

	response, err := p.agent.Execute(ctx, question, nil)
	if err != nil {
		return "", err
	}

	// Track conversation
	turn := ConversationTurn{
		Question: question,
		Answer:   response,
		Context:  context,
	}
	p.conversation = append(p.conversation, turn)

	slog.Debug("Conversational AI response",
		"context", context,
		"response_length", len(response))

	return response, nil
}

// saveConversation stores the conversation for later review
func (p *ConversationalPlanner) saveConversation(ctx context.Context, sessionID string) error {
	conversationData := map[string]interface{}{
		"turns":     p.conversation,
		"timestamp": time.Now(),
		"session":   sessionID,
	}

	data, err := json.Marshal(conversationData)
	if err != nil {
		return err
	}

	return p.storage.Save(ctx, "conversation.json", data)
}

func (p *ConversationalPlanner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if len(strings.TrimSpace(input.Request)) < 10 {
		return fmt.Errorf("request too short for conversational development")
	}
	return nil
}

func (p *ConversationalPlanner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil // Conversational output is flexible
}