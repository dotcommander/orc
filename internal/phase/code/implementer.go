package code

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// Implementer phase generates code based on plan
type Implementer struct {
	BasePhase
	agent      core.Agent
	storage    core.Storage
	promptPath string
	logger       *slog.Logger
	validator    core.Validator
	errorFactory core.ErrorFactory
	resilience   *core.PhaseResilience
}

// NewImplementer creates a new implementer phase
func NewImplementer(agent core.Agent, storage core.Storage, promptPath string, logger *slog.Logger) *Implementer {
	return &Implementer{
		BasePhase:    NewBasePhase("Implementation", 15*time.Minute),
		agent:        agent,
		storage:      storage,
		promptPath:   promptPath,
		logger:       logger,
		validator:    core.NewBaseValidator("Implementation"),
		errorFactory: core.NewDefaultErrorFactory(),
		resilience:   core.NewPhaseResilience(),
	}
}

// Execute generates code based on the plan
func (impl *Implementer) Execute(ctx context.Context, input core.PhaseInput) (core.PhaseOutput, error) {
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return core.PhaseOutput{}, fmt.Errorf("invalid input data")
	}
	
	// Extract plan data - handle both ImplementationPlan type and map
	var planData interface{}
	if plan, ok := data["plan"].(ImplementationPlan); ok {
		planData = plan
	} else if planMap, ok := data["plan"].(map[string]interface{}); ok {
		planData = planMap
	} else {
		return core.PhaseOutput{}, fmt.Errorf("missing or invalid plan in input: got %T", data["plan"])
	}
	
	// Extract analysis for full context
	analysisData, _ := data["analysis"].(map[string]interface{})
	
	// Create comprehensive context for code generation
	fullContext := map[string]interface{}{
		"analysis": analysisData,
		"plan":     planData,
		"request":  input.Request,
	}
	
	contextJSON, _ := json.MarshalIndent(fullContext, "", "  ")
	
	// Use resilient AI call with retry and fallback mechanisms
	var generated GeneratedCode
	
	result, err := impl.resilience.ExecuteWithFallbacks(ctx, "implementation", func() (interface{}, error) {
		return impl.executeImplementationWithRetry(ctx, string(contextJSON))
	}, string(contextJSON))
	
	if err != nil {
		return core.PhaseOutput{}, fmt.Errorf("implementation failed after retries and fallbacks: %w", err)
	}
	
	// Convert result to GeneratedCode
	switch v := result.(type) {
	case GeneratedCode:
		generated = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("failed to convert implementation result: %w", err)
		}
	default:
		return core.PhaseOutput{}, fmt.Errorf("unexpected implementation result type: %T", result)
	}
	
	// Validate we got actual code
	if len(generated.Files) == 0 {
		return core.PhaseOutput{}, fmt.Errorf("no code files generated")
	}
	
	// Save each generated file
	for _, file := range generated.Files {
		filePath := filepath.Join("generated_code", file.Path)
		if err := impl.storage.Save(ctx, filePath, []byte(file.Content)); err != nil {
			return core.PhaseOutput{}, fmt.Errorf("saving file %s: %w", file.Path, err)
		}
	}
	
	// Save generation metadata
	metaData, _ := json.MarshalIndent(generated, "", "  ")
	if err := impl.storage.Save(ctx, "generated_code/metadata.json", metaData); err != nil {
		return core.PhaseOutput{}, fmt.Errorf("saving metadata: %w", err)
	}
	
	return core.PhaseOutput{
		Data: map[string]interface{}{
			"analysis":  analysisData,
			"plan":      planData,
			"generated": generated,
		},
	}, nil
}

// executeImplementationWithRetry performs the primary AI-powered implementation with retry logic
func (impl *Implementer) executeImplementationWithRetry(ctx context.Context, contextJSON string) (GeneratedCode, error) {
	var generated GeneratedCode
	
	// Execute with retry logic
	err := impl.resilience.ExecuteWithRetry(ctx, func() error {
		// Build implementation prompt with context data
		prompt := impl.buildImplementationPrompt(contextJSON)
		response, err := impl.agent.ExecuteJSON(ctx, prompt, nil)
		if err != nil {
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 2 * time.Second,
			}
		}
		
		// Parse the response
		if err := json.Unmarshal([]byte(response), &generated); err != nil {
			impl.logger.Error("Failed to parse AI implementation response", "error", err, "response", response)
			return &core.RetryableError{
				Err:        err,
				RetryAfter: 1 * time.Second,
			}
		}
		
		// Basic validation to ensure we got meaningful data
		if len(generated.Files) == 0 {
			return &core.RetryableError{
				Err:        fmt.Errorf("no code files generated"),
				RetryAfter: 1 * time.Second,
			}
		}
		
		return nil
	}, "ai_implementation")
	
	if err != nil {
		return GeneratedCode{}, err
	}
	
	return generated, nil
}

