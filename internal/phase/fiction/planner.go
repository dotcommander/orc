package fiction

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
	"github.com/vampirenirmal/orchestrator/internal/phase"
)

type Planner struct {
	BasePhase
	agent        core.Agent
	storage      core.Storage
	promptPath   string
	validator    core.PhaseValidator
	errorFactory core.ErrorFactory
}

func NewPlanner(agent core.Agent, storage core.Storage, promptPath string) *Planner {
	return &Planner{
		BasePhase:    NewBasePhase("Planning", 5*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewStandardPhaseValidator("Planning", core.ValidationRules{MinRequestLength: 10, MaxRequestLength: 10000}),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func NewPlannerWithTimeout(agent core.Agent, storage core.Storage, promptPath string, timeout time.Duration) *Planner {
	return &Planner{
		BasePhase:    NewBasePhase("Planning", timeout),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		validator:    core.NewStandardPhaseValidator("Planning", core.ValidationRules{MinRequestLength: 10, MaxRequestLength: 10000}),
		errorFactory: core.NewDefaultErrorFactory(),
	}
}

func (p *Planner) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	slog.Debug("Validating planner input",
		"phase", p.Name(),
		"request_length", len(input.Request),
		"has_data", input.Data != nil,
	)
	
	// Use the consolidated validator
	err := p.validator.ValidateInput(ctx, input)
	if err != nil {
		slog.Error("Planner input validation failed",
			"phase", p.Name(),
			"error", err,
		)
	} else {
		slog.Debug("Planner input validation successful",
			"phase", p.Name(),
		)
	}
	return err
}

func (p *Planner) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	slog.Debug("Validating planner output",
		"phase", p.Name(),
		"has_data", output.Data != nil,
	)
	
	// First run standard validation
	if err := p.validator.ValidateOutput(ctx, output); err != nil {
		slog.Error("Planner standard output validation failed",
			"phase", p.Name(),
			"error", err,
		)
		return err
	}
	
	plan, ok := output.Data.(NovelPlan)
	if !ok {
		err := p.errorFactory.NewValidationError(p.Name(), "output", "data",
			fmt.Sprintf("output data must be a NovelPlan, got %T", output.Data), output.Data)
		slog.Error("Planner output type validation failed",
			"phase", p.Name(),
			"expected_type", "NovelPlan",
			"actual_type", fmt.Sprintf("%T", output.Data),
			"error", err,
		)
		return err
	}
	
	// Log plan summary
	slog.Debug("Planner output validation successful",
		"phase", p.Name(),
		"plan_title", plan.Title,
		"chapter_count", len(plan.Chapters),
		"theme_count", len(plan.Themes),
	)
	
	// Use basic validation since the plan structure is already validated
	return nil
}

func (p *Planner) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	slog.Info("Starting planner execution",
		"phase", p.Name(),
		"request_preview", truncateString(input.Request, 100),
		"prompt_path", p.promptPath,
	)
	
	// Debug prompt path
	if p.promptPath == "" {
		err := fmt.Errorf("prompt path is empty")
		slog.Error("Planner configuration error",
			"phase", p.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	// Execute prompt template with the request
	templateData := map[string]interface{}{
		"UserRequest": input.Request,
		"Request":     input.Request,
	}
	
	slog.Debug("Loading and executing prompt template",
		"phase", p.Name(),
		"template_data_keys", getMapKeys(templateData),
	)
	
	prompt, err := phase.LoadAndExecutePrompt(p.promptPath, templateData)
	if err != nil {
		slog.Error("Failed to load/execute prompt template",
			"phase", p.Name(),
			"prompt_path", p.promptPath,
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("loading prompt: %w", err)
	}
	
	slog.Debug("Calling AI agent",
		"phase", p.Name(),
		"prompt_length", len(prompt),
	)
	
	// Use template rendering with JSON enforcement
	response, err := p.agent.ExecuteJSON(ctx, prompt, "")
	if err != nil {
		slog.Error("AI agent execution failed",
			"phase", p.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("calling AI: %w", err)
	}
	
	slog.Debug("Received AI response",
		"phase", p.Name(),
		"response_length", len(response),
		"response_preview", truncateString(response, 200),
	)
	
	// Debug output
	if len(response) < 100 && strings.Contains(response, "Hello") {
		err := fmt.Errorf("AI returned greeting instead of JSON. Response: %s. PromptPath: %s", response, p.promptPath)
		slog.Error("Invalid AI response",
			"phase", p.Name(),
			"response", response,
			"prompt_path", p.promptPath,
			"error", err,
		)
		return core.PhaseOutput{}, err
	}
	
	var plan NovelPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		slog.Error("Failed to parse plan JSON",
			"phase", p.Name(),
			"error", err,
			"response_preview", truncateString(response, 500),
		)
		return core.PhaseOutput{}, fmt.Errorf("parsing plan: %w", err)
	}
	
	slog.Info("Successfully parsed novel plan",
		"phase", p.Name(),
		"title", plan.Title,
		"theme_count", len(plan.Themes),
		"chapter_count", len(plan.Chapters),
		"logline", truncateString(plan.Logline, 100),
	)
	
	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal plan for storage",
			"phase", p.Name(),
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("marshaling plan: %w", err)
	}
	
	slog.Debug("Saving plan to storage",
		"phase", p.Name(),
		"file", "plan.json",
		"size", len(planData),
	)
	
	if err := p.storage.Save(ctx, "plan.json", planData); err != nil {
		slog.Error("Failed to save plan",
			"phase", p.Name(),
			"file", "plan.json",
			"error", err,
		)
		return core.PhaseOutput{}, fmt.Errorf("saving plan: %w", err)
	}
	
	slog.Info("Planner execution completed successfully",
		"phase", p.Name(),
		"plan_title", plan.Title,
	)
	
	return core.PhaseOutput{
		Data: plan,
	}, nil
}