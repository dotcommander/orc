package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// ExpansionStrategy expands existing content to meet word count goals
type ExpansionStrategy struct {
	agent  Agent
	logger *slog.Logger
}

func NewExpansionStrategy(agent Agent, logger *slog.Logger) *ExpansionStrategy {
	return &ExpansionStrategy{
		agent:  agent,
		logger: logger,
	}
}

func (s *ExpansionStrategy) Name() string {
	return "expansion"
}

func (s *ExpansionStrategy) CanHandle(goals []*Goal) bool {
	for _, goal := range goals {
		if goal.Type == GoalTypeWordCount {
			gap, _ := goal.Gap().(int)
			// Good for small to medium gaps
			return gap > 0 && gap < 5000
		}
	}
	return false
}

func (s *ExpansionStrategy) EstimateEffectiveness(goals []*Goal) float64 {
	for _, goal := range goals {
		if goal.Type == GoalTypeWordCount {
			gap, _ := goal.Gap().(int)
			// Most effective for small gaps
			if gap < 1000 {
				return 0.95
			} else if gap < 3000 {
				return 0.80
			} else if gap < 5000 {
				return 0.60
			}
		}
	}
	return 0.0
}

func (s *ExpansionStrategy) Execute(ctx context.Context, input interface{}, goals []*Goal) (interface{}, error) {
	manuscript, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expansion strategy requires string input")
	}
	
	// Find word count goal
	var wordGap int
	for _, goal := range goals {
		if goal.Type == GoalTypeWordCount {
			wordGap, _ = goal.Gap().(int)
			break
		}
	}
	
	if wordGap <= 0 {
		return manuscript, nil
	}
	
	s.logger.Info("Expanding content", "words_needed", wordGap)
	
	// Split into scenes for expansion
	scenes := strings.Split(manuscript, "\n\n")
	if len(scenes) == 0 {
		return manuscript, nil
	}
	
	wordsPerScene := wordGap / len(scenes)
	if wordsPerScene < 50 {
		wordsPerScene = 50 // Minimum expansion
	}
	
	expandedScenes := make([]string, 0, len(scenes))
	
	for i, scene := range scenes {
		if strings.TrimSpace(scene) == "" {
			expandedScenes = append(expandedScenes, scene)
			continue
		}
		
		// Skip titles and short lines
		if strings.HasPrefix(scene, "#") || len(scene) < 100 {
			expandedScenes = append(expandedScenes, scene)
			continue
		}
		
		prompt := fmt.Sprintf(`Expand this scene by adding approximately %d words.
Focus on:
- Character internal thoughts and emotions
- Sensory details (sight, sound, smell, touch, taste)
- Environmental descriptions
- Dialogue that reveals character
- Action details and pacing

Original scene:
%s

Maintain the same tone, style, and narrative voice. The expansion should flow naturally and enhance the scene, not feel forced or repetitive.`, 
			wordsPerScene, scene)
		
		expanded, err := s.agent.Execute(ctx, prompt, nil)
		if err != nil {
			s.logger.Warn("Failed to expand scene", "index", i, "error", err)
			expandedScenes = append(expandedScenes, scene) // Keep original
			continue
		}
		
		expandedScenes = append(expandedScenes, expanded)
	}
	
	return strings.Join(expandedScenes, "\n\n"), nil
}