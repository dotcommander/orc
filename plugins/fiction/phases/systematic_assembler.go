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

// SystematicAssembler creates the final polished novel from editorial output
type SystematicAssembler struct {
	BasePhase
	storage core.Storage
}

type CompleteNovel struct {
	Metadata        NovelMetadata    `json:"metadata"`
	FullManuscript  string          `json:"full_manuscript"`
	Chapters        []ChapterOutput `json:"chapters"`
	Statistics      NovelStatistics `json:"statistics"`
	EditorialReport EditorialReport `json:"editorial_report"`
}

type NovelMetadata struct {
	Title           string    `json:"title"`
	Premise         string    `json:"premise"`
	WordCount       int       `json:"word_count"`
	ChapterCount    int       `json:"chapter_count"`
	SceneCount      int       `json:"scene_count"`
	GeneratedDate   time.Time `json:"generated_date"`
	EstimatedReading int      `json:"estimated_reading_minutes"`
}

type ChapterOutput struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	WordCount   int    `json:"word_count"`
	SceneCount  int    `json:"scene_count"`
}

type NovelStatistics struct {
	TargetWords       int     `json:"target_words"`
	ActualWords       int     `json:"actual_words"`
	WordCountAccuracy float64 `json:"word_count_accuracy"`
	AverageChapterLength int  `json:"average_chapter_length"`
	AverageSceneLength   int  `json:"average_scene_length"`
	QualityScore         float64 `json:"quality_score"`
}

type EditorialReport struct {
	PassesConducted    int      `json:"passes_conducted"`
	ImprovementsMade   []string `json:"improvements_made"`
	WordAdjustments    int      `json:"word_adjustments"`
	QualityMetrics     QualityMetrics `json:"quality_metrics"`
	EditorialNotes     string   `json:"editorial_notes"`
}

func NewSystematicAssembler(storage core.Storage) *SystematicAssembler {
	return &SystematicAssembler{
		BasePhase: NewBasePhase("Systematic Assembly", 15*time.Minute),
		storage:   storage,
	}
}

func (a *SystematicAssembler) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting systematic assembly of final novel",
		"phase", a.Name())

	// Extract final novel from contextual editor
	finalNovel, ok := input.Data.(FinalNovel)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("input must be FinalNovel from contextual editor")
	}

	slog.Info("Assembling final novel",
		"title", finalNovel.Title,
		"total_words", finalNovel.TotalWords,
		"target_words", finalNovel.TargetWords,
		"chapters", len(finalNovel.Chapters))

	// Create comprehensive novel package
	completeNovel := a.assembleCompleteNovel(finalNovel)

	// Generate formatted manuscript
	formattedManuscript := a.formatManuscript(completeNovel)
	
	// Save all outputs
	if err := a.saveAllOutputs(ctx, completeNovel, formattedManuscript, input.SessionID); err != nil {
		slog.Warn("Failed to save some outputs", "error", err)
	}

	// Generate final report
	report := a.generateFinalReport(completeNovel)

	slog.Info("Systematic assembly completed",
		"phase", a.Name(),
		"final_words", completeNovel.Statistics.ActualWords,
		"accuracy", fmt.Sprintf("%.1f%%", completeNovel.Statistics.WordCountAccuracy*100),
		"quality_score", completeNovel.Statistics.QualityScore)

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"novel":     completeNovel,
			"manuscript": formattedManuscript,
			"report":    report,
		},
	}, nil
}

