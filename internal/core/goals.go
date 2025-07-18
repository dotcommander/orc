package core

import (
	"fmt"
	"sync"
	"time"
)

// GoalType defines the type of goal to track
type GoalType string

const (
	GoalTypeWordCount    GoalType = "word_count"
	GoalTypeQuality      GoalType = "quality_score"
	GoalTypeCompleteness GoalType = "completeness"
	GoalTypeSceneCount   GoalType = "scene_count"
	GoalTypeChapterCount GoalType = "chapter_count"
)

// Goal represents a trackable objective for the orchestrator
type Goal struct {
	Type        GoalType
	Target      interface{}
	Current     interface{}
	Priority    int                    // 1-10, higher is more important
	Met         bool
	Strategy    string                 // Suggested strategy to meet goal
	Validator   func(interface{}) bool // Custom validation logic
	LastUpdated time.Time
}

// Progress returns the progress percentage for numeric goals
func (g *Goal) Progress() float64 {
	switch g.Type {
	case GoalTypeWordCount, GoalTypeSceneCount, GoalTypeChapterCount:
		if current, ok := g.Current.(int); ok {
			if target, ok := g.Target.(int); ok && target > 0 {
				return float64(current) / float64(target) * 100
			}
		}
	case GoalTypeQuality:
		if current, ok := g.Current.(float64); ok {
			if target, ok := g.Target.(float64); ok && target > 0 {
				return current / target * 100
			}
		}
	case GoalTypeCompleteness:
		if current, ok := g.Current.(bool); ok && current {
			return 100
		}
		return 0
	}
	return 0
}

// Gap returns the deficit for numeric goals
func (g *Goal) Gap() interface{} {
	switch g.Type {
	case GoalTypeWordCount, GoalTypeSceneCount, GoalTypeChapterCount:
		current, _ := g.Current.(int)
		target, _ := g.Target.(int)
		return target - current
	case GoalTypeQuality:
		current, _ := g.Current.(float64)
		target, _ := g.Target.(float64)
		return target - current
	}
	return nil
}

// GoalTracker manages and tracks multiple goals
type GoalTracker struct {
	goals map[string]*Goal
	mu    sync.RWMutex
}

// NewGoalTracker creates a new goal tracker
func NewGoalTracker() *GoalTracker {
	return &GoalTracker{
		goals: make(map[string]*Goal),
	}
}

// AddGoal adds a new goal to track
func (gt *GoalTracker) AddGoal(goal *Goal) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	goal.LastUpdated = time.Now()
	gt.goals[string(goal.Type)] = goal
}

// SetWordCountGoal sets a word count target
func (gt *GoalTracker) SetWordCountGoal(target int, priority int) {
	gt.AddGoal(&Goal{
		Type:     GoalTypeWordCount,
		Target:   target,
		Current:  0,
		Priority: priority,
		Validator: func(current interface{}) bool {
			if count, ok := current.(int); ok {
				return count >= target*9/10 // Accept 90% of target
			}
			return false
		},
	})
}

// SetQualityGoal sets a quality score target
func (gt *GoalTracker) SetQualityGoal(target float64, priority int) {
	gt.AddGoal(&Goal{
		Type:     GoalTypeQuality,
		Target:   target,
		Current:  0.0,
		Priority: priority,
		Validator: func(current interface{}) bool {
			if score, ok := current.(float64); ok {
				return score >= target
			}
			return false
		},
	})
}

// Update updates the current value for a goal
func (gt *GoalTracker) Update(goalType GoalType, current interface{}) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	if goal, exists := gt.goals[string(goalType)]; exists {
		goal.Current = current
		goal.LastUpdated = time.Now()
		
		// Check if goal is met
		if goal.Validator != nil {
			goal.Met = goal.Validator(current)
		} else {
			// Default validation for equality
			goal.Met = goal.Current == goal.Target
		}
		
		// Update strategy suggestion based on gap
		gt.updateStrategy(goal)
	}
}

// updateStrategy suggests a strategy based on the goal gap
func (gt *GoalTracker) updateStrategy(goal *Goal) {
	switch goal.Type {
	case GoalTypeWordCount:
		gap, _ := goal.Gap().(int)
		switch {
		case gap <= 0:
			goal.Strategy = "none_needed"
		case gap < 1000:
			goal.Strategy = "expand_scenes"
		case gap < 5000:
			goal.Strategy = "add_scenes"
		default:
			goal.Strategy = "add_chapters"
		}
	case GoalTypeQuality:
		if goal.Progress() < 80 {
			goal.Strategy = "enhance_quality"
		}
	}
}

// GetGoal retrieves a specific goal
func (gt *GoalTracker) GetGoal(goalType GoalType) (*Goal, bool) {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	goal, exists := gt.goals[string(goalType)]
	return goal, exists
}

// GetUnmetGoals returns all goals that haven't been met, sorted by priority
func (gt *GoalTracker) GetUnmetGoals() []*Goal {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	unmet := make([]*Goal, 0)
	for _, goal := range gt.goals {
		if !goal.Met {
			unmet = append(unmet, goal)
		}
	}
	
	// Sort by priority (higher first)
	for i := 0; i < len(unmet)-1; i++ {
		for j := i + 1; j < len(unmet); j++ {
			if unmet[j].Priority > unmet[i].Priority {
				unmet[i], unmet[j] = unmet[j], unmet[i]
			}
		}
	}
	
	return unmet
}

// AllMet returns true if all goals are met
func (gt *GoalTracker) AllMet() bool {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	for _, goal := range gt.goals {
		if !goal.Met {
			return false
		}
	}
	return true
}

// Progress returns a summary of all goal progress
func (gt *GoalTracker) Progress() string {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	summary := "Goal Progress:\n"
	for _, goal := range gt.goals {
		status := "❌"
		if goal.Met {
			status = "✅"
		}
		summary += fmt.Sprintf("%s %s: %.1f%% (Current: %v, Target: %v)\n",
			status, goal.Type, goal.Progress(), goal.Current, goal.Target)
	}
	return summary
}

// MetCount returns the number of goals that have been met
func (gt *GoalTracker) MetCount() int {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	count := 0
	for _, goal := range gt.goals {
		if goal.Met {
			count++
		}
	}
	return count
}

// TotalCount returns the total number of goals
func (gt *GoalTracker) TotalCount() int {
	gt.mu.RLock()
	defer gt.mu.RUnlock()
	
	return len(gt.goals)
}