package phase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// ValidationLevel defines the strictness of validation
type ValidationLevel int

const (
	ValidationLevelBasic ValidationLevel = iota
	ValidationLevelStrict
	ValidationLevelComprehensive
)

// PhaseValidator provides comprehensive validation utilities for phases
type PhaseValidator struct {
	phaseName string
	level     ValidationLevel
	rules     ValidationRules
}

// ValidationRules and ValidationFunc types removed - 
// Use core.ValidationRules and core.ValidationFunc from consolidated validation system instead
type ValidationRules = core.ValidationRules
type ValidationFunc = core.ValidationFunc

// DefaultValidationRules provides sensible defaults for most phases
var DefaultValidationRules = ValidationRules{
	MinRequestLength: 1,
	MaxRequestLength: 50000,
	TimeoutDuration:  30 * time.Second,
	CustomValidators: []ValidationFunc{},
}

// NewPhaseValidator creates a new phase validator with configuration
func NewPhaseValidator(phaseName string, level ValidationLevel, rules ValidationRules) *PhaseValidator {
	return &PhaseValidator{
		phaseName: phaseName,
		level:     level,
		rules:     rules,
	}
}

// NewBasicValidator creates a validator with basic validation rules
func NewBasicValidator(phaseName string) *PhaseValidator {
	return NewPhaseValidator(phaseName, ValidationLevelBasic, DefaultValidationRules)
}

// NewStrictValidator creates a validator with strict validation rules
func NewStrictValidator(phaseName string) *PhaseValidator {
	rules := DefaultValidationRules
	rules.MinRequestLength = 10
	rules.MaxRequestLength = 40000
	return NewPhaseValidator(phaseName, ValidationLevelStrict, rules)
}

// ValidateInput performs comprehensive input validation
func (v *PhaseValidator) ValidateInput(input core.PhaseInput) error {
	if err := v.validateBasicInput(input); err != nil {
		return err
	}

	if v.level >= ValidationLevelStrict {
		if err := v.validateStrictInput(input); err != nil {
			return err
		}
	}

	if v.level >= ValidationLevelComprehensive {
		if err := v.validateComprehensiveInput(input); err != nil {
			return err
		}
	}

	return nil
}

// ValidateOutput performs comprehensive output validation
func (v *PhaseValidator) ValidateOutput(output core.PhaseOutput) error {
	if err := v.validateBasicOutput(output); err != nil {
		return err
	}

	if v.level >= ValidationLevelStrict {
		if err := v.validateStrictOutput(output); err != nil {
			return err
		}
	}

	if v.level >= ValidationLevelComprehensive {
		if err := v.validateComprehensiveOutput(output); err != nil {
			return err
		}
	}

	return nil
}

// validateBasicInput performs basic input validation
func (v *PhaseValidator) validateBasicInput(input core.PhaseInput) error {
	// Check required fields
	for _, field := range v.rules.RequiredInputFields {
		if err := v.validateRequiredField(input, field); err != nil {
			return err
		}
	}

	// Validate request length
	if len(input.Request) < v.rules.MinRequestLength {
		return core.NewValidationError(v.phaseName, "input", "Request", 
			fmt.Sprintf("request too short (minimum %d characters)", v.rules.MinRequestLength), 
			len(input.Request))
	}

	if len(input.Request) > v.rules.MaxRequestLength {
		return core.NewValidationError(v.phaseName, "input", "Request", 
			fmt.Sprintf("request too long (maximum %d characters)", v.rules.MaxRequestLength), 
			len(input.Request))
	}

	return nil
}

// validateStrictInput performs strict input validation
func (v *PhaseValidator) validateStrictInput(input core.PhaseInput) error {
	// Validate session ID format
	if input.SessionID != "" {
		if len(input.SessionID) < 8 || len(input.SessionID) > 64 {
			return core.NewValidationError(v.phaseName, "input", "SessionID", 
				"session ID must be between 8 and 64 characters", input.SessionID)
		}
	}

	// Validate prompt structure
	if input.Prompt != "" {
		if err := v.validatePromptStructure(input.Prompt); err != nil {
			return err
		}
	}

	// Validate data types
	if input.Data != nil {
		if err := v.validateDataType(input.Data); err != nil {
			return err
		}
	}

	return nil
}

