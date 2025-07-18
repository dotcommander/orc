package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/phase"
)

// ResilientWriter implements a writer phase with enhanced timeout handling and resume capabilities
type ResilientWriter struct {
	BasePhase
	agent            core.Agent
	storage          core.Storage
	promptPath       string
	sceneTracker     *core.AtomicSceneTracker
	totalScenes      int
	checkpointMgr    *core.CheckpointManager
	sessionID        string
	logger           *slog.Logger
	
	// Configuration
	sceneTimeout     time.Duration
	maxRetries       int
	resumeEnabled    bool
	checkpointEvery  int // Checkpoint after every N scenes
}

type ResilientWriterOption func(*ResilientWriter)

func WithSceneTimeout(timeout time.Duration) ResilientWriterOption {
	return func(w *ResilientWriter) {
		w.sceneTimeout = timeout
	}
}

func WithCheckpointing(mgr *core.CheckpointManager, sessionID string) ResilientWriterOption {
	return func(w *ResilientWriter) {
		w.checkpointMgr = mgr
		w.sessionID = sessionID
		w.resumeEnabled = true
	}
}

func WithCheckpointFrequency(every int) ResilientWriterOption {
	return func(w *ResilientWriter) {
		w.checkpointEvery = every
	}
}

func NewResilientWriter(agent core.Agent, storage core.Storage, promptPath string, opts ...ResilientWriterOption) *ResilientWriter {
	w := &ResilientWriter{
		BasePhase:       NewBasePhase("Writing", 120*time.Minute), // Extended timeout
		agent:           agent,
		storage:         storage,
		promptPath:      promptPath,
		// sceneTracker will be initialized when we know total scenes
		logger:          slog.Default(),
		sceneTimeout:    5*time.Minute, // Per-scene timeout
		maxRetries:      3,
		checkpointEvery: 2, // Checkpoint every 2 scenes by default
	}
	
	for _, opt := range opts {
		opt(w)
	}
	
	return w
}

func (w *ResilientWriter) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	w.logger.Info("Starting resilient writer execution", 
		"session", w.sessionID,
		"resume_enabled", w.resumeEnabled)
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid input data")
	}
	
	plan, ok := data["plan"].(NovelPlan)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("missing plan in input")
	}
	
	arch, ok := data["architecture"].(NovelArchitecture)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("missing architecture in input")
	}
	
	// Check for existing progress if resuming
	startChapter := 0
	if w.resumeEnabled && w.checkpointMgr != nil {
		if checkpoint, err := w.checkpointMgr.Load(ctx, w.sessionID); err == nil {
			if checkpoint.SceneProgress != nil {
				startChapter = checkpoint.SceneProgress.Completed
				w.logger.Info("Resuming from checkpoint", 
					"completed_scenes", startChapter,
					"total_scenes", len(plan.Chapters))
			}
		}
	}
	
	// Process chapters with resilience
	results, err := w.processChaptersResilient(ctx, plan, arch, input.Request, startChapter)
	if err != nil {
		return core.PhaseOutput{}, err
	}
	
	// Save final results
	for _, result := range results {
		filename := fmt.Sprintf("scenes/chapter_%d_scene_%d.txt", result.ChapterNum, result.SceneNum)
		if err := w.storage.Save(ctx, filename, []byte(result.Content)); err != nil {
			w.logger.Error("Failed to save scene", 
				"chapter", result.ChapterNum,
				"error", err)
		}
	}
	
	w.logger.Info("Writing phase completed", 
		"total_scenes", len(results),
		"session", w.sessionID)
	
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"plan":         plan,
			"architecture": arch,
			"scenes":       results,
		},
	}, nil
}

func (w *ResilientWriter) processChaptersResilient(ctx context.Context, plan NovelPlan, arch NovelArchitecture, userRequest string, startFrom int) ([]SceneResult, error) {
	var results []SceneResult
	var mu sync.Mutex
	
	// Create scene tasks
	scenes := w.createScenes(plan, arch, userRequest)
	
	// Initialize scene tracker with total scenes
	if w.sceneTracker == nil && w.storage != nil && w.sessionID != "" {
		w.sceneTracker = core.NewAtomicSceneTracker(w.storage, w.sessionID, len(scenes))
		// Load any existing progress
		w.sceneTracker.LoadProgress(ctx)
	}
	
	// Skip already completed scenes if resuming
	if startFrom > 0 && startFrom < len(scenes) {
		w.logger.Info("Skipping completed scenes", "count", startFrom)
		scenes = scenes[startFrom:]
	}
	
	// Process scenes one by one with timeout and retry
	for idx, scene := range scenes {
		globalIdx := startFrom + idx
		
		w.logger.Info("Processing scene", 
			"chapter", scene.ChapterNum,
			"scene", scene.SceneNum,
			"progress", fmt.Sprintf("%d/%d", globalIdx+1, len(plan.Chapters)))
		
		// Process with retry logic
		result, err := w.processSceneWithRetry(ctx, scene)
		if err != nil {
			// Save partial progress before failing
			if w.resumeEnabled && w.checkpointMgr != nil {
				w.saveCheckpoint(ctx, globalIdx, plan, arch, results)
			}
			return results, fmt.Errorf("failed to process chapter %d after retries: %w", scene.ChapterNum, err)
		}
		
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
		
		// Update tracker
		if w.sceneTracker != nil {
			w.sceneTracker.MarkCompleted(ctx, result.ChapterNum, result.SceneNum, result.Content)
		}
		
		// Checkpoint periodically
		if w.resumeEnabled && w.checkpointMgr != nil && (globalIdx+1)%w.checkpointEvery == 0 {
			w.logger.Info("Creating checkpoint", "scenes_completed", globalIdx+1)
			w.saveCheckpoint(ctx, globalIdx+1, plan, arch, results)
		}
	}
	
	return results, nil
}

