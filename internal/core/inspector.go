package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// InspectorAgent performs deep analysis and quality assessment
type InspectorAgent struct {
	agent      Agent
	logger     *slog.Logger
	inspectors map[string]Inspector
	cache      *InspectionCache
	config     InspectorConfig
}

// Inspector represents a specialized quality inspector
type Inspector interface {
	Name() string
	Category() string
	Inspect(ctx context.Context, content interface{}) (InspectionResult, error)
	GenerateCriteria() []QualityCriteria
	CanInspect(content interface{}) bool
}

// InspectionResult contains detailed findings from an inspection
type InspectionResult struct {
	InspectorName string                 `json:"inspector_name"`
	Category      string                 `json:"category"`
	Score         float64                `json:"score"`
	Passed        bool                   `json:"passed"`
	Findings      []Finding              `json:"findings"`
	Metrics       map[string]float64     `json:"metrics"`
	Suggestions   []ImprovementSuggestion `json:"suggestions"`
	Evidence      []Evidence             `json:"evidence"`
	Timestamp     time.Time              `json:"timestamp"`
	Context       map[string]interface{} `json:"context"`
}

// Finding represents a specific issue or observation
type Finding struct {
	ID          string         `json:"id"`
	Type        FindingType    `json:"type"`
	Severity    Severity       `json:"severity"`
	Location    Location       `json:"location"`
	Description string         `json:"description"`
	Impact      string         `json:"impact"`
	Pattern     string         `json:"pattern,omitempty"`
	Occurrences int            `json:"occurrences"`
	Context     []string       `json:"context,omitempty"`
}

type FindingType string

const (
	ErrorFinding      FindingType = "error"
	WarningFinding    FindingType = "warning"
	SuggestionFinding FindingType = "suggestion"
	InfoFinding       FindingType = "info"
)

type Severity string

const (
	CriticalSeverity Severity = "critical"
	HighSeverity     Severity = "high"
	MediumSeverity   Severity = "medium"
	LowSeverity      Severity = "low"
)

// Location pinpoints where an issue exists
type Location struct {
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	StartIndex int    `json:"start_index,omitempty"`
	EndIndex   int    `json:"end_index,omitempty"`
	Context    string `json:"context,omitempty"`
}

// Evidence provides proof of findings
type Evidence struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Data        string `json:"data"`
	Source      string `json:"source"`
}

// InspectorConfig configures inspector behavior
type InspectorConfig struct {
	DeepAnalysis       bool              `json:"deep_analysis"`
	ParallelInspection bool              `json:"parallel_inspection"`
	CacheResults       bool              `json:"cache_results"`
	MaxDepth           int               `json:"max_depth"`
	TimeoutPerCheck    time.Duration     `json:"timeout_per_check"`
	CustomInspectors   []string          `json:"custom_inspectors"`
	Thresholds         map[string]float64 `json:"thresholds"`
}

// Built-in Inspectors

// CodeQualityInspector checks code quality metrics
type CodeQualityInspector struct {
	logger *slog.Logger
	agent  Agent
}

func NewCodeQualityInspector(agent Agent, logger *slog.Logger) *CodeQualityInspector {
	return &CodeQualityInspector{
		agent:  agent,
		logger: logger.With("inspector", "code_quality"),
	}
}

func (cqi *CodeQualityInspector) Name() string { return "CodeQuality" }
func (cqi *CodeQualityInspector) Category() string { return "quality" }

func (cqi *CodeQualityInspector) Inspect(ctx context.Context, content interface{}) (InspectionResult, error) {
	result := InspectionResult{
		InspectorName: cqi.Name(),
		Category:      cqi.Category(),
		Findings:      make([]Finding, 0),
		Metrics:       make(map[string]float64),
		Suggestions:   make([]ImprovementSuggestion, 0),
		Evidence:      make([]Evidence, 0),
		Timestamp:     time.Now(),
	}

	// Convert content to string for analysis
	code, ok := content.(string)
	if !ok {
		return result, fmt.Errorf("content must be string for code inspection")
	}

	// Perform various quality checks
	cqi.checkComplexity(code, &result)
	cqi.checkDuplication(code, &result)
	cqi.checkNaming(code, &result)
	cqi.checkStructure(code, &result)
	cqi.checkDocumentation(code, &result)
	cqi.checkErrorHandling(code, &result)
	cqi.checkSecurity(code, &result)
	cqi.checkPerformance(code, &result)

	// Calculate overall score
	result.Score = cqi.calculateScore(result.Metrics)
	result.Passed = result.Score >= 0.7

	return result, nil
}

