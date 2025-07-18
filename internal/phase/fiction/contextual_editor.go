package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// ContextualEditor improves the entire novel chapter by chapter with full story context
type ContextualEditor struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
}

type EditorialPass struct {
	PassNumber    int                    `json:"pass_number"`
	PassType      string                 `json:"pass_type"`
	ChapterEdits  map[int]ChapterEdit    `json:"chapter_edits"`
	OverallNotes  string                 `json:"overall_notes"`
	WordCountAdjustments []WordAdjustment `json:"word_count_adjustments"`
}

type ChapterEdit struct {
	ChapterNumber    int    `json:"chapter_number"`
	OriginalContent  string `json:"original_content"`
	EditedContent    string `json:"edited_content"`
	OriginalWords    int    `json:"original_words"`
	EditedWords      int    `json:"edited_words"`
	EditorialNotes   string `json:"editorial_notes"`
	ImprovementsMade []string `json:"improvements_made"`
}

type WordAdjustment struct {
	Chapter      int    `json:"chapter"`
	TargetWords  int    `json:"target_words"`
	ActualWords  int    `json:"actual_words"`
	Adjustment   int    `json:"adjustment"`
	Reason       string `json:"reason"`
}

type FinalNovel struct {
	Title            string              `json:"title"`
	FullManuscript   string              `json:"full_manuscript"`
	Chapters         []ChapterEdit       `json:"chapters"`
	TotalWords       int                 `json:"total_words"`
	TargetWords      int                 `json:"target_words"`
	EditorialPasses  []EditorialPass     `json:"editorial_passes"`
	QualityMetrics   QualityMetrics      `json:"quality_metrics"`
}

type QualityMetrics struct {
	WordCountAccuracy    float64 `json:"word_count_accuracy"`
	CharacterConsistency float64 `json:"character_consistency"`
	PlotCohesion        float64 `json:"plot_cohesion"`
	PacingQuality       float64 `json:"pacing_quality"`
	OverallRating       float64 `json:"overall_rating"`
}

func NewContextualEditor(agent core.Agent, storage core.Storage) *ContextualEditor {
	return &ContextualEditor{
		BasePhase: NewBasePhase("Contextual Editing", 60*time.Minute),
		agent:     agent,
		storage:   storage,
	}
}

func (e *ContextualEditor) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting contextual editing with full novel awareness",
		"phase", e.Name())

	// Extract novel progress from targeted writer
	progress, ok := input.Data.(NovelProgress)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("input must be NovelProgress from targeted writer")
	}

	slog.Info("Beginning editorial process",
		"total_scenes", len(progress.Scenes),
		"total_words", progress.TotalWordsSoFar,
		"target_words", progress.TargetWords,
		"chapters", len(progress.CompletedChapters))

	// Assemble full manuscript for context
	fullManuscript := e.assembleFullManuscript(progress)
	
	slog.Info("Full manuscript assembled for editorial review",
		"manuscript_length", len(fullManuscript),
		"manuscript_words", e.countWords(fullManuscript))

	// Editorial Pass 1: Continuity and Character Consistency
	pass1, err := e.editorialPassContinuity(ctx, fullManuscript, progress)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("continuity pass: %w", err)
	}

	// Editorial Pass 2: Pacing and Flow Enhancement
	pass2, err := e.editorialPassPacing(ctx, pass1, progress)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("pacing pass: %w", err)
	}

	// Editorial Pass 3: Word Count Optimization
	finalPass, err := e.editorialPassWordCount(ctx, pass2, progress)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("word count pass: %w", err)
	}

	// Assess final quality
	qualityMetrics := e.assessQuality(finalPass, progress)

	// Create final novel output
	finalNovel := FinalNovel{
		Title:           progress.NovelPlan.Title,
		FullManuscript:  e.assembleFromChapterEdits(finalPass.ChapterEdits),
		Chapters:        e.extractChapterEdits(finalPass.ChapterEdits),
		TotalWords:      e.countWords(e.assembleFromChapterEdits(finalPass.ChapterEdits)),
		TargetWords:     progress.TargetWords,
		EditorialPasses: []EditorialPass{pass1, pass2, finalPass},
		QualityMetrics:  qualityMetrics,
	}

	// Save final manuscript
	if err := e.storage.Save(ctx, "final_manuscript.md", []byte(finalNovel.FullManuscript)); err != nil {
		slog.Warn("Failed to save final manuscript", "error", err)
	}

	slog.Info("Contextual editing completed",
		"phase", e.Name(),
		"final_words", finalNovel.TotalWords,
		"target_words", finalNovel.TargetWords,
		"accuracy", fmt.Sprintf("%.1f%%", qualityMetrics.WordCountAccuracy*100),
		"overall_rating", qualityMetrics.OverallRating)

	return core.PhaseOutput{Data: finalNovel}, nil
}

