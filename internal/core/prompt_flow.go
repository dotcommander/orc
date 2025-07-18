package core

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"
)

// PromptFlow represents a dynamic, composable prompt system
type PromptFlow struct {
	templates  map[string]*FlowTemplate
	fragments  map[string]string
	functions  template.FuncMap
	optimizer  *PromptOptimizer
	mu         sync.RWMutex
}

// FlowTemplate is a flexible, composable template
type FlowTemplate struct {
	Name        string
	Content     string
	Fragments   []string
	Variables   map[string]interface{}
	Conditions  []TemplateCondition
	Variations  []TemplateVariation
	Performance TemplateMetrics
}

// TemplateCondition determines template selection
type TemplateCondition func(ctx context.Context, data interface{}) bool

// TemplateVariation provides alternative versions
type TemplateVariation struct {
	Name      string
	Condition TemplateCondition
	Content   string
	Weight    float64
}

// TemplateMetrics tracks template performance
type TemplateMetrics struct {
	UsageCount      int
	SuccessRate     float64
	AverageTokens   int
	ResponseQuality float64
	LastUsed        time.Time
}

// PromptOptimizer learns optimal prompt patterns
type PromptOptimizer struct {
	history    []PromptExecution
	patterns   map[string]*PromptPattern
	learning   bool
	mu         sync.RWMutex
}

// PromptExecution records prompt execution data
type PromptExecution struct {
	Template    string
	Input       interface{}
	Output      string
	Success     bool
	TokensUsed  int
	Duration    time.Duration
	Quality     float64
	Timestamp   time.Time
}

// PromptPattern represents learned prompt patterns
type PromptPattern struct {
	Pattern         string
	SuccessRate     float64
	OptimalLength   int
	BestFragments   []string
	ContextFactors  map[string]float64
}

// NewPromptFlow creates a flexible prompt system
func NewPromptFlow() *PromptFlow {
	pf := &PromptFlow{
		templates:  make(map[string]*FlowTemplate),
		fragments:  make(map[string]string),
		functions:  createDefaultFunctions(),
		optimizer:  NewPromptOptimizer(),
	}

	// Register default fragments
	pf.registerDefaultFragments()

	return pf
}