func (cqi *CodeQualityInspector) checkComplexity(code string, result *InspectionResult) {
	// Analyze cyclomatic complexity
	lines := strings.Split(code, "\n")
	
	// Simple heuristics for complexity
	ifCount := 0
	forCount := 0
	functionCount := 0
	deepNesting := 0
	maxNesting := 0
	currentNesting := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Count control structures
		if strings.Contains(trimmed, "if ") || strings.Contains(trimmed, "if(") {
			ifCount++
		}
		if strings.Contains(trimmed, "for ") || strings.Contains(trimmed, "while ") {
			forCount++
		}
		if strings.Contains(trimmed, "function ") || strings.Contains(trimmed, "func ") {
			functionCount++
		}

		// Track nesting
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		currentNesting += openBraces - closeBraces
		
		if currentNesting > maxNesting {
			maxNesting = currentNesting
		}
		
		if currentNesting > 3 {
			deepNesting++
			result.Findings = append(result.Findings, Finding{
				ID:          fmt.Sprintf("deep-nesting-%d", i),
				Type:        WarningFinding,
				Severity:    MediumSeverity,
				Location:    Location{Line: i + 1},
				Description: fmt.Sprintf("Deep nesting level %d detected", currentNesting),
				Impact:      "Reduces code readability and maintainability",
			})
		}
	}

	// Calculate complexity metrics
	complexity := float64(ifCount + forCount) / float64(len(lines)+1)
	result.Metrics["cyclomatic_complexity"] = complexity
	result.Metrics["max_nesting_depth"] = float64(maxNesting)
	result.Metrics["functions_per_file"] = float64(functionCount)

	if complexity > 0.15 {
		result.Suggestions = append(result.Suggestions, ImprovementSuggestion{
			Target:     "Code Structure",
			Action:     "Reduce complexity by extracting methods",
			Reason:     fmt.Sprintf("Complexity score %.2f exceeds threshold", complexity),
			Complexity: "medium",
		})
	}
}

func (cqi *CodeQualityInspector) checkDuplication(code string, result *InspectionResult) {
	lines := strings.Split(code, "\n")
	lineMap := make(map[string][]int)
	
	// Find duplicate lines
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 { // Only consider meaningful lines
			lineMap[trimmed] = append(lineMap[trimmed], i+1)
		}
	}

	duplicates := 0
	for line, occurrences := range lineMap {
		if len(occurrences) > 1 {
			duplicates++
			if duplicates < 5 { // Limit findings
				result.Findings = append(result.Findings, Finding{
					ID:          fmt.Sprintf("duplication-%d", duplicates),
					Type:        WarningFinding,
					Severity:    LowSeverity,
					Description: fmt.Sprintf("Duplicate line found: '%s'", line),
					Occurrences: len(occurrences),
					Impact:      "Code duplication reduces maintainability",
				})
			}
		}
	}

	duplicationRatio := float64(duplicates) / float64(len(lines)+1)
	result.Metrics["duplication_ratio"] = duplicationRatio
}

func (cqi *CodeQualityInspector) checkNaming(code string, result *InspectionResult) {
	// Check variable and function naming conventions
	camelCaseRegex := regexp.MustCompile(`[a-z][a-zA-Z0-9]*`)
	snakeCaseRegex := regexp.MustCompile(`[a-z]+(_[a-z]+)*`)
	
	goodNames := 0
	totalNames := 0
	
	// Simple heuristic: look for variable declarations
	varRegex := regexp.MustCompile(`(var|let|const)\s+(\w+)`)
	matches := varRegex.FindAllStringSubmatch(code, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			varName := match[2]
			totalNames++
			
			if camelCaseRegex.MatchString(varName) || snakeCaseRegex.MatchString(varName) {
				goodNames++
			} else if len(varName) < 3 {
				result.Findings = append(result.Findings, Finding{
					ID:          fmt.Sprintf("naming-%s", varName),
					Type:        SuggestionFinding,
					Severity:    LowSeverity,
					Description: fmt.Sprintf("Variable name '%s' is too short", varName),
					Impact:      "Short names reduce code clarity",
				})
			}
		}
	}

	if totalNames > 0 {
		result.Metrics["naming_quality"] = float64(goodNames) / float64(totalNames)
	}
}