// validateComprehensiveInput performs comprehensive input validation
func (v *PhaseValidator) validateComprehensiveInput(input core.PhaseInput) error {
	// Run custom validators
	for _, validator := range v.rules.CustomValidators {
		if err := validator(context.Background(), input); err != nil {
			return core.NewValidationError(v.phaseName, "input", "custom", 
				fmt.Sprintf("custom validation failed: %v", err), input)
		}
	}

	// Validate JSON structure if data is provided
	if input.Data != nil {
		if err := v.validateJSONStructure(input.Data); err != nil {
			return err
		}
	}

	return nil
}

// validateBasicOutput performs basic output validation
func (v *PhaseValidator) validateBasicOutput(output core.PhaseOutput) error {
	// Check if output has data or error
	if output.Data == nil && output.Error == nil {
		return core.NewValidationError(v.phaseName, "output", "Data", 
			"output must have either data or error", output)
	}

	// If there's an error, it should be properly formatted
	if output.Error != nil {
		if output.Error.Error() == "" {
			return core.NewValidationError(v.phaseName, "output", "Error", 
				"error message cannot be empty", output.Error)
		}
	}

	return nil
}

// validateStrictOutput performs strict output validation
func (v *PhaseValidator) validateStrictOutput(output core.PhaseOutput) error {
	// Check required output fields
	for _, field := range v.rules.RequiredOutputFields {
		if err := v.validateRequiredOutputField(output, field); err != nil {
			return err
		}
	}

	// Validate data type if present
	if output.Data != nil {
		if err := v.validateDataType(output.Data); err != nil {
			return err
		}
	}

	return nil
}

// validateComprehensiveOutput performs comprehensive output validation
func (v *PhaseValidator) validateComprehensiveOutput(output core.PhaseOutput) error {
	// Run custom validators
	for _, validator := range v.rules.CustomValidators {
		if err := validator(context.Background(), output); err != nil {
			return core.NewValidationError(v.phaseName, "output", "custom", 
				fmt.Sprintf("custom validation failed: %v", err), output)
		}
	}

	// Validate JSON structure if data is provided
	if output.Data != nil {
		if err := v.validateJSONStructure(output.Data); err != nil {
			return err
		}
	}

	return nil
}

// validateRequiredField validates that a required field is present in input
func (v *PhaseValidator) validateRequiredField(input core.PhaseInput, field string) error {
	switch field {
	case "Request":
		if strings.TrimSpace(input.Request) == "" {
			return core.NewValidationError(v.phaseName, "input", field, 
				"required field is empty", input.Request)
		}
	case "Prompt":
		if strings.TrimSpace(input.Prompt) == "" {
			return core.NewValidationError(v.phaseName, "input", field, 
				"required field is empty", input.Prompt)
		}
	case "Data":
		if input.Data == nil {
			return core.NewValidationError(v.phaseName, "input", field, 
				"required field is nil", input.Data)
		}
	case "SessionID":
		if strings.TrimSpace(input.SessionID) == "" {
			return core.NewValidationError(v.phaseName, "input", field, 
				"required field is empty", input.SessionID)
		}
	default:
		return core.NewValidationError(v.phaseName, "input", field, 
			"unknown required field", field)
	}
	return nil
}

// validateRequiredOutputField validates that a required field is present in output
func (v *PhaseValidator) validateRequiredOutputField(output core.PhaseOutput, field string) error {
	switch field {
	case "Data":
		if output.Data == nil {
			return core.NewValidationError(v.phaseName, "output", field, 
				"required field is nil", output.Data)
		}
	case "Error":
		if output.Error == nil {
			return core.NewValidationError(v.phaseName, "output", field, 
				"required field is nil", output.Error)
		}
	default:
		return core.NewValidationError(v.phaseName, "output", field, 
			"unknown required field", field)
	}
	return nil
}

// validatePromptStructure validates the structure of a prompt
func (v *PhaseValidator) validatePromptStructure(prompt string) error {
	// Check for common prompt injection patterns
	dangerousPatterns := []string{
		"ignore previous instructions",
		"forget everything",
		"system:",
		"<script>",
		"javascript:",
	}

	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return core.NewValidationError(v.phaseName, "input", "Prompt", 
				fmt.Sprintf("prompt contains potentially dangerous pattern: %s", pattern), prompt)
		}
	}

	// Check prompt length
	if len(prompt) > 100000 {
		return core.NewValidationError(v.phaseName, "input", "Prompt", 
			"prompt exceeds maximum length", len(prompt))
	}

	return nil
}

