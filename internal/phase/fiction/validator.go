package fiction

import (
	"context"
	"fmt"
	"strings"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// FictionValidator provides fiction-specific validation
type FictionValidator struct {
	*core.StandardPhaseValidator
	errorFactory core.ErrorFactory
}

// NewFictionValidator creates a validator with fiction-specific rules
func NewFictionValidator(phaseName string) *FictionValidator {
	rules := core.ValidationRules{
		MinRequestLength: core.DefaultMinLength,
		MaxRequestLength: core.DefaultMaxLength,
		CustomValidators: []core.ValidationFunc{
			validateFictionContent,
		},
	}

	return &FictionValidator{
		StandardPhaseValidator: core.NewStandardPhaseValidator(phaseName, rules),
		errorFactory:           core.NewDefaultErrorFactory(),
	}
}

// ValidatePlan validates a NovelPlan
func (v *FictionValidator) ValidatePlan(plan NovelPlan) error {
	if err := core.ValidateNonEmpty(plan.Title, "title"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "title", err.Error(), plan.Title)
	}

	if err := core.ValidateNonEmpty(plan.Logline, "logline"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "logline", err.Error(), plan.Logline)
	}

	if err := core.ValidateNonEmpty(plan.Synopsis, "synopsis"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "synopsis", err.Error(), plan.Synopsis)
	}

	if len(plan.Themes) == 0 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "themes", "at least one theme is required", plan.Themes)
	}

	if len(plan.MainCharacters) == 0 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "main_characters", "at least one main character is required", plan.MainCharacters)
	}

	return nil
}

// ValidateArchitecture validates a NovelArchitecture
func (v *FictionValidator) ValidateArchitecture(arch NovelArchitecture) error {
	if len(arch.Chapters) == 0 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapters", "at least one chapter is required", arch.Chapters)
	}

	if len(arch.Chapters) > core.MaxChapterCount {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapters",
			fmt.Sprintf("too many chapters (max: %d)", core.MaxChapterCount), len(arch.Chapters))
	}

	// Validate each chapter
	for i, chapter := range arch.Chapters {
		if err := v.validateChapter(chapter, i); err != nil {
			return err
		}
	}

	return nil
}

// ValidateScene validates a scene
func (v *FictionValidator) ValidateScene(scene Scene) error {
	if err := core.ValidateNonEmpty(scene.Title, "scene.title"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "scene.title", err.Error(), scene.Title)
	}

	if err := core.ValidateNonEmpty(scene.Content, "scene.content"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "scene.content", err.Error(), scene.Content)
	}

	// Validate content length
	if err := core.ValidateStringLength(scene.Content, 100, 50000, "scene.content"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "scene.content", err.Error(), len(scene.Content))
	}

	return nil
}

// ValidateManuscript validates a complete manuscript
func (v *FictionValidator) ValidateManuscript(manuscript Manuscript) error {
	if err := core.ValidateNonEmpty(manuscript.Title, "manuscript.title"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "manuscript.title", err.Error(), manuscript.Title)
	}

	if len(manuscript.Chapters) == 0 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "manuscript.chapters", "manuscript must have at least one chapter", manuscript.Chapters)
	}

	// Validate chapter count matches architecture
	if len(manuscript.Chapters) > core.MaxChapterCount {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "manuscript.chapters",
			fmt.Sprintf("too many chapters (max: %d)", core.MaxChapterCount), len(manuscript.Chapters))
	}

	// Validate each chapter has content
	for i, chapter := range manuscript.Chapters {
		if err := core.ValidateNonEmpty(chapter.Title, fmt.Sprintf("chapter[%d].title", i)); err != nil {
			return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.title", err.Error(), chapter.Title)
		}

		if err := core.ValidateNonEmpty(chapter.Content, fmt.Sprintf("chapter[%d].content", i)); err != nil {
			return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.content", err.Error(), chapter.Content)
		}
	}

	// Calculate and validate total word count
	totalWords := 0
	for _, chapter := range manuscript.Chapters {
		totalWords += len(strings.Fields(chapter.Content))
	}

	if totalWords < 1000 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "manuscript.word_count",
			fmt.Sprintf("manuscript too short (min: 1000 words, got: %d)", totalWords), totalWords)
	}

	return nil
}

// ValidateCritique validates a critique
func (v *FictionValidator) ValidateCritique(critique DetailedCritique) error {
	if critique.OverallScore < 1 || critique.OverallScore > 10 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "overall_score",
			"overall score must be between 1 and 10", critique.OverallScore)
	}

	if err := core.ValidateNonEmpty(critique.Summary, "critique.summary"); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "critique.summary", err.Error(), critique.Summary)
	}

	// Validate category scores
	scores := []struct {
		name  string
		score int
	}{
		{"plot", critique.PlotScore},
		{"characters", critique.CharacterScore},
		{"writing", critique.WritingScore},
		{"pacing", critique.PacingScore},
		{"dialogue", critique.DialogueScore},
	}

	for _, s := range scores {
		if s.score < 1 || s.score > 10 {
			return v.errorFactory.NewValidationError(v.PhaseName, "output", s.name+"_score",
				fmt.Sprintf("%s score must be between 1 and 10", s.name), s.score)
		}
	}

	return nil
}

// validateChapter validates a single chapter
func (v *FictionValidator) validateChapter(chapter Chapter, index int) error {
	if err := core.ValidateNonEmpty(chapter.Title, fmt.Sprintf("chapter[%d].title", index)); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.title", err.Error(), chapter.Title)
	}

	if err := core.ValidateNonEmpty(chapter.Summary, fmt.Sprintf("chapter[%d].summary", index)); err != nil {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.summary", err.Error(), chapter.Summary)
	}

	if len(chapter.Scenes) == 0 {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.scenes",
			fmt.Sprintf("chapter %d must have at least one scene", index+1), chapter.Scenes)
	}

	if len(chapter.Scenes) > core.MaxSceneCount {
		return v.errorFactory.NewValidationError(v.PhaseName, "output", "chapter.scenes",
			fmt.Sprintf("chapter %d has too many scenes (max: %d)", index+1, core.MaxSceneCount), len(chapter.Scenes))
	}

	return nil
}

// validateFictionContent is a custom validator for fiction content
func validateFictionContent(ctx context.Context, data interface{}) error {
	// This can be extended with fiction-specific content validation
	// For example: checking for inappropriate content, genre consistency, etc.
	return nil
}