func (cqi *CodeQualityInspector) checkStructure(code string, result *InspectionResult) {
	// Check file structure and organization
	lines := strings.Split(code, "\n")
	
	// Check line length
	longLines := 0
	for i, line := range lines {
		if len(line) > 120 {
			longLines++
			if longLines < 3 {
				result.Findings = append(result.Findings, Finding{
					ID:          fmt.Sprintf("long-line-%d", i),
					Type:        SuggestionFinding,
					Severity:    LowSeverity,
					Location:    Location{Line: i + 1},
					Description: fmt.Sprintf("Line exceeds 120 characters (%d)", len(line)),
					Impact:      "Long lines reduce readability",
				})
			}
		}
	}
	
	result.Metrics["long_line_ratio"] = float64(longLines) / float64(len(lines)+1)
}

func (cqi *CodeQualityInspector) checkDocumentation(code string, result *InspectionResult) {
	// Check for comments and documentation
	lines := strings.Split(code, "\n")
	commentLines := 0
	functionCount := 0
	documentedFunctions := 0
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Count comment lines
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || 
		   strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "\"\"\"") {
			commentLines++
		}
		
		// Check if functions are documented
		if strings.Contains(trimmed, "function ") || strings.Contains(trimmed, "func ") ||
		   strings.Contains(trimmed, "def ") {
			functionCount++
			// Check if previous line was a comment
			if i > 0 && strings.Contains(lines[i-1], "//") {
				documentedFunctions++
			}
		}
	}
	
	commentRatio := float64(commentLines) / float64(len(lines)+1)
	result.Metrics["comment_ratio"] = commentRatio
	
	if functionCount > 0 {
		result.Metrics["documented_function_ratio"] = float64(documentedFunctions) / float64(functionCount)
	}
	
	if commentRatio < 0.1 {
		result.Suggestions = append(result.Suggestions, ImprovementSuggestion{
			Target:     "Documentation",
			Action:     "Add comments to explain complex logic",
			Reason:     fmt.Sprintf("Comment ratio %.2f%% is below recommended 10%%", commentRatio*100),
			Complexity: "low",
		})
	}
}

func (cqi *CodeQualityInspector) checkErrorHandling(code string, result *InspectionResult) {
	// Check for proper error handling
	errorPatterns := []string{"try", "catch", "error", "err", "exception"}
	errorHandling := 0
	
	for _, pattern := range errorPatterns {
		errorHandling += strings.Count(strings.ToLower(code), pattern)
	}
	
	// Simple heuristic: should have some error handling
	if errorHandling == 0 {
		result.Findings = append(result.Findings, Finding{
			ID:          "no-error-handling",
			Type:        WarningFinding,
			Severity:    HighSeverity,
			Description: "No error handling detected in code",
			Impact:      "Unhandled errors can cause application crashes",
		})
		result.Metrics["error_handling_score"] = 0.0
	} else {
		result.Metrics["error_handling_score"] = 1.0
	}
}

func (cqi *CodeQualityInspector) checkSecurity(code string, result *InspectionResult) {
	// Check for common security issues
	securityPatterns := map[string]string{
		"eval(":           "Eval usage can lead to code injection",
		"innerHTML":       "Direct innerHTML assignment can lead to XSS",
		"password\":":     "Hardcoded passwords detected",
		"api_key\":":      "Hardcoded API keys detected",
		"SELECT * FROM":   "SQL queries should use parameterized queries",
		"system(":         "System calls can be dangerous",
		"exec(":           "Exec calls can lead to command injection",
	}
	
	securityScore := 1.0
	for pattern, issue := range securityPatterns {
		if strings.Contains(code, pattern) {
			securityScore -= 0.2
			result.Findings = append(result.Findings, Finding{
				ID:          fmt.Sprintf("security-%s", pattern),
				Type:        ErrorFinding,
				Severity:    HighSeverity,
				Description: issue,
				Pattern:     pattern,
				Impact:      "Security vulnerability",
			})
		}
	}
	
	result.Metrics["security_score"] = securityScore
}

func (cqi *CodeQualityInspector) checkPerformance(code string, result *InspectionResult) {
	// Check for performance issues
	performancePatterns := map[string]string{
		"SELECT \\*": "Avoid SELECT *, specify columns explicitly",
		"n\\+1":      "Potential N+1 query problem",
		"sleep\\(":   "Synchronous sleep blocks execution",
		"while\\(true": "Infinite loops can cause performance issues",
	}
	
	performanceScore := 1.0
	for pattern, issue := range performancePatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			performanceScore -= 0.1
			result.Findings = append(result.Findings, Finding{
				ID:          fmt.Sprintf("performance-%s", pattern),
				Type:        WarningFinding,
				Severity:    MediumSeverity,
				Description: issue,
				Pattern:     pattern,
				Impact:      "Performance degradation",
			})
		}
	}
	
	result.Metrics["performance_score"] = performanceScore
}

