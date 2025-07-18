package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ValidationError is now defined in errors.go

// Common validation constants
var (
	// ValidProgrammingLanguages is the canonical list of supported languages
	ValidProgrammingLanguages = []string{
		"PHP", "Python", "JavaScript", "Go", "Java", "C++",
		"TypeScript", "Ruby", "Rust", "C#", "Swift", "Kotlin",
		"JSON", "YAML", "XML",
	}

	// LanguageFileExtensions maps languages to their file extensions
	LanguageFileExtensions = map[string][]string{
		"PHP":        {".php"},
		"Python":     {".py"},
		"JavaScript": {".js", ".mjs"},
		"Go":         {".go"},
		"Java":       {".java"},
		"C++":        {".cpp", ".cc", ".cxx", ".h", ".hpp"},
		"TypeScript": {".ts", ".tsx"},
		"Ruby":       {".rb"},
		"Rust":       {".rs"},
		"C#":         {".cs"},
		"Swift":      {".swift"},
		"Kotlin":     {".kt", ".kts"},
		"JSON":       {".json"},
		"YAML":       {".yaml", ".yml"},
		"XML":        {".xml"},
	}

	// Common validation limits
	DefaultMinLength = 10
	DefaultMaxLength = 10000
	MaxSceneCount    = 50
	MaxChapterCount  = 30
)

// BaseValidator provides common validation utilities
type BaseValidator struct {
	PhaseName string
}

func NewBaseValidator(phaseName string) *BaseValidator {
	return &BaseValidator{PhaseName: phaseName}
}

func (v *BaseValidator) ValidateRequired(field string, value string, context string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(v.PhaseName, context, field, "required field is empty", value)
	}
	return nil
}

func (v *BaseValidator) ValidateJSON(field string, data interface{}, context string) error {
	if data == nil {
		return NewValidationError(v.PhaseName, context, field, "data is nil", data)
	}

	// Try to marshal/unmarshal to validate JSON structure
	jsonData, err := json.Marshal(data)
	if err != nil {
		return NewValidationError(v.PhaseName, context, field, fmt.Sprintf("failed to marshal to JSON: %v", err), data)
	}

	var temp interface{}
	if err := json.Unmarshal(jsonData, &temp); err != nil {
		return NewValidationError(v.PhaseName, context, field, fmt.Sprintf("failed to unmarshal JSON: %v", err), string(jsonData))
	}

	return nil
}

func (v *BaseValidator) ValidateLanguage(language string, context string) error {
	if language == "Other" || language == "" {
		return NewValidationError(v.PhaseName, context, "language", "language detection failed - got 'Other' or empty", language)
	}

	for _, valid := range ValidProgrammingLanguages {
		if strings.EqualFold(language, valid) {
			return nil
		}
	}

	return NewValidationError(v.PhaseName, context, "language", fmt.Sprintf("unsupported language: %s", language), language)
}

func (v *BaseValidator) ValidateFileExtension(filename string, expectedLanguage string, context string) error {
	extensions, exists := LanguageFileExtensions[expectedLanguage]
	if !exists {
		return nil // Skip validation for unknown languages
	}

	for _, ext := range extensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return nil
		}
	}

	return NewValidationError(v.PhaseName, context, "filename",
		fmt.Sprintf("filename '%s' doesn't match language '%s' (expected: %v)", filename, expectedLanguage, extensions),
		map[string]interface{}{"filename": filename, "language": expectedLanguage, "expected_extensions": extensions})
}

// ValidationLogger tracks validation events for debugging
type ValidationLogger struct {
	Events []ValidationEvent
}

type ValidationEvent struct {
	Phase     string      `json:"phase"`
	Type      string      `json:"type"` // "input", "output", "error"
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

func NewValidationLogger() *ValidationLogger {
	return &ValidationLogger{Events: make([]ValidationEvent, 0)}
}

func (l *ValidationLogger) LogValidation(phase, validationType string, success bool, err error, data interface{}) {
	event := ValidationEvent{
		Phase:     phase,
		Type:      validationType,
		Success:   success,
		Data:      data,
		Timestamp: getCurrentTimestamp(),
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	l.Events = append(l.Events, event)
}

func (l *ValidationLogger) GetValidationReport() string {
	report := "=== VALIDATION REPORT ===\n"
	for _, event := range l.Events {
		status := "✓ PASS"
		if !event.Success {
			status = "✗ FAIL"
		}
		
		report += fmt.Sprintf("%s [%s:%s] %s", status, event.Phase, event.Type, "")
		if event.Error != "" {
			report += fmt.Sprintf(" - %s", event.Error)
		}
		report += "\n"
	}
	return report
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// ValidationFunc defines a custom validation function
type ValidationFunc func(ctx context.Context, data interface{}) error

// ValidationRules defines validation rules for phases
type ValidationRules struct {
	RequiredInputFields  []string
	RequiredOutputFields []string
	AllowedDataTypes     []string
	MinRequestLength     int
	MaxRequestLength     int
	TimeoutDuration      time.Duration
	CustomValidators     []ValidationFunc
}


// StandardPhaseValidator implements common phase validation patterns
type StandardPhaseValidator struct {
	*BaseValidator
	Rules ValidationRules
}

func NewStandardPhaseValidator(phaseName string, rules ValidationRules) *StandardPhaseValidator {
	return &StandardPhaseValidator{
		BaseValidator: NewBaseValidator(phaseName),
		Rules:         rules,
	}
}

func (v *StandardPhaseValidator) ValidateInput(ctx context.Context, input PhaseInput) error {
	// Check required fields
	if input.Request == "" {
		return NewValidationError(v.PhaseName, "input", "request", "request is empty", input)
	}

	// Check request length
	if v.Rules.MinRequestLength > 0 && len(input.Request) < v.Rules.MinRequestLength {
		return NewValidationError(v.PhaseName, "input", "request",
			fmt.Sprintf("request too short (min: %d)", v.Rules.MinRequestLength), input)
	}

	if v.Rules.MaxRequestLength > 0 && len(input.Request) > v.Rules.MaxRequestLength {
		return NewValidationError(v.PhaseName, "input", "request",
			fmt.Sprintf("request too long (max: %d)", v.Rules.MaxRequestLength), input)
	}

	// Run custom validators
	for _, validator := range v.Rules.CustomValidators {
		if err := validator(ctx, input); err != nil {
			return err
		}
	}

	return nil
}

func (v *StandardPhaseValidator) ValidateOutput(ctx context.Context, output PhaseOutput) error {
	// Check output data exists
	if output.Data == nil {
		return NewValidationError(v.PhaseName, "output", "data", "output data is nil", output)
	}

	// Validate JSON structure
	if err := v.ValidateJSON("data", output.Data, "output"); err != nil {
		return err
	}

	// Run custom validators
	for _, validator := range v.Rules.CustomValidators {
		if err := validator(ctx, output); err != nil {
			return err
		}
	}

	return nil
}

// Common validation helpers
func ValidateStringLength(value string, min, max int, fieldName string) error {
	if len(value) < min {
		return fmt.Errorf("%s is too short (min: %d, got: %d)", fieldName, min, len(value))
	}
	if max > 0 && len(value) > max {
		return fmt.Errorf("%s is too long (max: %d, got: %d)", fieldName, max, len(value))
	}
	return nil
}

func ValidateNonEmpty(value string, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// GetFileExtension returns the file extension for a given language
func GetFileExtension(language string) string {
	extensions, exists := LanguageFileExtensions[language]
	if !exists || len(extensions) == 0 {
		return ".txt" // default fallback
	}
	return extensions[0] // return primary extension
}