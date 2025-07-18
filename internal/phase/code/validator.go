package code

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vampirenirmal/orchestrator/internal/core"
)

// Type definitions for validator compatibility
type Plan struct {
	ProjectName string `json:"project_name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Files       []File `json:"files"`
}

type Analysis struct {
	Summary           string `json:"summary"`
	ComplexityScore   int    `json:"complexity_score"`
	CodeQualityScore  int    `json:"code_quality_score"`
}

type Implementation struct {
	Files      []ImplementedFile `json:"files"`
	EntryPoint string            `json:"entry_point"`
}

type Review struct {
	OverallScore         int     `json:"overall_score"`
	Summary             string  `json:"summary"`
	CodeQualityScore    int     `json:"code_quality_score"`
	FunctionalityScore  int     `json:"functionality_score"`
	MaintainabilityScore int    `json:"maintainability_score"`
	PerformanceScore    int     `json:"performance_score"`
	SecurityScore       int     `json:"security_score"`
	Issues              []Issue `json:"issues"`
}

type File struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

type ImplementedFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

type Issue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	File        string `json:"file"`
}

// CodeValidator provides code-specific validation
type CodeValidator struct {
	*core.StandardPhaseValidator
}

// NewCodeValidator creates a validator with code-specific rules
func NewCodeValidator(phaseName string) *CodeValidator {
	rules := core.ValidationRules{
		MinRequestLength: core.DefaultMinLength,
		MaxRequestLength: core.DefaultMaxLength,
		CustomValidators: []core.ValidationFunc{
			validateCodeContent,
		},
	}

	return &CodeValidator{
		StandardPhaseValidator: core.NewStandardPhaseValidator(phaseName, rules),
	}
}

// ValidatePlan validates a CodePlan
func (v *CodeValidator) ValidatePlan(plan Plan) error {
	if err := core.ValidateNonEmpty(plan.ProjectName, "project_name"); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "project_name", err.Error(), plan.ProjectName)
	}

	if err := core.ValidateNonEmpty(plan.Description, "description"); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "description", err.Error(), plan.Description)
	}

	// Validate language
	if err := v.BaseValidator.ValidateLanguage(plan.Language, "output"); err != nil {
		return err
	}

	if len(plan.Files) == 0 {
		return core.NewValidationError(v.PhaseName, "output", "files", "at least one file must be planned", plan.Files)
	}

	// Validate each file
	for i, file := range plan.Files {
		if err := v.validateFile(file, i, plan.Language); err != nil {
			return err
		}
	}

	return nil
}

// ValidateAnalysis validates code analysis results
func (v *CodeValidator) ValidateAnalysis(analysis Analysis) error {
	if err := core.ValidateNonEmpty(analysis.Summary, "analysis.summary"); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "analysis.summary", err.Error(), analysis.Summary)
	}

	// Validate complexity score
	if analysis.ComplexityScore < 1 || analysis.ComplexityScore > 10 {
		return core.NewValidationError(v.PhaseName, "output", "complexity_score",
			"complexity score must be between 1 and 10", analysis.ComplexityScore)
	}

	// Validate code quality score
	if analysis.CodeQualityScore < 1 || analysis.CodeQualityScore > 10 {
		return core.NewValidationError(v.PhaseName, "output", "code_quality_score",
			"code quality score must be between 1 and 10", analysis.CodeQualityScore)
	}

	return nil
}

// ValidateImplementation validates code implementation
func (v *CodeValidator) ValidateImplementation(impl Implementation) error {
	if len(impl.Files) == 0 {
		return core.NewValidationError(v.PhaseName, "output", "files", "at least one file must be implemented", impl.Files)
	}

	// Validate each implemented file
	for i, file := range impl.Files {
		if err := v.validateImplementedFile(file, i); err != nil {
			return err
		}
	}

	// Validate entry point if specified
	if impl.EntryPoint != "" {
		found := false
		for _, file := range impl.Files {
			if file.Path == impl.EntryPoint {
				found = true
				break
			}
		}
		if !found {
			return core.NewValidationError(v.PhaseName, "output", "entry_point",
				fmt.Sprintf("entry point '%s' not found in implemented files", impl.EntryPoint), impl.EntryPoint)
		}
	}

	return nil
}

// ValidateReview validates code review results
func (v *CodeValidator) ValidateReview(review Review) error {
	// Validate overall score
	if review.OverallScore < 1 || review.OverallScore > 10 {
		return core.NewValidationError(v.PhaseName, "output", "overall_score",
			"overall score must be between 1 and 10", review.OverallScore)
	}

	if err := core.ValidateNonEmpty(review.Summary, "review.summary"); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "review.summary", err.Error(), review.Summary)
	}

	// Validate category scores
	scores := []struct {
		name  string
		score int
	}{
		{"code_quality", review.CodeQualityScore},
		{"functionality", review.FunctionalityScore},
		{"maintainability", review.MaintainabilityScore},
		{"performance", review.PerformanceScore},
		{"security", review.SecurityScore},
	}

	for _, s := range scores {
		if s.score < 1 || s.score > 10 {
			return core.NewValidationError(v.PhaseName, "output", s.name+"_score",
				fmt.Sprintf("%s score must be between 1 and 10", s.name), s.score)
		}
	}

	// Validate issues if any
	for i, issue := range review.Issues {
		if err := v.validateIssue(issue, i); err != nil {
			return err
		}
	}

	return nil
}

// validateFile validates a planned file
func (v *CodeValidator) validateFile(file File, index int, language string) error {
	if err := core.ValidateNonEmpty(file.Path, fmt.Sprintf("file[%d].path", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "file.path", err.Error(), file.Path)
	}

	if err := core.ValidateNonEmpty(file.Description, fmt.Sprintf("file[%d].description", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "file.description", err.Error(), file.Description)
	}

	// Validate file extension matches language
	if err := v.BaseValidator.ValidateFileExtension(file.Path, language, "output"); err != nil {
		return err
	}

	// Validate no directory traversal
	if strings.Contains(file.Path, "..") {
		return core.NewValidationError(v.PhaseName, "output", "file.path",
			"file path cannot contain directory traversal", file.Path)
	}

	return nil
}

// validateImplementedFile validates an implemented file
func (v *CodeValidator) validateImplementedFile(file ImplementedFile, index int) error {
	if err := core.ValidateNonEmpty(file.Path, fmt.Sprintf("file[%d].path", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "file.path", err.Error(), file.Path)
	}

	if err := core.ValidateNonEmpty(file.Content, fmt.Sprintf("file[%d].content", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "file.content", err.Error(), file.Path)
	}

	// Validate language if specified
	if file.Language != "" {
		if err := v.BaseValidator.ValidateLanguage(file.Language, "output"); err != nil {
			return err
		}

		// Validate file extension matches language
		if err := v.BaseValidator.ValidateFileExtension(file.Path, file.Language, "output"); err != nil {
			return err
		}
	}

	// Validate content length
	if len(file.Content) < 10 {
		return core.NewValidationError(v.PhaseName, "output", "file.content",
			fmt.Sprintf("file content too short for '%s'", file.Path), len(file.Content))
	}

	// Validate no directory traversal
	if strings.Contains(file.Path, "..") {
		return core.NewValidationError(v.PhaseName, "output", "file.path",
			"file path cannot contain directory traversal", file.Path)
	}

	return nil
}

// validateIssue validates a code review issue
func (v *CodeValidator) validateIssue(issue Issue, index int) error {
	if err := core.ValidateNonEmpty(issue.Type, fmt.Sprintf("issue[%d].type", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "issue.type", err.Error(), issue.Type)
	}

	// Validate issue severity
	validSeverities := []string{"critical", "high", "medium", "low", "info"}
	found := false
	for _, sev := range validSeverities {
		if strings.EqualFold(issue.Severity, sev) {
			found = true
			break
		}
	}
	if !found {
		return core.NewValidationError(v.PhaseName, "output", "issue.severity",
			fmt.Sprintf("invalid severity '%s' (must be one of: %v)", issue.Severity, validSeverities), issue.Severity)
	}

	if err := core.ValidateNonEmpty(issue.Description, fmt.Sprintf("issue[%d].description", index)); err != nil {
		return core.NewValidationError(v.PhaseName, "output", "issue.description", err.Error(), issue.Description)
	}

	// Validate file path if specified
	if issue.File != "" {
		if strings.Contains(issue.File, "..") {
			return core.NewValidationError(v.PhaseName, "output", "issue.file",
				"issue file path cannot contain directory traversal", issue.File)
		}
	}

	return nil
}

// validateCodeContent is a custom validator for code content
func validateCodeContent(ctx context.Context, data interface{}) error {
	// This can be extended with code-specific content validation
	// For example: checking for security vulnerabilities, code patterns, etc.
	return nil
}

// Helper function to validate project structure
func ValidateProjectStructure(files []ImplementedFile) error {
	// Check for common required files based on detected project type
	hasMainFile := false
	hasPackageFile := false
	projectType := detectProjectType(files)

	for _, file := range files {
		basename := filepath.Base(file.Path)
		
		// Check for main/entry files
		if strings.HasPrefix(basename, "main.") || strings.HasPrefix(basename, "index.") || strings.HasPrefix(basename, "app.") {
			hasMainFile = true
		}

		// Check for package files
		switch basename {
		case "package.json", "go.mod", "requirements.txt", "Cargo.toml", "pom.xml", "build.gradle":
			hasPackageFile = true
		}
	}

	// Validate based on project type
	if projectType != "script" && projectType != "unknown" {
		if !hasMainFile {
			return fmt.Errorf("missing main/entry point file for %s project", projectType)
		}
		if !hasPackageFile && projectType != "simple" {
			return fmt.Errorf("missing package/dependency file for %s project", projectType)
		}
	}

	return nil
}

// detectProjectType attempts to detect the project type from files
func detectProjectType(files []ImplementedFile) string {
	for _, file := range files {
		basename := filepath.Base(file.Path)
		switch basename {
		case "package.json":
			return "node"
		case "go.mod":
			return "go"
		case "requirements.txt", "setup.py":
			return "python"
		case "Cargo.toml":
			return "rust"
		case "pom.xml", "build.gradle":
			return "java"
		}
	}

	// Check if it's a simple script
	if len(files) == 1 {
		return "script"
	}

	return "unknown"
}