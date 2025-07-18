package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
)

// TargetedWriter writes individual scenes with specific word targets and full context
type TargetedWriter struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
}

type SceneOutput struct {
	ChapterNumber int    `json:"chapter_number"`
	SceneNumber   int    `json:"scene_number"`
	Content       string `json:"content"`
	ActualWords   int    `json:"actual_words"`
	TargetWords   int    `json:"target_words"`
	Title         string `json:"title"`
}

type NovelProgress struct {
	Scenes           map[string]SceneOutput `json:"scenes"`
	CompletedChapters []int                  `json:"completed_chapters"`
	TotalWordsSoFar   int                    `json:"total_words_so_far"`
	TargetWords       int                    `json:"target_words"`
	NovelPlan         NovelPlan              `json:"novel_plan"`
}

func NewTargetedWriter(agent core.Agent, storage core.Storage) *TargetedWriter {
	return &TargetedWriter{
		BasePhase: NewBasePhase("Targeted Writing", 90*time.Minute),
		agent:     agent,
		storage:   storage,
	}
}

func (w *TargetedWriter) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting targeted scene-by-scene writing",
		"phase", w.Name())

	// Extract novel plan
	plan, ok := input.Data.(NovelPlan)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("input must be a NovelPlan from systematic planner")
	}

	// Calculate target metrics for systematic writing
	targetWords := 20000 // Default systematic target
	wordsPerChapter := targetWords / len(plan.Chapters)
	
	slog.Info("Beginning systematic scene writing",
		"total_chapters", len(plan.Chapters),
		"target_words", targetWords,
		"words_per_chapter", wordsPerChapter)

	// Initialize progress tracking
	progress := NovelProgress{
		Scenes:            make(map[string]SceneOutput),
		CompletedChapters: make([]int, 0),
		TotalWordsSoFar:   0,
		TargetWords:       targetWords,
		NovelPlan:         plan,
	}

	// Write each scene systematically
	for _, chapter := range plan.Chapters {
		// Calculate target words for this chapter
		chapterTargetWords := wordsPerChapter
		sceneTargetWords := chapterTargetWords / len(chapter.Scenes)
		
		slog.Info("Writing chapter scenes",
			"chapter", chapter.Number,
			"chapter_title", chapter.Title,
			"target_words", chapterTargetWords,
			"scenes", len(chapter.Scenes))

		chapterWordCount := 0

		for _, scene := range chapter.Scenes {
			sceneKey := fmt.Sprintf("ch%d_sc%d", chapter.Number, scene.SceneNum)
			
			slog.Info("Writing individual scene",
				"chapter", chapter.Number,
				"scene", scene.SceneNum,
				"target_words", sceneTargetWords,
				"summary", truncateString(scene.Summary, 50))

			// Write the scene with full context
			sceneOutput, err := w.writeScene(ctx, chapter, scene, plan, progress, sceneTargetWords)
			if err != nil {
				return core.PhaseOutput{}, fmt.Errorf("writing scene %s: %w", sceneKey, err)
			}

			// Track progress
			progress.Scenes[sceneKey] = sceneOutput
			chapterWordCount += sceneOutput.ActualWords
			progress.TotalWordsSoFar += sceneOutput.ActualWords

			// Save individual scene
			sceneFile := fmt.Sprintf("scenes/%s.md", sceneKey)
			if err := w.storage.Save(ctx, sceneFile, []byte(sceneOutput.Content)); err != nil {
				slog.Warn("Failed to save scene", "scene", sceneKey, "error", err)
			}

			slog.Info("Scene completed",
				"scene", sceneKey,
				"actual_words", sceneOutput.ActualWords,
				"target_words", sceneTargetWords,
				"chapter_progress", fmt.Sprintf("%d/%d words", chapterWordCount, chapterTargetWords))
		}

		progress.CompletedChapters = append(progress.CompletedChapters, chapter.Number)

		slog.Info("Chapter completed",
			"chapter", chapter.Number,
			"actual_words", chapterWordCount,
			"target_words", chapterTargetWords,
			"total_progress", fmt.Sprintf("%d/%d words (%.1f%%)", 
				progress.TotalWordsSoFar, targetWords,
				float64(progress.TotalWordsSoFar)/float64(targetWords)*100))
	}

	slog.Info("Targeted writing completed",
		"phase", w.Name(),
		"total_scenes", len(progress.Scenes),
		"total_words", progress.TotalWordsSoFar,
		"target_words", targetWords,
		"accuracy", fmt.Sprintf("%.1f%%", float64(progress.TotalWordsSoFar)/float64(targetWords)*100))

	return core.PhaseOutput{Data: progress}, nil
}

