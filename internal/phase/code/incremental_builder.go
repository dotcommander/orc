package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotcommander/orc/internal/core"
	"github.com/dotcommander/orc/internal/phase"
)

// IncrementalBuilder builds code incrementally like our fiction writer builds scenes
type IncrementalBuilder struct {
	BasePhase
	agent   core.Agent
	storage core.Storage
	logger  *slog.Logger
}

// BuildPlan represents the systematic approach to building the project
type BuildPlan struct {
	Phases       []BuildPhase       `json:"phases"`
	FileStructure FileStructure     `json:"file_structure"`
	Dependencies  []Dependency      `json:"dependencies"`
	TestStrategy  TestStrategy      `json:"test_strategy"`
	Context      map[string]interface{} `json:"context"`
}

type BuildPhase struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Deliverables []Deliverable `json:"deliverables"`
	Dependencies []string     `json:"dependencies"`
	EstimatedEffort string    `json:"estimated_effort"`
	Success      []string     `json:"success_criteria"`
}

type Deliverable struct {
	Type        string   `json:"type"` // file, test, config, documentation
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Size        string   `json:"size"` // small, medium, large
	Content     string   `json:"content,omitempty"`
	Status      string   `json:"status"` // planned, in_progress, completed, validated
}

type FileStructure struct {
	RootDir     string                 `json:"root_dir"`
	Directories []string               `json:"directories"`
	Files       map[string]FileSpec    `json:"files"`
	Rationale   string                 `json:"rationale"`
}

type FileSpec struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Purpose     string   `json:"purpose"`
	Size        string   `json:"size"`
	Dependencies []string `json:"dependencies"`
}

type Dependency struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Type        string `json:"type"` // library, framework, tool
	Purpose     string `json:"purpose"`
	InstallCmd  string `json:"install_cmd,omitempty"`
}

type TestStrategy struct {
	Framework   string   `json:"framework"`
	Coverage    string   `json:"coverage_target"`
	Types       []string `json:"types"` // unit, integration, e2e
	Approach    string   `json:"approach"`
}

// BuildProgress tracks the current state of incremental building
type BuildProgress struct {
	CurrentPhase    int                 `json:"current_phase"`
	CompletedPhases []string            `json:"completed_phases"`
	ActiveFiles     []string            `json:"active_files"`
	GeneratedCode   map[string]string   `json:"generated_code"`
	TestResults     map[string]string   `json:"test_results"`
	ValidationNotes []ValidationNote    `json:"validation_notes"`
	Context         map[string]interface{} `json:"context"`
}

