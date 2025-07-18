package code

// CodeAnalysis represents the analysis phase output
type CodeAnalysis struct {
	Language       string   `json:"language"`
	Framework      string   `json:"framework,omitempty"`
	Complexity     string   `json:"complexity"`
	MainObjective  string   `json:"main_objective"`
	Requirements   []string `json:"requirements"`
	Constraints    []string `json:"constraints"`
	PotentialRisks []string `json:"potential_risks"`
}

// ImplementationPlan represents the planning phase output
type ImplementationPlan struct {
	Overview string      `json:"overview"`
	Steps    []CodeStep  `json:"steps"`
	Testing  TestingPlan `json:"testing"`
}

// CodeStep represents a single implementation step
type CodeStep struct {
	Order        int      `json:"order"`
	Description  string   `json:"description"`
	CodeFiles    []string `json:"code_files"`
	Rationale    string   `json:"rationale"`
	TimeEstimate string   `json:"time_estimate"`
}

// TestingPlan represents testing strategy
type TestingPlan struct {
	UnitTests        []string `json:"unit_tests"`
	IntegrationTests []string `json:"integration_tests"`
	EdgeCases        []string `json:"edge_cases"`
}

// GeneratedCode represents implemented code
type GeneratedCode struct {
	Files           []CodeFile `json:"files"`
	Summary         string     `json:"summary"`
	RunInstructions string     `json:"run_instructions"`
}

// CodeFile represents a single code file
type CodeFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Purpose  string `json:"purpose"`
}

// CodeReview represents review phase output
type CodeReview struct {
	Score          float64       `json:"score"`
	Summary        string        `json:"summary"`
	Strengths      []string      `json:"strengths"`
	Improvements   []Improvement `json:"improvements"`
	SecurityIssues []string      `json:"security_issues"`
	BestPractices  []string      `json:"best_practices"`
}

// Improvement represents a suggested improvement
type Improvement struct {
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Suggestion  string `json:"suggestion"`
}