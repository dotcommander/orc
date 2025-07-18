package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// NaturalWriter generates story content through flowing, natural requests
type NaturalWriter struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
}

type SceneContext struct {
	StoryPremise string                 `json:"story_premise"`
	Characters   []string               `json:"characters"`
	Setting      string                 `json:"setting"`
	ChapterInfo  map[string]interface{} `json:"chapter_info"`
	PreviousText string                 `json:"previous_text,omitempty"`
}

func NewNaturalWriter(agent core.Agent, storage core.Storage) *NaturalWriter {
	return &NaturalWriter{
		BasePhase: NewBasePhase("Natural Writing", 45*time.Minute),
		agent:     agent,
		storage:   storage,
	}
}

func (w *NaturalWriter) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting natural writing process", 
		"phase", w.Name())

	// Extract story context from previous phases
	planData, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid plan data format")
	}

	// Process each chapter naturally
	allScenes := make(map[string]interface{})
	manuscriptParts := make([]string, 0)

	chapters, ok := planData["chapters"].([]map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("no chapters found in plan")
	}

	storyCore := w.extractStoryCore(planData)

	for i, chapter := range chapters {
		chapterNumber := i + 1
		slog.Info("Writing chapter naturally",
			"chapter", chapterNumber,
			"title", chapter["title"])

		sceneContent, err := w.writeChapterNaturally(ctx, storyCore, chapter, manuscriptParts)
		if err != nil {
			return core.PhaseOutput{}, fmt.Errorf("writing chapter %d: %w", chapterNumber, err)
		}

		// Store scene and add to manuscript
		sceneKey := fmt.Sprintf("chapter_%d_scene_1", chapterNumber)
		allScenes[sceneKey] = map[string]interface{}{
			"content": sceneContent,
			"chapter": chapterNumber,
			"title":   chapter["title"],
		}

		manuscriptParts = append(manuscriptParts, sceneContent)

		// Save individual scene
		sceneFile := fmt.Sprintf("scenes/chapter_%d_scene_1.md", chapterNumber)
		if err := w.storage.Save(ctx, sceneFile, []byte(sceneContent)); err != nil {
			slog.Warn("Failed to save scene", "file", sceneFile, "error", err)
		}
	}

	result := map[string]interface{}{
		"scenes":     allScenes,
		"manuscript": strings.Join(manuscriptParts, "\n\n---\n\n"),
	}

	slog.Info("Natural writing completed",
		"phase", w.Name(),
		"chapters_written", len(chapters),
		"total_length", len(result["manuscript"].(string)))

	return core.PhaseOutput{Data: result}, nil
}

// writeChapterNaturally creates chapter content through conversational prompting
func (w *NaturalWriter) writeChapterNaturally(ctx context.Context, storyCore *SceneContext, chapter map[string]interface{}, previousParts []string) (string, error) {
	// Build context naturally
	contextPrompt := w.buildNaturalContext(storyCore, chapter, previousParts)
	
	// Write the chapter with natural flow
	writePrompt := fmt.Sprintf(`%s

Now, write this chapter. Let the story flow naturally. Focus on:
- Bringing the characters to life through their actions and dialogue
- Making the reader feel like they're there in the scene
- Moving the story forward naturally
- Writing prose that feels engaging and readable

Don't worry about hitting exact word counts or following rigid structures. 
Just tell this part of the story well. Write as if you're an author who 
loves storytelling and wants the reader to be captivated.

Begin writing the chapter:`, contextPrompt)

	content, err := w.agent.Execute(ctx, writePrompt, nil)
	if err != nil {
		return "", err
	}

	// Optional: Enhance the content through iteration
	enhanced, err := w.enhanceContent(ctx, content, storyCore)
	if err != nil {
		slog.Warn("Content enhancement failed, using original", "error", err)
		return content, nil
	}

	return enhanced, nil
}

// buildNaturalContext creates flowing context without rigid structure
func (w *NaturalWriter) buildNaturalContext(storyCore *SceneContext, chapter map[string]interface{}, previousParts []string) string {
	context := fmt.Sprintf("You're writing a story about: %s\n", storyCore.StoryPremise)
	
	if storyCore.Setting != "" {
		context += fmt.Sprintf("Set in: %s\n", storyCore.Setting)
	}
	
	if len(storyCore.Characters) > 0 {
		context += fmt.Sprintf("Main characters: %s\n", strings.Join(storyCore.Characters, ", "))
	}

	if chapterTitle, ok := chapter["title"].(string); ok {
		context += fmt.Sprintf("This chapter: %s\n", chapterTitle)
	}

	if chapterDesc, ok := chapter["description"].(string); ok {
		context += fmt.Sprintf("What happens: %s\n", chapterDesc)
	}

	// Add story continuation context
	if len(previousParts) > 0 {
		lastPart := previousParts[len(previousParts)-1]
		if len(lastPart) > 200 {
			lastPart = lastPart[len(lastPart)-200:] // Last 200 characters for context
		}
		context += fmt.Sprintf("\nContinuing from: ...%s\n", lastPart)
	} else {
		context += "\nThis is the beginning of the story.\n"
	}

	return context
}

// enhanceContent improves the generated content through natural feedback
func (w *NaturalWriter) enhanceContent(ctx context.Context, content string, storyCore *SceneContext) (string, error) {
	enhancePrompt := fmt.Sprintf(`
Here's a chapter from our story:

%s

Read through this and improve it. Make it more engaging, more vivid, 
more compelling. Fix any awkward phrasing. Add sensory details where 
they would help. Make the dialogue more natural. Make the reader care more.

Don't change the basic story or events - just make the writing better. 
Return the improved version:`, content)

	enhanced, err := w.agent.Execute(ctx, enhancePrompt, nil)
	if err != nil {
		return content, err // Return original on error
	}

	// Simple quality check - enhanced should be reasonably similar in length
	originalLength := len(content)
	enhancedLength := len(enhanced)
	
	// If too different in length, something went wrong
	if enhancedLength < originalLength/2 || enhancedLength > originalLength*2 {
		slog.Warn("Enhancement created suspiciously different length content",
			"original", originalLength,
			"enhanced", enhancedLength)
		return content, nil
	}

	return enhanced, nil
}

// extractStoryCore pulls essential info from plan data
func (w *NaturalWriter) extractStoryCore(planData map[string]interface{}) *SceneContext {
	context := &SceneContext{
		Characters: make([]string, 0),
	}

	// Extract story core if available
	if storyCore, ok := planData["story_core"].(map[string]interface{}); ok {
		if premise, ok := storyCore["premise"].(string); ok {
			context.StoryPremise = premise
		}
		if setting, ok := storyCore["setting"].(string); ok {
			context.Setting = setting
		}
		if chars, ok := storyCore["characters"].([]string); ok {
			context.Characters = chars
		}
	}

	// Fallback to extracting from conversation if needed
	if context.StoryPremise == "" {
		if conv, ok := planData["conversation"].([]interface{}); ok && len(conv) > 0 {
			// Extract premise from first conversation turn
			if turn, ok := conv[0].(map[string]interface{}); ok {
				if answer, ok := turn["answer"].(string); ok {
					context.StoryPremise = answer
				}
			}
		}
	}

	return context
}

func (w *NaturalWriter) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	planData, ok := input.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("input data must be a plan structure")
	}

	if _, ok := planData["chapters"]; !ok {
		return fmt.Errorf("plan must contain chapters")
	}

	return nil
}

func (w *NaturalWriter) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil // Natural writing output is flexible
}