func (w *ResilientWriter) processSceneWithRetry(ctx context.Context, scene Scene) (SceneResult, error) {
	var lastErr error
	
	for attempt := 1; attempt <= w.maxRetries; attempt++ {
		// Create a timeout context for this specific scene
		sceneCtx, cancel := context.WithTimeout(ctx, w.sceneTimeout)
		defer cancel()
		
		w.logger.Debug("Processing scene attempt", 
			"chapter", scene.ChapterNum,
			"attempt", attempt,
			"timeout", w.sceneTimeout)
		
		result, err := w.writeScene(sceneCtx, scene)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Check if context was cancelled (timeout or user cancellation)
		if sceneCtx.Err() != nil {
			if sceneCtx.Err() == context.DeadlineExceeded {
				w.logger.Warn("Scene timed out", 
					"chapter", scene.ChapterNum,
					"attempt", attempt,
					"timeout", w.sceneTimeout)
			} else {
				// User cancellation, don't retry
				return SceneResult{}, fmt.Errorf("cancelled: %w", err)
			}
		}
		
		// Exponential backoff before retry
		if attempt < w.maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			w.logger.Info("Retrying scene after backoff", 
				"chapter", scene.ChapterNum,
				"attempt", attempt,
				"backoff", backoff)
			
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return SceneResult{}, ctx.Err()
			}
		}
	}
	
	return SceneResult{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (w *ResilientWriter) writeScene(ctx context.Context, scene Scene) (SceneResult, error) {
	contextJSON, _ := json.Marshal(scene.Context)
	
	sceneData := map[string]interface{}{
		"ChapterNum":   scene.ChapterNum,
		"ChapterTitle": scene.ChapterTitle,
		"Summary":      scene.Summary,
		"Context":      scene.Context,
		"ContextJSON":  string(contextJSON),
		"UserRequest":  scene.Context["userRequest"],
	}
	
	prompt, err := phase.LoadAndExecutePrompt(w.promptPath, sceneData)
	if err != nil {
		return SceneResult{}, fmt.Errorf("loading prompt: %w", err)
	}
	
	content, err := w.agent.Execute(ctx, prompt, "")
	if err != nil {
		return SceneResult{}, err
	}
	
	return SceneResult{
		ChapterNum: scene.ChapterNum,
		SceneNum:   scene.SceneNum,
		Content:    content,
	}, nil
}

func (w *ResilientWriter) createScenes(plan NovelPlan, arch NovelArchitecture, userRequest string) []Scene {
	var scenes []Scene
	
	for i, chapter := range plan.Chapters {
		scenes = append(scenes, Scene{
			ChapterNum:   i + 1,
			SceneNum:     1,
			ChapterTitle: chapter.Title,
			Summary:      chapter.Summary,
			Context: map[string]interface{}{
				"characters":  arch.Characters,
				"settings":    arch.Settings,
				"themes":      arch.Themes,
				"userRequest": userRequest,
			},
		})
	}
	
	return scenes
}

func (w *ResilientWriter) saveCheckpoint(ctx context.Context, completedScenes int, plan NovelPlan, arch NovelArchitecture, results []SceneResult) {
	checkpoint := &core.Checkpoint{
		ID:         w.sessionID,
		PhaseIndex: 2, // Writing is typically the 3rd phase (0-indexed)
		PhaseName:  w.Name(),
		Timestamp:  time.Now(),
		Request:    "", // Would need to pass this through
		State: map[string]any{
			"last_output": map[string]interface{}{
				"plan":         plan,
				"architecture": arch,
				"scenes":       results,
			},
		},
		SceneProgress: &core.SceneProgressStats{
			Total:      len(plan.Chapters),
			Completed:  completedScenes,
			Failed:     0,
			Pending:    len(plan.Chapters) - completedScenes,
			StartTime:  time.Now(), // Would need to track this properly
			LastUpdate: time.Now(),
		},
		CanResumeWithin: true,
	}
	
	if err := w.checkpointMgr.SaveWithSceneProgress(ctx, checkpoint, w.sceneTracker); err != nil {
		w.logger.Error("Failed to save checkpoint", "error", err)
	}
}

// ValidateInput validates the writer input
func (w *ResilientWriter) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	// Use consolidated validation
	validator := core.NewBaseValidator("Writer")
	if err := validator.ValidateRequired("request", input.Request, "input"); err != nil {
		return err
	}
	return nil
}

// ValidateOutput validates the writer output
func (w *ResilientWriter) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	// Use consolidated validation
	validator := core.NewBaseValidator("Writer")
	if output.Data == nil {
		return core.NewValidationError("Writer", "output", "data", "output data cannot be nil", nil)
	}
	if err := validator.ValidateJSON("data", output.Data, "output"); err != nil {
		return err
	}
	return nil
}