// buildImplementationPrompt creates the implementation prompt with analysis and plan data
func (impl *Implementer) buildImplementationPrompt(contextJSON string) string {
	return fmt.Sprintf(`You are a senior software engineer implementing code based on the analysis and plan provided. Generate high-quality, production-ready code.

Analysis and Plan:
%s

Please implement the code and return ONLY a JSON response with the following structure:

{
  "files": [
    {
      "path": "main.php",
      "content": "<?php\n// Full file content here",
      "language": "php",
      "purpose": "Main entry point for the PHP application"
    }
  ],
  "summary": "Brief summary of what was implemented",
  "run_instructions": "Step-by-step instructions to run the code"
}

Implementation Guidelines:
1. Code Quality: Follow language-specific best practices and conventions
2. File Organization: Create well-organized file structure
3. Functionality: Implement all requirements from the analysis
4. Documentation: Add meaningful comments and documentation

Language-Specific Guidelines for PHP:
- Use proper PHP syntax and formatting
- Include proper error handling
- Follow PSR coding standards where applicable
- Use meaningful variable and function names
- Include proper HTML structure for web applications

Return ONLY valid JSON, no markdown formatting or explanations.`, contextJSON)
}

// ValidateInput validates input for the implementer phase
func (impl *Implementer) ValidateInput(ctx context.Context, input core.PhaseInput) error {
	impl.logger.Debug("Validating implementer input",
		"has_data", input.Data != nil,
		"data_type", fmt.Sprintf("%T", input.Data))
	
	if input.Data == nil {
		return impl.errorFactory.NewValidationError(impl.Name(), "input", "data", 
			"implementer requires plan data from previous phase", nil)
	}
	
	data, ok := input.Data.(map[string]interface{})
	if !ok {
		return impl.errorFactory.NewValidationError(impl.Name(), "input", "data", 
			"input data must be a map containing plan and analysis", fmt.Sprintf("%T", input.Data))
	}
	
	// Validate plan exists
	planData, hasPlan := data["plan"]
	if !hasPlan {
		return impl.errorFactory.NewValidationError(impl.Name(), "input", "plan", 
			"plan data missing from input", "missing")
	}
	
	// Validate plan structure
	var plan ImplementationPlan
	switch v := planData.(type) {
	case ImplementationPlan:
		plan = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &plan); err != nil {
			return impl.errorFactory.NewValidationError(impl.Name(), "input", "plan", 
				fmt.Sprintf("failed to parse plan: %v", err), planData)
		}
	default:
		return impl.errorFactory.NewValidationError(impl.Name(), "input", "plan", 
			"plan must be ImplementationPlan type or map", fmt.Sprintf("%T", planData))
	}
	
	// Validate plan has implementation steps
	if len(plan.Steps) == 0 {
		return impl.errorFactory.NewValidationError(impl.Name(), "input", "steps", 
			"plan contains no implementation steps", plan.Steps)
	}
	
	// Validate each step has code files
	for idx, step := range plan.Steps {
		if len(step.CodeFiles) == 0 {
			return impl.errorFactory.NewValidationError(impl.Name(), "input", fmt.Sprintf("steps[%d].code_files", idx), 
				"step specifies no code files to implement", step.CodeFiles)
		}
	}
	
	return nil
}

// ValidateOutput validates output from the implementer phase
func (impl *Implementer) ValidateOutput(ctx context.Context, output core.PhaseOutput) error {
	impl.logger.Debug("Validating implementer output",
		"has_error", output.Error != nil,
		"data_type", fmt.Sprintf("%T", output.Data))
	
	if output.Error != nil {
		return output.Error
	}
	
	if output.Data == nil {
		return impl.errorFactory.NewValidationError(impl.Name(), "output", "data", 
			"implementer output cannot be nil", nil)
	}
	
	outputMap, ok := output.Data.(map[string]interface{})
	if !ok {
		return impl.errorFactory.NewValidationError(impl.Name(), "output", "data", 
			"output data must be a map containing generated code", fmt.Sprintf("%T", output.Data))
	}
	
	// Validate generated code exists
	generatedData, hasGenerated := outputMap["generated"]
	if !hasGenerated {
		return impl.errorFactory.NewValidationError(impl.Name(), "output", "generated", 
			"generated code missing from output", "missing")
	}
	
	// Validate generated code structure
	var generated GeneratedCode
	switch v := generatedData.(type) {
	case GeneratedCode:
		generated = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return impl.errorFactory.NewValidationError(impl.Name(), "output", "generated", 
				fmt.Sprintf("failed to parse generated code: %v", err), generatedData)
		}
	default:
		return impl.errorFactory.NewValidationError(impl.Name(), "output", "generated", 
			"generated data must be GeneratedCode type", fmt.Sprintf("%T", generatedData))
	}
	
	// Validate files were generated
	if len(generated.Files) == 0 {
		return impl.errorFactory.NewValidationError(impl.Name(), "output", "files", 
			"no code files generated", generated.Files)
	}
	
	// Validate each file
	for idx, file := range generated.Files {
		if strings.TrimSpace(file.Path) == "" {
			return impl.errorFactory.NewValidationError(impl.Name(), "output", fmt.Sprintf("files[%d].path", idx), 
				"file path cannot be empty", file.Path)
		}
		
		if strings.TrimSpace(file.Content) == "" {
			return impl.errorFactory.NewValidationError(impl.Name(), "output", fmt.Sprintf("files[%d].content", idx), 
				"file content cannot be empty", "empty")
		}
		
		if strings.TrimSpace(file.Language) == "" {
			return impl.errorFactory.NewValidationError(impl.Name(), "output", fmt.Sprintf("files[%d].language", idx), 
				"file language cannot be empty", file.Language)
		}
		
		// Validate file extension matches language
		if err := impl.validator.ValidateFileExtension(file.Path, file.Language, "output"); err != nil {
			return err
		}
	}
	
	// Validate summary exists
	if err := impl.validator.ValidateRequired("summary", generated.Summary, "output"); err != nil {
		return err
	}
	
	impl.logger.Info("Implementer output validation passed",
		"files_count", len(generated.Files),
		"summary_length", len(generated.Summary),
		"has_run_instructions", generated.RunInstructions != "")
	
	return nil
}

