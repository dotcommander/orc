package fiction

import "fmt"

// Shared types used across fiction phase implementations

// NovelPlan represents the structured plan for a novel
type NovelPlan struct {
	Title          string      `json:"title"`
	Logline        string      `json:"logline"`
	Synopsis       string      `json:"synopsis"`
	Themes         []string    `json:"themes"`
	MainCharacters []Character `json:"main_characters"`
	Chapters       []Chapter   `json:"chapters"`
}

type Chapter struct {
	Number  int     `json:"number"`
	Title   string  `json:"title"`
	Summary string  `json:"summary"`
	Scenes  []Scene `json:"scenes"`
}

// NovelArchitecture represents the detailed structure of a novel
type NovelArchitecture struct {
	Characters []Character `json:"characters"`
	Settings   []Setting   `json:"settings"`
	Themes     []string    `json:"themes"`
	PlotArcs   []PlotArc   `json:"plot_arcs"`
	Chapters   []Chapter   `json:"chapters"`
}

type Character struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Arc         string `json:"arc"`
}

type Setting struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Importance  string `json:"importance"`
}

type PlotArc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Chapters    []int  `json:"chapters"`
}

// Scene represents a single scene to be written
type Scene struct {
	ChapterNum   int                    `json:"chapter_num"`
	SceneNum     int                    `json:"scene_num"`
	ChapterTitle string                 `json:"chapter_title"`
	Title        string                 `json:"title"`
	Summary      string                 `json:"summary"`
	Content      string                 `json:"content"`
	Context      map[string]interface{} `json:"context"`
}

// ID implements WorkItem interface
func (s Scene) ID() string {
	return fmt.Sprintf("chapter_%d_scene_%d", s.ChapterNum, s.SceneNum)
}

// Priority implements WorkItem interface - scenes are processed in order
func (s Scene) Priority() int {
	return s.ChapterNum*1000 + s.SceneNum
}

// SceneResult represents a completed scene
type SceneResult struct {
	ChapterNum int    `json:"chapter_num"`
	SceneNum   int    `json:"scene_num"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Err        error  `json:"-"` // Don't serialize errors
}

// ItemID implements WorkResult interface
func (sr SceneResult) ItemID() string {
	return fmt.Sprintf("chapter_%d_scene_%d", sr.ChapterNum, sr.SceneNum)
}

// Error implements WorkResult interface
func (sr SceneResult) Error() error {
	return sr.Err
}

// Critique represents the AI critique of the completed novel
type Critique struct {
	OverallRating float64              `json:"overall_rating"`
	Strengths     []string             `json:"strengths"`
	Weaknesses    []string             `json:"weaknesses"`
	Suggestions   []string             `json:"suggestions"`
	ChapterNotes  map[string][]string  `json:"chapter_notes"`
}

// NovelCritique represents the AI critique of the completed novel
type NovelCritique struct {
	Score       float64  `json:"score"`
	Summary     string   `json:"summary"`
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
	Suggestions []string `json:"suggestions"`
}

// Manuscript represents a complete novel manuscript
type Manuscript struct {
	Title    string                 `json:"title"`
	Chapters []ManuscriptChapter   `json:"chapters"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ManuscriptChapter represents a chapter in the manuscript
type ManuscriptChapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// DetailedCritique represents detailed critique with scores
type DetailedCritique struct {
	OverallScore   int      `json:"overall_score"`
	PlotScore      int      `json:"plot_score"`
	CharacterScore int      `json:"character_score"`
	WritingScore   int      `json:"writing_score"`
	PacingScore    int      `json:"pacing_score"`
	DialogueScore  int      `json:"dialogue_score"`
	Summary        string   `json:"summary"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Suggestions    []string `json:"suggestions"`
}