func (e *ContextualEditor) assembleFullManuscript(progress NovelProgress) string {
	chapters := make([]string, len(progress.NovelPlan.Chapters))
	
	for i, chapterPlan := range progress.NovelPlan.Chapters {
		chapterNum := chapterPlan.Number
		chapterContent := fmt.Sprintf("# %s\n\n", chapterPlan.Title)
		
		// Assemble scenes for this chapter
		for _, scenePlan := range chapterPlan.Scenes {
			sceneKey := fmt.Sprintf("ch%d_sc%d", chapterNum, scenePlan.SceneNum)
			if scene, exists := progress.Scenes[sceneKey]; exists {
				chapterContent += scene.Content + "\n\n"
			}
		}
		
		chapters[i] = chapterContent
	}
	
	return strings.Join(chapters, "---\n\n")
}

func (e *ContextualEditor) editorialPassContinuity(ctx context.Context, fullManuscript string, progress NovelProgress) (EditorialPass, error) {
	slog.Info("Editorial Pass 1: Continuity and Character Consistency")

	pass := EditorialPass{
		PassNumber:   1,
		PassType:     "Continuity and Character Consistency",
		ChapterEdits: make(map[int]ChapterEdit),
	}

	// First, read the entire novel and understand it
	overviewPrompt := fmt.Sprintf(`
As a professional editor, read this complete novel:

TITLE: %s
PREMISE: %s

FULL MANUSCRIPT:
%s

After reading the entire novel, provide:
1. Overall story assessment
2. Character consistency issues you notice
3. Plot continuity problems
4. Timeline or logical inconsistencies
5. Areas that need better transitions

Focus on big-picture story issues, not line editing yet.`, 
		progress.NovelPlan.Title, progress.NovelPlan.Synopsis, 
		truncateString(fullManuscript, 12000)) // Keep within token limits

	overallNotes, err := e.agent.Execute(ctx, overviewPrompt, nil)
	if err != nil {
		return pass, err
	}
	pass.OverallNotes = overallNotes

	slog.Info("Overall novel assessment completed", "notes_length", len(overallNotes))

	// Now edit each chapter with full novel context
	for _, chapter := range progress.NovelPlan.Chapters {
		chapterContent := e.extractChapterContent(fullManuscript, chapter.Number)
		
		editPrompt := fmt.Sprintf(`
You are editing Chapter %d of this novel. You have read the ENTIRE novel, so you know:

FULL STORY CONTEXT:
%s

OVERALL EDITORIAL NOTES:
%s

CURRENT CHAPTER %d ("%s"):
%s

Improve this chapter for:
1. Character consistency (personalities, voices, development)
2. Plot continuity (does it flow logically from previous chapters?)
3. Foreshadowing and callbacks (add subtle connections to other parts)
4. Transitions (smooth connection to what comes before/after)
5. Internal consistency (timeline, details, character knowledge)

Return the improved chapter, maintaining the same basic events but enhancing the storytelling:`,
			chapter.Number, truncateString(fullManuscript, 6000),
			truncateString(overallNotes, 1000), chapter.Number, chapter.Title, chapterContent)

		editedContent, err := e.agent.Execute(ctx, editPrompt, nil)
		if err != nil {
			slog.Warn("Failed to edit chapter", "chapter", chapter.Number, "error", err)
			editedContent = chapterContent // Keep original
		}

		pass.ChapterEdits[chapter.Number] = ChapterEdit{
			ChapterNumber:   chapter.Number,
			OriginalContent: chapterContent,
			EditedContent:   editedContent,
			OriginalWords:   e.countWords(chapterContent),
			EditedWords:     e.countWords(editedContent),
			EditorialNotes:  "Continuity and character consistency improvements",
			ImprovementsMade: []string{"Character consistency", "Plot continuity", "Foreshadowing"},
		}

		slog.Info("Chapter edited for continuity",
			"chapter", chapter.Number,
			"original_words", e.countWords(chapterContent),
			"edited_words", e.countWords(editedContent))
	}

	return pass, nil
}

