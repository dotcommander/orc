package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// FlowingAssembler creates cohesive manuscripts through natural composition
type FlowingAssembler struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
}

func NewFlowingAssembler(agent core.Agent, storage core.Storage) *FlowingAssembler {
	return &FlowingAssembler{
		BasePhase: NewBasePhase("Flowing Assembly", 10*time.Minute),
		agent:     agent,
		storage:   storage,
	}
}

func (a *FlowingAssembler) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting flowing assembly", "phase", a.Name())

	writerData, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid writer data format")
	}

	// Get the raw manuscript if available
	if manuscript, ok := writerData["manuscript"].(string); ok && len(manuscript) > 0 {
		// The natural writer already created a flowing manuscript
		enhanced, err := a.enhanceManuscriptFlow(ctx, manuscript)
		if err != nil {
			slog.Warn("Failed to enhance manuscript flow, using original", "error", err)
			enhanced = manuscript
		}

		// Save the final manuscript
		if err := a.storage.Save(ctx, "manuscript.md", []byte(enhanced)); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("saving manuscript: %w", err)
		}

		return core.PhaseOutput{
			Data: map[string]interface{}{
				"manuscript": enhanced,
			},
		}, nil
	}

	// Fallback: assemble from individual scenes
	scenes, ok := writerData["scenes"].(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("no manuscript or scenes found")
	}

	manuscript, err := a.assembleFromScenes(ctx, scenes)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("assembling from scenes: %w", err)
	}

	// Save the final manuscript
	if err := a.storage.Save(ctx, "manuscript.md", []byte(manuscript)); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("saving manuscript: %w", err)
	}

	slog.Info("Flowing assembly completed",
		"phase", a.Name(),
		"manuscript_length", len(manuscript))

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"manuscript": manuscript,
		},
	}, nil
}

// enhanceManuscriptFlow improves the overall flow and coherence
func (a *FlowingAssembler) enhanceManuscriptFlow(ctx context.Context, manuscript string) (string, error) {
	if len(manuscript) < 1000 {
		// Too short to meaningfully enhance
		return manuscript, nil
	}

	enhancePrompt := fmt.Sprintf(`
Here's a complete story manuscript:

%s

Please review this story for flow and coherence. Make these improvements:

1. Smooth any rough transitions between chapters or sections
2. Ensure consistent tone and voice throughout
3. Fix any continuity issues you notice
4. Polish the prose for better readability
5. Make sure the ending feels satisfying and connected to the beginning

Keep the same story, characters, and events. Just make it read more smoothly 
and feel more cohesive as a complete work. Return the improved manuscript:`, 
		truncateString(manuscript, 8000)) // Limit to avoid token limits

	enhanced, err := a.agent.Execute(ctx, enhancePrompt, nil)
	if err != nil {
		return manuscript, err
	}

	// Quality check - enhanced should be reasonably similar in length
	originalLength := len(manuscript)
	enhancedLength := len(enhanced)
	
	if enhancedLength < originalLength/3 || enhancedLength > originalLength*2 {
		slog.Warn("Enhanced manuscript length seems wrong",
			"original", originalLength,
			"enhanced", enhancedLength)
		return manuscript, nil
	}

	return enhanced, nil
}

// assembleFromScenes creates flowing manuscript from individual scenes
func (a *FlowingAssembler) assembleFromScenes(ctx context.Context, scenes map[string]interface{}) (string, error) {
	// Extract and order scenes
	orderedScenes := a.orderScenes(scenes)
	
	if len(orderedScenes) == 0 {
		return "", fmt.Errorf("no scenes to assemble")
	}

	// Build manuscript parts
	var parts []string
	
	// Add title and introduction
	title := a.extractTitle(scenes)
	if title != "" {
		parts = append(parts, fmt.Sprintf("# %s\n", title))
	}

	// Add each scene with natural transitions
	for i, scene := range orderedScenes {
		content := scene.Content
		
		// Add chapter heading if this is the start of a new chapter
		if scene.ChapterNumber > 0 && (i == 0 || orderedScenes[i-1].ChapterNumber != scene.ChapterNumber) {
			if scene.Title != "" {
				parts = append(parts, fmt.Sprintf("\n## %s\n", scene.Title))
			} else {
				parts = append(parts, fmt.Sprintf("\n## Chapter %d\n", scene.ChapterNumber))
			}
		}

		parts = append(parts, content)
	}

	manuscript := strings.Join(parts, "\n\n")

	// Enhance the flow between assembled scenes
	if len(orderedScenes) > 1 {
		enhanced, err := a.improveTransitions(ctx, manuscript)
		if err != nil {
			slog.Warn("Failed to improve transitions", "error", err)
			return manuscript, nil
		}
		return enhanced, nil
	}

	return manuscript, nil
}

type SceneInfo struct {
	Content       string
	ChapterNumber int
	Title         string
	SceneNumber   int
}

// orderScenes sorts scenes into logical order
func (a *FlowingAssembler) orderScenes(scenes map[string]interface{}) []SceneInfo {
	var orderedScenes []SceneInfo

	for key, sceneData := range scenes {
		scene, ok := sceneData.(map[string]interface{})
		if !ok {
			continue
		}

		info := SceneInfo{}
		
		if content, ok := scene["content"].(string); ok {
			info.Content = content
		}
		
		if chapter, ok := scene["chapter"].(int); ok {
			info.ChapterNumber = chapter
		} else if chapter, ok := scene["chapter"].(float64); ok {
			info.ChapterNumber = int(chapter)
		}
		
		if title, ok := scene["title"].(string); ok {
			info.Title = title
		}

		// Extract scene number from key if possible
		if strings.Contains(key, "scene_") {
			parts := strings.Split(key, "_")
			for _, part := range parts {
				if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
					// Simple scene number extraction
					info.SceneNumber = 1 // Default for now
					break
				}
			}
		}

		orderedScenes = append(orderedScenes, info)
	}

	// Simple sort by chapter number, then scene number
	// TODO: Implement proper sorting if needed
	return orderedScenes
}

// extractTitle attempts to find a story title
func (a *FlowingAssembler) extractTitle(scenes map[string]interface{}) string {
	// Look for title in first scene or any scene metadata
	for _, sceneData := range scenes {
		if scene, ok := sceneData.(map[string]interface{}); ok {
			if title, ok := scene["story_title"].(string); ok && title != "" {
				return title
			}
		}
	}
	return "" // No title found
}

// improveTransitions enhances flow between assembled parts
func (a *FlowingAssembler) improveTransitions(ctx context.Context, manuscript string) (string, error) {
	transitionPrompt := fmt.Sprintf(`
Here's a story that was assembled from separate parts:

%s

Please improve the transitions between sections to make it flow more naturally. 
Add bridging sentences or paragraphs where needed, but don't change the main 
content. Just make it read like one cohesive story instead of separate pieces 
stuck together.

Return the story with improved transitions:`, 
		truncateString(manuscript, 8000))

	improved, err := a.agent.Execute(ctx, transitionPrompt, nil)
	if err != nil {
		return manuscript, err
	}

	return improved, nil
}

func (a *FlowingAssembler) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	writerData, ok := input.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("input data must be writer output")
	}

	// Check for either manuscript or scenes
	if _, hasManuscript := writerData["manuscript"]; hasManuscript {
		return nil
	}
	
	if _, hasScenes := writerData["scenes"]; hasScenes {
		return nil
	}

	return fmt.Errorf("no manuscript or scenes found in writer output")
}

func (a *FlowingAssembler) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil // Assembly output is flexible
}