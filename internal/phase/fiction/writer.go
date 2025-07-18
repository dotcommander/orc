package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/phase"
)

type Writer struct {
	BasePhase
	agent        core.Agent
	storage      core.Storage
	promptPath   string
	pool         *phase.WorkerPool[Scene, SceneResult]
	validator    core.PhaseValidator
	errorFactory core.ErrorFactory
}

type WriterOption func(*Writer)

func WithWorkerPool(workers int) WriterOption {
	return func(w *Writer) {
		w.pool = phase.NewWorkerPool[Scene, SceneResult](
			phase.WithWorkers(workers),
			phase.WithBufferSize(10),
			phase.WithTimeout(5*time.Minute),
		)
	}
}

func NewWriter(agent core.Agent, storage core.Storage, promptPath string, opts ...WriterOption) *Writer {
	w := &Writer{
		BasePhase:  NewBasePhase("Writing", 30*time.Minute),
		agent:      agent,
		storage:    storage,
		promptPath: promptPath,
		pool: phase.NewWorkerPool[Scene, SceneResult](
			phase.WithWorkers(1),
			phase.WithBufferSize(10),
			phase.WithTimeout(5*time.Minute),
		),
		validator:    core.NewStandardPhaseValidator("Writing", core.ValidationRules{MinRequestLength: 10, MaxRequestLength: 10000}),
		errorFactory: core.NewDefaultErrorFactory(),
	}
	
	for _, opt := range opts {
		opt(w)
	}
	
	return w
}

func NewWriterWithTimeout(agent core.Agent, storage core.Storage, promptPath string, timeout time.Duration, opts ...WriterOption) *Writer {
	w := &Writer{
		BasePhase:  NewBasePhase("Writing", timeout),
		agent:      agent,
		storage:    storage,
		promptPath: promptPath,
		pool: phase.NewWorkerPool[Scene, SceneResult](
			phase.WithWorkers(1),
			phase.WithBufferSize(10),
			phase.WithTimeout(5*time.Minute),
		),
		validator:    core.NewStandardPhaseValidator("Writing", core.ValidationRules{MinRequestLength: 10, MaxRequestLength: 10000}),
		errorFactory: core.NewDefaultErrorFactory(),
	}
	
	for _, opt := range opts {
		opt(w)
	}
	
	return w
}

