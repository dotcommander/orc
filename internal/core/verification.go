package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StageVerifier ensures each stage completes sufficiently
type StageVerifier struct {
	verifiers     map[string]VerificationFunc
	retryLimit    int
	issueTracker  *IssueTracker
	logger        *slog.Logger
	strictMode    bool
}

// VerificationFunc checks if a stage output is sufficient
type VerificationFunc func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue)

// VerificationIssue describes why verification failed
type VerificationIssue struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"` // critical, major, minor
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// StageResult contains verification results
type StageResult struct {
	Stage         string                 `json:"stage"`
	Success       bool                   `json:"success"`
	Attempts      int                    `json:"attempts"`
	Issues        []VerificationIssue    `json:"issues"`
	Output        interface{}            `json:"output,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// IssueTracker documents failures for later analysis
type IssueTracker struct {
	issuesDir string
	sessionID string
	logger    *slog.Logger
}

// NewStageVerifier creates a verifier with issue tracking
func NewStageVerifier(sessionID string, outputDir string, logger *slog.Logger) *StageVerifier {
	issuesDir := filepath.Join(outputDir, "issues")
	
	return &StageVerifier{
		verifiers:    make(map[string]VerificationFunc),
		retryLimit:   3,
		issueTracker: NewIssueTracker(issuesDir, sessionID, logger),
		logger:       logger,
		strictMode:   true,
	}
}

// NewIssueTracker creates an issue documentation system
func NewIssueTracker(issuesDir, sessionID string, logger *slog.Logger) *IssueTracker {
	// Ensure issues directory exists
	if err := os.MkdirAll(issuesDir, 0755); err != nil {
		logger.Error("failed to create issues directory", "error", err)
	}
	
	return &IssueTracker{
		issuesDir: issuesDir,
		sessionID: sessionID,
		logger:    logger,
	}
}

// RegisterVerifier adds a stage-specific verifier
func (sv *StageVerifier) RegisterVerifier(stage string, verifier VerificationFunc) {
	sv.verifiers[stage] = verifier
}

// RegisterDefaultVerifiers sets up common verification patterns
func (sv *StageVerifier) RegisterDefaultVerifiers() {
	// Planning stage verifier
	sv.RegisterVerifier("Planning", func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
		issues := []VerificationIssue{}
		
		// Check output exists
		if output == nil {
			issues = append(issues, VerificationIssue{
				Type:        "missing_output",
				Severity:    "critical",
				Description: "Planning stage produced no output",
			})
			return false, issues
		}
		
		// Check for required planning elements
		outputStr := fmt.Sprintf("%v", output)
		requiredElements := []string{"outline", "characters", "plot", "theme"}
		missingElements := []string{}
		
		for _, element := range requiredElements {
			if !strings.Contains(strings.ToLower(outputStr), element) {
				missingElements = append(missingElements, element)
			}
		}
		
		if len(missingElements) > 0 {
			issues = append(issues, VerificationIssue{
				Type:        "incomplete_planning",
				Severity:    "major",
				Description: "Planning missing required elements",
				Details: map[string]interface{}{
					"missing_elements": missingElements,
				},
			})
		}
		
		// Check minimum length
		if len(outputStr) < 1000 {
			issues = append(issues, VerificationIssue{
				Type:        "insufficient_detail",
				Severity:    "major",
				Description: "Planning output too brief",
				Details: map[string]interface{}{
					"length":          len(outputStr),
					"minimum_expected": 1000,
				},
			})
		}
		
		return len(issues) == 0, issues
	})
	
	// Architecture stage verifier
	sv.RegisterVerifier("Architecture", func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
		issues := []VerificationIssue{}
		
		if output == nil {
			issues = append(issues, VerificationIssue{
				Type:        "missing_output",
				Severity:    "critical",
				Description: "Architecture stage produced no output",
			})
			return false, issues
		}
		
		// Check for structure elements
		outputStr := fmt.Sprintf("%v", output)
		if !strings.Contains(strings.ToLower(outputStr), "chapter") {
			issues = append(issues, VerificationIssue{
				Type:        "missing_structure",
				Severity:    "critical",
				Description: "Architecture missing chapter structure",
			})
		}
		
		return len(issues) == 0, issues
	})
	
	// Writing stage verifier
	sv.RegisterVerifier("Writing", func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
		issues := []VerificationIssue{}
		
		if output == nil {
			issues = append(issues, VerificationIssue{
				Type:        "missing_output",
				Severity:    "critical",
				Description: "Writing stage produced no output",
			})
			return false, issues
		}
		
		// Check word count
		outputStr := fmt.Sprintf("%v", output)
		wordCount := len(strings.Fields(outputStr))
		
		if wordCount < 100 {
			issues = append(issues, VerificationIssue{
				Type:        "insufficient_content",
				Severity:    "critical",
				Description: "Writing output too short",
				Details: map[string]interface{}{
					"word_count":      wordCount,
					"minimum_expected": 100,
				},
			})
		}
		
		return len(issues) == 0, issues
	})
	
	// Code stage verifier
	sv.RegisterVerifier("Implementation", func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
		issues := []VerificationIssue{}
		
		if output == nil {
			issues = append(issues, VerificationIssue{
				Type:        "missing_output",
				Severity:    "critical",
				Description: "Implementation stage produced no output",
			})
			return false, issues
		}
		
		// Check for code patterns
		outputStr := fmt.Sprintf("%v", output)
		codePatterns := []string{"func", "package", "import", "class", "def", "function", "const", "var"}
		hasCode := false
		
		for _, pattern := range codePatterns {
			if strings.Contains(outputStr, pattern) {
				hasCode = true
				break
			}
		}
		
		if !hasCode {
			issues = append(issues, VerificationIssue{
				Type:        "no_code_detected",
				Severity:    "critical",
				Description: "Implementation output doesn't appear to contain code",
			})
		}
		
		return len(issues) == 0, issues
	})
}

// VerifyStageWithRetry verifies a stage output with retry logic
func (sv *StageVerifier) VerifyStageWithRetry(
	ctx context.Context,
	stage string,
	executeFunc func() (interface{}, error),
) (*StageResult, error) {
	
	result := &StageResult{
		Stage:     stage,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Get verifier for this stage
	verifier, hasVerifier := sv.verifiers[stage]
	if !hasVerifier && sv.strictMode {
		sv.logger.Warn("no verifier registered for stage", "stage", stage)
		// Create a basic verifier
		verifier = func(ctx context.Context, stage string, output interface{}) (bool, []VerificationIssue) {
			if output == nil {
				return false, []VerificationIssue{{
					Type:        "missing_output",
					Severity:    "critical",
					Description: "Stage produced no output",
				}}
			}
			return true, nil
		}
	}
	
	startTime := time.Now()
	
	// Try up to retryLimit times
	for attempt := 1; attempt <= sv.retryLimit; attempt++ {
		sv.logger.Info("executing stage", "stage", stage, "attempt", attempt)
		
		// Execute the stage
		output, err := executeFunc()
		if err != nil {
			// Execution error
			result.Issues = append(result.Issues, VerificationIssue{
				Type:        "execution_error",
				Severity:    "critical",
				Description: fmt.Sprintf("Stage execution failed: %v", err),
				Details: map[string]interface{}{
					"attempt": attempt,
					"error":   err.Error(),
				},
			})
			
			if attempt < sv.retryLimit {
				sv.logger.Warn("stage execution failed, retrying",
					"stage", stage,
					"attempt", attempt,
					"error", err)
				time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
				continue
			}
		} else {
			// Verify the output
			if hasVerifier {
				passed, issues := verifier(ctx, stage, output)
				result.Issues = append(result.Issues, issues...)
				
				if passed {
					// Success!
					result.Success = true
					result.Output = output
					result.Attempts = attempt
					result.Duration = time.Since(startTime)
					
					sv.logger.Info("stage verified successfully",
						"stage", stage,
						"attempts", attempt,
						"duration", result.Duration)
					
					return result, nil
				}
				
				// Verification failed
				if attempt < sv.retryLimit {
					sv.logger.Warn("stage verification failed, retrying",
						"stage", stage,
						"attempt", attempt,
						"issues", len(issues))
					time.Sleep(time.Duration(attempt) * 3 * time.Second)
					continue
				}
			} else {
				// No verifier, assume success
				result.Success = true
				result.Output = output
				result.Attempts = attempt
				result.Duration = time.Since(startTime)
				return result, nil
			}
		}
	}
	
	// All attempts failed
	result.Success = false
	result.Attempts = sv.retryLimit
	result.Duration = time.Since(startTime)
	
	// Document the failure
	sv.issueTracker.DocumentFailure(result)
	
	return result, fmt.Errorf("stage %s failed after %d attempts", stage, sv.retryLimit)
}

// DocumentFailure saves failure details for later analysis
func (it *IssueTracker) DocumentFailure(result *StageResult) error {
	// Create filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s-%s.json", it.sessionID[:8], result.Stage, timestamp)
	filepath := filepath.Join(it.issuesDir, filename)
	
	// Create issue report
	report := map[string]interface{}{
		"session_id": it.sessionID,
		"stage":      result.Stage,
		"timestamp":  result.Timestamp,
		"attempts":   result.Attempts,
		"duration":   result.Duration.String(),
		"issues":     result.Issues,
		"metadata":   result.Metadata,
		"output":     result.Output, // Include output for debugging
	}
	
	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		it.logger.Error("failed to marshal issue report", "error", err)
		return err
	}
	
	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		it.logger.Error("failed to write issue report", "error", err)
		return err
	}
	
	it.logger.Info("documented stage failure", "stage", result.Stage, "file", filepath)
	
	// Also create a summary file
	it.updateIssueSummary(result)
	
	return nil
}

// updateIssueSummary maintains a summary of all issues
func (it *IssueTracker) updateIssueSummary(result *StageResult) {
	summaryPath := filepath.Join(it.issuesDir, fmt.Sprintf("%s-summary.md", it.sessionID[:8]))
	
	// Create or append to summary
	f, err := os.OpenFile(summaryPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		it.logger.Error("failed to open summary file", "error", err)
		return
	}
	defer f.Close()
	
	// Write summary entry
	fmt.Fprintf(f, "\n## %s - %s\n", result.Stage, result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "- Attempts: %d\n", result.Attempts)
	fmt.Fprintf(f, "- Duration: %s\n", result.Duration)
	fmt.Fprintf(f, "- Issues:\n")
	
	for _, issue := range result.Issues {
		fmt.Fprintf(f, "  - **%s** (%s): %s\n", issue.Type, issue.Severity, issue.Description)
		if issue.Details != nil {
			for k, v := range issue.Details {
				fmt.Fprintf(f, "    - %s: %v\n", k, v)
			}
		}
	}
	
	fmt.Fprintf(f, "\n---\n")
}

// LoadIssues loads all issues for a session
func (it *IssueTracker) LoadIssues() ([]StageResult, error) {
	pattern := filepath.Join(it.issuesDir, fmt.Sprintf("%s-*.json", it.sessionID[:8]))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	results := []StageResult{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		var report map[string]interface{}
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}
		
		// Convert back to StageResult
		result := StageResult{
			Stage:    report["stage"].(string),
			Success:  false,
			Attempts: int(report["attempts"].(float64)),
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// AnalyzeIssuePatterns looks for patterns in failures
func (it *IssueTracker) AnalyzeIssuePatterns() map[string]interface{} {
	issues, _ := it.LoadIssues()
	
	patterns := map[string]interface{}{
		"total_failures":     len(issues),
		"failures_by_stage":  make(map[string]int),
		"common_issue_types": make(map[string]int),
	}
	
	for _, issue := range issues {
		// Count by stage
		stageFailures := patterns["failures_by_stage"].(map[string]int)
		stageFailures[issue.Stage]++
		
		// Count issue types
		// Would need to parse issues from the files for detailed analysis
	}
	
	return patterns
}