func (cqi *CodeQualityInspector) calculateScore(metrics map[string]float64) float64 {
	// Weighted average of all metrics
	weights := map[string]float64{
		"cyclomatic_complexity":       0.2,
		"duplication_ratio":           0.1,
		"naming_quality":              0.1,
		"comment_ratio":               0.15,
		"documented_function_ratio":   0.15,
		"error_handling_score":        0.15,
		"security_score":              0.1,
		"performance_score":           0.05,
	}
	
	totalScore := 0.0
	totalWeight := 0.0
	
	for metric, weight := range weights {
		if value, exists := metrics[metric]; exists {
			// Invert some metrics where lower is better
			if metric == "cyclomatic_complexity" || metric == "duplication_ratio" || 
			   metric == "long_line_ratio" {
				value = 1.0 - value
			}
			totalScore += value * weight
			totalWeight += weight
		}
	}
	
	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0.5
}

func (cqi *CodeQualityInspector) GenerateCriteria() []QualityCriteria {
	return []QualityCriteria{
		{
			ID:          "code-complexity",
			Name:        "Code Complexity",
			Description: "Code should have manageable complexity",
			Category:    "quality",
			Priority:    HighPriority,
			Validator:   cqi.validateComplexity,
		},
		{
			ID:          "code-duplication",
			Name:        "Code Duplication",
			Description: "Minimize code duplication",
			Category:    "quality",
			Priority:    MediumPriority,
			Validator:   cqi.validateDuplication,
		},
		{
			ID:          "error-handling",
			Name:        "Error Handling",
			Description: "Proper error handling throughout",
			Category:    "reliability",
			Priority:    CriticalPriority,
			Validator:   cqi.validateErrorHandling,
		},
		{
			ID:          "security",
			Name:        "Security Best Practices",
			Description: "No security vulnerabilities",
			Category:    "security",
			Priority:    CriticalPriority,
			Validator:   cqi.validateSecurity,
		},
	}
}

func (cqi *CodeQualityInspector) validateComplexity(ctx context.Context, content interface{}) (CriteriaResult, error) {
	inspection, err := cqi.Inspect(ctx, content)
	if err != nil {
		return CriteriaResult{}, err
	}
	
	complexity := inspection.Metrics["cyclomatic_complexity"]
	passed := complexity < 0.15
	
	suggestions := make([]ImprovementSuggestion, 0)
	if !passed {
		suggestions = append(suggestions, ImprovementSuggestion{
			Target: "Complex functions",
			Action: "Extract smaller functions from complex logic",
			Reason: "High cyclomatic complexity reduces maintainability",
			Example: "Break down if-else chains into separate handler functions",
			Complexity: "medium",
		})
	}
	
	return CriteriaResult{
		Passed:      passed,
		Score:       1.0 - complexity,
		Details:     fmt.Sprintf("Cyclomatic complexity: %.2f", complexity),
		Suggestions: suggestions,
	}, nil
}

func (cqi *CodeQualityInspector) validateDuplication(ctx context.Context, content interface{}) (CriteriaResult, error) {
	inspection, err := cqi.Inspect(ctx, content)
	if err != nil {
		return CriteriaResult{}, err
	}
	
	duplication := inspection.Metrics["duplication_ratio"]
	passed := duplication < 0.05
	
	return CriteriaResult{
		Passed:  passed,
		Score:   1.0 - duplication,
		Details: fmt.Sprintf("Code duplication: %.1f%%", duplication*100),
	}, nil
}

func (cqi *CodeQualityInspector) validateErrorHandling(ctx context.Context, content interface{}) (CriteriaResult, error) {
	inspection, err := cqi.Inspect(ctx, content)
	if err != nil {
		return CriteriaResult{}, err
	}
	
	score := inspection.Metrics["error_handling_score"]
	passed := score > 0.5
	
	return CriteriaResult{
		Passed:  passed,
		Score:   score,
		Details: "Error handling presence check",
	}, nil
}

