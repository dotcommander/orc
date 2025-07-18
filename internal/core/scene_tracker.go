package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// SceneProgress tracks individual scene completion state
type SceneProgress struct {
	SessionID       string                 `json:"session_id"`
	TotalScenes     int                   `json:"total_scenes"`
	CompletedScenes map[string]SceneResult `json:"completed_scenes"` // Key: "chapter_X_scene_Y"
	FailedScenes    map[string]SceneError  `json:"failed_scenes"`
	StartTime       time.Time             `json:"start_time"`
	LastUpdate      time.Time             `json:"last_update"`
}

type SceneResult struct {
	ChapterNum  int    `json:"chapter_num"`
	SceneNum    int    `json:"scene_num"`
	Content     string `json:"content"`
	CompletedAt time.Time `json:"completed_at"`
}

type SceneError struct {
	ChapterNum  int       `json:"chapter_num"`
	SceneNum    int       `json:"scene_num"`
	Attempt     int       `json:"attempt"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	Retryable   bool      `json:"retryable"`
}

// AtomicSceneTracker provides atomic scene completion tracking with persistence
type AtomicSceneTracker struct {
	storage   Storage
	sessionID string
	mu        sync.RWMutex
	progress  *SceneProgress
}

func NewAtomicSceneTracker(storage Storage, sessionID string, totalScenes int) *AtomicSceneTracker {
	return &AtomicSceneTracker{
		storage:   storage,
		sessionID: sessionID,
		progress: &SceneProgress{
			SessionID:       sessionID,
			TotalScenes:     totalScenes,
			CompletedScenes: make(map[string]SceneResult),
			FailedScenes:    make(map[string]SceneError),
			StartTime:       time.Now(),
			LastUpdate:      time.Now(),
		},
	}
}

func (t *AtomicSceneTracker) LoadProgress(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	progressFile := fmt.Sprintf("progress/writing_progress_%s.json", t.sessionID)
	data, err := t.storage.Load(ctx, progressFile)
	if err != nil {
		// No existing progress - start fresh
		return nil
	}

	var progress SceneProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return fmt.Errorf("parsing progress: %w", err)
	}

	t.progress = &progress
	return nil
}

func (t *AtomicSceneTracker) saveProgress(ctx context.Context) error {
	t.progress.LastUpdate = time.Now()
	
	data, err := json.MarshalIndent(t.progress, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling progress: %w", err)
	}

	progressFile := fmt.Sprintf("progress/writing_progress_%s.json", t.sessionID)
	return t.storage.Save(ctx, progressFile, data)
}

func (t *AtomicSceneTracker) MarkCompleted(ctx context.Context, chapterNum, sceneNum int, content string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	sceneKey := fmt.Sprintf("chapter_%d_scene_%d", chapterNum, sceneNum)
	
	// Save scene content to individual file
	sceneFile := fmt.Sprintf("scenes/chapter_%d_scene_%d.txt", chapterNum, sceneNum)
	if err := t.storage.Save(ctx, sceneFile, []byte(content)); err != nil {
		return fmt.Errorf("saving scene content: %w", err)
	}

	// Update progress tracking
	t.progress.CompletedScenes[sceneKey] = SceneResult{
		ChapterNum:  chapterNum,
		SceneNum:    sceneNum,
		Content:     content,
		CompletedAt: time.Now(),
	}

	// Remove from failed scenes if it was there
	delete(t.progress.FailedScenes, sceneKey)

	// Persist progress atomically
	return t.saveProgress(ctx)
}

func (t *AtomicSceneTracker) MarkFailed(ctx context.Context, chapterNum, sceneNum int, attempt int, err error, retryable bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	sceneKey := fmt.Sprintf("chapter_%d_scene_%d", chapterNum, sceneNum)
	
	t.progress.FailedScenes[sceneKey] = SceneError{
		ChapterNum: chapterNum,
		SceneNum:   sceneNum,
		Attempt:    attempt,
		Error:      err.Error(),
		Timestamp:  time.Now(),
		Retryable:  retryable,
	}

	// Persist progress atomically
	return t.saveProgress(ctx)
}

func (t *AtomicSceneTracker) GetProgress() (completed, failed, total int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.progress.CompletedScenes), len(t.progress.FailedScenes), t.progress.TotalScenes
}

func (t *AtomicSceneTracker) GetCompletedScenes() map[string]SceneResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return copy to avoid race conditions
	result := make(map[string]SceneResult)
	for k, v := range t.progress.CompletedScenes {
		result[k] = v
	}
	return result
}

func (t *AtomicSceneTracker) GetFailedScenes() map[string]SceneError {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return copy to avoid race conditions
	result := make(map[string]SceneError)
	for k, v := range t.progress.FailedScenes {
		result[k] = v
	}
	return result
}

func (t *AtomicSceneTracker) IsCompleted(chapterNum, sceneNum int) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sceneKey := fmt.Sprintf("chapter_%d_scene_%d", chapterNum, sceneNum)
	_, exists := t.progress.CompletedScenes[sceneKey]
	return exists
}

func (t *AtomicSceneTracker) GetProgressStats() SceneProgressStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	completed := len(t.progress.CompletedScenes)
	failed := len(t.progress.FailedScenes)
	total := t.progress.TotalScenes
	pending := total - completed

	return SceneProgressStats{
		Total:           total,
		Completed:       completed,
		Failed:          failed,
		Pending:         pending,
		PercentComplete: float64(completed) / float64(total) * 100,
		StartTime:       t.progress.StartTime,
		LastUpdate:      t.progress.LastUpdate,
	}
}

type SceneProgressStats struct {
	Total           int       `json:"total"`
	Completed       int       `json:"completed"`
	Failed          int       `json:"failed"`
	Pending         int       `json:"pending"`
	PercentComplete float64   `json:"percent_complete"`
	StartTime       time.Time `json:"start_time"`
	LastUpdate      time.Time `json:"last_update"`
}