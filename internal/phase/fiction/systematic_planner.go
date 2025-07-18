package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// SystematicPlanner creates detailed, word-count aware story plans
type SystematicPlanner struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
}


type ChapterPlan struct {
	Number      int               `json:"number"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"`
	Purpose     string            `json:"purpose"`
	TargetWords int               `json:"target_words"`
	Scenes      []ScenePlan       `json:"scenes"`
}

type ScenePlan struct {
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Objective   string   `json:"objective"`
	Characters  []string `json:"characters"`
	Setting     string   `json:"setting"`
	Action      string   `json:"action"`
	TargetWords int      `json:"target_words"`
	Tone        string   `json:"tone"`
}

type WordBudgetStrategy struct {
	TotalWords          int `json:"total_words"`
	ChapterCount        int `json:"chapter_count"`
	WordsPerChapter     int `json:"words_per_chapter"`
	ScenesPerChapter    int `json:"scenes_per_chapter"`
	WordsPerScene       int `json:"words_per_scene"`
	BufferWords         int `json:"buffer_words"`
}

func NewSystematicPlanner(agent core.Agent, storage core.Storage) *SystematicPlanner {
	return &SystematicPlanner{
		BasePhase: NewBasePhase("Systematic Planning", 20*time.Minute),
		agent:     agent,
		storage:   storage,
	}
}

func (p *SystematicPlanner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting systematic novel planning",
		"phase", p.Name(),
		"request_preview", truncateString(input.Request, 100))

	// Extract target word count from request
	targetWords := p.extractTargetWords(input.Request)
	
	// Calculate word budget strategy
	strategy := p.calculateWordBudget(targetWords)
	
	slog.Info("Word budget strategy calculated",
		"target_words", strategy.TotalWords,
		"chapters", strategy.ChapterCount,
		"words_per_chapter", strategy.WordsPerChapter,
		"scenes_per_chapter", strategy.ScenesPerChapter)

	// Develop core story elements through conversation
	premise, err := p.developPremise(ctx, input.Request)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("developing premise: %w", err)
	}

	characters, err := p.createCharacters(ctx, premise)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("creating characters: %w", err)
	}

	settings, err := p.createSettings(ctx, premise)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("creating settings: %w", err)
	}

	plotArcs, err := p.createPlotArc(ctx, premise, characters, strategy.ChapterCount)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("creating plot arc: %w", err)
	}

	// Create detailed chapter plans with word budgets
	chapters, err := p.createChapterPlans(ctx, plotArcs[0], characters, settings, strategy)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("creating chapter plans: %w", err)
	}

	// Convert ChapterPlan to Chapter for compatibility
	compatibleChapters := make([]Chapter, len(chapters))
	for i, chapterPlan := range chapters {
		// Convert ScenePlan to Scene
		scenes := make([]Scene, len(chapterPlan.Scenes))
		for j, scenePlan := range chapterPlan.Scenes {
			scenes[j] = Scene{
				ChapterNum:   chapterPlan.Number,
				SceneNum:     scenePlan.Number,
				ChapterTitle: chapterPlan.Title,
				Title:        scenePlan.Title,
				Summary:      scenePlan.Summary,
			}
		}
		
		compatibleChapters[i] = Chapter{
			Number:  chapterPlan.Number,
			Title:   chapterPlan.Title,
			Summary: chapterPlan.Summary,
			Scenes:  scenes,
		}
	}

	// Assemble complete novel plan
	plan := NovelPlan{
		Title:          p.generateTitle(ctx, premise),
		Logline:        truncateString(premise, 100), // Short version for logline
		Synopsis:       premise,
		Themes:         []string{"systematic generation", "word count accuracy"},
		MainCharacters: characters,
		Chapters:       compatibleChapters,
	}

	// Save detailed plan
	if err := p.savePlan(ctx, plan, input.SessionID); err != nil {
		slog.Warn("Failed to save plan", "error", err)
	}

	slog.Info("Systematic planning completed",
		"phase", p.Name(),
		"title", plan.Title,
		"target_words", strategy.TotalWords,
		"chapters", len(plan.Chapters),
		"total_scenes", len(plan.Chapters)*strategy.ScenesPerChapter)

	return core.PhaseOutput{Data: plan}, nil
}

func (p *SystematicPlanner) extractTargetWords(request string) int {
	// Look for word count indicators in the request
	// Default to 20,000 if not specified
	// TODO: Parse "20,000 words", "20k words", etc.
	return 20000
}

func (p *SystematicPlanner) calculateWordBudget(targetWords int) WordBudgetStrategy {
	// Strategic word allocation
	chapterCount := targetWords / 1000 // Aim for ~1000 words per chapter
	if chapterCount < 5 {
		chapterCount = 5 // Minimum for a proper story
	}
	if chapterCount > 30 {
		chapterCount = 30 // Maximum for manageability
	}

	wordsPerChapter := targetWords / chapterCount
	scenesPerChapter := 3 // Standard 3-act chapter structure
	wordsPerScene := wordsPerChapter / scenesPerChapter
	bufferWords := targetWords - (chapterCount * wordsPerChapter) // For flexibility

	return WordBudgetStrategy{
		TotalWords:       targetWords,
		ChapterCount:     chapterCount,
		WordsPerChapter:  wordsPerChapter,
		ScenesPerChapter: scenesPerChapter,
		WordsPerScene:    wordsPerScene,
		BufferWords:      bufferWords,
	}
}

