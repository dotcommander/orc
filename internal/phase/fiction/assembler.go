package fiction

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
	
)

type Assembler struct {
	BasePhase
	storage      core.Storage
	validator    core.Validator
	errorFactory core.ErrorFactory
}

func NewAssembler(storage core.Storage) *Assembler {
	return &Assembler{
		BasePhase:    NewBasePhase("Assembly", 30*time.Second),
		storage:      storage,
		validator:    core.NewBaseValidator("Assembly"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func NewAssemblerWithTimeout(storage core.Storage, timeout time.Duration) *Assembler {
	return &Assembler{
		BasePhase:    NewBasePhase("Assembly", timeout),
		storage:      storage,
		validator:    core.NewBaseValidator("Assembly"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func (a *Assembler) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	slog.Debug("Validating assembler input",
		"phase", a.Name(),
		"request_length", len(input.Request),
		"has_data", input.Data != nil,
	)
	
	// First run consolidated validation
	if err := a.validator.ValidateRequired("request", input.Request, "input"); err != nil {
		slog.Error("Assembler request validation failed",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "data", 
			"input data must be a map containing plan and scenes", fmt.Sprintf("%T", input.Data))
		slog.Error("Assembler input type validation failed",
			"phase", a.Name(),
			"expected_type", "map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return err
	}
	
	plan, ok := data["plan"].(NovelPlan)
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "plan", 
			"plan data missing from input", "missing")
		slog.Error("Plan data missing from assembler input",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	scenes, ok := data["scenes"].([]SceneResult)
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "scenes", 
			"scenes data missing from input", "missing")
		slog.Error("Scenes data missing from assembler input",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate scenes exist
	if len(scenes) == 0 {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "scenes", 
			"no scenes to assemble", scenes)
		slog.Error("No scenes to assemble",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate plan has chapters matching scenes
	if len(plan.Chapters) == 0 {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "chapters", 
			"plan contains no chapters", plan.Chapters)
		slog.Error("Plan contains no chapters",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	slog.Debug("Assembler input validation successful",
		"phase", a.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
		"scene_count", len(scenes),
	)
	
	return nil
}

func (a *Assembler) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	slog.Debug("Validating assembler output",
		"phase", a.Name(),
		"has_data", output.Data != nil,
	)
	
	// First run consolidated validation
	if output.Data == nil {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "data", "output data cannot be nil", nil)
		slog.Error("Assembler output data is nil",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	if err := a.validator.ValidateJSON("data", output.Data, "output"); err != nil {
		slog.Error("Assembler output JSON validation failed",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	// Check if output is a map (which is what we return)
	if outputMap, ok := output.Data.(map[string]interface{}); ok {
		manuscript, ok := outputMap["manuscript"].(string)
		if !ok {
			err := a.errorFactory.NewValidationError(a.Name(), "output", "manuscript", 
				"manuscript missing from output map", "missing")
			slog.Error("Manuscript missing from output map",
				"phase", a.Name(),
				"error", err,
			)
			return err
		}
		
		// Validate manuscript has content
		if err := a.validator.ValidateRequired("manuscript", manuscript, "output"); err != nil {
			slog.Error("Manuscript validation failed",
				"phase", a.Name(),
				"error", err,
			)
			return err
		}
		
		// Validate manuscript has minimum length
		if len(manuscript) < 500 {
			err := a.errorFactory.NewValidationError(a.Name(), "output", "manuscript", 
				"assembled manuscript appears too short", len(manuscript))
			slog.Error("Assembled manuscript too short",
				"phase", a.Name(),
				"length", len(manuscript),
				"minimum", 500,
				"error", err,
			)
			return err
		}
		
		slog.Debug("Assembler output validation successful",
			"phase", a.Name(),
			"manuscript_length", len(manuscript),
		)
	} else {
		// Legacy check for string output
		manuscript, ok := output.Data.(string)
		if !ok {
			err := a.errorFactory.NewValidationError(a.Name(), "output", "data", 
				"output data must be a string containing the manuscript or a map with manuscript", fmt.Sprintf("%T", output.Data))
			slog.Error("Assembler output type validation failed",
				"phase", a.Name(),
				"expected_types", "string or map[string]interface{}",
				"actual_type", fmt.Sprintf("%T", output.Data),
				"error", err,
			)
			return err
		}
		
		// Validate manuscript has content
		if err := a.validator.ValidateRequired("manuscript", manuscript, "output"); err != nil {
			slog.Error("Manuscript validation failed",
				"phase", a.Name(),
				"error", err,
			)
			return err
		}
		
		// Validate manuscript has minimum length
		if len(manuscript) < 500 {
			err := a.errorFactory.NewValidationError(a.Name(), "output", "manuscript", 
				"assembled manuscript appears too short", len(manuscript))
			slog.Error("Assembled manuscript too short",
				"phase", a.Name(),
				"length", len(manuscript),
				"minimum", 500,
				"error", err,
			)
			return err
		}
		
		slog.Debug("Assembler output validation successful",
			"phase", a.Name(),
			"manuscript_length", len(manuscript),
		)
	}
	
	return nil
}

func (a *Assembler) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting assembler execution",
		"phase", a.Name(),
	)
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("invalid input data")
		slog.Error("Assembler input data type error",
			"phase", a.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	plan, ok := data["plan"].(NovelPlan)
	if !ok {
		err := fmt.Errorf("missing plan in input")
		slog.Error("Plan missing from assembler input",
			"phase", a.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	scenes, ok := data["scenes"].([]SceneResult)
	if !ok {
		err := fmt.Errorf("missing scenes in input")
		slog.Error("Scenes missing from assembler input",
			"phase", a.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	slog.Debug("Sorting scenes for assembly",
		"phase", a.Name(),
		"scene_count", len(scenes),
	)
	
	sort.Slice(scenes, func(i, j int) bool {
		if scenes[i].ChapterNum != scenes[j].ChapterNum {
			return scenes[i].ChapterNum < scenes[j].ChapterNum
		}
		return scenes[i].SceneNum < scenes[j].SceneNum
	})
	
	slog.Info("Assembling manuscript",
		"phase", a.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
		"scene_count", len(scenes),
	)
	
	var manuscript strings.Builder
	
	manuscript.WriteString(fmt.Sprintf("# %s\n\n", plan.Title))
	manuscript.WriteString(fmt.Sprintf("*%s*\n\n", plan.Logline))
	manuscript.WriteString("---\n\n")
	
	currentChapter := 0
	sceneCountByChapter := make(map[int]int)
	
	for _, scene := range scenes {
		if scene.ChapterNum != currentChapter {
			currentChapter = scene.ChapterNum
			if currentChapter <= len(plan.Chapters) {
				chapter := plan.Chapters[currentChapter-1]
				manuscript.WriteString(fmt.Sprintf("\n## Chapter %d: %s\n\n", currentChapter, chapter.Title))
				slog.Debug("Starting new chapter in manuscript",
					"phase", a.Name(),
					"chapter_num", currentChapter,
					"chapter_title", chapter.Title,
				)
			}
		}
		
		manuscript.WriteString(scene.Content)
		manuscript.WriteString("\n\n")
		sceneCountByChapter[scene.ChapterNum]++
	}
	
	// Log chapter statistics
	for chapterNum, count := range sceneCountByChapter {
		slog.Debug("Chapter scene count",
			"phase", a.Name(),
			"chapter_num", chapterNum,
			"scene_count", count,
		)
	}
	
	manuscriptData := []byte(manuscript.String())
	
	slog.Debug("Saving assembled manuscript",
		"phase", a.Name(),
		"file", "manuscript.md",
		"size", len(manuscriptData),
	)
	
	if err := a.storage.Save(ctx, "manuscript.md", manuscriptData); err != nil {
		slog.Error("Failed to save manuscript",
			"phase", a.Name(),
			"file", "manuscript.md",
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("saving manuscript: %w", err)
	}
	
	slog.Info("Assembler execution completed successfully",
		"phase", a.Name(),
		"manuscript_length", len(manuscriptData),
		"chapter_count", len(sceneCountByChapter),
	)
	
	// Return manuscript in a map to match Critic phase expectations
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"manuscript": manuscript.String(),
		},
	}, nil
}