func (w *TargetedWriter) writeScene(ctx context.Context, chapter Chapter, scene Scene, plan NovelPlan, progress NovelProgress, targetWords int) (SceneOutput, error) {
	// Build comprehensive context for the scene
	contextPrompt := w.buildSceneContext(chapter, scene, plan, progress)
	
	// Create targeted writing prompt
	writingPrompt := fmt.Sprintf(`%s

Now write this specific scene. Requirements:
- Target length: %d words (this is important for pacing)
- Scene objective: %s
- Make it engaging and well-written
- Include dialogue, action, and description as appropriate
- End at a natural stopping point that flows to the next scene

Write the scene now:`, contextPrompt, targetWords, scene.Summary)

	slog.Debug("Scene writing prompt prepared",
		"chapter", chapter.Number,
		"scene", scene.SceneNum,
		"prompt_length", len(writingPrompt),
		"target_words", targetWords)

	// Generate scene content
	content, err := w.agent.Execute(ctx, writingPrompt, nil)
	if err != nil {
		return SceneOutput{}, err
	}

	// Count actual words
	actualWords := w.countWords(content)

	// Check if we need length adjustment
	if actualWords < targetWords*3/4 {
		// Too short - ask for expansion
		content, err = w.expandScene(ctx, content, targetWords, actualWords)
		if err != nil {
			slog.Warn("Failed to expand scene", "error", err)
		} else {
			actualWords = w.countWords(content)
		}
	} else if actualWords > targetWords*5/4 {
		// Too long - ask for tightening
		content, err = w.tightenScene(ctx, content, targetWords, actualWords)
		if err != nil {
			slog.Warn("Failed to tighten scene", "error", err)
		} else {
			actualWords = w.countWords(content)
		}
	}

	return SceneOutput{
		ChapterNumber: chapter.Number,
		SceneNumber:   scene.SceneNum,
		Content:       content,
		ActualWords:   actualWords,
		TargetWords:   targetWords,
		Title:         fmt.Sprintf("%s - Scene %d", chapter.Title, scene.SceneNum),
	}, nil
}

func (w *TargetedWriter) buildSceneContext(chapter Chapter, scene Scene, plan NovelPlan, progress NovelProgress) string {
	context := fmt.Sprintf(`NOVEL CONTEXT:
Title: %s
Synopsis: %s
Target Length: %d words total

CHARACTERS:
`, plan.Title, plan.Synopsis, progress.TargetWords)

	for _, char := range plan.MainCharacters {
		context += fmt.Sprintf("- %s (%s): %s\n", char.Name, char.Role, char.Description)
	}

	context += fmt.Sprintf(`
STORY THEMES:
%s

CURRENT CHAPTER (%d of %d):
Title: %s
Summary: %s

CURRENT SCENE (%d of %d in this chapter):
Title: %s
Summary: %s

`, strings.Join(plan.Themes, ", "),
	chapter.Number, len(plan.Chapters), chapter.Title, chapter.Summary,
	scene.SceneNum, len(chapter.Scenes), scene.Title, scene.Summary)

	// Add story context from previous scenes
	if progress.TotalWordsSoFar > 0 {
		context += fmt.Sprintf("STORY PROGRESS SO FAR: %d words written\n", progress.TotalWordsSoFar)
		
		// Add context from recent scenes
		recentContext := w.getRecentSceneContext(progress, chapter.Number, scene.SceneNum)
		if recentContext != "" {
			context += fmt.Sprintf("RECENT STORY CONTEXT:\n%s\n", recentContext)
		}
	}

	return context
}

func (w *TargetedWriter) getRecentSceneContext(progress NovelProgress, currentChapter, currentScene int) string {
	// Get last 1-2 scenes for context
	context := ""
	
	// Look for the most recent completed scene
	if currentScene > 1 {
		// Previous scene in same chapter
		prevKey := fmt.Sprintf("ch%d_sc%d", currentChapter, currentScene-1)
		if scene, exists := progress.Scenes[prevKey]; exists {
			excerpt := truncateString(scene.Content, 200)
			context += fmt.Sprintf("Previous scene ending: ...%s\n", excerpt)
		}
	} else if currentChapter > 1 {
		// Last scene of previous chapter
		prevChapterLastScene := fmt.Sprintf("ch%d_sc3", currentChapter-1) // Assuming 3 scenes per chapter
		if scene, exists := progress.Scenes[prevChapterLastScene]; exists {
			excerpt := truncateString(scene.Content, 200)
			context += fmt.Sprintf("Previous chapter ending: ...%s\n", excerpt)
		}
	}

	return context
}

func (w *TargetedWriter) expandScene(ctx context.Context, content string, targetWords, actualWords int) (string, error) {
	expandPrompt := fmt.Sprintf(`
This scene is currently %d words but needs to be closer to %d words.

Current scene:
%s

Please expand this scene to reach the target length. Add:
- More descriptive details
- Additional dialogue
- Character thoughts or reactions
- Sensory details
- More development of the action

Keep the same basic events and flow, just make it richer and more detailed:`,
		actualWords, targetWords, content)

	return w.agent.Execute(ctx, expandPrompt, nil)
}

func (w *TargetedWriter) tightenScene(ctx context.Context, content string, targetWords, actualWords int) (string, error) {
	tightenPrompt := fmt.Sprintf(`
This scene is currently %d words but should be closer to %d words.

Current scene:
%s

Please tighten this scene to reach the target length. Remove:
- Unnecessary descriptions
- Redundant dialogue
- Overly long passages

Keep all the important story elements, just make it more concise and punchy:`,
		actualWords, targetWords, content)

	return w.agent.Execute(ctx, tightenPrompt, nil)
}

func (w *TargetedWriter) countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func (w *TargetedWriter) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	_, ok := input.Data.(NovelPlan)
	if !ok {
		return fmt.Errorf("input must be a NovelPlan from systematic planner")
	}
	return nil
}

func (w *TargetedWriter) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil
}