package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotcommander/orc/internal/core"
	"github.com/dotcommander/orc/internal/phase"
)

type Critic struct {
	BasePhase
	agent        core.Agent
	storage      core.Storage
	promptPath   string
	validator    core.Validator
	errorFactory core.ErrorFactory
}

func NewCritic(agent core.Agent, storage core.Storage, promptPath string) *Critic {
	return &Critic{
		BasePhase:    NewBasePhase("Critique", 5*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewBaseValidator("Critique"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func NewCriticWithTimeout(agent core.Agent, storage core.Storage, promptPath string, timeout time.Duration) *Critic {
	return &Critic{
		BasePhase:    NewBasePhase("Critique", timeout),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewBaseValidator("Critique"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func (c *Critic) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	slog.Debug("Validating critic input",
		"phase", c.Name(),
		"request_length", len(input.Request),
		"has_data", input.Data != nil,
	)
	
	// First run consolidated validation
	if err := c.validator.ValidateRequired("request", input.Request, "input"); err != nil {
		slog.Error("Critic request validation failed",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		err := c.errorFactory.NewValidationError(c.Name(), "input", "data", 
			"input data must be a map containing manuscript", fmt.Sprintf("%T", input.Data))
		slog.Error("Critic input type validation failed",
			"phase", c.Name(),
			"expected_type", "map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return err
	}
	
	manuscript, ok := data["manuscript"].(string)
	if !ok {
		err := c.errorFactory.NewValidationError(c.Name(), "input", "manuscript", 
			"manuscript data missing from input", "missing")
		slog.Error("Manuscript data missing from critic input",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate manuscript has content
	if err := c.validator.ValidateRequired("manuscript", manuscript, "input"); err != nil {
		slog.Error("Manuscript validation failed",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate manuscript has minimum length
	if len(manuscript) < 500 {
		err := c.errorFactory.NewValidationError(c.Name(), "input", "manuscript", 
			"manuscript appears too short for meaningful critique", len(manuscript))
		slog.Error("Manuscript too short for critique",
			"phase", c.Name(),
			"length", len(manuscript),
			"minimum", 500,
			"error", err,
		)
		return err
	}
	
	slog.Debug("Critic input validation successful",
		"phase", c.Name(),
		"manuscript_length", len(manuscript),
	)
	
	return nil
}

func (c *Critic) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	slog.Debug("Validating critic output",
		"phase", c.Name(),
		"has_data", output.Data != nil,
	)
	
	// First run consolidated validation
	if output.Data == nil {
		err := c.errorFactory.NewValidationError(c.Name(), "output", "data", "output data cannot be nil", nil)
		slog.Error("Critic output data is nil",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	if err := c.validator.ValidateJSON("data", output.Data, "output"); err != nil {
		slog.Error("Critic output JSON validation failed",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	// Check if output is a Critique directly (what Execute returns)
	if critique, ok := output.Data.(Critique); ok {
		// Validate critique has required content
		if len(critique.Strengths) == 0 && len(critique.Weaknesses) == 0 && len(critique.Suggestions) == 0 {
			err := c.errorFactory.NewValidationError(c.Name(), "output", "critique", 
				"critique must have at least one strength, weakness, or suggestion", "empty")
			slog.Error("Critique validation failed - empty critique",
				"phase", c.Name(),
				"error", err,
			)
			return err
		}
		
		slog.Debug("Critic output validation successful",
			"phase", c.Name(),
			"overall_rating", critique.OverallRating,
			"strength_count", len(critique.Strengths),
			"weakness_count", len(critique.Weaknesses),
			"suggestion_count", len(critique.Suggestions),
		)
		return nil
	}
	
	// Legacy check for map output
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		err := c.errorFactory.NewValidationError(c.Name(), "output", "data", 
			"output data must be a Critique or map containing critique results", fmt.Sprintf("%T", output.Data))
		slog.Error("Critic output type validation failed",
			"phase", c.Name(),
			"expected_types", "Critique or map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", output.Data),
			"error", err,
		)
		return err
	}
	
	// Validate critique results exist
	critiqueData, hasCritique := outputMap["critique"]
	if !hasCritique {
		err := c.errorFactory.NewValidationError(c.Name(), "output", "critique", 
			"critique results missing from output", "missing")
		slog.Error("Critique results missing from output map",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate critique structure
	var critique NovelCritique
	switch v := critiqueData.(type) {
	case NovelCritique:
		critique = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &critique); err != nil {
			err := c.errorFactory.NewValidationError(c.Name(), "output", "critique", 
				fmt.Sprintf("failed to parse critique: %v", err), critiqueData)
			slog.Error("Failed to parse critique from map",
				"phase", c.Name(),
				"error", err,
			)
			return err
		}
	default:
		err := c.errorFactory.NewValidationError(c.Name(), "output", "critique", 
			"critique must be NovelCritique type", fmt.Sprintf("%T", critiqueData))
		slog.Error("Invalid critique type in output map",
			"phase", c.Name(),
			"expected_type", "NovelCritique",
			"actual_type", fmt.Sprintf("%T", critiqueData),
			"error", err,
		)
		return err
	}
	
	// Validate critique has summary
	if err := c.validator.ValidateRequired("summary", critique.Summary, "output"); err != nil {
		slog.Error("Critique summary validation failed",
			"phase", c.Name(),
			"error", err,
		)
		return err
	}
	
	slog.Debug("Critic output validation successful",
		"phase", c.Name(),
		"has_summary", critique.Summary != "",
	)
	
	return nil
}

func (c *Critic) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting critic execution",
		"phase", c.Name(),
		"prompt_path", c.promptPath,
	)
	
	var manuscript string
	
	// Handle both string and map input from assembler
	switch data := input.Data.(type) {
	case string:
		manuscript = data
		slog.Debug("Received manuscript as string",
			"phase", c.Name(),
			"length", len(manuscript),
		)
	case map[string]interface{}:
		if ms, ok := data["manuscript"].(string); ok {
			manuscript = ms
			slog.Debug("Extracted manuscript from map",
				"phase", c.Name(),
				"length", len(manuscript),
			)
		} else {
			err := fmt.Errorf("missing manuscript in input map")
			slog.Error("Manuscript missing from input map",
				"phase", c.Name(),
				"error", err,
			)
			return core.PhaseOutput{}, err
		}
	default:
		err := fmt.Errorf("invalid input data type: %T", input.Data)
		slog.Error("Invalid input data type for critic",
			"phase", c.Name(),
			"expected_types", "string or map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	// Load and execute prompt template
	templateData := map[string]interface{}{
		"Manuscript": manuscript,
	}
	
	slog.Debug("Loading and executing prompt template",
		"phase", c.Name(),
		"manuscript_preview", truncateString(manuscript, 200),
	)
	
	prompt, err := phase.LoadAndExecutePrompt(c.promptPath, templateData)
	if err != nil {
		slog.Error("Failed to load/execute prompt template",
			"phase", c.Name(),
			"prompt_path", c.promptPath,
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("loading prompt: %w", err)
	}
	
	slog.Debug("Calling AI agent for critique",
		"phase", c.Name(),
		"prompt_length", len(prompt),
	)
	
	response, err := c.agent.ExecuteJSON(ctx, prompt, "")
	if err != nil {
		slog.Error("AI agent execution failed",
			"phase", c.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("calling AI: %w", err)
	}
	
	slog.Debug("Received AI response",
		"phase", c.Name(),
		"response_length", len(response),
	)
	
	var critique Critique
	if err := json.Unmarshal([]byte(response), &critique); err != nil {
		slog.Error("Failed to parse critique JSON",
			"phase", c.Name(),
			"error", err,
			"response_preview", truncateString(response, 500),
		)
		return core.PhaseOutput{}, fmt.Errorf("parsing critique: %w", err)
	}
	
	slog.Info("Successfully parsed critique",
		"phase", c.Name(),
		"overall_rating", critique.OverallRating,
		"strength_count", len(critique.Strengths),
		"weakness_count", len(critique.Weaknesses),
		"suggestion_count", len(critique.Suggestions),
	)
	
	// Log some critique details
	if len(critique.Strengths) > 0 {
		slog.Debug("Critique strengths",
			"phase", c.Name(),
			"first_strength", truncateString(critique.Strengths[0], 100),
		)
	}
	if len(critique.Weaknesses) > 0 {
		slog.Debug("Critique weaknesses",
			"phase", c.Name(),
			"first_weakness", truncateString(critique.Weaknesses[0], 100),
		)
	}
	
	critiqueData, err := json.MarshalIndent(critique, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal critique for storage",
			"phase", c.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("marshaling critique: %w", err)
	}
	
	slog.Debug("Saving critique to storage",
		"phase", c.Name(),
		"file", "critique.json",
		"size", len(critiqueData),
	)
	
	if err := c.storage.Save(ctx, "critique.json", critiqueData); err != nil {
		slog.Error("Failed to save critique",
			"phase", c.Name(),
			"file", "critique.json",
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("saving critique: %w", err)
	}
	
	slog.Info("Critic execution completed successfully",
		"phase", c.Name(),
	)
	
	return core.PhaseOutput{
		Data: critique,
	}, nil
}