func (e *ContextualEditor) editorialPassPacing(ctx context.Context, previousPass EditorialPass, progress NovelProgress) (EditorialPass, error) {
	slog.Info("Editorial Pass 2: Pacing and Flow Enhancement")

	pass := EditorialPass{
		PassNumber:   2,
		PassType:     "Pacing and Flow Enhancement",
		ChapterEdits: make(map[int]ChapterEdit),
	}

	// Assemble manuscript from previous pass
	manuscript := e.assembleFromChapterEdits(previousPass.ChapterEdits)

	for chapterNum, prevEdit := range previousPass.ChapterEdits {
		pacingPrompt := fmt.Sprintf(`
You are doing a pacing and flow pass on Chapter %d. You know the full story context.

FULL NOVEL CONTEXT (for pacing awareness):
%s

CHAPTER %d CURRENT VERSION:
%s

Improve this chapter for:
1. Pacing (does it move at the right speed for this point in the story?)
2. Tension and engagement (keep readers hooked)
3. Scene transitions (smooth flow between scenes)
4. Dialogue flow (natural, engaging conversations)
5. Action/description balance (right mix for pacing)

Enhance the pacing and flow while keeping the same basic content:`,
			chapterNum, truncateString(manuscript, 6000), chapterNum, prevEdit.EditedContent)

		editedContent, err := e.agent.Execute(ctx, pacingPrompt, nil)
		if err != nil {
			slog.Warn("Failed to improve pacing", "chapter", chapterNum, "error", err)
			editedContent = prevEdit.EditedContent // Keep previous version
		}

		pass.ChapterEdits[chapterNum] = ChapterEdit{
			ChapterNumber:   chapterNum,
			OriginalContent: prevEdit.EditedContent, // Previous pass version
			EditedContent:   editedContent,
			OriginalWords:   prevEdit.EditedWords,
			EditedWords:     e.countWords(editedContent),
			EditorialNotes:  "Pacing and flow improvements",
			ImprovementsMade: []string{"Pacing optimization", "Flow enhancement", "Tension building"},
		}

		slog.Info("Chapter pacing improved",
			"chapter", chapterNum,
			"words_before", prevEdit.EditedWords,
			"words_after", e.countWords(editedContent))
	}

	return pass, nil
}

func (e *ContextualEditor) editorialPassWordCount(ctx context.Context, previousPass EditorialPass, progress NovelProgress) (EditorialPass, error) {
	slog.Info("Editorial Pass 3: Word Count Optimization")

	pass := EditorialPass{
		PassNumber:            3,
		PassType:             "Word Count Optimization",
		ChapterEdits:         make(map[int]ChapterEdit),
		WordCountAdjustments: make([]WordAdjustment, 0),
	}

	// Calculate current word distribution
	currentTotal := 0
	for _, edit := range previousPass.ChapterEdits {
		currentTotal += edit.EditedWords
	}

	targetTotal := progress.TargetWords
	wordsPerChapter := targetTotal / len(progress.NovelPlan.Chapters)

	slog.Info("Word count analysis",
		"current_total", currentTotal,
		"target_total", targetTotal,
		"target_per_chapter", wordsPerChapter)

	for chapterNum, prevEdit := range previousPass.ChapterEdits {
		targetWords := wordsPerChapter
		currentWords := prevEdit.EditedWords
		adjustment := targetWords - currentWords

		var adjustedContent string
		var err error

		if adjustment > 100 { // Need to expand significantly
			expandPrompt := fmt.Sprintf(`
This chapter is currently %d words but needs to be closer to %d words (add ~%d words).

CHAPTER %d:
%s

Expand this chapter to reach the target length by:
- Adding more sensory details and atmosphere
- Developing character thoughts and emotions
- Expanding dialogue with subtext
- Adding relevant backstory or world-building
- Enhancing action sequences with more detail

Keep the same story beats but make it richer and more immersive:`,
				currentWords, targetWords, adjustment, chapterNum, prevEdit.EditedContent)

			adjustedContent, err = e.agent.Execute(ctx, expandPrompt, nil)
		} else if adjustment < -100 { // Need to tighten significantly
			tightenPrompt := fmt.Sprintf(`
This chapter is currently %d words but should be closer to %d words (cut ~%d words).

CHAPTER %d:
%s

Tighten this chapter to reach the target length by:
- Removing redundant descriptions
- Streamlining dialogue
- Cutting unnecessary scenes or beats
- Making prose more concise
- Eliminating repetitive elements

Keep all essential story elements but make it more focused:`,
				currentWords, targetWords, -adjustment, chapterNum, prevEdit.EditedContent)

			adjustedContent, err = e.agent.Execute(ctx, tightenPrompt, nil)
		} else {
			// Minor adjustment or no change needed
			adjustedContent = prevEdit.EditedContent
		}

		if err != nil {
			slog.Warn("Word count adjustment failed", "chapter", chapterNum, "error", err)
			adjustedContent = prevEdit.EditedContent
		}

		finalWords := e.countWords(adjustedContent)
		actualAdjustment := finalWords - currentWords

		pass.ChapterEdits[chapterNum] = ChapterEdit{
			ChapterNumber:   chapterNum,
			OriginalContent: prevEdit.EditedContent,
			EditedContent:   adjustedContent,
			OriginalWords:   currentWords,
			EditedWords:     finalWords,
			EditorialNotes:  fmt.Sprintf("Word count adjusted from %d to %d words", currentWords, finalWords),
			ImprovementsMade: []string{"Word count optimization", "Length adjustment"},
		}

		pass.WordCountAdjustments = append(pass.WordCountAdjustments, WordAdjustment{
			Chapter:     chapterNum,
			TargetWords: targetWords,
			ActualWords: finalWords,
			Adjustment:  actualAdjustment,
			Reason:      fmt.Sprintf("Target: %d, was: %d", targetWords, currentWords),
		})

		slog.Info("Chapter word count optimized",
			"chapter", chapterNum,
			"target", targetWords,
			"before", currentWords,
			"after", finalWords,
			"adjustment", actualAdjustment)
	}

	return pass, nil
}