func (a *SystematicAssembler) assembleCompleteNovel(finalNovel FinalNovel) CompleteNovel {
	// Extract chapter outputs
	chapters := make([]ChapterOutput, len(finalNovel.Chapters))
	totalScenes := 0
	
	for i, chapter := range finalNovel.Chapters {
		// Count scenes in chapter (estimate based on content breaks)
		sceneCount := a.estimateSceneCount(chapter.EditedContent)
		totalScenes += sceneCount
		
		chapters[i] = ChapterOutput{
			Number:     chapter.ChapterNumber,
			Title:      fmt.Sprintf("Chapter %d", chapter.ChapterNumber),
			Content:    chapter.EditedContent,
			WordCount:  chapter.EditedWords,
			SceneCount: sceneCount,
		}
	}

	// Calculate statistics
	avgChapterLength := 0
	if len(chapters) > 0 {
		avgChapterLength = finalNovel.TotalWords / len(chapters)
	}
	
	avgSceneLength := 0
	if totalScenes > 0 {
		avgSceneLength = finalNovel.TotalWords / totalScenes
	}

	accuracy := float64(finalNovel.TotalWords) / float64(finalNovel.TargetWords)
	if accuracy > 1.0 {
		accuracy = 2.0 - accuracy // Penalize being over as much as under
	}

	return CompleteNovel{
		Metadata: NovelMetadata{
			Title:           finalNovel.Title,
			Premise:         "Generated novel", // TODO: Extract from plan
			WordCount:       finalNovel.TotalWords,
			ChapterCount:    len(chapters),
			SceneCount:      totalScenes,
			GeneratedDate:   time.Now(),
			EstimatedReading: EstimateReadingTime(finalNovel.TotalWords),
		},
		FullManuscript: finalNovel.FullManuscript,
		Chapters:       chapters,
		Statistics: NovelStatistics{
			TargetWords:       finalNovel.TargetWords,
			ActualWords:       finalNovel.TotalWords,
			WordCountAccuracy: accuracy,
			AverageChapterLength: avgChapterLength,
			AverageSceneLength:   avgSceneLength,
			QualityScore:        finalNovel.QualityMetrics.OverallRating,
		},
		EditorialReport: EditorialReport{
			PassesConducted:  len(finalNovel.EditorialPasses),
			ImprovementsMade: a.extractImprovements(finalNovel.EditorialPasses),
			WordAdjustments:  a.countWordAdjustments(finalNovel.EditorialPasses),
			QualityMetrics:   finalNovel.QualityMetrics,
			EditorialNotes:   a.summarizeEditorialNotes(finalNovel.EditorialPasses),
		},
	}
}

func (a *SystematicAssembler) formatManuscript(novel CompleteNovel) string {
	formatted := fmt.Sprintf(`# %s

*A %d-word novel generated by systematic AI orchestration*

---

## Table of Contents

`, novel.Metadata.Title, novel.Metadata.WordCount)

	// Add table of contents
	for _, chapter := range novel.Chapters {
		formatted += fmt.Sprintf("- %s (%d words)\n", chapter.Title, chapter.WordCount)
	}

	formatted += "\n---\n\n"

	// Add full content
	for i, chapter := range novel.Chapters {
		if i > 0 {
			formatted += "\n\n---\n\n"
		}
		formatted += fmt.Sprintf("## %s\n\n%s", chapter.Title, chapter.Content)
	}

	// Add statistics footer
	formatted += fmt.Sprintf(`

---

## Generation Statistics

- **Total Words:** %d (target: %d)
- **Accuracy:** %.1f%%
- **Chapters:** %d
- **Estimated Reading Time:** %d minutes
- **Quality Score:** %.1f/10
- **Editorial Passes:** %d

*Generated with systematic AI orchestration for optimal length and quality.*
`,
		novel.Statistics.ActualWords,
		novel.Statistics.TargetWords,
		novel.Statistics.WordCountAccuracy*100,
		novel.Metadata.ChapterCount,
		novel.Metadata.EstimatedReading,
		novel.Statistics.QualityScore*10,
		novel.EditorialReport.PassesConducted)

	return formatted
}