// RegisterTemplate adds a new template with dynamic composition
func (pf *PromptFlow) RegisterTemplate(name string, content string, opts ...TemplateOption) error {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	tmpl := &FlowTemplate{
		Name:       name,
		Content:    content,
		Fragments:  make([]string, 0),
		Variables:  make(map[string]interface{}),
		Variations: make([]TemplateVariation, 0),
		Performance: TemplateMetrics{
			UsageCount: 0,
			SuccessRate: 0.0,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(tmpl)
	}

	pf.templates[name] = tmpl
	return nil
}

// TemplateOption configures template behavior
type TemplateOption func(*FlowTemplate)

// WithFragments includes reusable fragments
func WithFragments(fragments ...string) TemplateOption {
	return func(t *FlowTemplate) {
		t.Fragments = append(t.Fragments, fragments...)
	}
}

// WithVariation adds template variations
func WithVariation(name string, content string, condition TemplateCondition) TemplateOption {
	return func(t *FlowTemplate) {
		t.Variations = append(t.Variations, TemplateVariation{
			Name:      name,
			Content:   content,
			Condition: condition,
			Weight:    1.0,
		})
	}
}

// WithVariables sets default variables
func WithVariables(vars map[string]interface{}) TemplateOption {
	return func(t *FlowTemplate) {
		for k, v := range vars {
			t.Variables[k] = v
		}
	}
}

// Generate creates a prompt using dynamic composition
func (pf *PromptFlow) Generate(ctx context.Context, templateName string, data interface{}) (string, error) {
	pf.mu.RLock()
	tmpl, exists := pf.templates[templateName]
	pf.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	// Select best variation based on context
	selectedContent := pf.selectBestVariation(ctx, tmpl, data)

	// Compose with fragments
	composed := pf.composeWithFragments(selectedContent, tmpl.Fragments)

	// Parse and execute template
	t, err := template.New(templateName).
		Funcs(pf.functions).
		Parse(composed)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	// Merge data with template variables
	mergedData := pf.mergeData(data, tmpl.Variables)

	// Execute template
	var buf bytes.Buffer
	if err := t.Execute(&buf, mergedData); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	result := buf.String()

	// Optimize if learning is enabled
	if pf.optimizer.learning {
		result = pf.optimizer.Optimize(result, data)
	}

	// Track execution
	pf.trackExecution(templateName, data, result)

	return result, nil
}

// selectBestVariation chooses optimal template variation
func (pf *PromptFlow) selectBestVariation(ctx context.Context, tmpl *FlowTemplate, data interface{}) string {
	// Check conditions for variations
	for _, variation := range tmpl.Variations {
		if variation.Condition != nil && variation.Condition(ctx, data) {
			// Weight by performance
			if variation.Weight > 0.8 {
				return variation.Content
			}
		}
	}

	// Use optimizer suggestions if available
	if suggestion := pf.optimizer.SuggestVariation(tmpl.Name, data); suggestion != "" {
		return suggestion
	}

	// Default to base template
	return tmpl.Content
}

// composeWithFragments builds complete prompt from fragments
func (pf *PromptFlow) composeWithFragments(base string, fragmentNames []string) string {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	composed := base

	for _, name := range fragmentNames {
		if fragment, exists := pf.fragments[name]; exists {
			// Replace fragment placeholder
			placeholder := fmt.Sprintf("{{fragment:%s}}", name)
			composed = strings.Replace(composed, placeholder, fragment, -1)
		}
	}

	return composed
}

// RegisterFragment adds a reusable prompt fragment
func (pf *PromptFlow) RegisterFragment(name, content string) {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	
	pf.fragments[name] = content
}

// registerDefaultFragments sets up common fragments
func (pf *PromptFlow) registerDefaultFragments() {
	// Role fragments
	pf.RegisterFragment("expert_developer", "You are an expert software developer with deep knowledge of best practices, design patterns, and clean code principles.")
	pf.RegisterFragment("code_reviewer", "You are a thorough code reviewer focused on security, performance, and maintainability.")
	pf.RegisterFragment("architect", "You are a senior software architect who designs scalable, maintainable systems.")

	// Instruction fragments
	pf.RegisterFragment("think_step_by_step", "Think through this step-by-step, considering all implications and edge cases.")
	pf.RegisterFragment("explain_reasoning", "Explain your reasoning clearly for each decision.")
	pf.RegisterFragment("consider_tradeoffs", "Consider the tradeoffs between different approaches.")

	// Output format fragments
	pf.RegisterFragment("json_output", "Return your response in valid JSON format with the following structure:")
	pf.RegisterFragment("markdown_output", "Format your response using clear Markdown with appropriate headers and code blocks.")
	
	// Quality fragments
	pf.RegisterFragment("production_quality", "Ensure all code is production-ready with proper error handling, logging, and documentation.")
	pf.RegisterFragment("security_focus", "Pay special attention to security concerns including input validation, authentication, and data protection.")
}

// Chain creates a multi-step prompt flow
func (pf *PromptFlow) Chain(steps ...PromptStep) *PromptChain {
	return &PromptChain{
		steps: steps,
		flow:  pf,
	}
}

// PromptStep represents a step in a prompt chain
type PromptStep struct {
	Template  string
	Transform func(interface{}) interface{}
	Condition func(interface{}) bool
}

// PromptChain represents a sequence of prompts
type PromptChain struct {
	steps []PromptStep
	flow  *PromptFlow
}

// Execute runs the prompt chain
func (pc *PromptChain) Execute(ctx context.Context, initialData interface{}) ([]string, error) {
	results := make([]string, 0)
	data := initialData

	for _, step := range pc.steps {
		// Check condition
		if step.Condition != nil && !step.Condition(data) {
			continue
		}

		// Transform data if needed
		if step.Transform != nil {
			data = step.Transform(data)
		}

		// Generate prompt
		result, err := pc.flow.Generate(ctx, step.Template, data)
		if err != nil {
			return results, fmt.Errorf("chain step %s failed: %w", step.Template, err)
		}

		results = append(results, result)
		
		// Use result as input for next step
		data = result
	}

	return results, nil
}

// PromptOptimizer implementation

func NewPromptOptimizer() *PromptOptimizer {
	return &PromptOptimizer{
		history:  make([]PromptExecution, 0, 1000),
		patterns: make(map[string]*PromptPattern),
		learning: true,
	}
}

// Optimize improves prompt based on learned patterns
func (po *PromptOptimizer) Optimize(prompt string, data interface{}) string {
	po.mu.RLock()
	defer po.mu.RUnlock()

	// Find applicable patterns
	for _, pattern := range po.patterns {
		if pattern.SuccessRate > 0.8 {
			// Apply successful patterns
			prompt = po.applyPattern(prompt, pattern)
		}
	}

	// Optimize length based on learning
	if avgLength := po.getOptimalLength(data); avgLength > 0 {
		prompt = po.optimizeLength(prompt, avgLength)
	}

	return prompt
}

// SuggestVariation suggests best template variation
func (po *PromptOptimizer) SuggestVariation(templateName string, data interface{}) string {
	po.mu.RLock()
	defer po.mu.RUnlock()

	// Analyze historical performance
	bestPerformance := 0.0
	bestVariation := ""

	for _, execution := range po.history {
		if execution.Template == templateName && execution.Success {
			if execution.Quality > bestPerformance {
				bestPerformance = execution.Quality
				bestVariation = execution.Output
			}
		}
	}

	return bestVariation
}

// RecordExecution tracks prompt execution for learning
func (po *PromptOptimizer) RecordExecution(execution PromptExecution) {
	po.mu.Lock()
	defer po.mu.Unlock()

	po.history = append(po.history, execution)

	// Maintain history size
	if len(po.history) > 10000 {
		po.history = po.history[5000:]
	}

	// Update patterns
	po.updatePatterns(execution)
}

// updatePatterns learns from successful executions
func (po *PromptOptimizer) updatePatterns(execution PromptExecution) {
	if !execution.Success {
		return
	}

	// Extract patterns from successful prompts
	pattern := &PromptPattern{
		Pattern:       extractPattern(execution.Output),
		SuccessRate:   execution.Quality,
		OptimalLength: len(execution.Output),
		BestFragments: extractFragments(execution.Output),
	}

	po.patterns[pattern.Pattern] = pattern
}

// Helper functions

func createDefaultFunctions() template.FuncMap {
	return template.FuncMap{
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"join":      strings.Join,
		"split":     strings.Split,
		"contains":  strings.Contains,
		"replace":   strings.Replace,
		"now":       time.Now,
		"date":      formatDate,
		"json":      toJSON,
		"indent":    indent,
		"wrap":      wordWrap,
		"limit":     limitLength,
	}
}

func (pf *PromptFlow) mergeData(data interface{}, defaults map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Add defaults
	for k, v := range defaults {
		merged[k] = v
	}

	// Override with provided data
	if m, ok := data.(map[string]interface{}); ok {
		for k, v := range m {
			merged[k] = v
		}
	} else {
		merged["data"] = data
	}

	return merged
}

func (pf *PromptFlow) trackExecution(template string, data interface{}, result string) {
	execution := PromptExecution{
		Template:   template,
		Input:      data,
		Output:     result,
		Success:    true, // Would be determined by response
		TokensUsed: len(strings.Fields(result)), // Simplified
		Timestamp:  time.Now(),
	}

	pf.optimizer.RecordExecution(execution)

	// Update template metrics
	pf.mu.Lock()
	if tmpl, exists := pf.templates[template]; exists {
		tmpl.Performance.UsageCount++
		tmpl.Performance.LastUsed = time.Now()
		tmpl.Performance.AverageTokens = (tmpl.Performance.AverageTokens + execution.TokensUsed) / 2
	}
	pf.mu.Unlock()
}

func (po *PromptOptimizer) applyPattern(prompt string, pattern *PromptPattern) string {
	// Apply learned improvements
	return prompt
}

func (po *PromptOptimizer) getOptimalLength(data interface{}) int {
	// Determine optimal length based on data type
	return 0
}

func (po *PromptOptimizer) optimizeLength(prompt string, targetLength int) string {
	// Optimize prompt length while preserving meaning
	return prompt
}

func extractPattern(prompt string) string {
	// Extract reusable pattern from prompt
	return ""
}

func extractFragments(prompt string) []string {
	// Extract reusable fragments
	return []string{}
}

// Template functions
func formatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func toJSON(v interface{}) string {
	// Convert to JSON
	return ""
}

func indent(s string, n int) string {
	// Indent string
	return s
}

func wordWrap(s string, width int) string {
	// Word wrap string
	return s
}

func limitLength(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}