func (e *ContextualEditor) assessQuality(finalPass EditorialPass, progress NovelProgress) QualityMetrics {
	finalTotal := 0
	for _, edit := range finalPass.ChapterEdits {
		finalTotal += edit.EditedWords
	}

	wordCountAccuracy := float64(finalTotal) / float64(progress.TargetWords)
	if wordCountAccuracy > 1.0 {
		wordCountAccuracy = 2.0 - wordCountAccuracy // Penalize being over as much as under
	}

	return QualityMetrics{
		WordCountAccuracy:    wordCountAccuracy,
		CharacterConsistency: 0.9, // TODO: Implement AI assessment
		PlotCohesion:        0.85, // TODO: Implement AI assessment
		PacingQuality:       0.8,  // TODO: Implement AI assessment
		OverallRating:       (wordCountAccuracy + 0.9 + 0.85 + 0.8) / 4.0,
	}
}

func (e *ContextualEditor) extractChapterContent(fullManuscript string, chapterNumber int) string {
	// Extract content for specific chapter from full manuscript
	// This is a simplified implementation
	lines := strings.Split(fullManuscript, "\n")
	chapterStart := -1
	chapterEnd := len(lines)

	for i, line := range lines {
		if strings.Contains(line, fmt.Sprintf("Chapter %d", chapterNumber)) {
			chapterStart = i
		} else if chapterStart >= 0 && strings.Contains(line, "# Chapter") && !strings.Contains(line, fmt.Sprintf("Chapter %d", chapterNumber)) {
			chapterEnd = i
			break
		}
	}

	if chapterStart >= 0 {
		return strings.Join(lines[chapterStart:chapterEnd], "\n")
	}
	return fmt.Sprintf("Chapter %d content not found", chapterNumber)
}

func (e *ContextualEditor) assembleFromChapterEdits(edits map[int]ChapterEdit) string {
	chapters := make([]string, 0, len(edits))
	
	for i := 1; i <= len(edits); i++ {
		if edit, exists := edits[i]; exists {
			chapters = append(chapters, edit.EditedContent)
		}
	}
	
	return strings.Join(chapters, "\n\n---\n\n")
}

func (e *ContextualEditor) extractChapterEdits(edits map[int]ChapterEdit) []ChapterEdit {
	result := make([]ChapterEdit, 0, len(edits))
	for i := 1; i <= len(edits); i++ {
		if edit, exists := edits[i]; exists {
			result = append(result, edit)
		}
	}
	return result
}

func (e *ContextualEditor) countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func (e *ContextualEditor) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	_, ok := input.Data.(NovelProgress)
	if !ok {
		return fmt.Errorf("input must be NovelProgress from targeted writer")
	}
	return nil
}

func (e *ContextualEditor) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil
}