// validateFileExtension checks if file extension matches the language
func (impl *Implementer) validateFileExtension(path, language string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	lang := strings.ToLower(language)
	
	languageExtensions := map[string][]string{
		"go":         {".go"},
		"golang":     {".go"},
		"python":     {".py"},
		"javascript": {".js", ".mjs"},
		"typescript": {".ts"},
		"java":       {".java"},
		"c++":        {".cpp", ".cc", ".cxx"},
		"c#":         {".cs"},
		"rust":       {".rs"},
		"ruby":       {".rb"},
		"php":        {".php"},
		"swift":      {".swift"},
		"kotlin":     {".kt"},
		"html":       {".html", ".htm"},
		"css":        {".css"},
		"json":       {".json"},
		"yaml":       {".yaml", ".yml"},
		"xml":        {".xml"},
		"sql":        {".sql"},
		"shell":      {".sh"},
		"bash":       {".sh", ".bash"},
		"powershell": {".ps1"},
		"dockerfile": {".dockerfile"},
		"makefile":   {".mk"},
	}
	
	validExtensions, exists := languageExtensions[lang]
	if !exists {
		return true // Unknown language, skip validation
	}
	
	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}
	
	return false
}

// GetValidationRules returns validation rules for the implementer
func (impl *Implementer) GetValidationRules() core.ValidationRules {
	return core.ValidationRules{
		RequiredInputFields:  []string{"data", "plan"},
		RequiredOutputFields: []string{"generated", "files"},
		AllowedDataTypes:     []string{"map[string]interface{}", "GeneratedCode"},
		CustomValidators: []core.ValidationFunc{
			impl.validateCodeGeneration,
			impl.validateFileExtensions,
		},
	}
}

// validateCodeGeneration ensures actual code was generated
func (impl *Implementer) validateCodeGeneration(ctx context.Context, data interface{}) error {
	outputMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map output, got %T", data)
	}
	
	generatedData, exists := outputMap["generated"]
	if !exists {
		return fmt.Errorf("generated code not found in output")
	}
	
	var generated GeneratedCode
	switch v := generatedData.(type) {
	case GeneratedCode:
		generated = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return fmt.Errorf("failed to parse generated code: %w", err)
		}
	default:
		return fmt.Errorf("invalid generated code type: %T", generatedData)
	}
	
	if len(generated.Files) == 0 {
		return fmt.Errorf("no code files generated")
	}
	
	return nil
}

// validateFileExtensions ensures file extensions match languages
func (impl *Implementer) validateFileExtensions(ctx context.Context, data interface{}) error {
	outputMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected map output, got %T", data)
	}
	
	generatedData, exists := outputMap["generated"]
	if !exists {
		return fmt.Errorf("generated code not found in output")
	}
	
	var generated GeneratedCode
	switch v := generatedData.(type) {
	case GeneratedCode:
		generated = v
	case map[string]interface{}:
		jsonData, _ := json.Marshal(v)
		if err := json.Unmarshal(jsonData, &generated); err != nil {
			return fmt.Errorf("failed to parse generated code: %w", err)
		}
	default:
		return fmt.Errorf("invalid generated code type: %T", generatedData)
	}
	
	for _, file := range generated.Files {
		if !impl.validateFileExtension(file.Path, file.Language) {
			return fmt.Errorf("file %s extension does not match language %s", file.Path, file.Language)
		}
	}
	
	return nil
}