func (cqi *CodeQualityInspector) validateSecurity(ctx context.Context, content interface{}) (CriteriaResult, error) {
	inspection, err := cqi.Inspect(ctx, content)
	if err != nil {
		return CriteriaResult{}, err
	}
	
	score := inspection.Metrics["security_score"]
	passed := score > 0.8
	
	return CriteriaResult{
		Passed:  passed,
		Score:   score,
		Details: "Security vulnerability scan",
	}, nil
}

func (cqi *CodeQualityInspector) CanInspect(content interface{}) bool {
	_, ok := content.(string)
	return ok
}

// InspectionCache caches inspection results
type InspectionCache struct {
	cache map[string]InspectionResult
	mu    sync.RWMutex
	ttl   time.Duration
}

func NewInspectionCache(ttl time.Duration) *InspectionCache {
	return &InspectionCache{
		cache: make(map[string]InspectionResult),
		ttl:   ttl,
	}
}

// NewInspectorAgent creates a new inspector agent
func NewInspectorAgent(agent Agent, logger *slog.Logger, config InspectorConfig) *InspectorAgent {
	ia := &InspectorAgent{
		agent:      agent,
		logger:     logger.With("component", "inspector_agent"),
		inspectors: make(map[string]Inspector),
		config:     config,
	}
	
	if config.CacheResults {
		ia.cache = NewInspectionCache(5 * time.Minute)
	}
	
	// Register built-in inspectors
	codeInspector := NewCodeQualityInspector(agent, logger)
	ia.RegisterInspector(codeInspector)
	
	return ia
}

// RegisterInspector adds a new inspector
func (ia *InspectorAgent) RegisterInspector(inspector Inspector) {
	ia.inspectors[inspector.Name()] = inspector
}

// InspectContent runs all applicable inspectors on content
func (ia *InspectorAgent) InspectContent(ctx context.Context, content interface{}) (map[string]InspectionResult, error) {
	results := make(map[string]InspectionResult)
	
	if ia.config.ParallelInspection {
		return ia.inspectParallel(ctx, content)
	}
	
	// Sequential inspection
	for name, inspector := range ia.inspectors {
		if !inspector.CanInspect(content) {
			continue
		}
		
		result, err := inspector.Inspect(ctx, content)
		if err != nil {
			ia.logger.Error("Inspection failed", "inspector", name, "error", err)
			continue
		}
		
		results[name] = result
	}
	
	return results, nil
}

// inspectParallel runs inspections concurrently
func (ia *InspectorAgent) inspectParallel(ctx context.Context, content interface{}) (map[string]InspectionResult, error) {
	results := make(map[string]InspectionResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for name, inspector := range ia.inspectors {
		if !inspector.CanInspect(content) {
			continue
		}
		
		wg.Add(1)
		go func(n string, i Inspector) {
			defer wg.Done()
			
			result, err := i.Inspect(ctx, content)
			if err != nil {
				ia.logger.Error("Parallel inspection failed", "inspector", n, "error", err)
				return
			}
			
			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(name, inspector)
	}
	
	wg.Wait()
	return results, nil
}

// GenerateAllCriteria collects criteria from all inspectors
func (ia *InspectorAgent) GenerateAllCriteria() []QualityCriteria {
	allCriteria := make([]QualityCriteria, 0)
	
	for _, inspector := range ia.inspectors {
		criteria := inspector.GenerateCriteria()
		allCriteria = append(allCriteria, criteria...)
	}
	
	return allCriteria
}

// AnalyzeWithAI performs AI-powered deep analysis
func (ia *InspectorAgent) AnalyzeWithAI(ctx context.Context, content interface{}, focus string) (InspectionResult, error) {
	prompt := fmt.Sprintf(`You are an expert code inspector performing deep analysis.

Content to analyze:
%v

Focus area: %s

Perform a thorough inspection and identify:
1. Quality issues with severity levels
2. Security vulnerabilities
3. Performance bottlenecks
4. Maintainability concerns
5. Best practice violations

Return findings in JSON format with:
- findings: array of issues found
- metrics: quality metrics as numbers
- suggestions: specific improvements
- evidence: supporting evidence for findings`,
		content, focus)
	
	response, err := ia.agent.ExecuteJSON(ctx, prompt, nil)
	if err != nil {
		return InspectionResult{}, fmt.Errorf("AI analysis failed: %w", err)
	}
	
	var result InspectionResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return InspectionResult{}, fmt.Errorf("failed to parse AI analysis: %w", err)
	}
	
	result.InspectorName = "AI-Deep-Analysis"
	result.Category = focus
	result.Timestamp = time.Now()
	
	return result, nil
}