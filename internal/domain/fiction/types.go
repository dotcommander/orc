package fiction

// NovelPlan represents the structured plan for a novel
type NovelPlan struct {
	Title    string    `json:"title"`
	Logline  string    `json:"logline"`
	Chapters []Chapter `json:"chapters"`
}

type Chapter struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// NovelArchitecture represents the detailed structure of a novel
type NovelArchitecture struct {
	Characters []Character `json:"characters"`
	Settings   []Setting   `json:"settings"`
	Themes     []string    `json:"themes"`
	PlotArcs   []PlotArc   `json:"plot_arcs"`
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
	Summary      string                 `json:"summary"`
	Context      map[string]interface{} `json:"context"`
}

// SceneResult represents a completed scene
type SceneResult struct {
	ChapterNum int    `json:"chapter_num"`
	SceneNum   int    `json:"scene_num"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

// Critique represents the AI critique of the completed novel
type Critique struct {
	OverallRating float64             `json:"overall_rating"`
	Strengths     []string            `json:"strengths"`
	Weaknesses    []string            `json:"weaknesses"`
	Suggestions   []string            `json:"suggestions"`
	ChapterNotes  map[string][]string `json:"chapter_notes"`
}