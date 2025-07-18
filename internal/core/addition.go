package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// AdditionStrategy adds new content to meet goals
type AdditionStrategy struct {
	agent  Agent
	logger *slog.Logger
}

func NewAdditionStrategy(agent Agent, logger *slog.Logger) *AdditionStrategy {
	return &AdditionStrategy{
		agent:  agent,
		logger: logger,
	}
}

func (s *AdditionStrategy) Name() string {
	return "addition"
}

func (s *AdditionStrategy) CanHandle(goals []*Goal) bool {
	for _, goal := range goals {
		if goal.Type == GoalTypeWordCount {
			gap, _ := goal.Gap().(int)
			// Good for medium to large gaps
			return gap >= 5000
		}
	}
	return false
}

func (s *AdditionStrategy) EstimateEffectiveness(goals []*Goal) float64 {
	for _, goal := range goals {
		if goal.Type == GoalTypeWordCount {
			gap, _ := goal.Gap().(int)
			// Most effective for large gaps
			if gap >= 10000 {
				return 0.95
			} else if gap >= 5000 {
				return 0.85
			}
		}
	}
	return 0.0
}

func (s *AdditionStrategy) Execute(ctx context.Context, input interface{}, goals []*Goal) (interface{}, error) {
	// Extract manuscript and metadata
	data, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("addition strategy requires map input with manuscript and plan")
	}
	
	manuscript, _ := data["manuscript"].(string)
	plan, _ := data["plan"].(map[string]interface{})
	
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
	
	s.logger.Info("Adding new content", "words_needed", wordGap)
	
	// Calculate how many new chapters/scenes needed
	avgWordsPerChapter := 2500
	chaptersNeeded := (wordGap + avgWordsPerChapter - 1) / avgWordsPerChapter
	
	prompt := fmt.Sprintf(`Based on this existing story, generate %d additional chapters to continue the narrative.

Existing story summary and plan:
%v

Current story ends with:
%s

Generate %d new chapters (approximately %d words each) that:
1. Continue naturally from where the story left off
2. Maintain consistent character voices and development
3. Progress the plot toward resolution
4. Keep the same tone and style
5. Add new complications or depth while moving toward conclusion

Format each chapter with:
## Chapter [Number]: [Title]

[Chapter content]`,
		chaptersNeeded, plan, getLastParagraphs(manuscript, 3), 
		chaptersNeeded, avgWordsPerChapter)
	
	newContent, err := s.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate additional content: %w", err)
	}
	
	// Combine original and new content
	combined := manuscript + "\n\n" + newContent
	
	return combined, nil
}

// getLastParagraphs returns the last 'count' paragraphs from text
func getLastParagraphs(text string, count int) string {
	paragraphs := strings.Split(text, "\n\n")
	if len(paragraphs) <= count {
		return text
	}
	
	lastParagraphs := paragraphs[len(paragraphs)-count:]
	return strings.Join(lastParagraphs, "\n\n")
}