func (a *SystematicAssembler) saveAllOutputs(ctx context.Context, novel CompleteNovel, manuscript, sessionID string) error {
	// Save formatted manuscript
	if err := a.storage.Save(ctx, "complete_novel.md", []byte(manuscript)); err != nil {
		return fmt.Errorf("saving manuscript: %w", err)
	}

	// Save novel metadata
	metadata, err := json.MarshalIndent(novel, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := a.storage.Save(ctx, "novel_metadata.json", metadata); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	// Save individual chapters
	for _, chapter := range novel.Chapters {
		filename := fmt.Sprintf("chapters/chapter_%02d.md", chapter.Number)
		chapterContent := fmt.Sprintf("# %s\n\n%s", chapter.Title, chapter.Content)
		if err := a.storage.Save(ctx, filename, []byte(chapterContent)); err != nil {
			slog.Warn("Failed to save chapter", "chapter", chapter.Number, "error", err)
		}
	}

	// Save statistics
	stats, err := json.MarshalIndent(novel.Statistics, "", "  ")
	if err == nil {
		a.storage.Save(ctx, "generation_statistics.json", stats)
	}

	return nil
}

func (a *SystematicAssembler) generateFinalReport(novel CompleteNovel) string {
	return fmt.Sprintf(`# Novel Generation Report

## Summary
- **Title:** %s
- **Final Word Count:** %d words
- **Target Word Count:** %d words
- **Accuracy:** %.1f%%
- **Quality Score:** %.1f/10

## Structure
- **Chapters:** %d
- **Average Chapter Length:** %d words
- **Estimated Reading Time:** %d minutes

## Editorial Process
- **Passes Conducted:** %d
- **Word Adjustments:** %d
- **Character Consistency:** %.1f/10
- **Plot Cohesion:** %.1f/10
- **Pacing Quality:** %.1f/10

## Improvements Made
%s

## Quality Assessment
This novel was generated using systematic AI orchestration with:
1. **Strategic Planning** - Word-count aware story architecture
2. **Targeted Writing** - Scene-by-scene composition with specific targets
3. **Contextual Editing** - Full-novel awareness for consistency and flow

The final work achieves %.1f%% word count accuracy and demonstrates strong structural integrity.

---
*Generated by Systematic AI Novel Orchestration*`,
		novel.Metadata.Title,
		novel.Statistics.ActualWords,
		novel.Statistics.TargetWords,
		novel.Statistics.WordCountAccuracy*100,
		novel.Statistics.QualityScore*10,
		novel.Metadata.ChapterCount,
		novel.Statistics.AverageChapterLength,
		novel.Metadata.EstimatedReading,
		novel.EditorialReport.PassesConducted,
		novel.EditorialReport.WordAdjustments,
		novel.EditorialReport.QualityMetrics.CharacterConsistency*10,
		novel.EditorialReport.QualityMetrics.PlotCohesion*10,
		novel.EditorialReport.QualityMetrics.PacingQuality*10,
		a.formatImprovements(novel.EditorialReport.ImprovementsMade),
		novel.Statistics.WordCountAccuracy*100)
}

func (a *SystematicAssembler) estimateSceneCount(content string) int {
	// Simple heuristic: count paragraph breaks or scene transitions
	lines := strings.Split(content, "\n")
	scenes := 1 // At least one scene
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// Empty line might indicate scene break
			continue
		}
		if strings.Contains(line, "***") || strings.Contains(line, "---") {
			scenes++
		}
	}
	
	// Cap at reasonable number
	if scenes > 5 {
		scenes = 5
	}
	
	return scenes
}

func (a *SystematicAssembler) extractImprovements(passes []EditorialPass) []string {
	improvements := make([]string, 0)
	
	for _, pass := range passes {
		for _, edit := range pass.ChapterEdits {
			improvements = append(improvements, edit.ImprovementsMade...)
		}
	}
	
	// Deduplicate
	unique := make(map[string]bool)
	result := make([]string, 0)
	
	for _, improvement := range improvements {
		if !unique[improvement] {
			unique[improvement] = true
			result = append(result, improvement)
		}
	}
	
	return result
}

func (a *SystematicAssembler) countWordAdjustments(passes []EditorialPass) int {
	total := 0
	for _, pass := range passes {
		total += len(pass.WordCountAdjustments)
	}
	return total
}

func (a *SystematicAssembler) summarizeEditorialNotes(passes []EditorialPass) string {
	notes := make([]string, 0)
	for _, pass := range passes {
		if pass.OverallNotes != "" {
			notes = append(notes, fmt.Sprintf("Pass %d: %s", pass.PassNumber, truncateString(pass.OverallNotes, 200)))
		}
	}
	return strings.Join(notes, "\n")
}

func (a *SystematicAssembler) formatImprovements(improvements []string) string {
	if len(improvements) == 0 {
		return "- No specific improvements recorded"
	}
	
	formatted := ""
	for _, improvement := range improvements {
		formatted += fmt.Sprintf("- %s\n", improvement)
	}
	return formatted
}

func (a *SystematicAssembler) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	_, ok := input.Data.(FinalNovel)
	if !ok {
		return fmt.Errorf("input must be FinalNovel from contextual editor")
	}
	return nil
}

func (a *SystematicAssembler) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil
}