func (w *Writer) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	slog.Debug("Validating writer input",
		"phase", w.Name(),
		"request_length", len(input.Request),
		"has_data", input.Data != nil,
	)
	
	// Use consolidated validator first
	if err := w.validator.ValidateInput(ctx, input); err != nil {
		slog.Error("Writer standard input validation failed",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		err := &core.ValidationError{
			Phase:      w.Name(),
			Type:       "input",
			Field:      "data",
			Value:      fmt.Sprintf("%T", input.Data),
			Message:    "input data must be a map containing plan and architecture",
			Suggestion: "ensure data structure from previous phases is correct",
			Timestamp:  time.Now(),
		}
		slog.Error("Writer input type validation failed",
			"phase", w.Name(),
			"expected_type", "map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return err
	}
	
	plan, ok := data["plan"].(NovelPlan)
	if !ok {
		err := &core.ValidationError{
			Phase:      w.Name(),
			Type:       "input",
			Field:      "plan",
			Value:      "missing",
			Message:    "plan data missing from input",
			Suggestion: "ensure planner phase provides plan data",
			Timestamp:  time.Now(),
		}
		slog.Error("Plan data missing from writer input",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	arch, ok := data["architecture"].(NovelArchitecture)
	if !ok {
		err := &core.ValidationError{
			Phase:      w.Name(),
			Type:       "input",
			Field:      "architecture",
			Value:      "missing",
			Message:    "architecture data missing from input",
			Suggestion: "ensure architect phase provides architecture data",
			Timestamp:  time.Now(),
		}
		slog.Error("Architecture data missing from writer input",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate plan has chapters
	if len(plan.Chapters) == 0 {
		err := &core.ValidationError{
			Phase:      w.Name(),
			Type:       "input",
			Field:      "chapters",
			Value:      plan.Chapters,
			Message:    "plan contains no chapters to write",
			Suggestion: "ensure planner creates chapters for the novel",
			Timestamp:  time.Now(),
		}
		slog.Error("Plan contains no chapters",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate architecture has characters
	if len(arch.Characters) == 0 {
		err := &core.ValidationError{
			Phase:      w.Name(),
			Type:       "input",
			Field:      "characters",
			Value:      arch.Characters,
			Message:    "architecture contains no characters",
			Suggestion: "ensure architect creates character definitions",
			Timestamp:  time.Now(),
		}
		slog.Error("Architecture contains no characters",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	slog.Debug("Writer input validation successful",
		"phase", w.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
		"character_count", len(arch.Characters),
	)
	
	return nil
}

func (w *Writer) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	slog.Debug("Validating writer output",
		"phase", w.Name(),
		"has_data", output.Data != nil,
	)
	
	// Use consolidated validator first
	if err := w.validator.ValidateOutput(ctx, output); err != nil {
		slog.Error("Writer standard output validation failed",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		err := w.errorFactory.NewValidationError(w.Name(), "output", "data",
			fmt.Sprintf("output data must be a map containing scenes, got %T", output.Data), output.Data)
		slog.Error("Writer output type validation failed",
			"phase", w.Name(),
			"expected_type", "map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", output.Data),
			"error", err,
		)
		return err
	}
	
	scenes, ok := outputMap["scenes"].([]SceneResult)
	if !ok {
		err := w.errorFactory.NewValidationError(w.Name(), "output", "scenes",
			"scenes data missing from output", outputMap)
		slog.Error("Scenes data missing from writer output",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate scenes were generated
	if len(scenes) == 0 {
		err := w.errorFactory.NewValidationError(w.Name(), "output", "scenes",
			"no scenes generated", scenes)
		slog.Error("No scenes generated",
			"phase", w.Name(),
			"error", err,
		)
		return err
	}
	
	// Basic scene validation
	for i, scene := range scenes {
		if scene.Title == "" {
			err := w.errorFactory.NewValidationError(w.Name(), "output", "scene.title",
				"scene title cannot be empty", scene)
			slog.Error("Scene has empty title",
				"phase", w.Name(),
				"scene_index", i,
				"chapter_num", scene.ChapterNum,
				"scene_num", scene.SceneNum,
				"error", err,
			)
			return err
		}
		if scene.Content == "" {
			err := w.errorFactory.NewValidationError(w.Name(), "output", "scene.content",
				"scene content cannot be empty", scene)
			slog.Error("Scene has empty content",
				"phase", w.Name(),
				"scene_index", i,
				"chapter_num", scene.ChapterNum,
				"scene_num", scene.SceneNum,
				"error", err,
			)
			return err
		}
	}
	
	slog.Debug("Writer output validation successful",
		"phase", w.Name(),
		"scene_count", len(scenes),
	)
	
	return nil
}

func (w *Writer) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting writer execution",
		"phase", w.Name(),
		"worker_count", w.pool.GetMetrics().Workers,
	)
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("invalid input data")
		slog.Error("Writer input data type error",
			"phase", w.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	plan, ok := data["plan"].(NovelPlan)
	if !ok {
		err := fmt.Errorf("missing plan in input")
		slog.Error("Plan missing from writer input",
			"phase", w.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	arch, ok := data["architecture"].(NovelArchitecture)
	if !ok {
		err := fmt.Errorf("missing architecture in input")
		slog.Error("Architecture missing from writer input",
			"phase", w.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	slog.Debug("Creating scenes for writing",
		"phase", w.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
	)
	
	scenes := w.createScenes(plan, arch, input.Request)
	
	slog.Info("Processing scenes with worker pool",
		"phase", w.Name(),
		"scene_count", len(scenes),
		"worker_count", w.pool.GetMetrics().Workers,
	)
	
	// Use the generic worker pool with error group processing
	results, err := w.pool.ProcessWithErrGroup(ctx, scenes, w.processScene)
	
	if err != nil {
		slog.Error("Failed to process scenes",
			"phase", w.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("processing scenes: %w", err)
	}
	
	slog.Info("Successfully processed all scenes",
		"phase", w.Name(),
		"result_count", len(results),
	)
	
	for _, result := range results {
		filename := fmt.Sprintf("scenes/chapter_%d_scene_%d.txt", result.ChapterNum, result.SceneNum)
		slog.Debug("Saving scene to storage",
			"phase", w.Name(),
			"filename", filename,
			"chapter_num", result.ChapterNum,
			"scene_num", result.SceneNum,
			"content_length", len(result.Content),
		)
		if err := w.storage.Save(ctx, filename, []byte(result.Content)); err != nil {
			slog.Error("Failed to save scene",
				"phase", w.Name(),
				"filename", filename,
				"error", err,
			)
			return core.PhaseOutput{}, fmt.Errorf("saving scene: %w", err)
		}
	}
	
	slog.Info("Writer execution completed successfully",
		"phase", w.Name(),
		"scene_count", len(results),
	)
	
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"plan":         plan,
			"architecture": arch,
			"scenes":       results,
		},
	}, nil
}

func (w *Writer) createScenes(plan NovelPlan, arch NovelArchitecture, userRequest string) []Scene {
	var scenes []Scene
	
	for i, chapter := range plan.Chapters {
		scenes = append(scenes, Scene{
			ChapterNum:   i + 1,
			SceneNum:     1,
			ChapterTitle: chapter.Title,
			Summary:      chapter.Summary,
			Context: map[string]interface{}{
				"characters":   arch.Characters,
				"settings":     arch.Settings,
				"themes":       arch.Themes,
				"userRequest":  userRequest,
			},
		})
	}
	
	return scenes
}

func (w *Writer) processScene(ctx context.Context, scene Scene) (SceneResult, error) {
	slog.Debug("Processing scene",
		"phase", w.Name(),
		"chapter_num", scene.ChapterNum,
		"scene_num", scene.SceneNum,
		"chapter_title", scene.ChapterTitle,
	)
	
	contextJSON, _ := json.Marshal(scene.Context)
	
	// Pass scene data as input for template variables
	sceneData := map[string]interface{}{
		"ChapterNum":   scene.ChapterNum,
		"ChapterTitle": scene.ChapterTitle,
		"Summary":      scene.Summary,
		"Context":      scene.Context,
		"ContextJSON": string(contextJSON),
		"UserRequest":  scene.Context["userRequest"],
	}
	
	slog.Debug("Loading scene prompt template",
		"phase", w.Name(),
		"chapter_num", scene.ChapterNum,
		"scene_num", scene.SceneNum,
	)
	
	// Load and execute prompt template
	prompt, err := phase.LoadAndExecutePrompt(w.promptPath, sceneData)
	if err != nil {
		slog.Error("Failed to load scene prompt",
			"phase", w.Name(),
			"chapter_num", scene.ChapterNum,
			"scene_num", scene.SceneNum,
			"error", err,
		)
		return SceneResult{}, fmt.Errorf("loading prompt: %w", err)
	}
	
	slog.Debug("Calling AI agent for scene",
		"phase", w.Name(),
		"chapter_num", scene.ChapterNum,
		"scene_num", scene.SceneNum,
		"prompt_length", len(prompt),
	)
	
	content, err := w.agent.Execute(ctx, prompt, "")
	if err != nil {
		slog.Error("Failed to generate scene content",
			"phase", w.Name(),
			"chapter_num", scene.ChapterNum,
			"scene_num", scene.SceneNum,
			"error", err,
		)
		return SceneResult{}, err
	}
	
	slog.Debug("Received scene content",
		"phase", w.Name(),
		"chapter_num", scene.ChapterNum,
		"scene_num", scene.SceneNum,
		"content_length", len(content),
	)
	
	// Parse the response to extract title and content
	title := ""
	sceneContent := content
	
	// Look for "SCENE TITLE:" pattern
	if strings.HasPrefix(content, "SCENE TITLE:") {
		lines := strings.SplitN(content, "\n", 3)
		if len(lines) >= 2 {
			// Extract title from first line
			titleLine := strings.TrimPrefix(lines[0], "SCENE TITLE:")
			title = strings.TrimSpace(titleLine)
			
			// Reconstruct content without the title line
			if len(lines) >= 3 {
				sceneContent = strings.TrimSpace(lines[2])
			} else if len(lines) == 2 {
				sceneContent = strings.TrimSpace(lines[1])
			}
		}
		slog.Debug("Extracted scene title",
			"phase", w.Name(),
			"chapter_num", scene.ChapterNum,
			"scene_num", scene.SceneNum,
			"title", title,
		)
	}
	
	// If title is still empty, generate a default one
	if title == "" {
		title = fmt.Sprintf("Chapter %d, Scene %d", scene.ChapterNum, scene.SceneNum)
		slog.Debug("Using default scene title",
			"phase", w.Name(),
			"chapter_num", scene.ChapterNum,
			"scene_num", scene.SceneNum,
			"title", title,
		)
	}
	
	slog.Info("Successfully processed scene",
		"phase", w.Name(),
		"chapter_num", scene.ChapterNum,
		"scene_num", scene.SceneNum,
		"title", title,
		"content_length", len(sceneContent),
	)
	
	return SceneResult{
		ChapterNum: scene.ChapterNum,
		SceneNum:   scene.SceneNum,
		Title:      title,
		Content:    sceneContent,
	}, nil
}