// validateDataType validates that data matches allowed types
func (v *PhaseValidator) validateDataType(data interface{}) error {
	// For now, accept any non-nil data as this is handled by the consolidated validation system
	if data == nil {
		return core.NewValidationError(v.phaseName, "data", "Type", "data cannot be nil", data)
	}
	return nil
}

// validateJSONStructure function removed - 
// Use core.ValidateJSON from consolidated validation system instead
func (v *PhaseValidator) validateJSONStructure(data interface{}) error {
	validator := core.NewBaseValidator(v.phaseName)
	return validator.ValidateJSON("data", data, "validation")
}

// ValidateContext validates the execution context
func (v *PhaseValidator) ValidateContext(ctx context.Context) error {
	if ctx == nil {
		return core.NewValidationError(v.phaseName, "context", "Context", 
			"context cannot be nil", ctx)
	}

	// Check if context is already done
	select {
	case <-ctx.Done():
		return core.NewValidationError(v.phaseName, "context", "Context", 
			fmt.Sprintf("context already cancelled: %v", ctx.Err()), ctx.Err())
	default:
		return nil
	}
}

// ValidateTimeout validates timeout settings
func (v *PhaseValidator) ValidateTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return core.NewValidationError(v.phaseName, "timeout", "Duration", 
			"timeout must be positive", timeout)
	}

	if timeout > v.rules.TimeoutDuration {
		return core.NewValidationError(v.phaseName, "timeout", "Duration", 
			fmt.Sprintf("timeout exceeds maximum allowed duration (%v)", v.rules.TimeoutDuration), timeout)
	}

	return nil
}

// AddCustomValidator adds a custom validation function
func (v *PhaseValidator) AddCustomValidator(validator ValidationFunc) {
	v.rules.CustomValidators = append(v.rules.CustomValidators, validator)
}

// SetValidationLevel changes the validation level
func (v *PhaseValidator) SetValidationLevel(level ValidationLevel) {
	v.level = level
}

// GetValidationMetrics returns metrics about validation performance
func (v *PhaseValidator) GetValidationMetrics() ValidationMetrics {
	return ValidationMetrics{
		PhaseName:             v.phaseName,
		Level:                 v.level,
		RequiredInputFields:   len(v.rules.RequiredInputFields),
		RequiredOutputFields:  len(v.rules.RequiredOutputFields),
		AllowedDataTypes:      len(v.rules.AllowedDataTypes),
		CustomValidators:      len(v.rules.CustomValidators),
		MinRequestLength:      v.rules.MinRequestLength,
		MaxRequestLength:      v.rules.MaxRequestLength,
		TimeoutDuration:       v.rules.TimeoutDuration,
	}
}

// ValidationMetrics contains metrics about validation configuration
type ValidationMetrics struct {
	PhaseName             string
	Level                 ValidationLevel
	RequiredInputFields   int
	RequiredOutputFields  int
	AllowedDataTypes      int
	CustomValidators      int
	MinRequestLength      int
	MaxRequestLength      int
	TimeoutDuration       time.Duration
}

// Common validation functions for specific phase types

// ValidateFictionInput validates input for fiction-related phases
func ValidateFictionInput(input core.PhaseInput) error {
	validator := NewBasicValidator("fiction")
	
	// Add fiction-specific validation
	if input.Data != nil {
		// Check for common fiction data structures
		if dataMap, ok := input.Data.(map[string]interface{}); ok {
			if _, hasTitle := dataMap["title"]; !hasTitle {
				return core.NewValidationError("fiction", "input", "title", 
					"fiction data must contain title", input.Data)
			}
		}
	}
	
	return validator.ValidateInput(input)
}

// ValidateCodeInput validates input for code-related phases
func ValidateCodeInput(input core.PhaseInput) error {
	validator := NewStrictValidator("code")
	
	// Add code-specific validation
	if input.Data != nil {
		// Check for common code data structures
		if dataMap, ok := input.Data.(map[string]interface{}); ok {
			if language, hasLanguage := dataMap["language"]; hasLanguage {
				if langStr, ok := language.(string); ok {
					validator := core.NewBaseValidator("code")
					if err := validator.ValidateLanguage(langStr, "input"); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return validator.ValidateInput(input)
}

// validateProgrammingLanguage function removed - 
// Use core.ValidateLanguage from consolidated validation system instead