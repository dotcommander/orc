package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Checkpoint struct {
	ID         string         `json:"id"`
	PhaseIndex int            `json:"phase_index"`
	PhaseName  string         `json:"phase_name"`
	Timestamp  time.Time      `json:"timestamp"`
	State      map[string]any `json:"state"`
	Request    string         `json:"request"`
	
	// Enhanced resumeability state
	SceneProgress    *SceneProgressStats `json:"scene_progress,omitempty"`
	TemplateCache    map[string]string   `json:"template_cache,omitempty"`
	PhaseStates      map[string]any      `json:"phase_states,omitempty"`
	ResumeCount      int                 `json:"resume_count"`
	LastResumeTime   *time.Time         `json:"last_resume_time,omitempty"`
	CanResumeWithin  bool               `json:"can_resume_within"`
}

type CheckpointManager struct {
	storage Storage
}

func NewCheckpointManager(storage Storage) *CheckpointManager {
	return &CheckpointManager{
		storage: storage,
	}
}

func (cm *CheckpointManager) Save(ctx context.Context, sessionID string, phaseIndex int, phaseName string, data interface{}) error {
	checkpoint := &Checkpoint{
		ID:         sessionID,
		PhaseIndex: phaseIndex,
		PhaseName:  phaseName,
		Timestamp:  time.Now(),
		State:      map[string]any{"data": data},
	}
	// Increment resume count if this is a resume operation
	if checkpoint.LastResumeTime != nil {
		checkpoint.ResumeCount++
	}
	
	checkpointData, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}
	
	filename := fmt.Sprintf("checkpoints/%s.json", sessionID)
	return cm.storage.Save(ctx, filename, checkpointData)
}

// SaveCheckpoint saves a checkpoint struct directly (for internal use)
func (cm *CheckpointManager) SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	// Increment resume count if this is a resume operation
	if checkpoint.LastResumeTime != nil {
		checkpoint.ResumeCount++
	}
	
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}
	
	filename := fmt.Sprintf("checkpoints/%s.json", checkpoint.ID)
	return cm.storage.Save(ctx, filename, data)
}

// SaveWithSceneProgress saves checkpoint with scene-level progress
func (cm *CheckpointManager) SaveWithSceneProgress(ctx context.Context, checkpoint *Checkpoint, tracker *AtomicSceneTracker) error {
	if tracker != nil {
		stats := tracker.GetProgressStats()
		checkpoint.SceneProgress = &stats
		checkpoint.CanResumeWithin = true
	}
	return cm.SaveCheckpoint(ctx, checkpoint)
}

// MarkAsResumed updates checkpoint to indicate it was resumed
func (cm *CheckpointManager) MarkAsResumed(ctx context.Context, id string) error {
	checkpoint, err := cm.Load(ctx, id)
	if err != nil {
		return err
	}
	
	now := time.Now()
	checkpoint.LastResumeTime = &now
	return cm.SaveCheckpoint(ctx, checkpoint)
}

func (cm *CheckpointManager) Load(ctx context.Context, sessionID string) (*Checkpoint, error) {
	filename := fmt.Sprintf("checkpoints/%s.json", sessionID)
	data, err := cm.storage.Load(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("loading checkpoint: %w", err)
	}
	
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint: %w", err)
	}
	
	return &checkpoint, nil
}

func (cm *CheckpointManager) List(ctx context.Context) ([]*Checkpoint, error) {
	files, err := cm.storage.List(ctx, "checkpoints/*.json")
	if err != nil {
		return nil, fmt.Errorf("listing checkpoints: %w", err)
	}
	
	var checkpoints []*Checkpoint
	for _, file := range files {
		data, err := cm.storage.Load(ctx, file)
		if err != nil {
			continue
		}
		
		var checkpoint Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			continue
		}
		
		checkpoints = append(checkpoints, &checkpoint)
	}
	
	return checkpoints, nil
}

func (cm *CheckpointManager) Delete(ctx context.Context, sessionID string) error {
	filename := fmt.Sprintf("checkpoints/%s.json", sessionID)
	return cm.storage.Delete(ctx, filename)
}