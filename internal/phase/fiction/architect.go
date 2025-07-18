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

type Architect struct {
	BasePhase
	agent       core.Agent
	storage     core.Storage
	promptPath  string
	validator   core.Validator
	errorFactory core.ErrorFactory
}

func NewArchitect(agent core.Agent, storage core.Storage, promptPath string) *Architect {
	return &Architect{
		BasePhase:    NewBasePhase("Architecture", 10*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewBaseValidator("Architecture"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func NewArchitectWithTimeout(agent core.Agent, storage core.Storage, promptPath string, timeout time.Duration) *Architect {
	return &Architect{
		BasePhase:    NewBasePhase("Architecture", timeout),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewBaseValidator("Architecture"),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func (a *Architect) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	slog.Debug("Validating architect input",
		"phase", a.Name(),
		"request_length", len(input.Request),
		"has_data", input.Data != nil,
	)
	
	// First run consolidated validation
	if err := a.validator.ValidateRequired("request", input.Request, "input"); err != nil {
		slog.Error("Architect request validation failed",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	plan, ok := input.Data.(NovelPlan)
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "data", 
			"input data must be a NovelPlan", fmt.Sprintf("%T", input.Data))
		slog.Error("Architect input type validation failed",
			"phase", a.Name(),
			"expected_type", "NovelPlan",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return err
	}
	
	// Validate plan has required fields
	if err := a.validator.ValidateRequired("title", plan.Title, "input"); err != nil {
		slog.Error("Architect plan title validation failed",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	if len(plan.Chapters) == 0 {
		err := a.errorFactory.NewValidationError(a.Name(), "input", "chapters", 
			"plan contains no chapters", plan.Chapters)
		slog.Error("Architect plan chapters validation failed",
			"phase", a.Name(),
			"chapter_count", 0,
			"error", err,
		)
		return err
	}
	
	slog.Debug("Architect input validation successful",
		"phase", a.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
	)
	
	return nil
}

func (a *Architect) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	slog.Debug("Validating architect output",
		"phase", a.Name(),
		"has_data", output.Data != nil,
	)
	
	// First run consolidated validation
	if output.Data == nil {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "data", "output data cannot be nil", nil)
		slog.Error("Architect output data is nil",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	if err := a.validator.ValidateJSON("data", output.Data, "output"); err != nil {
		slog.Error("Architect output JSON validation failed",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "data", 
			"output data must be a map containing architecture", fmt.Sprintf("%T", output.Data))
		slog.Error("Architect output type validation failed",
			"phase", a.Name(),
			"expected_type", "map[string]interface{}",
			"actual_type", fmt.Sprintf("%T", output.Data),
			"error", err,
		)
		return err
	}
	
	architecture, ok := outputMap["architecture"].(NovelArchitecture)
	if !ok {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "architecture", 
			"architecture data missing from output", "missing")
		slog.Error("Architecture data missing from output",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	// Validate architecture has required elements
	if len(architecture.Characters) == 0 {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "characters", 
			"architecture contains no characters", architecture.Characters)
		slog.Error("Architecture contains no characters",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	if len(architecture.Settings) == 0 {
		err := a.errorFactory.NewValidationError(a.Name(), "output", "settings", 
			"architecture contains no settings", architecture.Settings)
		slog.Error("Architecture contains no settings",
			"phase", a.Name(),
			"error", err,
		)
		return err
	}
	
	slog.Debug("Architect output validation successful",
		"phase", a.Name(),
		"character_count", len(architecture.Characters),
		"setting_count", len(architecture.Settings),
		"theme_count", len(architecture.Themes),
	)
	
	return nil
}

func (a *Architect) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting architect execution",
		"phase", a.Name(),
		"prompt_path", a.promptPath,
	)
	
	plan, ok := input.Data.(NovelPlan)
	if !ok {
		err := fmt.Errorf("invalid input: expected NovelPlan")
		slog.Error("Architect input type error",
			"phase", a.Name(),
			"expected_type", "NovelPlan",
			"actual_type", fmt.Sprintf("%T", input.Data),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	slog.Debug("Processing plan for architecture",
		"phase", a.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
	)
	
	planJSON, _ := json.Marshal(plan)
	
	// Execute prompt template
	templateData := map[string]interface{}{
		"Plan":     plan,
		"PlanJSON": string(planJSON),
	}
	
	slog.Debug("Loading and executing prompt template",
		"phase", a.Name(),
		"template_data_keys", []string{"Plan", "PlanJSON"},
	)
	
	prompt, err := phase.LoadAndExecutePrompt(a.promptPath, templateData)
	if err != nil {
		slog.Error("Failed to load/execute prompt template",
			"phase", a.Name(),
			"prompt_path", a.promptPath,
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("loading prompt: %w", err)
	}
	
	slog.Debug("Calling AI agent for architecture",
		"phase", a.Name(),
		"prompt_length", len(prompt),
	)
	
	response, err := a.agent.ExecuteJSON(ctx, prompt, "")
	if err != nil {
		slog.Error("AI agent execution failed",
			"phase", a.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("calling AI: %w", err)
	}
	
	slog.Debug("Received AI response",
		"phase", a.Name(),
		"response_length", len(response),
	)
	
	var architecture NovelArchitecture
	if err := json.Unmarshal([]byte(response), &architecture); err != nil {
		slog.Error("Failed to parse architecture JSON",
			"phase", a.Name(),
			"error", err,
			"response_preview", truncateString(response, 500),
		)
		return core.PhaseOutput{}, fmt.Errorf("parsing architecture: %w", err)
	}
	
	slog.Info("Successfully parsed novel architecture",
		"phase", a.Name(),
		"character_count", len(architecture.Characters),
		"setting_count", len(architecture.Settings),
		"theme_count", len(architecture.Themes),
	)
	
	// Log character summaries
	for i, char := range architecture.Characters {
		if i < 3 { // Only log first 3 characters
			slog.Debug("Character details",
				"phase", a.Name(),
				"character_name", char.Name,
				"role", char.Role,
				"description_preview", truncateString(char.Description, 100),
			)
		}
	}
	
	archData, err := json.MarshalIndent(architecture, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal architecture for storage",
			"phase", a.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("marshaling architecture: %w", err)
	}
	
	slog.Debug("Saving architecture to storage",
		"phase", a.Name(),
		"file", "architecture.json",
		"size", len(archData),
	)
	
	if err := a.storage.Save(ctx, "architecture.json", archData); err != nil {
		slog.Error("Failed to save architecture",
			"phase", a.Name(),
			"file", "architecture.json",
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("saving architecture: %w", err)
	}
	
	slog.Info("Architect execution completed successfully",
		"phase", a.Name(),
	)
	
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"plan":         plan,
			"architecture": architecture,
		},
	}, nil
}