type ValidationNote struct {
	File        string    `json:"file"`
	Type        string    `json:"type"` // success, warning, improvement
	Message     string    `json:"message"`
	Suggestion  string    `json:"suggestion,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

func NewIncrementalBuilder(agent core.Agent, storage core.Storage, logger *slog.Logger) *IncrementalBuilder {
	return &IncrementalBuilder{
		BasePhase: NewBasePhase("IncrementalBuilder", 15*time.Minute),
		agent:     agent,
		storage:   storage,
		logger:    logger.With("component", "incremental_builder"),
	}
}

func (ib *IncrementalBuilder) Name() string {
	return "IncrementalBuilder"
}

func (ib *IncrementalBuilder) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	ib.logger.Info("Starting incremental building")

	// Extract exploration from previous phase
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid input data format")
	}
	
	explorationData, ok := data["exploration"]
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("exploration data not found in input")
	}

	exploration, ok := explorationData.(*ProjectExploration)
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid exploration data type")
	}

	// Create systematic build plan
	buildPlan, err := ib.createSystematicBuildPlan(ctx, exploration)
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("failed to create build plan: %w", err)
	}

	// Execute incremental building
	progress := &BuildProgress{
		CurrentPhase:    0,
		CompletedPhases: make([]string, 0),
		ActiveFiles:     make([]string, 0),
		GeneratedCode:   make(map[string]string),
		TestResults:     make(map[string]string),
		ValidationNotes: make([]ValidationNote, 0),
		Context:         make(map[string]interface{}),
	}

	// Execute each phase incrementally
	for i, phase := range buildPlan.Phases {
		ib.logger.Info("Executing build phase", "phase", phase.Name, "index", i)
		
		progress.CurrentPhase = i
		if err := ib.executePhase(ctx, phase, buildPlan, exploration, progress); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to execute phase %s: %w", phase.Name, err)
		}
		
		progress.CompletedPhases = append(progress.CompletedPhases, phase.Name)
	}

	ib.logger.Info("Incremental building completed",
		"phases_completed", len(progress.CompletedPhases),
		"files_generated", len(progress.GeneratedCode),
		"validation_notes", len(progress.ValidationNotes))

	return core.PhaseOutput{
		Data: map[string]interface{}{
			"exploration":    exploration,     // Pass along exploration data
			"build_plan":     buildPlan,
			"build_progress": progress,
			"generated_code": progress.GeneratedCode,
		},
	}, nil
}

func (ib *IncrementalBuilder) createSystematicBuildPlan(ctx context.Context, exploration *ProjectExploration) (*BuildPlan, error) {
	prompt := ib.buildPlanningPrompt(exploration)
	
	response, err := ib.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get build plan: %w", err)
	}

	buildPlan, err := ib.parseBuildPlan(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build plan: %w", err)
	}

	// Add project context to build plan
	buildPlan.Context = map[string]interface{}{
		"exploration":      exploration,
		"planning_response": response,
	}

	return buildPlan, nil
}

func (ib *IncrementalBuilder) executePhase(ctx context.Context, phase BuildPhase, plan *BuildPlan, exploration *ProjectExploration, progress *BuildProgress) error {
	for _, deliverable := range phase.Deliverables {
		if deliverable.Type == "file" {
			if err := ib.generateFile(ctx, deliverable, phase, plan, exploration, progress); err != nil {
				return fmt.Errorf("failed to generate file %s: %w", deliverable.Name, err)
			}
		}
	}
	return nil
}

func (ib *IncrementalBuilder) generateFile(ctx context.Context, deliverable Deliverable, phase BuildPhase, plan *BuildPlan, exploration *ProjectExploration, progress *BuildProgress) error {
	prompt := ib.buildFileGenerationPrompt(deliverable, phase, plan, exploration, progress)
	
	response, err := ib.agent.Execute(ctx, prompt, nil)
	if err != nil {
		return fmt.Errorf("failed to generate file content: %w", err)
	}

	fileContent, validationNotes, err := ib.parseFileGeneration(response)
	if err != nil {
		return fmt.Errorf("failed to parse file generation: %w", err)
	}

	// Store generated code
	progress.GeneratedCode[deliverable.Path] = fileContent
	progress.ActiveFiles = append(progress.ActiveFiles, deliverable.Path)
	
	// Save file to disk
	if err := ib.saveFileToDisk(ctx, deliverable.Path, fileContent); err != nil {
		return fmt.Errorf("failed to save file to disk: %w", err)
	}
	
	// Add validation notes
	for _, note := range validationNotes {
		note.File = deliverable.Path
		note.Timestamp = time.Now()
		progress.ValidationNotes = append(progress.ValidationNotes, note)
	}

	ib.logger.Info("Generated file", "path", deliverable.Path, "size", len(fileContent))
	return nil
}

func (ib *IncrementalBuilder) saveFileToDisk(ctx context.Context, filePath, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := ib.storage.Save(ctx, dir+"/.gitkeep", []byte("")); err != nil {
			// Try to create directory structure using os.MkdirAll as fallback
			if err := os.MkdirAll(filepath.Join(".", dir), 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}
	
	// Save the actual file
	return ib.storage.Save(ctx, filePath, []byte(content))
}

func (ib *IncrementalBuilder) buildPlanningPrompt(exploration *ProjectExploration) string {
	features := make([]string, len(exploration.Features))
	for i, f := range exploration.Features {
		features[i] = fmt.Sprintf("- %s (%s priority): %s", f.Name, f.Priority, f.Description)
	}

	return fmt.Sprintf(`You are a senior software architect creating a systematic build plan for a %s project.

Project Overview:
%s

Features to implement:
%s

Tech Stack:
- Language: %s
- Framework: %s
- Architecture: %s

Quality Goals:
- Security: %s
- Performance: %s
- Maintainability: %s

Create a systematic, incremental build plan that works like water - flowing naturally from simple to complex.

Break the project into logical phases where each phase:
1. Builds on previous phases
2. Delivers working, testable components
3. Maintains full project context
4. Allows for validation and refinement

Return your response in this JSON format:
{
  "phases": [
    {
      "name": "Phase name",
      "description": "What this phase accomplishes",
      "deliverables": [
        {
          "type": "file",
          "name": "filename.ext",
          "path": "relative/path/filename.ext",
          "description": "Purpose of this file",
          "size": "small|medium|large"
        }
      ],
      "dependencies": ["previous phase names"],
      "estimated_effort": "time estimate",
      "success_criteria": ["criteria1", "criteria2"]
    }
  ],
  "file_structure": {
    "root_dir": "project_name",
    "directories": ["dir1", "dir2", "dir3"],
    "files": {
      "filename": {
        "path": "relative/path/filename.ext",
        "type": "source|config|test|documentation",
        "purpose": "What this file does",
        "size": "small|medium|large",
        "dependencies": ["other files this depends on"]
      }
    },
    "rationale": "Why this structure makes sense"
  },
  "dependencies": [
    {
      "name": "dependency name",
      "version": "version if applicable",
      "type": "library|framework|tool",
      "purpose": "Why this is needed",
      "install_cmd": "installation command if applicable"
    }
  ],
  "test_strategy": {
    "framework": "testing framework to use",
    "coverage_target": "coverage percentage",
    "types": ["unit", "integration"],
    "approach": "how testing will be approached"
  }
}`,
		exploration.ProjectType,
		strings.Join(exploration.Requirements, "; "),
		strings.Join(features, "\n"),
		exploration.TechStack.Language,
		exploration.TechStack.Framework,
		exploration.Architecture.Pattern,
		exploration.QualityGoals.Security,
		exploration.QualityGoals.Performance,
		exploration.QualityGoals.Maintainability)
}

func (ib *IncrementalBuilder) buildFileGenerationPrompt(deliverable Deliverable, phase BuildPhase, plan *BuildPlan, exploration *ProjectExploration, progress *BuildProgress) string {
	// Build context from previous files
	contextFiles := make([]string, 0)
	for path, content := range progress.GeneratedCode {
		if len(content) > 0 {
			contextFiles = append(contextFiles, fmt.Sprintf("%s:\n"+"`"+"``"+"\n%s\n"+"`"+"``", path, content))
		}
	}

	return fmt.Sprintf(`You are implementing file "%s" as part of phase "%s" in a %s project.

File Specification:
- Path: %s
- Purpose: %s
- Size: %s
- Type: Source code file

Project Context:
%s

Tech Stack: %s with %s architecture
Quality Focus: %s

Current Phase Context:
%s

Previously Generated Files:
%s

Phase Success Criteria:
%s

Generate high-quality, production-ready code that:
1. Implements the specified functionality completely
2. Follows best practices for %s
3. Includes proper error handling and validation
4. Has clear, maintainable structure
5. Integrates well with existing files
6. Meets the quality goals specified

Return your response in this JSON format:
{
  "file_content": "Complete file content here",
  "implementation_notes": "Key implementation decisions and rationale",
  "validation_notes": [
    {
      "type": "success|warning|improvement",
      "message": "validation message",
      "suggestion": "improvement suggestion if applicable"
    }
  ],
  "next_steps": ["What should be done next"],
  "integration_points": ["How this integrates with other files"]
}`,
		deliverable.Name,
		phase.Name,
		exploration.ProjectType,
		deliverable.Path,
		deliverable.Description,
		deliverable.Size,
		strings.Join(exploration.Requirements, "; "),
		exploration.TechStack.Language,
		exploration.Architecture.Pattern,
		exploration.QualityGoals.Security,
		phase.Description,
		strings.Join(contextFiles, "\n\n"),
		strings.Join(phase.Success, "; "),
		exploration.TechStack.Language)
}

func (ib *IncrementalBuilder) parseBuildPlan(response string) (*BuildPlan, error) {
	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	var plan BuildPlan
	if err := json.Unmarshal([]byte(cleanedResponse), &plan); err != nil {
		ib.logger.Error("Failed to parse build plan JSON",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return nil, fmt.Errorf("failed to parse build plan JSON: %w", err)
	}
	return &plan, nil
}

func (ib *IncrementalBuilder) parseFileGeneration(response string) (string, []ValidationNote, error) {
	// Clean the response to remove markdown code blocks
	cleanedResponse := phase.CleanJSONResponse(response)
	
	var result struct {
		FileContent         string `json:"file_content"`
		ImplementationNotes string `json:"implementation_notes"`
		ValidationNotes     []struct {
			Type       string `json:"type"`
			Message    string `json:"message"`
			Suggestion string `json:"suggestion"`
		} `json:"validation_notes"`
		NextSteps         []string `json:"next_steps"`
		IntegrationPoints []string `json:"integration_points"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		ib.logger.Error("Failed to parse file generation JSON",
			"error", err,
			"original_response", response,
			"cleaned_response", cleanedResponse)
		return "", nil, fmt.Errorf("failed to parse file generation JSON: %w", err)
	}

	validationNotes := make([]ValidationNote, len(result.ValidationNotes))
	for i, note := range result.ValidationNotes {
		validationNotes[i] = ValidationNote{
			Type:       note.Type,
			Message:    note.Message,
			Suggestion: note.Suggestion,
		}
	}

	return result.FileContent, validationNotes, nil
}

func (ib *IncrementalBuilder) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	if input.Data == nil {
		return fmt.Errorf("exploration data is required for incremental building")
	}
	return nil
}

func (ib *IncrementalBuilder) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	if output.Data == nil {
		return fmt.Errorf("build data is required")
	}
	return nil
}

func (ib *IncrementalBuilder) EstimatedDuration() time.Duration {
	return 5 * time.Minute
}

func (ib *IncrementalBuilder) CanRetry(err error) bool {
	return true
}