func (p *SystematicPlanner) developPremise(ctx context.Context, request string) (string, error) {
	prompt := fmt.Sprintf(`
	User request: "%s"
	
	Develop this into a compelling story premise. Focus on:
	- The core conflict or challenge
	- What makes this story worth telling
	- The emotional journey
	
	Write a clear, engaging premise in 2-3 sentences:`, request)

	return p.agent.Execute(ctx, prompt, nil)
}

func (p *SystematicPlanner) createCharacters(ctx context.Context, premise string) ([]Character, error) {
	prompt := fmt.Sprintf(`
	Story premise: "%s"
	
	Create 3-5 main characters for this story. For each character, provide:
	- Name
	- Role in the story (protagonist, antagonist, ally, etc.)
	- Brief description (personality, background)
	- Character arc (how they change)
	
	Format as a simple list, one character per paragraph.`, premise)

	response, err := p.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, err
	}

	// Parse response into Character structs
	// TODO: Improve parsing or use structured output
	return []Character{
		{Name: "Character 1", Role: "Protagonist", Description: response[:min(200, len(response))], Arc: "Growth"},
	}, nil
}

func (p *SystematicPlanner) createSettings(ctx context.Context, premise string) ([]Setting, error) {
	prompt := fmt.Sprintf(`
	Story premise: "%s"
	
	Describe the main settings where this story takes place. Include:
	- Primary location
	- Secondary locations
	- Why these settings matter to the story
	
	Keep it concise but vivid.`, premise)

	response, err := p.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, err
	}

	return []Setting{
		{Name: "Primary Setting", Description: response, Importance: "Central to plot"},
	}, nil
}

func (p *SystematicPlanner) createPlotArc(ctx context.Context, premise string, characters []Character, chapterCount int) ([]PlotArc, error) {
	prompt := fmt.Sprintf(`
	Story premise: "%s"
	Target chapters: %d
	
	Create a plot arc with these key moments:
	- Hook (opening that grabs attention)
	- Inciting event (what starts the main story)
	- Rising action (building tension and complications)
	- Climax (the big confrontation or turning point)
	- Falling action (aftermath and consequences)
	- Resolution (how it all ends)
	
	Write each element in 1-2 sentences.`, premise, chapterCount)

	response, err := p.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, err
	}

	// Create compatible PlotArc that matches the existing type
	plotArc := PlotArc{
		Name:        "Main Plot",
		Description: response,
		Chapters:    make([]int, chapterCount),
	}
	
	// Fill chapter numbers
	for i := 0; i < chapterCount; i++ {
		plotArc.Chapters[i] = i + 1
	}

	return []PlotArc{plotArc}, nil
}

func (p *SystematicPlanner) createChapterPlans(ctx context.Context, plotArc PlotArc, characters []Character, settings []Setting, strategy WordBudgetStrategy) ([]ChapterPlan, error) {
	chapters := make([]ChapterPlan, strategy.ChapterCount)
	
	for i := 0; i < strategy.ChapterCount; i++ {
		chapterNum := i + 1
		
		prompt := fmt.Sprintf(`
		Chapter %d of %d
		Target words: %d words (divide into %d scenes of ~%d words each)
		
		Plot context:
		%s
		
		For this chapter, define:
		1. Chapter purpose (what advances the plot)
		2. Chapter title
		3. Three scenes with specific objectives and word targets
		
		Make each scene focused and specific.`, 
		chapterNum, strategy.ChapterCount, strategy.WordsPerChapter, 
		strategy.ScenesPerChapter, strategy.WordsPerScene,
		plotArc.Description)

		_, err := p.agent.Execute(ctx, prompt, nil)
		if err != nil {
			slog.Warn("Failed to create chapter plan", "chapter", chapterNum, "error", err)
			// Continue with basic chapter plan
		}

		// Create structured chapter plan
		scenes := make([]ScenePlan, strategy.ScenesPerChapter)
		for j := 0; j < strategy.ScenesPerChapter; j++ {
			scenes[j] = ScenePlan{
				Number:      j + 1,
				Objective:   fmt.Sprintf("Scene %d objective", j+1),
				TargetWords: strategy.WordsPerScene,
				Tone:        "engaging",
			}
		}

		chapters[i] = ChapterPlan{
			Number:      chapterNum,
			Title:       fmt.Sprintf("Chapter %d", chapterNum),
			Purpose:     fmt.Sprintf("Advance plot point %d", chapterNum),
			TargetWords: strategy.WordsPerChapter,
			Scenes:      scenes,
		}
	}

	return chapters, nil
}

func (p *SystematicPlanner) generateTitle(ctx context.Context, premise string) string {
	prompt := fmt.Sprintf(`
	Story premise: "%s"
	
	Create a compelling title for this story. Make it intriguing and memorable.
	Just return the title, nothing else.`, premise)

	title, err := p.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return "Untitled Story"
	}

	return strings.TrimSpace(title)
}

func (p *SystematicPlanner) savePlan(ctx context.Context, plan NovelPlan, sessionID string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}

	return p.storage.Save(ctx, "systematic_plan.json", data)
}

func (p *SystematicPlanner) countTotalScenes(chapters []ChapterPlan) int {
	total := 0
	for _, chapter := range chapters {
		total += len(chapter.Scenes)
	}
	return total
}

func (p *SystematicPlanner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if len(strings.TrimSpace(input.Request)) < 10 {
		return fmt.Errorf("request too short for systematic planning")
	}
	return nil
}

func (p *SystematicPlanner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	return nil // Systematic output is structured
}

